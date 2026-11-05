package main

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
