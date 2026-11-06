package main

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccGarageKey_basic(t *testing.T) {
	name := acctest.RandomWithPrefix("acc-key")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckGarageKeyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testAccGarageKeyConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_key.test", "name", name),
					resource.TestCheckResourceAttrSet("garage_key.test", "access_key_id"),
					resource.TestCheckResourceAttrSet("garage_key.test", "secret_access_key"),
					resource.TestCheckResourceAttrPair("garage_key.test", "id", "garage_key.test", "access_key_id"),
				),
			},
			{
				Config: testAccGarageKeyConfig(name + "-renamed"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_key.test", "name", name+"-renamed"),
				),
			},
		},
	})
}

func TestAccGarageKey_importCredentials(t *testing.T) {
	// Garage keeps a deleted key as a tombstone and refuses to import its ID
	// again, so each run needs a new ID.
	accessKeyID := "GK" + acctest.RandStringFromCharSet(24, "0123456789abcdef")
	secretAccessKey := acctest.RandStringFromCharSet(64, "0123456789abcdef")
	name := acctest.RandomWithPrefix("acc-key-import")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckGarageKeyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testAccGarageKeyImportConfig(name, accessKeyID, secretAccessKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_key.test", "id", accessKeyID),
					resource.TestCheckResourceAttr("garage_key.test", "access_key_id", accessKeyID),
					resource.TestCheckResourceAttr("garage_key.test", "secret_access_key", secretAccessKey),
					testAccCheckGarageKeyInGarage(t, accessKeyID, secretAccessKey, name),
				),
			},
			{
				Config:   testAccGarageKeyImportConfig(name, accessKeyID, secretAccessKey),
				PlanOnly: true,
			},
			{
				Config: testAccGarageKeyImportConfig(name+"-renamed", accessKeyID, secretAccessKey),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("garage_key.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_key.test", "name", name+"-renamed"),
					testAccCheckGarageKeyInGarage(t, accessKeyID, secretAccessKey, name+"-renamed"),
				),
			},
			{
				Config:      testAccGarageKeyImportConfig(name+"-renamed", accessKeyID, acctest.RandStringFromCharSet(64, "0123456789abcdef")),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`cannot change secret_access_key`),
			},
			// Without the values in the configuration, the key stays as it is.
			{
				Config: testAccGarageKeyConfig(name + "-renamed"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("garage_key.test", plancheck.ResourceActionNoop),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_key.test", "access_key_id", accessKeyID),
					resource.TestCheckResourceAttr("garage_key.test", "secret_access_key", secretAccessKey),
					testAccCheckGarageKeyInGarage(t, accessKeyID, secretAccessKey, name+"-renamed"),
				),
			},
		},
	})
}

func TestAccGarageKey_changeAccessKeyID(t *testing.T) {
	name := acctest.RandomWithPrefix("acc-key-change")
	secretAccessKey := acctest.RandStringFromCharSet(64, "0123456789abcdef")
	newSecretAccessKey := acctest.RandStringFromCharSet(64, "0123456789abcdef")
	firstID := "GK" + acctest.RandStringFromCharSet(24, "0123456789abcdef")
	secondID := "GK" + acctest.RandStringFromCharSet(24, "0123456789abcdef")
	thirdID := "GK" + acctest.RandStringFromCharSet(24, "0123456789abcdef")

	expectReplace := resource.ConfigPlanChecks{
		PreApply: []plancheck.PlanCheck{
			plancheck.ExpectResourceAction("garage_key.test", plancheck.ResourceActionReplace),
		},
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckGarageKeyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testAccGarageKeyImportConfig(name, firstID, secretAccessKey),
				Check:  testAccCheckGarageKeyInGarage(t, firstID, secretAccessKey, name),
			},
			{
				Config:           testAccGarageKeyImportConfig(name, secondID, secretAccessKey),
				ConfigPlanChecks: expectReplace,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_key.test", "id", secondID),
					testAccCheckGarageKeyInGarage(t, secondID, secretAccessKey, name),
					testAccCheckGarageKeyAbsent(t, firstID),
				),
			},
			{
				Config:           testAccGarageKeyImportConfig(name, thirdID, newSecretAccessKey),
				ConfigPlanChecks: expectReplace,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckGarageKeyInGarage(t, thirdID, newSecretAccessKey, name),
					testAccCheckGarageKeyAbsent(t, secondID),
				),
			},
			// The tombstone of the first key is not visible at plan time.
			{
				Config:      testAccGarageKeyImportConfig(name, firstID, secretAccessKey),
				ExpectError: regexp.MustCompile(`failed to import key`),
			},
		},
	})
}

// TestAccGarageKey_importCredentialsUnknownAtPlan checks values that pass
// validation because they are unknown at plan time. Such a key must not get
// generated credentials.
func TestAccGarageKey_importCredentialsUnknownAtPlan(t *testing.T) {
	validID := "GK" + acctest.RandStringFromCharSet(24, "0123456789abcdef")
	validSecret := acctest.RandStringFromCharSet(64, "0123456789abcdef")

	config := func(input, accessKeyID, secretAccessKey string) string {
		return testAccProviderConfig() + fmt.Sprintf(`
resource "terraform_data" "value" {
  input = %s
}

resource "garage_key" "test" {
  name              = "acc-key-unknown"
  access_key_id     = %s
  secret_access_key = %s
}
`, input, accessKeyID, secretAccessKey)
	}
	const value = "terraform_data.value.output"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckGarageKeyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config:      config(`""`, value, strconv.Quote(validSecret)),
				ExpectError: regexp.MustCompile(`must be at least 8 characters`),
			},
			{
				Config:      config(`""`, strconv.Quote(validID), value),
				ExpectError: regexp.MustCompile(`must be at least 16 characters`),
			},
			{
				Config:      config("null", value, strconv.Quote(validSecret)),
				ExpectError: regexp.MustCompile(`all of\s+` + "`access_key_id,secret_access_key`"),
			},
			{
				Config:      config("null", strconv.Quote(validID), value),
				ExpectError: regexp.MustCompile(`all of\s+` + "`access_key_id,secret_access_key`"),
			},
		},
	})
}

func TestAccGarageKey_importCredentialsValidation(t *testing.T) {
	validID := "GK" + acctest.RandStringFromCharSet(24, "0123456789abcdef")
	validSecret := acctest.RandStringFromCharSet(64, "0123456789abcdef")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccGarageKeyImportConfig("acc-key-invalid", "GK1234", validSecret),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`must be at least 8 characters`),
			},
			{
				Config:      testAccGarageKeyImportConfig("acc-key-invalid", "GK/"+validID, validSecret),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`must contain only ASCII letters`),
			},
			{
				Config:      testAccGarageKeyImportConfig("acc-key-invalid", validID, "too-short"),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`must be at least 16 characters`),
			},
			{
				Config:      testAccGarageKeyImportConfig("acc-key-invalid", validID, "with spaces in the secret"),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`must contain only graphic ASCII characters`),
			},
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "garage_key" "test" {
  name          = "acc-key-invalid"
  access_key_id = %q
}
`, validID),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`"access_key_id": all of\s+` + "`access_key_id,secret_access_key`"),
			},
		},
	})
}

func testAccGarageKeyImportConfig(name, accessKeyID, secretAccessKey string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "garage_key" "test" {
  name              = %q
  access_key_id     = %q
  secret_access_key = %q
}
`, name, accessKeyID, secretAccessKey)
}

// testAccCheckGarageKeyInGarage checks the key in Garage itself, not in the
// Terraform state.
func testAccCheckGarageKeyInGarage(t *testing.T, accessKeyID, secretAccessKey, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccClient(t)
		key, resp, err := client.Client.AccessKeyAPI.GetKeyInfo(client.WithAuth(context.Background())).Id(accessKeyID).ShowSecretKey(true).Execute()
		if err != nil {
			return fmt.Errorf("failed to read key %s: %w", accessKeyID, err)
		}
		defer func() {
			if resp.Body != nil {
				_ = resp.Body.Close()
			}
		}()

		if got := key.GetAccessKeyId(); got != accessKeyID {
			return fmt.Errorf("access key ID is %q, want %q", got, accessKeyID)
		}
		if got := key.GetSecretAccessKey(); got != secretAccessKey {
			return fmt.Errorf("secret access key in Garage does not match the configured secret")
		}
		if got := key.GetName(); got != name {
			return fmt.Errorf("key name is %q, want %q", got, name)
		}
		return nil
	}
}

// testAccCheckGarageKeyAbsent checks that Garage does not have the key.
func testAccCheckGarageKeyAbsent(t *testing.T, accessKeyID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccClient(t)
		_, resp, err := client.Client.AccessKeyAPI.GetKeyInfo(client.WithAuth(context.Background())).Id(accessKeyID).Execute()
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		if err == nil {
			return fmt.Errorf("key %s still exists", accessKeyID)
		}
		if resp == nil || resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("failed to read key %s: %w", accessKeyID, err)
		}
		return nil
	}
}

func testAccGarageKeyConfig(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "garage_key" "test" {
  name = %q
}
`, name)
}

func testAccCheckGarageKeyDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccClient(t)
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "garage_key" {
				continue
			}
			_, resp, err := client.Client.AccessKeyAPI.GetKeyInfo(client.WithAuth(context.Background())).Id(rs.Primary.ID).Execute()
			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}
			gone, probeErr := classifyDeletionProbe(resp, err)
			if probeErr != nil {
				return fmt.Errorf("key %s: %w", rs.Primary.ID, probeErr)
			}
			if !gone {
				return fmt.Errorf("key %s still exists", rs.Primary.ID)
			}
		}
		return nil
	}
}
