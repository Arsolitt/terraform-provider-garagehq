package main

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// nodeIDRegex matches a full node ID, as the Garage admin API gives it.
var nodeIDRegex = regexp.MustCompile(`^[0-9a-f]{64}$`)

func dataSourceGarageNode() *schema.Resource {
	return &schema.Resource{
		Description: "Reads a node of the Garage cluster, by default the node that serves the admin API.",
		ReadContext: dataSourceGarageNodeRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ValidateFunc: validation.StringMatch(
					nodeIDRegex,
					"must be 64 lowercase hexadecimal characters",
				),
				Description: "Node ID (64 lowercase hexadecimal characters). Defaults to the ID of the node that serves the admin API.",
			},
			"hostname": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Hostname of the node",
			},
			"address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "RPC address of the node",
			},
			"garage_version": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Garage version of the node",
			},
			"is_up": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the node is connected to the cluster",
			},
			"draining": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the node has no role in the newest layout, but still holds data of an older layout version",
			},
			"zone": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Zone of the node in the current layout. Empty if the node has no role.",
			},
			"capacity": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Storage capacity of the node in the current layout. Empty for a gateway node or a node without a role.",
			},
			"tags": {
				Type:        schema.TypeList,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Tags of the node in the current layout",
			},
		},
	}
}

func dataSourceGarageNodeRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)

	nodeID := d.Get("id").(string)
	if nodeID == "" {
		id, err := garageSelfNodeID(ctx, client)
		if err != nil {
			return diag.FromErr(err)
		}
		nodeID = id
	}

	status, resp, err := client.Client.ClusterAPI.GetClusterStatus(client.WithAuth(ctx)).Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to get cluster status: %w", err))
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	for _, node := range status.GetNodes() {
		if node.GetId() != nodeID {
			continue
		}

		d.SetId(node.GetId())
		if err := d.Set("hostname", node.GetHostname()); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("address", node.GetAddr()); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("garage_version", node.GetGarageVersion()); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("is_up", node.GetIsUp()); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("draining", node.GetDraining()); err != nil {
			return diag.FromErr(err)
		}

		zone, capacity, tags := "", "", []string{}
		if role, ok := node.GetRoleOk(); ok && role != nil {
			zone = role.GetZone()
			if c, ok := role.GetCapacityOk(); ok && c != nil {
				capacity = formatCapacityExact(*c)
			}
			tags = role.GetTags()
		}
		if err := d.Set("zone", zone); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("capacity", capacity); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("tags", tags); err != nil {
			return diag.FromErr(err)
		}

		return nil
	}

	return diag.Errorf("node %q not found in the cluster", nodeID)
}

// garageSelfNodeID returns the ID of the node that serves the admin API.
func garageSelfNodeID(ctx context.Context, client *GarageClient) (string, error) {
	info, resp, err := client.Client.NodeAPI.GetNodeInfo(client.WithAuth(ctx)).Node("self").Execute()
	if err != nil {
		return "", fmt.Errorf("failed to get node info: %w", err)
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	for _, node := range info.GetSuccess() {
		return node.GetNodeId(), nil
	}
	for id, msg := range info.GetError() {
		return "", fmt.Errorf("failed to get node info of %s: %s", id, msg)
	}
	return "", fmt.Errorf("failed to get node info: empty response")
}
