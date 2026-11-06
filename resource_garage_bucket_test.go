package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func websiteBlock(enabled bool, index, errDoc string) []interface{} {
	return []interface{}{
		map[string]interface{}{
			"enabled":        enabled,
			"index_document": index,
			"error_document": errDoc,
		},
	}
}

func TestBuildWebsiteAccess(t *testing.T) {
	t.Run("absent block disables access", func(t *testing.T) {
		wa, err := buildWebsiteAccess(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if wa.GetEnabled() {
			t.Error("expected enabled=false")
		}
		if wa.HasIndexDocument() || wa.HasErrorDocument() {
			t.Error("expected no documents when block is absent")
		}
	})

	t.Run("empty list disables access", func(t *testing.T) {
		wa, err := buildWebsiteAccess([]interface{}{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if wa.GetEnabled() {
			t.Error("expected enabled=false")
		}
	})

	t.Run("enabled without index_document errors", func(t *testing.T) {
		_, err := buildWebsiteAccess(websiteBlock(true, "", ""))
		if err == nil {
			t.Fatal("expected error when enabled with no index_document")
		}
	})

	t.Run("enabled with index and error documents", func(t *testing.T) {
		wa, err := buildWebsiteAccess(websiteBlock(true, "index.html", "error.html"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !wa.GetEnabled() {
			t.Error("expected enabled=true")
		}
		if wa.GetIndexDocument() != "index.html" {
			t.Errorf("index_document = %q, want index.html", wa.GetIndexDocument())
		}
		if wa.GetErrorDocument() != "error.html" {
			t.Errorf("error_document = %q, want error.html", wa.GetErrorDocument())
		}
	})

	t.Run("enabled with index only", func(t *testing.T) {
		wa, err := buildWebsiteAccess(websiteBlock(true, "index.html", ""))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if wa.HasErrorDocument() {
			t.Error("expected no error_document when not set")
		}
	})

	t.Run("disabled drops leftover documents", func(t *testing.T) {
		wa, err := buildWebsiteAccess(websiteBlock(false, "index.html", "error.html"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if wa.GetEnabled() {
			t.Error("expected enabled=false")
		}
		if wa.HasIndexDocument() || wa.HasErrorDocument() {
			t.Error("expected documents dropped when disabled (Garage rejects them)")
		}
	})
}

func TestSelectGlobalAlias(t *testing.T) {
	cases := []struct {
		name       string
		aliases    []string
		configured string
		want       string
	}{
		{"no alias", nil, "", ""},
		{"no alias, configured", nil, "a", ""},
		{"configured alias kept", []string{"a", "b"}, "b", "b"},
		{"configured alias missing", []string{"a", "b"}, "c", ""},
		{"nothing configured", []string{"a", "b"}, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := selectGlobalAlias(tc.aliases, tc.configured); got != tc.want {
				t.Errorf("selectGlobalAlias(%q, %q) = %q, want %q", tc.aliases, tc.configured, got, tc.want)
			}
		})
	}
}

func TestBuildBucketQuotas(t *testing.T) {
	t.Run("zero sends null quotas", func(t *testing.T) {
		quotas := buildBucketQuotas(0, 0)
		body, err := quotas.MarshalJSON()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got, want := string(body), `{"maxObjects":null,"maxSize":null}`; got != want {
			t.Errorf("got %s, want %s", got, want)
		}
	})

	t.Run("positive values are sent", func(t *testing.T) {
		quotas := buildBucketQuotas(1024, 10)
		body, err := quotas.MarshalJSON()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got, want := string(body), `{"maxObjects":10,"maxSize":1024}`; got != want {
			t.Errorf("got %s, want %s", got, want)
		}
	})

	t.Run("one zero sends one null quota", func(t *testing.T) {
		quotas := buildBucketQuotas(1024, 0)
		body, err := quotas.MarshalJSON()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got, want := string(body), `{"maxObjects":null,"maxSize":1024}`; got != want {
			t.Errorf("got %s, want %s", got, want)
		}
	})
}

// TestSetBucketQuotasRequestBody checks the JSON that goes to Garage. The
// generated SDK adds the outer object, and buildBucketQuotas cannot see it.
func TestSetBucketQuotasRequestBody(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/UpdateBucket" || r.URL.Query().Get("id") != "bucket-id" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		got = strings.TrimSpace(string(body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "bucket-id", "created": "2026-01-01T00:00:00Z", "globalAliases": [],
			"websiteAccess": false, "keys": [], "objects": 0, "bytes": 0,
			"unfinishedUploads": 0, "unfinishedMultipartUploads": 0,
			"unfinishedMultipartUploadParts": 0, "unfinishedMultipartUploadBytes": 0,
			"quotas": {"maxSize": 1024, "maxObjects": null}
		}`))
	}))
	defer srv.Close()

	client, err := NewGarageClient("http", strings.TrimPrefix(srv.URL, "http://"), "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := setBucketQuotas(context.Background(), client, "bucket-id", 1024, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := `{"quotas":{"maxObjects":null,"maxSize":1024}}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

// TestBucketUpdateQuotaFailure applies a quota change through the SDK, as
// Terraform does, against an API that refuses it. The state must keep the
// prior quotas, and the error must give the reason from Garage.
func TestBucketUpdateQuotaFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/UpdateBucket" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"code":"InternalError","message":"quorum not reached"}`))
	}))
	defer srv.Close()

	client, err := NewGarageClient("http", strings.TrimPrefix(srv.URL, "http://"), "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	state := &terraform.InstanceState{
		ID: "bucket-id",
		Attributes: map[string]string{
			"id":          "bucket-id",
			"max_size":    "1024",
			"max_objects": "10",
		},
	}
	diff := &terraform.InstanceDiff{
		Attributes: map[string]*terraform.ResourceAttrDiff{
			"max_size":    {Old: "1024", New: "2048"},
			"max_objects": {Old: "10", New: "20"},
		},
	}

	got, diags := resourceGarageBucket().Apply(context.Background(), state, diff, client)
	if !diags.HasError() {
		t.Fatal("expected an error, got none")
	}
	if msg := diags[0].Summary; !strings.Contains(msg, "quorum not reached") {
		t.Errorf("error %q does not give the reason from Garage", msg)
	}
	if got == nil {
		t.Fatal("expected a state, got nil")
	}
	if v := got.Attributes["max_size"]; v != "1024" {
		t.Errorf("max_size in state = %q, want 1024", v)
	}
	if v := got.Attributes["max_objects"]; v != "10" {
		t.Errorf("max_objects in state = %q, want 10", v)
	}
}
