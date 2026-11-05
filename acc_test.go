package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// The acceptance tests run against a live Garage v2 server. Start one with
// `scripts/acc-garage.sh`, then run `make testacc`.

var testAccProtoV5ProviderFactories = map[string]func() (tfprotov5.ProviderServer, error){
	"garage": func() (tfprotov5.ProviderServer, error) {
		return schema.NewGRPCProviderServer(Provider()), nil
	},
}

func testAccEnv(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("GARAGE_TOKEN") == "" {
		t.Fatal("GARAGE_TOKEN must be set for acceptance tests")
	}
}

// testAccProviderConfig returns the provider block for the test server.
func testAccProviderConfig() string {
	return fmt.Sprintf(`
provider "garage" {
  scheme = %q
  host   = %q
  token  = %q
}
`, testAccEnv("GARAGE_SCHEME", "http"), testAccEnv("GARAGE_HOST", "127.0.0.1:3903"), os.Getenv("GARAGE_TOKEN"))
}

// testAccClient returns an API client for the test server, to examine the
// state of Garage directly.
func testAccClient(t *testing.T) *GarageClient {
	t.Helper()
	client, err := NewGarageClient(testAccEnv("GARAGE_SCHEME", "http"), testAccEnv("GARAGE_HOST", "127.0.0.1:3903"), os.Getenv("GARAGE_TOKEN"))
	if err != nil {
		t.Fatal(err)
	}
	return client
}

// classifyDeletionProbe interprets an admin API lookup issued after the
// resource was supposedly destroyed. A successful call means the resource is
// still there. A 404 is the only acceptable failure and means it is gone. A
// transport failure or any other status must fail the test.
func classifyDeletionProbe(resp *http.Response, apiErr error) (gone bool, unexpected error) {
	if apiErr == nil {
		return false, nil
	}
	if resp == nil {
		return false, fmt.Errorf("unexpected error: %w", apiErr)
	}
	if resp.StatusCode != http.StatusNotFound {
		return false, fmt.Errorf("unexpected status %d: %w", resp.StatusCode, apiErr)
	}
	return true, nil
}

func TestClassifyDeletionProbe(t *testing.T) {
	tests := []struct {
		name     string
		resp     *http.Response
		apiErr   error
		wantGone bool
		wantErr  bool
	}{
		{
			name:   "successful lookup means the resource still exists",
			resp:   &http.Response{StatusCode: http.StatusOK},
			apiErr: nil,
		},
		{
			name:     "404 means the resource is gone",
			resp:     &http.Response{StatusCode: http.StatusNotFound},
			apiErr:   errors.New("404 Not Found"),
			wantGone: true,
		},
		{
			name:    "500 is an unexpected error",
			resp:    &http.Response{StatusCode: http.StatusInternalServerError},
			apiErr:  errors.New("500 Internal Server Error"),
			wantErr: true,
		},
		{
			name:    "403 is an unexpected error",
			resp:    &http.Response{StatusCode: http.StatusForbidden},
			apiErr:  errors.New("403 Forbidden"),
			wantErr: true,
		},
		{
			name:    "missing response is an unexpected error",
			resp:    nil,
			apiErr:  errors.New("connection refused"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gone, err := classifyDeletionProbe(tt.resp, tt.apiErr)
			if tt.wantErr && err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gone != tt.wantGone {
				t.Fatalf("gone = %v, want %v", gone, tt.wantGone)
			}
		})
	}
}
