package main

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	garage "git.deuxfleurs.fr/garage-sdk/garage-admin-sdk-golang"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// The test server has one node, and the node must keep its storage role.
// The tests change the zone, capacity and tags of this node, and give a
// gateway role to testAccFakeNodeID, a node that does not exist. A gateway
// holds no data, thus the layout stays valid. Each test puts the layout back
// as it was before the test.

var testAccFakeNodeID = strings.Repeat("a", 64)

// testAccLayoutRole is the role of a node in the layout. A nil capacity is
// a gateway role.
type testAccLayoutRole struct {
	id       string
	zone     string
	capacity *int64
	tags     []string
}

func testAccGetLayout(t *testing.T, client *GarageClient) *clusterLayoutResponse {
	t.Helper()
	layout, err := getClusterLayoutRaw(context.Background(), client)
	if err != nil {
		t.Fatalf("failed to get cluster layout: %v", err)
	}
	return layout
}

// testAccLayoutPreCheck returns the role of the node of the test server. It
// makes sure that the test puts this role back at the end.
func testAccLayoutPreCheck(t *testing.T) testAccLayoutRole {
	t.Helper()
	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance tests are skipped unless TF_ACC is set")
	}
	testAccPreCheck(t)
	client := testAccClient(t)
	ctx := client.WithAuth(context.Background())

	info, resp, err := client.Client.NodeAPI.GetNodeInfo(ctx).Node("self").Execute()
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("failed to get node info: %v", err)
	}
	var id string
	for _, node := range info.GetSuccess() {
		id = node.GetNodeId()
	}
	if id == "" {
		t.Fatalf("failed to get the id of the node: %v", info.GetError())
	}

	layout := testAccGetLayout(t, client)
	if len(layout.StagedRoleChanges) > 0 {
		t.Fatalf("the cluster layout has staged changes before the test")
	}
	for _, role := range layout.Roles {
		if role.Id == id && role.Capacity != nil {
			self := testAccLayoutRole{id: id, zone: role.Zone, capacity: role.Capacity, tags: role.Tags}
			t.Cleanup(func() { testAccRestoreLayout(t, client, self) })
			return self
		}
	}
	t.Fatalf("node %s has no storage role in the cluster layout", id)
	return testAccLayoutRole{}
}

// testAccApplyLayout stages changes and applies them, without the provider.
func testAccApplyLayout(t *testing.T, client *GarageClient, changes ...nodeRoleChange) {
	t.Helper()
	layout, err := updateClusterLayoutRaw(context.Background(), client, changes)
	if err != nil {
		t.Fatal(err)
	}
	req := garage.NewApplyClusterLayoutRequest(layout.Version + 1)
	_, resp, err := client.Client.ClusterLayoutAPI.ApplyClusterLayout(client.WithAuth(context.Background())).ApplyClusterLayoutRequest(*req).Execute()
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("failed to apply cluster layout: %v", garageAPIError(err))
	}
}

// testAccRestoreLayout gives self its role again and removes the role of
// testAccFakeNodeID. The fake node never acknowledges a layout version, thus
// the versions that it is in stay in the "Draining" status. Then
// testAccRestoreLayout skips the dead nodes, and checks that these versions
// are not "Draining" after that.
func testAccRestoreLayout(t *testing.T, client *GarageClient, self testAccLayoutRole) {
	t.Helper()
	ctx := client.WithAuth(context.Background())
	var changes []nodeRoleChange
	for _, role := range testAccGetLayout(t, client).Roles {
		switch role.Id {
		case self.id:
			if role.Zone != self.zone || !reflect.DeepEqual(role.Capacity, self.capacity) || !reflect.DeepEqual(role.Tags, self.tags) {
				changes = append(changes, nodeRoleChange{Id: self.id, Zone: self.zone, Capacity: self.capacity, Tags: self.tags})
			}
		case testAccFakeNodeID:
			remove := true
			changes = append(changes, nodeRoleChange{Id: testAccFakeNodeID, Remove: &remove})
		}
	}
	if len(changes) > 0 {
		testAccApplyLayout(t, client, changes...)
	}

	for skipped := false; ; skipped = true {
		history, resp, err := client.Client.ClusterLayoutAPI.GetClusterLayoutHistory(ctx).Execute()
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		if err != nil {
			t.Fatalf("failed to get cluster layout history: %v", err)
		}
		fake := false
		for _, v := range history.GetVersions() {
			// self has a storage role, thus a gateway is the fake node.
			fake = fake || (v.GetStatus() == garage.DRAINING && v.GetGatewayNodes() > 0)
		}
		if !fake {
			return
		}
		if skipped {
			t.Fatalf("cluster layout versions with the fake node stay in the Draining status")
		}
		// The first call updates the ack tracker, the second the sync tracker.
		for _, allowMissingData := range []bool{false, true} {
			req := garage.NewClusterLayoutSkipDeadNodesRequest(allowMissingData, history.GetCurrentVersion())
			_, resp, err := client.Client.ClusterLayoutAPI.ClusterLayoutSkipDeadNodes(ctx).ClusterLayoutSkipDeadNodesRequest(*req).Execute()
			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}
			if err != nil {
				t.Fatalf("failed to skip dead nodes: %v", garageAPIError(err))
			}
		}
	}
}

// testAccCheckLayoutRole checks the role of a node in the layout.
func testAccCheckLayoutRole(client *GarageClient, want testAccLayoutRole) resource.TestCheckFunc {
	return func(*terraform.State) error {
		layout, err := getClusterLayoutRaw(context.Background(), client)
		if err != nil {
			return err
		}
		for _, role := range layout.Roles {
			if role.Id != want.id {
				continue
			}
			got := testAccLayoutRole{id: role.Id, zone: role.Zone, capacity: role.Capacity, tags: role.Tags}
			if want.tags == nil {
				want.tags = []string{}
			}
			if !reflect.DeepEqual(got, want) {
				return fmt.Errorf("node %s: got role %s, want %s", want.id, got, want)
			}
			return nil
		}
		return fmt.Errorf("node %s has no role in the cluster layout", want.id)
	}
}

func (r testAccLayoutRole) String() string {
	capacity := "gateway"
	if r.capacity != nil {
		capacity = strconv.FormatInt(*r.capacity, 10)
	}
	return fmt.Sprintf("{zone=%q capacity=%s tags=%q}", r.zone, capacity, r.tags)
}

// testAccRevertLayout reverts the staged changes of the cluster layout.
func testAccRevertLayout(t *testing.T, client *GarageClient) {
	t.Helper()
	_, resp, err := client.Client.ClusterLayoutAPI.RevertClusterLayout(client.WithAuth(context.Background())).Execute()
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		t.Errorf("failed to revert cluster layout: %v", err)
	}
}

// testAccStageLayout stages changes to the cluster layout, and reverts
// them at the end of the test.
func testAccStageLayout(t *testing.T, client *GarageClient, changes ...nodeRoleChange) {
	t.Helper()
	t.Cleanup(func() { testAccRevertLayout(t, client) })
	if _, err := updateClusterLayoutRaw(context.Background(), client, changes); err != nil {
		t.Fatal(err)
	}
}

// testAccCheckNoLayoutRole checks that the node has no role in the cluster
// layout.
func testAccCheckNoLayoutRole(client *GarageClient, id string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		layout, err := getClusterLayoutRaw(context.Background(), client)
		if err != nil {
			return err
		}
		for _, role := range layout.Roles {
			if role.Id == id {
				return fmt.Errorf("node %s still has a role in the cluster layout", id)
			}
		}
		return nil
	}
}

// testAccCheckLayoutVersion checks the version of the layout, and that the
// `version` attribute of the resource is the same.
func testAccCheckLayoutVersion(client *GarageClient, name string, want int64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		layout, err := getClusterLayoutRaw(context.Background(), client)
		if err != nil {
			return err
		}
		if layout.Version != want {
			return fmt.Errorf("got layout version %d, want %d", layout.Version, want)
		}
		return resource.TestCheckResourceAttr(name, "version", strconv.FormatInt(want, 10))(s)
	}
}

func testAccClusterLayoutRole(id, zone, capacity string, tags ...string) string {
	block := fmt.Sprintf("  roles {\n    id   = %q\n    zone = %q\n", id, zone)
	if capacity != "" {
		block += fmt.Sprintf("    capacity = %q\n", capacity)
	}
	if len(tags) > 0 {
		quoted := make([]string, len(tags))
		for i, tag := range tags {
			quoted[i] = strconv.Quote(tag)
		}
		block += fmt.Sprintf("    tags = [%s]\n", strings.Join(quoted, ", "))
	}
	return block + "  }\n"
}

func testAccClusterLayoutConfig(roles ...string) string {
	return testAccProviderConfig() + "resource \"garage_cluster_layout\" \"test\" {\n" + strings.Join(roles, "") + "}\n"
}

func int64Ptr(v int64) *int64 { return &v }

func TestAccGarageClusterLayout_basic(t *testing.T) {
	self := testAccLayoutPreCheck(t)
	client := testAccClient(t)
	name := "garage_cluster_layout.test"
	version := testAccGetLayout(t, client).Version

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClusterLayoutConfig(testAccClusterLayoutRole(self.id, "acc-zone", "2G", "acc", "b")),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLayoutRole(client, testAccLayoutRole{id: self.id, zone: "acc-zone", capacity: int64Ptr(2 << 30), tags: []string{"acc", "b"}}),
					testAccCheckLayoutVersion(client, name, version+1),
					resource.TestCheckResourceAttr(name, "id", "cluster-layout"),
					resource.TestCheckResourceAttr(name, "roles.0.capacity", "2G"),
				),
			},
			{
				// A node without tags.
				Config: testAccClusterLayoutConfig(testAccClusterLayoutRole(self.id, "acc-zone", "1536M")),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLayoutRole(client, testAccLayoutRole{id: self.id, zone: "acc-zone", capacity: int64Ptr(1536 << 20)}),
					testAccCheckLayoutVersion(client, name, version+2),
					resource.TestCheckResourceAttr(name, "roles.0.tags.#", "0"),
				),
			},
			{
				Config:   testAccClusterLayoutConfig(testAccClusterLayoutRole(self.id, "acc-zone", "1536M")),
				PlanOnly: true,
			},
			{
				// The same capacity in a different notation does not
				// apply a new layout version.
				Config: testAccClusterLayoutConfig(testAccClusterLayoutRole(self.id, "acc-zone", "1.5G")),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLayoutVersion(client, name, version+2),
					resource.TestCheckResourceAttr(name, "roles.0.capacity", "1.5G"),
				),
			},
		},
	})
}

func TestAccGarageClusterLayout_noChange(t *testing.T) {
	self := testAccLayoutPreCheck(t)
	client := testAccClient(t)
	version := testAccGetLayout(t, client).Version

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				// The layout is already as configured, thus there is no
				// new layout version.
				Config: testAccClusterLayoutConfig(testAccClusterLayoutRole(self.id, self.zone, strconv.FormatInt(*self.capacity, 10), self.tags...)),
				Check:  testAccCheckLayoutVersion(client, "garage_cluster_layout.test", version),
			},
		},
	})
}

func TestAccGarageClusterLayout_stagedChanges(t *testing.T) {
	self := testAccLayoutPreCheck(t)
	client := testAccClient(t)
	version := testAccGetLayout(t, client).Version

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					testAccStageLayout(t, client, nodeRoleChange{Id: testAccFakeNodeID, Zone: "acc-zone"})
				},
				Config:      testAccClusterLayoutConfig(testAccClusterLayoutRole(self.id, "acc-zone", "2G")),
				ExpectError: regexp.MustCompile("staged changes"),
			},
			{
				// The provider did not apply the staged changes.
				PreConfig: func() {
					layout := testAccGetLayout(t, client)
					if layout.Version != version || len(layout.StagedRoleChanges) != 1 {
						t.Errorf("got layout version %d with %d staged changes, want version %d with 1", layout.Version, len(layout.StagedRoleChanges), version)
					}
				},
				Config:   testAccClusterLayoutConfig(testAccClusterLayoutRole(self.id, "acc-zone", "2G")),
				PlanOnly: true,
				// Create failed, thus there is still a resource to create.
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// TestAccGarageClusterLayout_noRoleChangeWithStagedChanges checks that staged
// changes do not stop an update that changes no role. The two spellings of
// the capacity cause the update, but they have the same value.
func TestAccGarageClusterLayout_noRoleChangeWithStagedChanges(t *testing.T) {
	self := testAccLayoutPreCheck(t)
	client := testAccClient(t)
	version := testAccGetLayout(t, client).Version
	capacity := strconv.FormatInt(*self.capacity, 10)
	sameCapacity := capacity + ".0"

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClusterLayoutConfig(testAccClusterLayoutRole(self.id, self.zone, capacity, self.tags...)),
			},
			{
				PreConfig: func() {
					testAccStageLayout(t, client, nodeRoleChange{Id: testAccFakeNodeID, Zone: "acc-zone"})
				},
				Config: testAccClusterLayoutConfig(testAccClusterLayoutRole(self.id, self.zone, sameCapacity, self.tags...)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("garage_cluster_layout.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLayoutVersion(client, "garage_cluster_layout.test", version),
					resource.TestCheckResourceAttr("garage_cluster_layout.test", "roles.0.capacity", sameCapacity),
					func(*terraform.State) error {
						layout, err := getClusterLayoutRaw(context.Background(), client)
						if err != nil {
							return err
						}
						if len(layout.StagedRoleChanges) != 1 {
							return fmt.Errorf("got %d staged changes, want 1", len(layout.StagedRoleChanges))
						}
						return nil
					},
				),
			},
		},
	})
}

func TestAccGarageClusterLayout_removeNode(t *testing.T) {
	self := testAccLayoutPreCheck(t)
	client := testAccClient(t)
	version := testAccGetLayout(t, client).Version
	selfRole := testAccClusterLayoutRole(self.id, self.zone, strconv.FormatInt(*self.capacity, 10), self.tags...)

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClusterLayoutConfig(selfRole, testAccClusterLayoutRole(testAccFakeNodeID, "acc-zone", "")),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLayoutRole(client, self),
					testAccCheckLayoutRole(client, testAccLayoutRole{id: testAccFakeNodeID, zone: "acc-zone"}),
					testAccCheckLayoutVersion(client, "garage_cluster_layout.test", version+1),
				),
			},
			{
				// The gateway is no longer in the configuration, thus it
				// leaves the layout. Self keeps its role.
				Config: testAccClusterLayoutConfig(selfRole),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNoLayoutRole(client, testAccFakeNodeID),
					testAccCheckLayoutRole(client, self),
					testAccCheckLayoutVersion(client, "garage_cluster_layout.test", version+2),
					resource.TestCheckResourceAttr("garage_cluster_layout.test", "roles.#", "1"),
				),
			},
			{
				Config:   testAccClusterLayoutConfig(selfRole),
				PlanOnly: true,
			},
		},
	})
}

// TestAccGarageClusterLayout_updateFails checks that an update that fails
// keeps the prior state, thus the next apply tries the change again.
func TestAccGarageClusterLayout_updateFails(t *testing.T) {
	self := testAccLayoutPreCheck(t)
	client := testAccClient(t)
	version := testAccGetLayout(t, client).Version
	selfRole := testAccClusterLayoutRole(self.id, self.zone, strconv.FormatInt(*self.capacity, 10), self.tags...)
	bothRoles := testAccClusterLayoutConfig(selfRole, testAccClusterLayoutRole(testAccFakeNodeID, "acc-zone", ""))

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: bothRoles,
				Check:  testAccCheckLayoutVersion(client, "garage_cluster_layout.test", version+1),
			},
			{
				// The staged change is on a node that is not in the
				// configuration, thus it does not hide the role change.
				PreConfig: func() {
					testAccStageLayout(t, client, nodeRoleChange{Id: strings.Repeat("b", 64), Zone: "acc-zone"})
				},
				Config:      testAccClusterLayoutConfig(selfRole),
				ExpectError: regexp.MustCompile("staged changes"),
			},
			{
				PreConfig: func() { testAccRevertLayout(t, client) },
				Config:    testAccClusterLayoutConfig(selfRole),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNoLayoutRole(client, testAccFakeNodeID),
					testAccCheckLayoutRole(client, self),
					testAccCheckLayoutVersion(client, "garage_cluster_layout.test", version+2),
					resource.TestCheckResourceAttr("garage_cluster_layout.test", "roles.#", "1"),
				),
			},
			{
				Config:   testAccClusterLayoutConfig(selfRole),
				PlanOnly: true,
			},
		},
	})
}

func TestAccGarageClusterLayout_import(t *testing.T) {
	self := testAccLayoutPreCheck(t)
	config := testAccClusterLayoutConfig(testAccClusterLayoutRole(self.id, self.zone, formatCapacityExact(*self.capacity), self.tags...))

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
			},
			{
				ResourceName:      "garage_cluster_layout.test",
				ImportState:       true,
				ImportStateId:     "cluster-layout",
				ImportStateVerify: true,
			},
			{
				ResourceName:  "garage_cluster_layout.test",
				ImportState:   true,
				ImportStateId: "layout",
				ExpectError:   regexp.MustCompile(`unexpected import ID`),
			},
		},
	})
}

func TestAccGarageClusterLayout_destroy(t *testing.T) {
	self := testAccLayoutPreCheck(t)
	client := testAccClient(t)
	version := testAccGetLayout(t, client).Version
	fake := testAccLayoutRole{id: testAccFakeNodeID, zone: "acc-zone"}
	config := testAccClusterLayoutConfig(testAccClusterLayoutRole(testAccFakeNodeID, "acc-zone", ""))
	staged := strings.Repeat("b", 64)

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		// Destroy keeps the layout, and the changes that others staged.
		CheckDestroy: func(s *terraform.State) error {
			layout, err := getClusterLayoutRaw(context.Background(), client)
			if err != nil {
				return err
			}
			if layout.Version != version+1 || len(layout.StagedRoleChanges) != 1 {
				return fmt.Errorf("got layout version %d with %d staged changes, want version %d with 1", layout.Version, len(layout.StagedRoleChanges), version+1)
			}
			return resource.ComposeTestCheckFunc(
				testAccCheckLayoutRole(client, fake),
				testAccCheckLayoutRole(client, self),
			)(s)
		},
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  testAccCheckLayoutVersion(client, "garage_cluster_layout.test", version+1),
			},
			{
				PreConfig: func() {
					testAccStageLayout(t, client, nodeRoleChange{Id: staged, Zone: "acc-zone"})
				},
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}

func TestAccGarageClusterLayout_nodeLeavesLayout(t *testing.T) {
	self := testAccLayoutPreCheck(t)
	client := testAccClient(t)
	config := testAccClusterLayoutConfig(testAccClusterLayoutRole(testAccFakeNodeID, "acc-zone", ""))
	fake := testAccLayoutRole{id: testAccFakeNodeID, zone: "acc-zone"}

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				// The resource manages only the gateway, and keeps the
				// role of self.
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLayoutRole(client, fake),
					testAccCheckLayoutRole(client, self),
					resource.TestCheckResourceAttr("garage_cluster_layout.test", "roles.#", "1"),
					resource.TestCheckResourceAttr("garage_cluster_layout.test", "roles.0.capacity", ""),
				),
			},
			{
				// The gateway leaves the layout outside Terraform.
				PreConfig: func() {
					remove := true
					testAccApplyLayout(t, client, nodeRoleChange{Id: testAccFakeNodeID, Remove: &remove})
				},
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				Config: config,
				Check:  testAccCheckLayoutRole(client, fake),
			},
		},
	})
}

// TestAccGarageClusterLayout_applyFails tries to remove the storage role of
// the only storage node. Garage refuses to apply such a layout. Because a
// mistake in the provider could remove the role, the test runs only when
// GARAGE_ACC_DISPOSABLE is set, on a test server that can lose its layout.
func TestAccGarageClusterLayout_applyFails(t *testing.T) {
	if os.Getenv("GARAGE_ACC_DISPOSABLE") == "" {
		t.Skip("set GARAGE_ACC_DISPOSABLE to run this test on a disposable Garage")
	}
	self := testAccLayoutPreCheck(t)
	client := testAccClient(t)
	version := testAccGetLayout(t, client).Version

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccClusterLayoutConfig(testAccClusterLayoutRole(self.id, self.zone, "")),
				ExpectError: regexp.MustCompile("positive capacity"),
			},
			{
				// The provider reverted the change that it staged.
				PreConfig: func() {
					layout := testAccGetLayout(t, client)
					if layout.Version != version || len(layout.StagedRoleChanges) != 0 {
						t.Errorf("got layout version %d with %d staged changes, want version %d with 0", layout.Version, len(layout.StagedRoleChanges), version)
					}
				},
				Config:             testAccClusterLayoutConfig(testAccClusterLayoutRole(self.id, self.zone, "")),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// TestAccGarageClusterLayout_updateApplyFails removes the storage role of
// the only storage node and a gateway node in one update. Garage refuses to
// apply such a layout. Because a mistake in the provider could remove the
// storage role, the test runs only when GARAGE_ACC_DISPOSABLE is set, on a
// test server that can lose its layout.
func TestAccGarageClusterLayout_updateApplyFails(t *testing.T) {
	if os.Getenv("GARAGE_ACC_DISPOSABLE") == "" {
		t.Skip("set GARAGE_ACC_DISPOSABLE to run this test on a disposable Garage")
	}
	self := testAccLayoutPreCheck(t)
	client := testAccClient(t)
	version := testAccGetLayout(t, client).Version
	selfRole := testAccClusterLayoutRole(self.id, self.zone, strconv.FormatInt(*self.capacity, 10), self.tags...)
	config := testAccClusterLayoutConfig(selfRole, testAccClusterLayoutRole(testAccFakeNodeID, "acc-zone", ""))

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				// The fake gateway is added in a new layout version.
				Config: config,
				Check:  testAccCheckLayoutVersion(client, "garage_cluster_layout.test", version+1),
			},
			{
				// The update removes the fake node and the last storage role.
				Config:      testAccClusterLayoutConfig(testAccClusterLayoutRole(self.id, self.zone, "")),
				ExpectError: regexp.MustCompile("positive capacity"),
			},
			{
				// The failed update restored the prior state, including the
				// fake node that a refresh alone cannot recover, thus the
				// original configuration plans no change. The provider also
				// reverted the staged changes.
				PreConfig: func() {
					layout := testAccGetLayout(t, client)
					if layout.Version != version+1 || len(layout.StagedRoleChanges) != 0 {
						t.Errorf("got layout version %d with %d staged changes, want version %d with 0", layout.Version, len(layout.StagedRoleChanges), version+1)
					}
				},
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}
