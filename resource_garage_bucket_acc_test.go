package main

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"testing"

	garage "git.deuxfleurs.fr/garage-sdk/garage-admin-sdk-golang"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccGarageBucket_basic(t *testing.T) {
	alias := acctest.RandomWithPrefix("acc-bucket")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckGarageBucketDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "garage_bucket" "test" {
  global_alias = %q
}
`, alias),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_bucket.test", "global_alias", alias),
					resource.TestCheckResourceAttrSet("garage_bucket.test", "id"),
					resource.TestCheckResourceAttr("garage_bucket.test", "objects", "0"),
				),
			},
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "garage_bucket" "test" {
  global_alias = %q

  website_access {
    enabled        = true
    index_document = "index.html"
    error_document = "error.html"
  }
}
`, alias),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_bucket.test", "website_access.0.enabled", "true"),
					resource.TestCheckResourceAttr("garage_bucket.test", "website_access.0.index_document", "index.html"),
					resource.TestCheckResourceAttr("garage_bucket.test", "website_access.0.error_document", "error.html"),
				),
			},
		},
	})
}

func TestAccGarageBucket_globalAlias(t *testing.T) {
	prefix := "accb-" + acctest.RandString(8)
	first := prefix + "-first"
	second := prefix + "-second"
	extra := prefix + "-0extra"
	var bucketID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckGarageBucketDestroy(t),
		Steps: []resource.TestStep{
			// Create the bucket without a global alias.
			{
				Config: testAccGarageBucketAliasConfig(""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_bucket.test", "global_alias", ""),
					testAccCheckGarageBucketID("garage_bucket.test", &bucketID),
					testAccCheckGarageBucketGlobalAliases(t, &bucketID),
				),
			},
			// Set the global alias.
			{
				Config: testAccGarageBucketAliasConfig(first),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_bucket.test", "global_alias", first),
					resource.TestCheckResourceAttrPtr("garage_bucket.test", "id", &bucketID),
					testAccCheckGarageBucketGlobalAliases(t, &bucketID, first),
				),
			},
			// Rename the global alias. The old alias must be gone.
			{
				Config: testAccGarageBucketAliasConfig(second),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_bucket.test", "global_alias", second),
					resource.TestCheckResourceAttrPtr("garage_bucket.test", "id", &bucketID),
					testAccCheckGarageBucketGlobalAliases(t, &bucketID, second),
					testAccCheckGarageGlobalAliasAbsent(t, first),
				),
			},
			// An alias that is added outside Terraform causes no diff.
			{
				PreConfig: func() {
					if err := addBucketGlobalAlias(context.Background(), testAccClient(t), bucketID, extra); err != nil {
						t.Fatal(err)
					}
				},
				Config:   testAccGarageBucketAliasConfig(second),
				PlanOnly: true,
			},
			// Rename to an alias that the bucket already has.
			{
				Config: testAccGarageBucketAliasConfig(extra),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_bucket.test", "global_alias", extra),
					testAccCheckGarageBucketGlobalAliases(t, &bucketID, extra),
				),
			},
			// Garage does not remove the last alias of a bucket.
			{
				Config:      testAccGarageBucketAliasConfig(""),
				ExpectError: regexp.MustCompile("Garage does not remove the last alias of a bucket"),
			},
			// The failed apply must have left the global alias in place. With a
			// local alias, the global alias can then be removed.
			{
				PreConfig: func() {
					if err := testAccCheckGarageBucketGlobalAliases(t, &bucketID, extra)(nil); err != nil {
						t.Fatal(err)
					}
					testAccAddGarageBucketLocalAlias(t, bucketID, prefix+"-local")
				},
				Config: testAccGarageBucketAliasConfig(""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_bucket.test", "global_alias", ""),
					testAccCheckGarageBucketGlobalAliases(t, &bucketID),
					testAccCheckGarageGlobalAliasAbsent(t, extra),
				),
			},
		},
	})
}

// TestAccGarageBucket_otherGlobalAliases checks that the provider does not
// adopt or remove a global alias that is not in the configuration.
func TestAccGarageBucket_otherGlobalAliases(t *testing.T) {
	prefix := "accb-" + acctest.RandString(8)
	configured := prefix + "-configured"
	// Garage sorts the aliases, so this one is first.
	other := prefix + "-0other"
	var bucketID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckGarageBucketDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testAccGarageBucketAliasConfig(configured),
				Check:  testAccCheckGarageBucketID("garage_bucket.test", &bucketID),
			},
			// Remove the configured alias. The other alias is not adopted.
			{
				PreConfig: func() {
					if err := addBucketGlobalAlias(context.Background(), testAccClient(t), bucketID, other); err != nil {
						t.Fatal(err)
					}
				},
				Config: testAccGarageBucketAliasConfig(""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_bucket.test", "global_alias", ""),
					testAccCheckGarageBucketGlobalAliases(t, &bucketID, other),
					testAccCheckGarageGlobalAliasAbsent(t, configured),
				),
			},
			{
				Config:   testAccGarageBucketAliasConfig(""),
				PlanOnly: true,
			},
			{
				Config: testAccGarageBucketAliasConfig(configured),
				Check:  testAccCheckGarageBucketGlobalAliases(t, &bucketID, other, configured),
			},
			// When the configured alias is removed outside Terraform, the
			// apply adds it again and keeps the other alias.
			{
				PreConfig: func() {
					if _, err := removeBucketGlobalAlias(context.Background(), testAccClient(t), bucketID, configured); err != nil {
						t.Fatal(err)
					}
				},
				Config: testAccGarageBucketAliasConfig(configured),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_bucket.test", "global_alias", configured),
					testAccCheckGarageBucketGlobalAliases(t, &bucketID, other, configured),
				),
			},
		},
	})
}

func testAccGarageBucketAliasConfig(alias string) string {
	if alias == "" {
		return testAccProviderConfig() + `
resource "garage_bucket" "test" {}
`
	}
	return testAccProviderConfig() + fmt.Sprintf(`
resource "garage_bucket" "test" {
  global_alias = %q
}
`, alias)
}

// testAccCheckGarageBucketID stores the ID of the bucket in id.
func testAccCheckGarageBucketID(name string, id *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("resource %s not found", name)
		}
		*id = rs.Primary.ID
		return nil
	}
}

// testAccCheckGarageBucketGlobalAliases makes sure that Garage has exactly the
// given global aliases for the bucket.
func testAccCheckGarageBucketGlobalAliases(t *testing.T, id *string, want ...string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccClient(t)
		bucket, resp, err := client.Client.BucketAPI.GetBucketInfo(client.WithAuth(context.Background())).Id(*id).Execute()
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		if err != nil {
			return fmt.Errorf("failed to read bucket %s: %w", *id, err)
		}
		got := bucket.GetGlobalAliases()
		if !slices.Equal(got, want) {
			return fmt.Errorf("bucket %s has global aliases %q, want %q", *id, got, want)
		}
		return nil
	}
}

// testAccCheckGarageGlobalAliasAbsent makes sure that no bucket has the
// global alias.
func testAccCheckGarageGlobalAliasAbsent(t *testing.T, alias string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccClient(t)
		_, resp, err := client.Client.BucketAPI.GetBucketInfo(client.WithAuth(context.Background())).GlobalAlias(alias).Execute()
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		gone, probeErr := classifyDeletionProbe(resp, err)
		if probeErr != nil {
			return fmt.Errorf("global alias %s: %w", alias, probeErr)
		}
		if !gone {
			return fmt.Errorf("global alias %s still exists", alias)
		}
		return nil
	}
}

// testAccAddGarageBucketLocalAlias gives the bucket a local alias in a new
// access key. The key is deleted when the test ends.
func testAccAddGarageBucketLocalAlias(t *testing.T, bucketID, alias string) {
	t.Helper()
	client := testAccClient(t)
	ctx := client.WithAuth(context.Background())

	keyReq := garage.NewUpdateKeyRequestBody()
	keyReq.SetName(alias)
	key, resp, err := client.Client.AccessKeyAPI.CreateKey(ctx).Body(*keyReq).Execute()
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("failed to create key: %v", err)
	}
	keyID := key.GetAccessKeyId()
	t.Cleanup(func() {
		resp, err := client.Client.AccessKeyAPI.DeleteKey(client.WithAuth(context.Background())).Id(keyID).Execute()
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		if err != nil {
			t.Errorf("failed to delete key %s: %v", keyID, err)
		}
	})

	perms := garage.NewApiBucketKeyPerm()
	perms.SetRead(true)
	_, resp, err = client.Client.PermissionAPI.AllowBucketKey(ctx).Body(*garage.NewBucketKeyPermChangeRequest(keyID, bucketID, *perms)).Execute()
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("failed to allow key: %v", err)
	}

	aliasEnum := garage.BucketAliasEnumOneOf1AsBucketAliasEnum(garage.NewBucketAliasEnumOneOf1(keyID, bucketID, alias))
	_, resp, err = client.Client.BucketAliasAPI.AddBucketAlias(ctx).BucketAliasEnum(aliasEnum).Execute()
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("failed to add local alias: %v", err)
	}
}

func testAccCheckGarageBucketDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccClient(t)
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "garage_bucket" {
				continue
			}
			_, resp, err := client.Client.BucketAPI.GetBucketInfo(client.WithAuth(context.Background())).Id(rs.Primary.ID).Execute()
			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}
			gone, probeErr := classifyDeletionProbe(resp, err)
			if probeErr != nil {
				return fmt.Errorf("bucket %s: %w", rs.Primary.ID, probeErr)
			}
			if !gone {
				return fmt.Errorf("bucket %s still exists", rs.Primary.ID)
			}
		}
		return nil
	}
}
