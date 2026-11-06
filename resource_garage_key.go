package main

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	garage "git.deuxfleurs.fr/garage-sdk/garage-admin-sdk-golang"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// Garage accepts these formats for an imported key, see `Key::import` in
// https://git.deuxfleurs.fr/Deuxfleurs/garage/src/tag/v2.3.0/src/model/key_table.rs
// Update them when Garage changes these rules. Keys that Garage generates
// have the narrower form `GK` + 24 hex digits and 64 hex digits.
var (
	accessKeyIDLengthRegex     = regexp.MustCompile(`(?s)^.{8,}$`)
	accessKeyIDCharsRegex      = regexp.MustCompile(`^[A-Za-z0-9._-]*$`)
	secretAccessKeyLengthRegex = regexp.MustCompile(`(?s)^.{16,}$`)
	secretAccessKeyCharsRegex  = regexp.MustCompile(`^[\x21-\x7e]*$`)
)

func resourceGarageKey() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceGarageKeyCreate,
		ReadContext:   resourceGarageKeyRead,
		UpdateContext: resourceGarageKeyUpdate,
		DeleteContext: resourceGarageKeyDelete,
		CustomizeDiff: resourceGarageKeyCustomizeDiff,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the access key",
			},
			"access_key_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				RequiredWith: []string{"secret_access_key"},
				ValidateDiagFunc: validation.ToDiagFunc(validation.All(
					validation.StringMatch(accessKeyIDLengthRegex, "must be at least 8 characters"),
					validation.StringMatch(accessKeyIDCharsRegex, "must contain only ASCII letters, digits, '-', '_' or '.'"),
				)),
				Description: "The access key ID. Set it together with `secret_access_key` to import a key with known credentials; otherwise Garage generates it",
			},
			"secret_access_key": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				Sensitive:    true,
				RequiredWith: []string{"access_key_id"},
				ValidateDiagFunc: validation.ToDiagFunc(validation.All(
					validation.StringMatch(secretAccessKeyLengthRegex, "must be at least 16 characters"),
					validation.StringMatch(secretAccessKeyCharsRegex, "must contain only graphic ASCII characters (U+0021 to U+007E)"),
				)),
				Description: "The secret access key. Set it together with `access_key_id` to import a key with known credentials; otherwise Garage generates it (the provider records it only at create)",
			},
		},
	}
}

// resourceGarageKeyCustomizeDiff rejects a new secret for an existing access
// key ID. The replacement deletes the key first, and Garage then refuses to
// import the same ID again.
func resourceGarageKeyCustomizeDiff(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
	if d.Id() != "" && d.HasChange("secret_access_key") && !d.HasChange("access_key_id") {
		return fmt.Errorf("cannot change secret_access_key of key %s: Garage does not import a deleted access key ID again, so also set a new access_key_id", d.Id())
	}
	return nil
}

func resourceGarageKeyCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)
	name := d.Get("name").(string)
	accessKeyID := d.Get("access_key_id").(string)
	secretAccessKey := d.Get("secret_access_key").(string)

	var key *garage.GetKeyInfoResponse
	var resp *http.Response
	var err error
	// Use the configuration, not the values, so that an empty resolved pair
	// cannot silently fall back to generated credentials.
	config := d.GetRawConfig()
	configured := !config.IsNull() &&
		(!config.GetAttr("access_key_id").IsNull() || !config.GetAttr("secret_access_key").IsNull())
	if configured {
		if accessKeyID == "" || secretAccessKey == "" {
			return diag.Errorf("access_key_id and secret_access_key must both be set")
		}
		importReq := garage.NewImportKeyRequest(accessKeyID, secretAccessKey)
		importReq.SetName(name)

		key, resp, err = client.Client.AccessKeyAPI.ImportKey(client.WithAuth(ctx)).ImportKeyRequest(*importReq).Execute()
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to import key: %w", garageAPIError(err)))
		}
	} else {
		keyReq := garage.NewUpdateKeyRequestBody()
		keyReq.SetName(name)

		key, resp, err = client.Client.AccessKeyAPI.CreateKey(client.WithAuth(ctx)).Body(*keyReq).Execute()
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to create key: %w", err))
		}
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
