package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	garage "git.deuxfleurs.fr/garage-sdk/garage-admin-sdk-golang"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceGarageAdminToken() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceGarageAdminTokenCreate,
		ReadContext:   resourceGarageAdminTokenRead,
		UpdateContext: resourceGarageAdminTokenUpdate,
		DeleteContext: resourceGarageAdminTokenDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the admin API token",
			},
			"scope": {
				Type:        schema.TypeList,
				Required:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Scope of the admin API token (list of endpoint names or '*' for all)",
			},
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Identifier of the admin token",
			},
			"secret_token": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "The secret bearer token (only available on create)",
			},
			"expiration": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRFC3339,
				Description:  "Expiration time (RFC3339 format)",
			},
			"never_expires": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Set the admin token to never expire",
			},
			"expired": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether this admin token is expired",
			},
			"created": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Creation date",
			},
		},
	}
}

func validateRFC3339(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if value == "" {
		return
	}
	_, err := time.Parse(time.RFC3339, value)
	if err != nil {
		errors = append(errors, fmt.Errorf("%q must be in RFC3339 format", k))
	}
	return
}

func resourceGarageAdminTokenCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)

	req := garage.NewUpdateAdminTokenRequestBody()
	req.SetName(d.Get("name").(string))

	scope := expandStringList(d.Get("scope").([]interface{}))
	req.SetScope(scope)

	if v, ok := d.GetOk("expiration"); ok {
		t, err := time.Parse(time.RFC3339, v.(string))
		if err != nil {
			return diag.FromErr(fmt.Errorf("invalid expiration format: %w", err))
		}
		req.SetExpiration(t)
	}

	if v, ok := d.GetOk("never_expires"); ok && v.(bool) {
		req.SetNeverExpires(true)
	}

	token, resp, err := client.Client.AdminAPITokenAPI.CreateAdminToken(client.WithAuth(ctx)).UpdateAdminTokenRequestBody(*req).Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create admin token: %w", err))
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if token.HasId() {
		d.SetId(token.GetId())
	} else {
		d.SetId(token.GetName())
	}

	if err := d.Set("id", token.GetId()); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("name", token.GetName()); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("scope", token.GetScope()); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("secret_token", token.GetSecretToken()); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("expired", token.GetExpired()); err != nil {
		return diag.FromErr(err)
	}
	if token.HasCreated() {
		if err := d.Set("created", token.GetCreated().Format(time.RFC3339)); err != nil {
			return diag.FromErr(err)
		}
	}
	if token.HasExpiration() {
		if err := d.Set("expiration", token.GetExpiration().Format(time.RFC3339)); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourceGarageAdminTokenRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)
	tokenID := d.Id()

	token, resp, err := client.Client.AdminAPITokenAPI.GetAdminTokenInfo(client.WithAuth(ctx)).Id(tokenID).Execute()
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("failed to read admin token: %w", err))
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if token.HasId() {
		if err := d.Set("id", token.GetId()); err != nil {
			return diag.FromErr(err)
		}
	}
	if err := d.Set("name", token.GetName()); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("scope", token.GetScope()); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("expired", token.GetExpired()); err != nil {
		return diag.FromErr(err)
	}
	if token.HasCreated() {
		if err := d.Set("created", token.GetCreated().Format(time.RFC3339)); err != nil {
			return diag.FromErr(err)
		}
	}
	if token.HasExpiration() {
		if err := d.Set("expiration", token.GetExpiration().Format(time.RFC3339)); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourceGarageAdminTokenUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)
	tokenID := d.Id()

	req := garage.NewUpdateAdminTokenRequestBody()

	if d.HasChange("name") {
		req.SetName(d.Get("name").(string))
	}

	if d.HasChange("scope") {
		scope := expandStringList(d.Get("scope").([]interface{}))
		req.SetScope(scope)
	}

	if d.HasChange("expiration") {
		if v, ok := d.GetOk("expiration"); ok {
			t, err := time.Parse(time.RFC3339, v.(string))
			if err != nil {
				return diag.FromErr(fmt.Errorf("invalid expiration format: %w", err))
			}
			req.SetExpiration(t)
		}
	}

	if d.HasChange("never_expires") {
		if v, ok := d.GetOk("never_expires"); ok && v.(bool) {
			req.SetNeverExpires(true)
		}
	}

	_, resp, err := client.Client.AdminAPITokenAPI.UpdateAdminToken(client.WithAuth(ctx)).Id(tokenID).UpdateAdminTokenRequestBody(*req).Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to update admin token: %w", err))
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	return resourceGarageAdminTokenRead(ctx, d, m)
}

func resourceGarageAdminTokenDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)
	tokenID := d.Id()

	resp, err := client.Client.AdminAPITokenAPI.DeleteAdminToken(client.WithAuth(ctx)).Id(tokenID).Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete admin token: %w", err))
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	d.SetId("")
	return nil
}

func expandStringList(l []interface{}) []string {
	result := make([]string, len(l))
	for i, v := range l {
		result[i] = v.(string)
	}
	return result
}
