package main

import (
	"fmt"
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
