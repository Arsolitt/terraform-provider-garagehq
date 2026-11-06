package main

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	garage "git.deuxfleurs.fr/garage-sdk/garage-admin-sdk-golang"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceGarageBucketKey() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceGarageBucketKeyCreate,
		ReadContext:   resourceGarageBucketKeyRead,
		UpdateContext: resourceGarageBucketKeyUpdate,
		DeleteContext: resourceGarageBucketKeyDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceGarageBucketKeyImport,
		},
		Schema: map[string]*schema.Schema{
			"bucket_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The bucket ID",
			},
			"access_key_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The access key ID",
			},
			"read": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Grant read permission",
			},
			"write": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Grant write permission",
			},
			"owner": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Grant owner permission",
			},
		},
	}
}

func resourceGarageBucketKeyCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)
	bucketID := d.Get("bucket_id").(string)
	keyID := d.Get("access_key_id").(string)
	read := d.Get("read").(bool)
	write := d.Get("write").(bool)
	owner := d.Get("owner").(bool)

	perms := garage.NewApiBucketKeyPerm()
	perms.SetRead(read)
	perms.SetWrite(write)
	perms.SetOwner(owner)

	updateReq := garage.NewBucketKeyPermChangeRequest(keyID, bucketID, *perms)

	_, resp, err := client.Client.PermissionAPI.AllowBucketKey(client.WithAuth(ctx)).Body(*updateReq).Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to update bucket key permissions: %w", err))
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	d.SetId(fmt.Sprintf("%s/%s", bucketID, keyID))

	return resourceGarageBucketKeyRead(ctx, d, m)
}

func resourceGarageBucketKeyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)
	bucketID := d.Get("bucket_id").(string)
	keyID := d.Get("access_key_id").(string)

	// Get key to check bucket associations
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

	// Find this bucket in the key's bucket list
	for _, bucket := range key.GetBuckets() {
		if bucket.GetId() == bucketID {
			perms := bucket.GetPermissions()
			if err := d.Set("read", perms.GetRead()); err != nil {
				return diag.FromErr(err)
			}
			if err := d.Set("write", perms.GetWrite()); err != nil {
				return diag.FromErr(err)
			}
			if err := d.Set("owner", perms.GetOwner()); err != nil {
				return diag.FromErr(err)
			}
			return nil
		}
	}

	// Garage lists a bucket on a key only while the key has a permission on it.
	// A resource that sets no permission therefore has no entry, and is not gone.
	if !d.Get("read").(bool) && !d.Get("write").(bool) && !d.Get("owner").(bool) {
		return nil
	}

	// Key doesn't have permissions on this bucket
	d.SetId("")
	return nil
}

// resourceGarageBucketKeyImport accepts `<bucket>/<access_key_id>`.
// `<bucket>` is a bucket ID or a global alias. The resource ID is
// `<bucket_id>/<access_key_id>`.
func resourceGarageBucketKeyImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	client := m.(*GarageClient)
	idOrAlias, keyID, ok := strings.Cut(d.Id(), "/")
	if !ok || idOrAlias == "" || keyID == "" || strings.Contains(keyID, "/") {
		return nil, fmt.Errorf("invalid import ID %q: expected <bucket>/<access_key_id>", d.Id())
	}

	bucket, err := lookupBucket(ctx, client, idOrAlias)
	if err != nil {
		return nil, err
	}
	bucketID := bucket.GetId()

	key, resp, err := client.Client.AccessKeyAPI.GetKeyInfo(client.WithAuth(ctx)).Id(keyID).Execute()
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()
	if err != nil {
		return nil, fmt.Errorf("failed to read key %s: %w", keyID, garageAPIError(err))
	}

	hasBucket := func(b garage.KeyInfoBucketResponse) bool { return b.GetId() == bucketID }
	if !slices.ContainsFunc(key.GetBuckets(), hasBucket) {
		return nil, fmt.Errorf("key %q has no permissions on bucket %q", keyID, idOrAlias)
	}

	d.SetId(fmt.Sprintf("%s/%s", bucketID, keyID))
	if err := d.Set("bucket_id", bucketID); err != nil {
		return nil, err
	}
	if err := d.Set("access_key_id", keyID); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}

func resourceGarageBucketKeyUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)
	bucketID := d.Get("bucket_id").(string)
	keyID := d.Get("access_key_id").(string)
	read := d.Get("read").(bool)
	write := d.Get("write").(bool)
	owner := d.Get("owner").(bool)

	// A request sets only the permissions that it marks true: `AllowBucketKey`
	// cannot remove a permission, and `DenyBucketKey` cannot add one. Deny first,
	// so that a failed update does not leave access that the configuration removed.
	denied := garage.NewApiBucketKeyPerm()
	denied.SetRead(!read)
	denied.SetWrite(!write)
	denied.SetOwner(!owner)

	denyReq := garage.NewBucketKeyPermChangeRequest(keyID, bucketID, *denied)

	_, resp, err := client.Client.PermissionAPI.DenyBucketKey(client.WithAuth(ctx)).Body(*denyReq).Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to remove bucket key permissions: %w", err))
	}
	if resp.Body != nil {
		_ = resp.Body.Close()
	}

	allowed := garage.NewApiBucketKeyPerm()
	allowed.SetRead(read)
	allowed.SetWrite(write)
	allowed.SetOwner(owner)

	allowReq := garage.NewBucketKeyPermChangeRequest(keyID, bucketID, *allowed)

	_, resp, err = client.Client.PermissionAPI.AllowBucketKey(client.WithAuth(ctx)).Body(*allowReq).Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to update bucket key permissions: %w", err))
	}
	if resp.Body != nil {
		_ = resp.Body.Close()
	}

	return resourceGarageBucketKeyRead(ctx, d, m)
}

func resourceGarageBucketKeyDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)
	bucketID := d.Get("bucket_id").(string)
	keyID := d.Get("access_key_id").(string)

	// Remove all permissions by denying them
	perms := garage.NewApiBucketKeyPerm()
	perms.SetRead(true)
	perms.SetWrite(true)
	perms.SetOwner(true)

	updateReq := garage.NewBucketKeyPermChangeRequest(keyID, bucketID, *perms)

	_, resp, err := client.Client.PermissionAPI.DenyBucketKey(client.WithAuth(ctx)).Body(*updateReq).Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to remove bucket key permissions: %w", err))
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	d.SetId("")
	return nil
}
