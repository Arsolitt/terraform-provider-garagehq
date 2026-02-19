package main

import (
	"context"
	"fmt"
	"net/http"

	garage "git.deuxfleurs.fr/garage-sdk/garage-admin-sdk-golang"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceGarageClusterLayout() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceGarageClusterLayoutCreate,
		ReadContext:   resourceGarageClusterLayoutRead,
		UpdateContext: resourceGarageClusterLayoutUpdate,
		DeleteContext: resourceGarageClusterLayoutDelete,
		Schema: map[string]*schema.Schema{
			"roles": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "List of node roles in the cluster layout",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Node ID",
						},
						"zone": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Zone assigned to the node",
						},
						"capacity": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Capacity in bytes (omit for gateway nodes)",
						},
						"tags": {
							Type:        schema.TypeList,
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Description: "Tags assigned to the node",
						},
					},
				},
			},
			"version": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Current layout version",
			},
		},
	}
}

func resourceGarageClusterLayoutCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)

	roles := d.Get("roles").([]interface{})
	nodeRoles := make([]garage.NodeRoleChangeRequest, len(roles))

	for i, role := range roles {
		r := role.(map[string]interface{})
		nodeRole := garage.NewNodeRoleChangeRequestOneOf1(
			expandStringList(r["tags"].([]interface{})),
			r["zone"].(string),
			r["id"].(string),
		)
		if capacity, ok := r["capacity"].(int); ok && capacity > 0 {
			nodeRole.SetCapacity(int64(capacity))
		}
		nodeRoles[i] = garage.NodeRoleChangeRequestOneOf1AsNodeRoleChangeRequest(nodeRole)
	}

	req := garage.NewUpdateClusterLayoutRequest()
	req.SetRoles(nodeRoles)

	layout, resp, err := client.Client.ClusterLayoutAPI.UpdateClusterLayout(client.WithAuth(ctx)).UpdateClusterLayoutRequest(*req).Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to update cluster layout: %w", err))
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	// Apply the layout changes
	applyReq := garage.NewApplyClusterLayoutRequest(int64(layout.GetVersion()))
	_, resp, err = client.Client.ClusterLayoutAPI.ApplyClusterLayout(client.WithAuth(ctx)).ApplyClusterLayoutRequest(*applyReq).Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to apply cluster layout: %w", err))
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	d.SetId("cluster-layout")
	if err := d.Set("version", layout.GetVersion()); err != nil {
		return diag.FromErr(err)
	}

	return resourceGarageClusterLayoutRead(ctx, d, m)
}

func resourceGarageClusterLayoutRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)

	layout, resp, err := client.Client.ClusterLayoutAPI.GetClusterLayout(client.WithAuth(ctx)).Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to get cluster layout: %w", err))
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if err := d.Set("version", layout.GetVersion()); err != nil {
		return diag.FromErr(err)
	}

	// Read current roles from the layout
	existingRoles := d.Get("roles").([]interface{})
	currentRoles := make([]map[string]interface{}, 0)

	for _, role := range existingRoles {
		r := role.(map[string]interface{})
		nodeID := r["id"].(string)

		// Check if node still exists in layout
		for _, node := range layout.GetRoles() {
			if node.GetId() == nodeID {
				currentRole := map[string]interface{}{
					"id":   node.GetId(),
					"zone": node.GetZone(),
				}
				if capacity, ok := node.GetCapacityOk(); ok {
					currentRole["capacity"] = int(*capacity)
				}
				if tags := node.GetTags(); len(tags) > 0 {
					currentRole["tags"] = tags
				}
				currentRoles = append(currentRoles, currentRole)
				break
			}
		}
	}

	if len(currentRoles) > 0 {
		if err := d.Set("roles", currentRoles); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourceGarageClusterLayoutUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)

	if d.HasChange("roles") {
		roles := d.Get("roles").([]interface{})
		nodeRoles := make([]garage.NodeRoleChangeRequest, len(roles))

		for i, role := range roles {
			r := role.(map[string]interface{})
			nodeRole := garage.NewNodeRoleChangeRequestOneOf1(
				expandStringList(r["tags"].([]interface{})),
				r["zone"].(string),
				r["id"].(string),
			)
			if capacity, ok := r["capacity"].(int); ok && capacity > 0 {
				nodeRole.SetCapacity(int64(capacity))
			}
			nodeRoles[i] = garage.NodeRoleChangeRequestOneOf1AsNodeRoleChangeRequest(nodeRole)
		}

		req := garage.NewUpdateClusterLayoutRequest()
		req.SetRoles(nodeRoles)

		layout, resp, err := client.Client.ClusterLayoutAPI.UpdateClusterLayout(client.WithAuth(ctx)).UpdateClusterLayoutRequest(*req).Execute()
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to update cluster layout: %w", err))
		}
		defer func() {
			if resp.Body != nil {
				_ = resp.Body.Close()
			}
		}()

		// Apply the layout changes
		applyReq := garage.NewApplyClusterLayoutRequest(int64(layout.GetVersion()))
		_, resp, err = client.Client.ClusterLayoutAPI.ApplyClusterLayout(client.WithAuth(ctx)).ApplyClusterLayoutRequest(*applyReq).Execute()
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to apply cluster layout: %w", err))
		}
		defer func() {
			if resp.Body != nil {
				_ = resp.Body.Close()
			}
		}()
	}

	return resourceGarageClusterLayoutRead(ctx, d, m)
}

func resourceGarageClusterLayoutDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Cluster layout is a singleton resource, we can't really "delete" it
	// Instead, we revert any staged changes
	client := m.(*GarageClient)

	_, resp, err := client.Client.ClusterLayoutAPI.RevertClusterLayout(client.WithAuth(ctx)).Execute()
	if err != nil {
		if resp != nil && resp.StatusCode != http.StatusNotFound {
			return diag.FromErr(fmt.Errorf("failed to revert cluster layout: %w", err))
		}
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	d.SetId("")
	return nil
}
