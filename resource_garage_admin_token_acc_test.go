package main

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccGarageAdminToken_expiration(t *testing.T) {
	name := acctest.RandomWithPrefix("acck-admin-token")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckGarageAdminTokenDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testAccGarageAdminTokenConfig(name, `never_expires = true`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_admin_token.test", "name", name),
					resource.TestCheckResourceAttrSet("garage_admin_token.test", "secret_token"),
					resource.TestCheckResourceAttr("garage_admin_token.test", "expiration", ""),
					testAccCheckGarageAdminTokenExpiration(t, nil),
				),
			},
			{
				Config: testAccGarageAdminTokenConfig(name, `expiration = "2099-01-01T00:00:00Z"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_admin_token.test", "expiration", "2099-01-01T00:00:00Z"),
					testAccCheckGarageAdminTokenExpiration(t, &testAccAdminTokenExpiration),
				),
			},
			{
				Config: testAccGarageAdminTokenConfig(name, `never_expires = true`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_admin_token.test", "expiration", ""),
					testAccCheckGarageAdminTokenExpiration(t, nil),
				),
			},
			{
				Config:      testAccGarageAdminTokenConfig(name, "expiration = \"2099-01-01T00:00:00Z\"\n  never_expires = true"),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`expiration and never_expires conflict`),
			},
		},
	})
}

var testAccAdminTokenExpiration = time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)

func testAccGarageAdminTokenConfig(name, expiry string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "garage_admin_token" "test" {
  name  = %q
  scope = ["GetClusterStatus", "GetClusterHealth"]
  %s
}
`, name, expiry)
}

// testAccCheckGarageAdminTokenExpiration checks the expiration of the token in
// Garage itself. A nil want means that the token never expires.
func testAccCheckGarageAdminTokenExpiration(t *testing.T, want *time.Time) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["garage_admin_token.test"]
		if !ok {
			return fmt.Errorf("garage_admin_token.test not found in state")
		}
		client := testAccClient(t)
		token, resp, err := client.Client.AdminAPITokenAPI.GetAdminTokenInfo(client.WithAuth(context.Background())).Id(rs.Primary.ID).Execute()
		if err != nil {
			return fmt.Errorf("failed to read admin token %s: %w", rs.Primary.ID, err)
		}
		defer func() {
			if resp.Body != nil {
				_ = resp.Body.Close()
			}
		}()

		got, _ := token.GetExpirationOk()
		switch {
		case want == nil && got != nil:
			return fmt.Errorf("token expires at %s, want no expiration", got)
		case want != nil && (got == nil || !got.Equal(*want)):
			return fmt.Errorf("token expires at %v, want %s", got, want)
		}
		return nil
	}
}

func testAccCheckGarageAdminTokenDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccClient(t)
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "garage_admin_token" {
				continue
			}
			_, resp, err := client.Client.AdminAPITokenAPI.GetAdminTokenInfo(client.WithAuth(context.Background())).Id(rs.Primary.ID).Execute()
			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}
			gone, probeErr := classifyDeletionProbe(resp, err)
			if probeErr != nil {
				return fmt.Errorf("admin token %s: %w", rs.Primary.ID, probeErr)
			}
			if !gone {
				return fmt.Errorf("admin token %s still exists", rs.Primary.ID)
			}
		}
		return nil
	}
}

func TestNormalizeRFC3339(t *testing.T) {
	tests := map[string]string{
		"":                          "",
		"2099-01-01T00:00:00Z":      "2099-01-01T00:00:00Z",
		"2099-01-01T03:00:00+03:00": "2099-01-01T00:00:00Z",
		"2099-01-01T00:00:00.500Z":  "2099-01-01T00:00:00Z",
		"not-a-time":                "not-a-time",
	}
	for in, want := range tests {
		if got := normalizeRFC3339(in); got != want {
			t.Errorf("normalizeRFC3339(%q) = %q, want %q", in, got, want)
		}
	}
}
