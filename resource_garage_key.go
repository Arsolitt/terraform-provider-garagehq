package main

import (
	"context"
	"fmt"
	"net/http"

	garage "git.deuxfleurs.fr/garage-sdk/garage-admin-sdk-golang"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceGarageKey() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceGarageKeyCreate,
		ReadContext:   resourceGarageKeyRead,
		UpdateContext: resourceGarageKeyUpdate,
		DeleteContext: resourceGarageKeyDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the access key",
			},
			"access_key_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The access key ID",
			},
			"secret_access_key": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "The secret access key (only available on create)",
			},
		},
	}
}

func resourceGarageKeyCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)
	name := d.Get("name").(string)

	keyReq := garage.NewUpdateKeyRequestBody()
	keyReq.SetName(name)

	key, resp, err := client.Client.AccessKeyAPI.CreateKey(client.WithAuth(ctx)).Body(*keyReq).Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create key: %w", err))
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	d.SetId(key.GetAccessKeyId())
	if err := d.Set("access_key_id", key.GetAccessKeyId()); err != nil {
		return diag.FromErr(err)
	}
	if secret, ok := key.GetSecretAccessKeyOk(); ok && secret != nil {
		if err := d.Set("secret_access_key", *secret); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourceGarageKeyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)
	keyID := d.Id()

	key, resp, err := client.Client.AccessKeyAPI.GetKeyInfo(client.WithAuth(ctx)).Id(keyID).Execute()
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("failed to read key: %w", err))
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if err := d.Set("access_key_id", key.GetAccessKeyId()); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("name", key.GetName()); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceGarageKeyUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)
	keyID := d.Id()

	if d.HasChange("name") {
		name := d.Get("name").(string)
		updateReq := garage.NewUpdateKeyRequestBody()
		updateReq.SetName(name)

		_, resp, err := client.Client.AccessKeyAPI.UpdateKey(client.WithAuth(ctx)).Id(keyID).UpdateKeyRequestBody(*updateReq).Execute()
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to update key: %w", err))
		}
		defer func() {
			if resp.Body != nil {
				_ = resp.Body.Close()
			}
		}()
	}

	return resourceGarageKeyRead(ctx, d, m)
}

func resourceGarageKeyDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)
	keyID := d.Id()

	resp, err := client.Client.AccessKeyAPI.DeleteKey(client.WithAuth(ctx)).Id(keyID).Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete key: %w", err))
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	d.SetId("")
	return nil
}
