package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	Id       string
	Zone     string
	Tags     []string
	Capacity *int64
	Remove   *bool
}

// MarshalJSON encodes a removal as `id` and `remove` only. It encodes all
// other changes with `zone` and `tags`, also when `tags` is empty, because
// Garage refuses a role without `tags`. It omits `capacity` for a gateway.
func (c nodeRoleChange) MarshalJSON() ([]byte, error) {
	if c.Remove != nil {
		return json.Marshal(struct {
			Id     string `json:"id"`
			Remove bool   `json:"remove"`
		}{c.Id, *c.Remove})
	}
	tags := c.Tags
	if tags == nil {
		tags = []string{}
	}
	return json.Marshal(struct {
		Id       string   `json:"id"`
		Zone     string   `json:"zone"`
		Tags     []string `json:"tags"`
		Capacity *int64   `json:"capacity,omitempty"`
	}{c.Id, c.Zone, tags, c.Capacity})
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

// clusterLayoutResponse is a custom response type for cluster layout. The
// SDK cannot decode a response with staged role changes, because it
// expects `remove` in each change.
type clusterLayoutResponse struct {
	Version           int64                  `json:"version"`
	Roles             []layoutNodeRole       `json:"roles"`
	StagedRoleChanges []stagedRoleChange     `json:"stagedRoleChanges,omitempty"`
	StagedParameters  map[string]interface{} `json:"stagedParameters,omitempty"`
}

// updateClusterLayoutRequest is a custom request type for cluster layout updates.
type updateClusterLayoutRequest struct {
	Roles []nodeRoleChange `json:"roles,omitempty"`
}

// updateClusterLayoutRaw sends a cluster layout update request using raw JSON.
// This works around the Garage v2.2 oneOf validation issue where the server
// expects the "remove" field even for add/update operations.
func updateClusterLayoutRaw(ctx context.Context, client *GarageClient, roles []nodeRoleChange) (*clusterLayoutResponse, error) {
	return clusterLayoutRaw(ctx, client, http.MethodPost, "UpdateClusterLayout", updateClusterLayoutRequest{Roles: roles})
}

// getClusterLayoutRaw gets the cluster layout using raw JSON.
func getClusterLayoutRaw(ctx context.Context, client *GarageClient) (*clusterLayoutResponse, error) {
	return clusterLayoutRaw(ctx, client, http.MethodGet, "GetClusterLayout", nil)
}

// clusterLayoutRaw sends a request with the JSON of reqBody, if not nil, to
// an endpoint that returns the cluster layout.
func clusterLayoutRaw(ctx context.Context, client *GarageClient, method, endpoint string, reqBody interface{}) (*clusterLayoutResponse, error) {
	var reqReader io.Reader
	if reqBody != nil {
		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		reqReader = bytes.NewReader(jsonData)
	}

	url := fmt.Sprintf("%s://%s/v2/%s", client.Scheme, client.Host, endpoint)
	req, err := http.NewRequestWithContext(ctx, method, url, reqReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
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

	version, err := applyClusterLayoutRoles(ctx, client, buildNodeRoles(d.Get("roles").([]interface{})), nil)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("cluster-layout")
	if err := d.Set("version", version); err != nil {
		return diag.FromErr(err)
	}

	return resourceGarageClusterLayoutRead(ctx, d, m)
}

// applyClusterLayoutRoles stages the roles in desired that are different
// from the current layout, and applies them as the next layout version. It
// does not apply a new version when all roles are already current. It
// returns the layout version after the change.
//
// It also removes the nodes in removed from the layout. It ignores a node
// in removed that is not in the layout.
//
// It refuses to continue when the layout already has staged changes,
// because the apply would also apply these changes. If the apply fails,
// it reverts the changes that it staged.
func applyClusterLayoutRoles(ctx context.Context, client *GarageClient, desired []nodeRoleChange, removed []string) (int64, error) {
	layout, err := getClusterLayoutRaw(ctx, client)
	if err != nil {
		return 0, fmt.Errorf("failed to get cluster layout: %w", err)
	}
	if len(layout.StagedRoleChanges) > 0 || layout.StagedParameters != nil {
		return 0, fmt.Errorf("the cluster layout has staged changes that are not applied; apply or revert them first (`garage layout show`)")
	}

	current := make(map[string]layoutNodeRole, len(layout.Roles))
	for _, role := range layout.Roles {
		current[role.Id] = role
	}
	var changes []nodeRoleChange
	for _, role := range desired {
		if cur, ok := current[role.Id]; ok && layoutRoleEqual(cur, role) {
			continue
		}
		changes = append(changes, role)
	}
	remove := true
	for _, id := range removed {
		if _, ok := current[id]; ok {
			changes = append(changes, nodeRoleChange{Id: id, Remove: &remove})
		}
	}
	if len(changes) == 0 {
		return layout.Version, nil
	}

	if _, err := updateClusterLayoutRaw(ctx, client, changes); err != nil {
		return 0, fmt.Errorf("failed to update cluster layout: %w", err)
	}

	// Apply the layout changes - version must be current + 1
	version := layout.Version + 1
	applyReq := garage.NewApplyClusterLayoutRequest(version)
	_, resp, err := client.Client.ClusterLayoutAPI.ApplyClusterLayout(client.WithAuth(ctx)).ApplyClusterLayoutRequest(*applyReq).Execute()
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		err = fmt.Errorf("failed to apply cluster layout: %w", garageAPIError(err))
		_, resp, revertErr := client.Client.ClusterLayoutAPI.RevertClusterLayout(client.WithAuth(ctx)).Execute()
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		if revertErr != nil {
			return 0, fmt.Errorf("%w; failed to revert the staged changes: %v", err, revertErr)
		}
		return 0, err
	}

	return version, nil
}

// garageAPIError adds the response body of an API error to err, because the
// body holds the reason that Garage gives.
func garageAPIError(err error) error {
	var apiErr *garage.GenericOpenAPIError
	if errors.As(err, &apiErr) && len(apiErr.Body()) > 0 {
		return fmt.Errorf("%w: %s", err, apiErr.Body())
	}
	return err
}

// layoutRoleEqual tells if the role of a node in the layout is the same as
// the role that a change gives it.
func layoutRoleEqual(cur layoutNodeRole, change nodeRoleChange) bool {
	if change.Remove != nil || cur.Zone != change.Zone {
		return false
	}
	if (cur.Capacity == nil) != (change.Capacity == nil) || (cur.Capacity != nil && *cur.Capacity != *change.Capacity) {
		return false
	}
	if len(cur.Tags) != len(change.Tags) {
		return false
	}
	for i := range cur.Tags {
		if cur.Tags[i] != change.Tags[i] {
			return false
		}
	}
	return true
}

func resourceGarageClusterLayoutRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)

	layout, err := getClusterLayoutRaw(ctx, client)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to get cluster layout: %w", err))
	}

	if err := d.Set("version", layout.Version); err != nil {
		return diag.FromErr(err)
	}

	current := make(map[string]layoutNodeRole, len(layout.Roles))
	for _, node := range layout.Roles {
		current[node.Id] = node
	}

	// Read the roles of the nodes in the state. A node that is not in the
	// layout is not in the result, thus Terraform shows it as drift.
	currentRoles := make([]map[string]interface{}, 0)
	for _, role := range d.Get("roles").([]interface{}) {
		r := role.(map[string]interface{})
		node, ok := current[r["id"].(string)]
		if !ok {
			continue
		}
		currentRole := map[string]interface{}{
			"id":   node.Id,
			"zone": node.Zone,
		}
		if node.Capacity != nil {
			// Keep the notation of the configuration when the value is the same.
			if c, err := ParseCapacity(r["capacity"].(string)); err == nil && c == *node.Capacity {
				currentRole["capacity"] = r["capacity"]
			} else {
				currentRole["capacity"] = formatCapacityExact(*node.Capacity)
			}
		}
		if len(node.Tags) > 0 {
			currentRole["tags"] = node.Tags
		}
		currentRoles = append(currentRoles, currentRole)
	}

	if err := d.Set("roles", currentRoles); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceGarageClusterLayoutUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)

	if d.HasChange("roles") {
		oldRoles, newRoles := d.GetChange("roles")
		desired := buildNodeRoles(newRoles.([]interface{}))
		version, err := applyClusterLayoutRoles(ctx, client, desired, removedNodeIDs(buildNodeRoles(oldRoles.([]interface{})), desired))
		if err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("version", version); err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceGarageClusterLayoutRead(ctx, d, m)
}

// removedNodeIDs returns the IDs of the nodes in oldRoles that are not in
// newRoles. These are the nodes that the configuration no longer manages.
func removedNodeIDs(oldRoles, newRoles []nodeRoleChange) []string {
	keep := make(map[string]bool, len(newRoles))
	for _, role := range newRoles {
		keep[role.Id] = true
	}
	var removed []string
	for _, role := range oldRoles {
		if !keep[role.Id] {
			removed = append(removed, role.Id)
		}
	}
	return removed
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
