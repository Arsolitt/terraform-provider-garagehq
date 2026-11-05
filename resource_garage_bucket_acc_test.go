package main

import (
	"context"
	"fmt"
	"testing"

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
			if err == nil {
				return fmt.Errorf("bucket %s still exists", rs.Primary.ID)
			}
		}
		return nil
	}
}
