package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const testBucketInfo = `{
	"id": "bucket-id", "created": "2026-01-01T00:00:00Z", "globalAliases": ["my-bucket"],
	"websiteAccess": false, "keys": [], "objects": 0, "bytes": 0,
	"unfinishedUploads": 0, "unfinishedMultipartUploads": 0,
	"unfinishedMultipartUploadParts": 0, "unfinishedMultipartUploadBytes": 0,
	"quotas": {"maxSize": null, "maxObjects": null}
}`

func testImportClient(t *testing.T, srv *httptest.Server) *GarageClient {
	t.Helper()
	client, err := NewGarageClient("http", strings.TrimPrefix(srv.URL, "http://"), "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return client
}

func testImport(r *schema.Resource, client *GarageClient, id string) error {
	d := r.TestResourceData()
	d.SetId(id)
	_, err := r.Importer.StateContext(context.Background(), d, client)
	return err
}

var testImporters = []struct {
	name     string
	resource *schema.Resource
	id       string
	// bucketFound lets the first request of the bucket key import pass, so
	// that the second request fails.
	bucketFound bool
}{
	{name: "bucket", resource: resourceGarageBucket(), id: "my-bucket"},
	{name: "bucket key, bucket request", resource: resourceGarageBucketKey(), id: "my-bucket/GK1"},
	{name: "bucket key, key request", resource: resourceGarageBucketKey(), id: "my-bucket/GK1", bucketFound: true},
	{name: "key", resource: resourceGarageKey(), id: "GK1"},
}

// A status code alone does not tell the user why Garage refused the request.
func TestImportErrorHasGarageReason(t *testing.T) {
	for _, tc := range testImporters {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.bucketFound && r.URL.Path == "/v2/GetBucketInfo" {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(testBucketInfo))
					return
				}
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"code":"AccessDenied","message":"token scope is too small"}`))
			}))
			defer srv.Close()

			err := testImport(tc.resource, testImportClient(t, srv), tc.id)
			if err == nil || !strings.Contains(err.Error(), "token scope is too small") {
				t.Errorf("got error %v, want the reason from Garage", err)
			}
		})
	}
}

// With no response from Garage, the SDK returns no response object.
func TestImportServerUnavailable(t *testing.T) {
	for _, tc := range testImporters {
		if tc.bucketFound {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.NotFoundHandler())
			client := testImportClient(t, srv)
			srv.Close()

			if err := testImport(tc.resource, client, tc.id); err == nil {
				t.Error("got no error")
			}
		})
	}
}

func TestKeyImportSecretMissing(t *testing.T) {
	for name, secret := range map[string]string{
		"absent": ``,
		"null":   `"secretAccessKey": null,`,
		"empty":  `"secretAccessKey": "",`,
	} {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{` + secret + `"accessKeyId": "GK1", "buckets": [], "expired": false, "name": "k", "permissions": {}}`))
			}))
			defer srv.Close()

			err := testImport(resourceGarageKey(), testImportClient(t, srv), "GK1")
			if err == nil || !strings.Contains(err.Error(), "response has no secret access key") {
				t.Errorf("got error %v, want a missing secret error", err)
			}
		})
	}
}

// The read keeps only the alias that is in the state, so the importer must
// write the alias that the import ID selects. A bucket ID must not become an
// alias.
func TestBucketImportGlobalAlias(t *testing.T) {
	bucketID := strings.Repeat("ab", 32)
	tests := []struct {
		name    string
		id      string
		aliases string
		want    string
	}{
		{name: "alias", id: "second", aliases: `["first", "second"]`, want: "second"},
		{name: "bucket ID", id: bucketID, aliases: `["first", "second"]`, want: "first"},
		{name: "bucket ID, no alias", id: bucketID, aliases: `[]`, want: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := strings.ReplaceAll(testBucketInfo, `"bucket-id"`, `"`+bucketID+`"`)
			body = strings.ReplaceAll(body, `["my-bucket"]`, tc.aliases)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(body))
			}))
			defer srv.Close()

			r := resourceGarageBucket()
			d := r.TestResourceData()
			d.SetId(tc.id)
			if _, err := r.Importer.StateContext(context.Background(), d, testImportClient(t, srv)); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if d.Id() != bucketID {
				t.Errorf("ID = %q, want %q", d.Id(), bucketID)
			}
			if got := d.Get("global_alias").(string); got != tc.want {
				t.Errorf("global_alias = %q, want %q", got, tc.want)
			}
		})
	}
}
