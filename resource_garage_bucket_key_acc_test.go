package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	garage "git.deuxfleurs.fr/garage-sdk/garage-admin-sdk-golang"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccGarageBucketKey_basic(t *testing.T) {
	name := acctest.RandomWithPrefix("acc-bucket-key")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGarageBucketKeyConfig(name, true, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("garage_bucket_key.test", "bucket_id", "garage_bucket.test", "id"),
					resource.TestCheckResourceAttrPair("garage_bucket_key.test", "access_key_id", "garage_key.test", "access_key_id"),
					resource.TestCheckResourceAttr("garage_bucket_key.test", "read", "true"),
					resource.TestCheckResourceAttr("garage_bucket_key.test", "write", "false"),
				),
			},
			{
				Config: testAccGarageBucketKeyConfig(name, true, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_bucket_key.test", "write", "true"),
				),
			},
		},
	})
}

func testAccGarageBucketKeyConfig(name string, read, write bool) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "garage_bucket" "test" {
  global_alias = %[1]q
}

resource "garage_key" "test" {
  name = %[1]q
}

resource "garage_bucket_key" "test" {
  bucket_id     = garage_bucket.test.id
  access_key_id = garage_key.test.access_key_id
  read          = %[2]t
  write         = %[3]t
  owner         = false
}
`, name, read, write)
}

// TestAccGarageBucketKey_destroy checks that destroying the resource removes
// the bucket from the key's bucket list. The bucket and the access key are
// provisioned directly through the admin API and only torn down once the test
// is over, so CheckDestroy still finds the key and can assert on its
// permissions.
func TestAccGarageBucketKey_destroy(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance tests are skipped unless TF_ACC is set")
	}
	testAccPreCheck(t)

	client := testAccClient(t)
	bucketID, keyID := testAccCreateBucketAndKey(t, client)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckGarageBucketKeyDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testAccGarageBucketKeyOnlyConfig(bucketID, keyID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("garage_bucket_key.test", "bucket_id", bucketID),
					resource.TestCheckResourceAttr("garage_bucket_key.test", "access_key_id", keyID),
					resource.TestCheckResourceAttr("garage_bucket_key.test", "read", "true"),
					resource.TestCheckResourceAttr("garage_bucket_key.test", "write", "false"),
					resource.TestCheckResourceAttr("garage_bucket_key.test", "owner", "false"),
				),
			},
		},
	})
}

func testAccGarageBucketKeyOnlyConfig(bucketID, keyID string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "garage_bucket_key" "test" {
  bucket_id     = %[1]q
  access_key_id = %[2]q
  read          = true
  write         = false
  owner         = false
}
`, bucketID, keyID)
}

// testAccCreateBucketAndKey creates a bucket and an access key outside
// Terraform, and schedules their removal after the test.
func testAccCreateBucketAndKey(t *testing.T, client *GarageClient) (bucketID, keyID string) {
	t.Helper()
	ctx := context.Background()

	bucketReq := garage.NewCreateBucketRequest()
	bucketReq.SetGlobalAlias(acctest.RandomWithPrefix("acc-bucket-key-destroy"))
	bucket, resp, err := client.Client.BucketAPI.CreateBucket(client.WithAuth(ctx)).CreateBucketRequest(*bucketReq).Execute()
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("failed to create bucket: %v", err)
	}
	bucketID = bucket.GetId()
	t.Cleanup(func() {
		resp, err := client.Client.BucketAPI.DeleteBucket(client.WithAuth(ctx)).Id(bucketID).Execute()
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		if err != nil {
			t.Errorf("failed to delete bucket %s: %v", bucketID, err)
		}
	})

	keyReq := garage.NewUpdateKeyRequestBody()
	keyReq.SetName(acctest.RandomWithPrefix("acc-bucket-key-destroy-key"))
	key, resp, err := client.Client.AccessKeyAPI.CreateKey(client.WithAuth(ctx)).Body(*keyReq).Execute()
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("failed to create key: %v", err)
	}
	keyID = key.GetAccessKeyId()
	t.Cleanup(func() {
		resp, err := client.Client.AccessKeyAPI.DeleteKey(client.WithAuth(ctx)).Id(keyID).Execute()
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		if err != nil {
			t.Errorf("failed to delete key %s: %v", keyID, err)
		}
	})

	return bucketID, keyID
}

// testAccCheckGarageBucketKeyDestroy asserts that the key no longer carries
// any permission on the bucket named by the resource ID.
func testAccCheckGarageBucketKeyDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccClient(t)
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "garage_bucket_key" {
				continue
			}
			bucketID, keyID, ok := strings.Cut(rs.Primary.ID, "/")
			if !ok || bucketID == "" || keyID == "" {
				return fmt.Errorf("malformed garage_bucket_key ID %q", rs.Primary.ID)
			}
			key, resp, err := client.Client.AccessKeyAPI.GetKeyInfo(client.WithAuth(context.Background())).Id(keyID).Execute()
			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}
			gone, probeErr := classifyDeletionProbe(resp, err)
			if probeErr != nil {
				return fmt.Errorf("bucket key %s: %w", rs.Primary.ID, probeErr)
			}
			if gone {
				continue
			}
			for _, bucket := range key.GetBuckets() {
				if bucket.GetId() != bucketID {
					continue
				}
				perms := bucket.GetPermissions()
				if perms.GetRead() || perms.GetWrite() || perms.GetOwner() {
					return fmt.Errorf("bucket key %s still exists", rs.Primary.ID)
				}
			}
		}
		return nil
	}
}
