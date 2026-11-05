package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
			if err == nil {
				return fmt.Errorf("key %s still exists", rs.Primary.ID)
			}
		}
		return nil
	}
}
