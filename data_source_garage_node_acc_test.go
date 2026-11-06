package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccGarageNodeDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
data "garage_node" "self" {}

data "garage_node" "by_id" {
  id = data.garage_node.self.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckGarageNodeIsSelf(t, "data.garage_node.self"),
					resource.TestCheckResourceAttrSet("data.garage_node.self", "hostname"),
					resource.TestCheckResourceAttrSet("data.garage_node.self", "address"),
					resource.TestCheckResourceAttrSet("data.garage_node.self", "garage_version"),
					resource.TestCheckResourceAttr("data.garage_node.self", "is_up", "true"),
					resource.TestCheckResourceAttr("data.garage_node.self", "draining", "false"),
					resource.TestCheckResourceAttrPair("data.garage_node.by_id", "id", "data.garage_node.self", "id"),
					resource.TestCheckResourceAttrPair("data.garage_node.by_id", "hostname", "data.garage_node.self", "hostname"),
					resource.TestCheckResourceAttrPair("data.garage_node.by_id", "address", "data.garage_node.self", "address"),
					resource.TestCheckResourceAttrPair("data.garage_node.by_id", "zone", "data.garage_node.self", "zone"),
					resource.TestCheckResourceAttrPair("data.garage_node.by_id", "capacity", "data.garage_node.self", "capacity"),
					resource.TestCheckResourceAttrPair("data.garage_node.by_id", "is_up", "data.garage_node.self", "is_up"),
					resource.TestCheckResourceAttrPair("data.garage_node.by_id", "draining", "data.garage_node.self", "draining"),
					resource.TestCheckResourceAttrPair("data.garage_node.by_id", "garage_version", "data.garage_node.self", "garage_version"),
				),
			},
		},
	})
}

func TestAccGarageNodeDataSource_invalidID(t *testing.T) {
	for name, id := range map[string]string{
		"short":     "abc123",
		"long":      strings.Repeat("a", 65),
		"uppercase": strings.Repeat("A", 64),
		"not hex":   strings.Repeat("g", 64),
		"newline":   strings.Repeat("a", 64) + `\n`,
		"empty":     "",
	} {
		t.Run(name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: testAccProviderConfig() + fmt.Sprintf(`
data "garage_node" "invalid" {
  id = "%s"
}
`, id),
						ExpectError: regexp.MustCompile(`64 lowercase hexadecimal characters`),
					},
				},
			})
		})
	}
}

func TestAccGarageNodeDataSource_notFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
data "garage_node" "missing" {
  id = "0000000000000000000000000000000000000000000000000000000000000000"
}
`,
				ExpectError: regexp.MustCompile(`not found in the cluster`),
			},
		},
	})
}

// testAccCheckGarageNodeIsSelf checks that the data source has the ID and
// the role of the node that serves the admin API.
func testAccCheckGarageNodeIsSelf(t *testing.T, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("%s not found in state", name)
		}

		ctx := context.Background()
		client := testAccClient(t)
		id, err := garageSelfNodeID(ctx, client)
		if err != nil {
			return err
		}
		if rs.Primary.ID != id {
			return fmt.Errorf("%s: id is %q, expected %q", name, rs.Primary.ID, id)
		}

		layout, resp, err := client.Client.ClusterLayoutAPI.GetClusterLayout(client.WithAuth(ctx)).Execute()
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		if err != nil {
			return err
		}
		for _, role := range layout.GetRoles() {
			if role.GetId() != id {
				continue
			}
			attrs := rs.Primary.Attributes
			if attrs["zone"] != role.GetZone() {
				return fmt.Errorf("%s: zone is %q, expected %q", name, attrs["zone"], role.GetZone())
			}
			capacity := ""
			if c, ok := role.GetCapacityOk(); ok && c != nil {
				capacity = formatCapacityExact(*c)
			}
			if attrs["capacity"] != capacity {
				return fmt.Errorf("%s: capacity is %q, expected %q", name, attrs["capacity"], capacity)
			}
			if attrs["tags.#"] != fmt.Sprint(len(role.GetTags())) {
				return fmt.Errorf("%s: %s tags, expected %d", name, attrs["tags.#"], len(role.GetTags()))
			}
			for i, tag := range role.GetTags() {
				if got := attrs[fmt.Sprintf("tags.%d", i)]; got != tag {
					return fmt.Errorf("%s: tag %d is %q, expected %q", name, i, got, tag)
				}
			}
			return nil
		}
		return fmt.Errorf("node %s has no role in the layout", id)
	}
}

func TestGarageSelfNodeID(t *testing.T) {
	const (
		selfID  = "1111111111111111111111111111111111111111111111111111111111111111"
		otherID = "2222222222222222222222222222222222222222222222222222222222222222"
	)
	nodeInfo := func(id string) string {
		return `{"dbEngine":"lmdb","garageVersion":"v2.3.0","nodeId":"` + id + `","rustVersion":"1.90.0"}`
	}

	tests := []struct {
		name    string
		body    string
		want    string
		wantErr string
	}{
		{
			name: "one node",
			body: `{"success":{"` + selfID + `":` + nodeInfo(selfID) + `},"error":{}}`,
			want: selfID,
		},
		{
			name:    "multiple nodes",
			body:    `{"success":{"` + selfID + `":` + nodeInfo(selfID) + `,"` + otherID + `":` + nodeInfo(otherID) + `},"error":{}}`,
			wantErr: `expected one node for "self", got 2`,
		},
		{
			name:    "node error",
			body:    `{"success":{},"error":{"` + otherID + `":"node is not available"}}`,
			wantErr: "failed to get node info of " + otherID + ": node is not available",
		},
		{
			name:    "empty response",
			body:    `{"success":{},"error":{}}`,
			wantErr: "failed to get node info: empty response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := tt.body
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v2/GetNodeInfo" || r.URL.Query().Get("node") != "self" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(body))
			}))
			defer srv.Close()

			client, err := NewGarageClient("http", strings.TrimPrefix(srv.URL, "http://"), "test-token")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got, err := garageSelfNodeID(context.Background(), client)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("id = %q, want %q", got, tt.want)
			}
		})
	}
}
