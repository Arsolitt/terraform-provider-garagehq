package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	garage "git.deuxfleurs.fr/garage-sdk/garage-admin-sdk-golang"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func validateCapacity(v interface{}, k string) (ws []string, errors []error) {
	val, ok := v.(string)
	if !ok {
		errors = append(errors, fmt.Errorf("expected type of %q to be string", k))
		return
	}
	if val == "" {
		return
	}
	_, err := ParseCapacity(val)
	if err != nil {
		errors = append(errors, fmt.Errorf("%q: %w", k, err))
	}
	return
}

// nodeRoleChange is a custom type to work around Garage v2.2 oneOf validation issue.
// The server validates oneOf schemas in order and expects "remove" field in the first schema.
type nodeRoleChange struct {
	Id       string   `json:"id"`
	Zone     string   `json:"zone,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Capacity *int64   `json:"capacity,omitempty"`
	Remove   *bool    `json:"remove,omitempty"`
}

// layoutNodeRole represents a node role in the cluster layout response.
type layoutNodeRole struct {
	Id       string   `json:"id"`
	Zone     string   `json:"zone"`
	Tags     []string `json:"tags"`
	Capacity *int64   `json:"capacity,omitempty"`
}

// stagedRoleChange represents a staged role change in the response.
type stagedRoleChange struct {
	Id       string   `json:"id"`
	Zone     string   `json:"zone,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Capacity *int64   `json:"capacity,omitempty"`
	Remove   *bool    `json:"remove,omitempty"`
}

// clusterLayoutResponse is a custom response type for cluster layout.
type clusterLayoutResponse struct {
	Version           int64              `json:"version"`
	Roles             []layoutNodeRole   `json:"roles"`
	StagedRoleChanges []stagedRoleChange `json:"stagedRoleChanges,omitempty"`
}

// updateClusterLayoutRequest is a custom request type for cluster layout updates.
type updateClusterLayoutRequest struct {
	Roles []nodeRoleChange `json:"roles,omitempty"`
}

// updateClusterLayoutRaw sends a cluster layout update request using raw JSON.
// This works around the Garage v2.2 oneOf validation issue where the server
// expects the "remove" field even for add/update operations.
func updateClusterLayoutRaw(ctx context.Context, client *GarageClient, roles []nodeRoleChange) (*clusterLayoutResponse, error) {
	reqBody := updateClusterLayoutRequest{Roles: roles}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s://%s/v2/UpdateClusterLayout", client.Scheme, client.Host)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+client.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() {
		if resp.Body != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var layout clusterLayoutResponse
	if err := json.Unmarshal(body, &layout); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &layout, nil
}

// buildNodeRoles converts terraform schema roles to nodeRoleChange slice.
func buildNodeRoles(roles []interface{}) []nodeRoleChange {
	nodeRoles := make([]nodeRoleChange, len(roles))
	for i, role := range roles {
		r := role.(map[string]interface{})
		nodeRole := nodeRoleChange{
			Id:   r["id"].(string),
			Zone: r["zone"].(string),
			Tags: expandStringList(r["tags"].([]interface{})),
		}
		if capacity, ok := r["capacity"].(string); ok && capacity != "" {
			c, err := ParseCapacity(capacity)
			if err == nil {
				nodeRole.Capacity = &c
			}
		}
		nodeRoles[i] = nodeRole
	}
	return nodeRoles
}

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
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "Storage capacity with unit suffix (e.g., 1G, 500M, 2TiB). Omit for gateway nodes.",
							ValidateFunc: validateCapacity,
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
	nodeRoles := buildNodeRoles(roles)

	layout, err := updateClusterLayoutRaw(ctx, client, nodeRoles)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to update cluster layout: %w", err))
	}

	// Apply the layout changes - version must be current + 1
	applyReq := garage.NewApplyClusterLayoutRequest(layout.Version + 1)
	_, resp, err := client.Client.ClusterLayoutAPI.ApplyClusterLayout(client.WithAuth(ctx)).ApplyClusterLayoutRequest(*applyReq).Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to apply cluster layout: %w", err))
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	d.SetId("cluster-layout")
	if err := d.Set("version", layout.Version+1); err != nil {
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
					// Try to preserve the user's original input format
					originalInput, hasOriginal := r["capacity"].(string)
					if hasOriginal && originalInput != "" {
						parsedOriginal, err := ParseCapacity(originalInput)
						if err == nil && parsedOriginal == *capacity {
							currentRole["capacity"] = originalInput
						} else {
							currentRole["capacity"] = FormatCapacity(*capacity)
						}
					} else {
						currentRole["capacity"] = FormatCapacity(*capacity)
					}
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
		nodeRoles := buildNodeRoles(roles)

		layout, err := updateClusterLayoutRaw(ctx, client, nodeRoles)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to update cluster layout: %w", err))
		}

		// Apply the layout changes - version must be current + 1
		applyReq := garage.NewApplyClusterLayoutRequest(layout.Version + 1)
		_, resp, err := client.Client.ClusterLayoutAPI.ApplyClusterLayout(client.WithAuth(ctx)).ApplyClusterLayoutRequest(*applyReq).Execute()
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
