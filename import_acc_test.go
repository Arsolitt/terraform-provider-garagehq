package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"testing"
	"time"

	garage "git.deuxfleurs.fr/garage-sdk/garage-admin-sdk-golang"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// These tests create the objects directly in Garage and adopt them with an
// `import` block, so the first plan is the plan that a user sees after an
// import. The cleanups accept a missing object, because the test can destroy
// the object that it imported.

func testAccImportClient(t *testing.T) *GarageClient {
	t.Helper()
	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance tests are skipped unless TF_ACC is set")
	}
	testAccPreCheck(t)
	return testAccClient(t)
}

func testAccImportBlock(to, id string) string {
	return fmt.Sprintf(`
import {
  to = %s
  id = %q
}
`, to, id)
}

func testAccPlanAction(address string, action plancheck.ResourceActionType) resource.ConfigPlanChecks {
	return resource.ConfigPlanChecks{
		PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, action)},
	}
}

func testAccCleanup(t *testing.T, what string, remove func() (*http.Response, error)) {
	t.Helper()
	t.Cleanup(func() {
		resp, err := remove()
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		if err != nil && (resp == nil || resp.StatusCode != http.StatusNotFound) {
			t.Errorf("failed to delete %s: %v", what, err)
		}
	})
}

// testAccCreateImportBucket creates a bucket with the given global aliases.
func testAccCreateImportBucket(t *testing.T, client *GarageClient, aliases ...string) string {
	t.Helper()
	ctx := context.Background()

	req := garage.NewCreateBucketRequest()
	if len(aliases) > 0 {
		req.SetGlobalAlias(aliases[0])
	}
	bucket, resp, err := client.Client.BucketAPI.CreateBucket(client.WithAuth(ctx)).CreateBucketRequest(*req).Execute()
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("failed to create bucket: %v", err)
	}
	id := bucket.GetId()
	testAccCleanup(t, "bucket "+id, func() (*http.Response, error) {
		return client.Client.BucketAPI.DeleteBucket(client.WithAuth(ctx)).Id(id).Execute()
	})

	for _, alias := range aliases[min(1, len(aliases)):] {
		if err := addBucketGlobalAlias(ctx, client, id, alias); err != nil {
			t.Fatal(err)
		}
	}
	return id
}

func testAccCheckGarageBucketWebsiteDisabled(t *testing.T, id string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccClient(t)
		bucket, resp, err := client.Client.BucketAPI.GetBucketInfo(client.WithAuth(context.Background())).Id(id).Execute()
		if err != nil {
			return fmt.Errorf("failed to read bucket %s: %w", id, err)
		}
		defer func() {
			if resp.Body != nil {
				_ = resp.Body.Close()
			}
		}()
		if bucket.GetWebsiteAccess() {
			return fmt.Errorf("bucket %s still has website access", id)
		}
		return nil
	}
}

func TestAccGarageBucket_importBlock(t *testing.T) {
	const addr = "garage_bucket.test"
	config := func(id, body string) string {
		return testAccProviderConfig() + testAccImportBlock(addr, id) + fmt.Sprintf("resource \"garage_bucket\" \"test\" {\n%s\n}\n", body)
	}
	test := func(t *testing.T, step resource.TestStep) {
		t.Helper()
		resource.Test(t, resource.TestCase{
			ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
			CheckDestroy:             testAccCheckGarageBucketDestroy(t),
			Steps:                    []resource.TestStep{step},
		})
	}

	t.Run("no alias", func(t *testing.T) {
		id := testAccCreateImportBucket(t, testAccImportClient(t))
		test(t, resource.TestStep{
			Config:           config(id, ""),
			ConfigPlanChecks: testAccPlanAction(addr, plancheck.ResourceActionNoop),
		})
	})

	t.Run("configured alias", func(t *testing.T) {
		alias := acctest.RandomWithPrefix("acc-bucket-block")
		id := testAccCreateImportBucket(t, testAccImportClient(t), alias)
		test(t, resource.TestStep{
			Config:           config(id, fmt.Sprintf("global_alias = %q", alias)),
			ConfigPlanChecks: testAccPlanAction(addr, plancheck.ResourceActionNoop),
		})
	})

	t.Run("omitted last alias", func(t *testing.T) {
		alias := acctest.RandomWithPrefix("acc-bucket-block")
		testAccCreateImportBucket(t, testAccImportClient(t), alias)
		test(t, resource.TestStep{
			Config:           config(alias, ""),
			ConfigPlanChecks: testAccPlanAction(addr, plancheck.ResourceActionUpdate),
			ExpectError:      regexp.MustCompile(`Garage does not remove the last alias of a bucket`),
		})
	})

	t.Run("omitted alias with another alias", func(t *testing.T) {
		alias := acctest.RandomWithPrefix("acc-bucket-block")
		other := alias + "-other"
		id := testAccCreateImportBucket(t, testAccImportClient(t), alias, other)
		test(t, resource.TestStep{
			Config:           config(alias, ""),
			ConfigPlanChecks: testAccPlanAction(addr, plancheck.ResourceActionUpdate),
			Check:            testAccCheckGarageBucketGlobalAliases(t, &id, other),
		})
	})

	t.Run("omitted website access", func(t *testing.T) {
		client := testAccImportClient(t)
		alias := acctest.RandomWithPrefix("acc-bucket-block")
		id := testAccCreateImportBucket(t, client, alias)
		website := []interface{}{map[string]interface{}{"enabled": true, "index_document": "index.html", "error_document": ""}}
		if err := setBucketWebsiteAccess(context.Background(), client, id, website); err != nil {
			t.Fatal(err)
		}
		test(t, resource.TestStep{
			Config:           config(alias, fmt.Sprintf("global_alias = %q", alias)),
			ConfigPlanChecks: testAccPlanAction(addr, plancheck.ResourceActionUpdate),
			Check:            testAccCheckGarageBucketWebsiteDisabled(t, id),
		})
	})
}

// testAccCreateImportKey creates a key with known credentials, so that the
// configuration can repeat them. Garage does not import a deleted key ID
// again, so each call uses a new ID.
func testAccCreateImportKey(t *testing.T, client *GarageClient, name string) (id, secret string) {
	t.Helper()
	ctx := context.Background()

	id = "GK" + acctest.RandStringFromCharSet(24, "0123456789abcdef")
	secret = acctest.RandStringFromCharSet(64, "0123456789abcdef")
	req := garage.NewImportKeyRequest(id, secret)
	req.SetName(name)
	_, resp, err := client.Client.AccessKeyAPI.ImportKey(client.WithAuth(ctx)).ImportKeyRequest(*req).Execute()
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("failed to create key: %v", err)
	}
	testAccCleanup(t, "key "+id, func() (*http.Response, error) {
		return client.Client.AccessKeyAPI.DeleteKey(client.WithAuth(ctx)).Id(id).Execute()
	})
	return id, secret
}

func TestAccGarageKey_importBlock(t *testing.T) {
	const addr = "garage_key.test"
	nameConfig := func(id, name string) string {
		return testAccProviderConfig() + testAccImportBlock(addr, id) + fmt.Sprintf(`
resource "garage_key" "test" {
  name = %q
}
`, name)
	}
	test := func(t *testing.T, step resource.TestStep) {
		t.Helper()
		resource.Test(t, resource.TestCase{
			ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
			CheckDestroy:             testAccCheckGarageKeyDestroy(t),
			Steps:                    []resource.TestStep{step},
		})
	}

	t.Run("same credentials", func(t *testing.T) {
		name := acctest.RandomWithPrefix("acc-key-block")
		id, secret := testAccCreateImportKey(t, testAccImportClient(t), name)
		test(t, resource.TestStep{
			Config:           testAccImportBlock(addr, id) + testAccGarageKeyImportConfig(name, id, secret),
			ConfigPlanChecks: testAccPlanAction(addr, plancheck.ResourceActionNoop),
		})
	})

	t.Run("omitted credentials", func(t *testing.T) {
		name := acctest.RandomWithPrefix("acc-key-block")
		id, secret := testAccCreateImportKey(t, testAccImportClient(t), name)
		test(t, resource.TestStep{
			Config:           nameConfig(id, name),
			ConfigPlanChecks: testAccPlanAction(addr, plancheck.ResourceActionNoop),
			Check:            resource.TestCheckResourceAttr(addr, "secret_access_key", secret),
		})
	})

	t.Run("other name", func(t *testing.T) {
		name := acctest.RandomWithPrefix("acc-key-block")
		id, secret := testAccCreateImportKey(t, testAccImportClient(t), name)
		test(t, resource.TestStep{
			Config:           nameConfig(id, name+"-new"),
			ConfigPlanChecks: testAccPlanAction(addr, plancheck.ResourceActionUpdate),
			Check:            testAccCheckGarageKeyInGarage(t, id, secret, name+"-new"),
		})
	})

	t.Run("other secret", func(t *testing.T) {
		name := acctest.RandomWithPrefix("acc-key-block")
		id, _ := testAccCreateImportKey(t, testAccImportClient(t), name)
		test(t, resource.TestStep{
			Config:      testAccImportBlock(addr, id) + testAccGarageKeyImportConfig(name, id, acctest.RandStringFromCharSet(64, "0123456789abcdef")),
			ExpectError: regexp.MustCompile(`cannot change secret_access_key`),
		})
	})
}

func testAccCreateImportAdminToken(t *testing.T, client *GarageClient, name string, expiration *time.Time) string {
	t.Helper()
	ctx := context.Background()

	req := garage.NewUpdateAdminTokenRequestBody()
	req.SetName(name)
	req.SetScope([]string{"ListBuckets"})
	if expiration != nil {
		req.SetExpiration(*expiration)
	}
	token, resp, err := client.Client.AdminAPITokenAPI.CreateAdminToken(client.WithAuth(ctx)).UpdateAdminTokenRequestBody(*req).Execute()
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("failed to create admin token: %v", err)
	}
	id := token.GetId()
	testAccCleanup(t, "admin token "+id, func() (*http.Response, error) {
		return client.Client.AdminAPITokenAPI.DeleteAdminToken(client.WithAuth(ctx)).Id(id).Execute()
	})
	return id
}

func TestAccGarageAdminToken_importBlock(t *testing.T) {
	const addr = "garage_admin_token.test"
	expiration := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)
	formatted := expiration.Format(time.RFC3339)

	for _, tc := range []struct {
		name       string
		expiration *time.Time
		attributes string
		action     plancheck.ResourceActionType
		want       *time.Time
	}{
		{
			name:       "omitted expiration keeps it",
			expiration: &expiration,
			action:     plancheck.ResourceActionNoop,
			want:       &expiration,
		},
		{
			name:       "same expiration",
			expiration: &expiration,
			attributes: fmt.Sprintf("expiration = %q", formatted),
			action:     plancheck.ResourceActionNoop,
			want:       &expiration,
		},
		{
			name:       "never_expires on a token with no expiration",
			attributes: "never_expires = true",
			action:     plancheck.ResourceActionUpdate,
		},
		{
			name:       "never_expires removes the expiration",
			expiration: &expiration,
			attributes: "never_expires = true",
			action:     plancheck.ResourceActionUpdate,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			name := acctest.RandomWithPrefix("acc-admin-token-block")
			id := testAccCreateImportAdminToken(t, testAccImportClient(t), name, tc.expiration)

			resource.Test(t, resource.TestCase{
				ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
				CheckDestroy:             testAccCheckGarageAdminTokenDestroy(t),
				Steps: []resource.TestStep{{
					Config: testAccProviderConfig() + testAccImportBlock(addr, id) + fmt.Sprintf(`
resource "garage_admin_token" "test" {
  name  = %q
  scope = ["ListBuckets"]
  %s
}
`, name, tc.attributes),
					ConfigPlanChecks: testAccPlanAction(addr, tc.action),
					Check:            testAccCheckGarageAdminTokenExpiration(t, tc.want),
				}},
			})
		})
	}
}
