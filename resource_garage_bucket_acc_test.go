package main

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
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

func TestAccGarageBucket_import(t *testing.T) {
	alias := acctest.RandomWithPrefix("acc-bucket-import")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckGarageBucketDestroy(t),
		Steps: []resource.TestStep{
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
			},
			{
				ResourceName:      "garage_bucket.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// ImportStateVerify also compares the ID, so the alias must
				// resolve to the ID of the bucket.
				ResourceName:      "garage_bucket.test",
				ImportState:       true,
				ImportStateId:     alias,
				ImportStateVerify: true,
			},
			{
				ResourceName:  "garage_bucket.test",
				ImportState:   true,
				ImportStateId: alias + "-missing",
				ExpectError:   regexp.MustCompile(`no bucket has the global alias`),
			},
			{
				ResourceName:  "garage_bucket.test",
				ImportState:   true,
				ImportStateId: strings.Repeat("0", 64),
				ExpectError:   regexp.MustCompile(`no bucket has the ID`),
			},
		},
	})
}

// TestAccGarageBucket_importOtherGlobalAlias checks which global alias the
// import writes to the state when the bucket has more than one.
func TestAccGarageBucket_importOtherGlobalAlias(t *testing.T) {
	prefix := "accb-" + acctest.RandString(8)
	configured := prefix + "-configured"
	// Garage sorts the aliases, so this one is first.
	other := prefix + "-0other"
	var bucketID string

	checkImportedAlias := func(want string) resource.ImportStateCheckFunc {
		return func(states []*terraform.InstanceState) error {
			if len(states) != 1 {
				return fmt.Errorf("got %d states, want 1", len(states))
			}
			state := states[0]
			if state.ID != bucketID {
				return fmt.Errorf("got bucket ID %q, want %q", state.ID, bucketID)
			}
			if got := state.Attributes["global_alias"]; got != want {
				return fmt.Errorf("got global_alias %q, want %q", got, want)
			}
			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckGarageBucketDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testAccGarageBucketAliasConfig(configured),
				Check:  testAccCheckGarageBucketID("garage_bucket.test", &bucketID),
			},
			{
				PreConfig: func() {
					if err := addBucketGlobalAlias(context.Background(), testAccClient(t), bucketID, other); err != nil {
						t.Fatal(err)
					}
				},
				Config:   testAccGarageBucketAliasConfig(configured),
				PlanOnly: true,
			},
			// An import by alias keeps that alias, so the state matches the
			// configuration.
			{
				ResourceName:      "garage_bucket.test",
				ImportState:       true,
				ImportStateId:     configured,
				ImportStateVerify: true,
			},
			{
				ResourceName:     "garage_bucket.test",
				ImportState:      true,
				ImportStateId:    other,
				ImportStateCheck: checkImportedAlias(other),
			},
			// An import by bucket ID keeps the first alias.
			{
				ResourceName:     "garage_bucket.test",
				ImportState:      true,
				ImportStateCheck: checkImportedAlias(other),
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

func TestAccGarageBucket_quotas(t *testing.T) {
	alias := "accb-" + acctest.RandString(8) + "-quotas"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckGarageBucketDestroy(t),
		Steps: []resource.TestStep{
			// Create the bucket with both quotas.
			{
				Config: testAccGarageBucketQuotasConfig(alias, "max_size = 1048576\n  max_objects = 100"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_bucket.test", "max_size", "1048576"),
					resource.TestCheckResourceAttr("garage_bucket.test", "max_objects", "100"),
					testAccCheckGarageBucketQuotas(t, "garage_bucket.test", 1048576, 100),
				),
			},
			// Change one quota. The other stays.
			{
				Config: testAccGarageBucketQuotasConfig(alias, "max_size = 2097152\n  max_objects = 100"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_bucket.test", "max_size", "2097152"),
					resource.TestCheckResourceAttr("garage_bucket.test", "max_objects", "100"),
					testAccCheckGarageBucketQuotas(t, "garage_bucket.test", 2097152, 100),
				),
			},
			// Remove one quota with 0. The other stays.
			{
				Config: testAccGarageBucketQuotasConfig(alias, "max_size = 2097152\n  max_objects = 0"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_bucket.test", "max_size", "2097152"),
					resource.TestCheckResourceAttr("garage_bucket.test", "max_objects", "0"),
					testAccCheckGarageBucketQuotas(t, "garage_bucket.test", 2097152, 0),
				),
			},
			// Remove the quota attributes.
			{
				Config: testAccGarageBucketQuotasConfig(alias, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_bucket.test", "max_size", "0"),
					resource.TestCheckResourceAttr("garage_bucket.test", "max_objects", "0"),
					testAccCheckGarageBucketQuotas(t, "garage_bucket.test", 0, 0),
				),
			},
			// Set a quota on an existing bucket.
			{
				Config: testAccGarageBucketQuotasConfig(alias, "max_objects = 5"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_bucket.test", "max_size", "0"),
					resource.TestCheckResourceAttr("garage_bucket.test", "max_objects", "5"),
					testAccCheckGarageBucketQuotas(t, "garage_bucket.test", 0, 5),
				),
			},
		},
	})
}

func testAccGarageBucketQuotasConfig(alias, quotas string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "garage_bucket" "test" {
  global_alias = %q
  %s
}
`, alias, quotas)
}

// testAccCheckGarageBucketQuotas makes sure that Garage has the given quotas
// for the bucket. A value of 0 means that Garage must report no quota (null).
func testAccCheckGarageBucketQuotas(t *testing.T, name string, maxSize, maxObjects int64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("resource %s not found", name)
		}
		client := testAccClient(t)
		bucket, resp, err := client.Client.BucketAPI.GetBucketInfo(client.WithAuth(context.Background())).Id(rs.Primary.ID).Execute()
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		if err != nil {
			return fmt.Errorf("failed to read bucket %s: %w", rs.Primary.ID, err)
		}
		quotas := bucket.GetQuotas()
		for _, q := range []struct {
			name string
			got  *int64
			want int64
		}{
			{"maxSize", quotas.MaxSize.Get(), maxSize},
			{"maxObjects", quotas.MaxObjects.Get(), maxObjects},
		} {
			switch {
			case q.want == 0 && q.got != nil:
				return fmt.Errorf("bucket %s has %s %d, want null", rs.Primary.ID, q.name, *q.got)
			case q.want != 0 && q.got == nil:
				return fmt.Errorf("bucket %s has %s null, want %d", rs.Primary.ID, q.name, q.want)
			case q.want != 0 && *q.got != q.want:
				return fmt.Errorf("bucket %s has %s %d, want %d", rs.Primary.ID, q.name, *q.got, q.want)
			}
		}
		return nil
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
