package main

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"slices"
	"strconv"

	garage "git.deuxfleurs.fr/garage-sdk/garage-admin-sdk-golang"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceGarageBucket() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceGarageBucketCreate,
		ReadContext:   resourceGarageBucketRead,
		UpdateContext: resourceGarageBucketUpdate,
		DeleteContext: resourceGarageBucketDelete,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The bucket ID (computed if not provided)",
			},
			"global_alias": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Global alias for the bucket (this appears as the name in garage bucket list). A change adds the new alias, then removes the old alias. Garage does not remove the last alias of a bucket, so to remove the global alias, the bucket must have another global alias or a local alias.",
			},
			"bytes": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Total number of bytes used by objects in this bucket",
			},
			"objects": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of objects in this bucket",
			},
			"expiration_days": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Number of days after which objects in this bucket will be automatically deleted. Set to 0 to disable expiration.",
			},
			"website_access": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Website hosting configuration for the bucket. When enabled, the bucket is served over the Garage S3 web endpoint.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "Whether website access is enabled for this bucket.",
						},
						"index_document": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Name of the index document served at the root of the website (e.g. index.html).",
						},
						"error_document": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Name of the error document served on errors (e.g. error.html).",
						},
					},
				},
			},
		},
	}
}

func resourceGarageBucketCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)
	globalAlias := d.Get("global_alias").(string)

	bucketReq := garage.NewCreateBucketRequest()
	if globalAlias != "" {
		bucketReq.SetGlobalAlias(globalAlias)
	}

	bucket, resp, err := client.Client.BucketAPI.CreateBucket(client.WithAuth(ctx)).CreateBucketRequest(*bucketReq).Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create bucket: %w", err))
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	d.SetId(bucket.GetId())
	if err := d.Set("id", bucket.GetId()); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("bytes", bucket.GetBytes()); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("objects", bucket.GetObjects()); err != nil {
		return diag.FromErr(err)
	}

	// Set expiration policy if specified
	if expirationDays, ok := d.GetOk("expiration_days"); ok && expirationDays.(int) > 0 {
		if err := setBucketLifecyclePolicy(ctx, client, bucket.GetId(), expirationDays.(int)); err != nil {
			return diag.FromErr(fmt.Errorf("failed to set expiration policy: %w", err))
		}
		if err := d.Set("expiration_days", expirationDays.(int)); err != nil {
			return diag.FromErr(err)
		}
	}

	// Set website access configuration if specified
	if _, ok := d.GetOk("website_access"); ok {
		if err := setBucketWebsiteAccess(ctx, client, bucket.GetId(), d.Get("website_access").([]interface{})); err != nil {
			return diag.FromErr(fmt.Errorf("failed to set website access: %w", err))
		}
	}

	return resourceGarageBucketRead(ctx, d, m)
}

func resourceGarageBucketRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)
	bucketID := d.Id()

	bucket, resp, err := client.Client.BucketAPI.GetBucketInfo(client.WithAuth(ctx)).Id(bucketID).Execute()
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("failed to read bucket: %w", err))
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if err := d.Set("id", bucket.GetId()); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("bytes", bucket.GetBytes()); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("objects", bucket.GetObjects()); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("global_alias", selectGlobalAlias(bucket.GetGlobalAliases(), d.Get("global_alias").(string))); err != nil {
		return diag.FromErr(err)
	}

	// Read expiration policy if it exists
	expirationDays, err := getBucketLifecyclePolicy(ctx, client, bucket.GetId())
	if err == nil && expirationDays > 0 {
		if err := d.Set("expiration_days", expirationDays); err != nil {
			return diag.FromErr(err)
		}
	} else if d.HasChange("expiration_days") {
		// If expiration_days was removed, clear it
		if err := d.Set("expiration_days", 0); err != nil {
			return diag.FromErr(err)
		}
	}

	// Read website access configuration
	if bucket.GetWebsiteAccess() {
		websiteAccess := map[string]interface{}{
			"enabled": true,
		}
		if config, ok := bucket.GetWebsiteConfigOk(); ok && config != nil {
			websiteAccess["index_document"] = config.GetIndexDocument()
			websiteAccess["error_document"] = config.GetErrorDocument()
		}
		if err := d.Set("website_access", []interface{}{websiteAccess}); err != nil {
			return diag.FromErr(err)
		}
	} else {
		if err := d.Set("website_access", []interface{}{}); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourceGarageBucketUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)
	bucketID := d.Id()

	// Handle global alias changes
	if d.HasChange("global_alias") {
		oldAlias, newAlias := d.GetChange("global_alias")
		if err := updateBucketGlobalAlias(ctx, client, bucketID, oldAlias.(string), newAlias.(string)); err != nil {
			// Keep the prior state: without this, the SDK records the
			// configured alias although Garage did not apply it.
			d.Partial(true)
			return diag.FromErr(fmt.Errorf("failed to update global alias: %w", err))
		}
	}

	// Handle expiration policy changes
	if d.HasChange("expiration_days") {
		expirationDays := d.Get("expiration_days").(int)
		if expirationDays > 0 {
			if err := setBucketLifecyclePolicy(ctx, client, bucketID, expirationDays); err != nil {
				return diag.FromErr(fmt.Errorf("failed to update expiration policy: %w", err))
			}
		} else {
			// Remove lifecycle policy if expiration_days is 0 or removed
			if err := deleteBucketLifecyclePolicy(ctx, client, bucketID); err != nil {
				return diag.FromErr(fmt.Errorf("failed to remove expiration policy: %w", err))
			}
		}
	}

	// Handle website access changes
	if d.HasChange("website_access") {
		if err := setBucketWebsiteAccess(ctx, client, bucketID, d.Get("website_access").([]interface{})); err != nil {
			return diag.FromErr(fmt.Errorf("failed to update website access: %w", err))
		}
	}

	return resourceGarageBucketRead(ctx, d, m)
}

func resourceGarageBucketDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)
	bucketID := d.Id()

	resp, err := client.Client.BucketAPI.DeleteBucket(client.WithAuth(ctx)).Id(bucketID).Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete bucket: %w", err))
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	d.SetId("")
	return nil
}

// selectGlobalAlias returns the global alias to record in state: configured
// if the bucket has it, else "". The provider manages only the configured
// alias, so other global aliases of the bucket have no effect on the state.
func selectGlobalAlias(aliases []string, configured string) string {
	if slices.Contains(aliases, configured) {
		return configured
	}
	return ""
}

// updateBucketGlobalAlias replaces the global alias oldAlias of a bucket with
// newAlias. An empty alias means no alias. The new alias is added before the
// old alias is removed, because Garage refuses to remove the last alias of a
// bucket.
func updateBucketGlobalAlias(ctx context.Context, client *GarageClient, bucketID, oldAlias, newAlias string) error {
	if newAlias != "" {
		if err := addBucketGlobalAlias(ctx, client, bucketID, newAlias); err != nil {
			return fmt.Errorf("failed to add global alias %q: %w", newAlias, err)
		}
	}

	if oldAlias != "" {
		status, err := removeBucketGlobalAlias(ctx, client, bucketID, oldAlias)
		if err != nil {
			if newAlias == "" && status == http.StatusBadRequest {
				return fmt.Errorf("failed to remove global alias %q: %w: Garage does not remove the last alias of a bucket, so give the bucket another global alias or a local alias first, or delete the bucket", oldAlias, err)
			}
			return fmt.Errorf("failed to remove global alias %q: %w", oldAlias, err)
		}
	}

	return nil
}

// addBucketGlobalAlias adds a global alias to a bucket. Garage accepts an
// alias that the bucket already has.
func addBucketGlobalAlias(ctx context.Context, client *GarageClient, bucketID, alias string) error {
	aliasEnum := garage.BucketAliasEnumOneOfAsBucketAliasEnum(garage.NewBucketAliasEnumOneOf(bucketID, alias))

	_, resp, err := client.Client.BucketAliasAPI.AddBucketAlias(client.WithAuth(ctx)).BucketAliasEnum(aliasEnum).Execute()
	if err != nil {
		return garageAPIError(err)
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	return nil
}

// removeBucketGlobalAlias removes a global alias from a bucket. Removing an
// alias that the bucket does not have succeeds, so a retry after a partially
// applied change reaches the target state. It also returns the HTTP status
// code of the removal request, or 0 if there is no response.
func removeBucketGlobalAlias(ctx context.Context, client *GarageClient, bucketID, alias string) (int, error) {
	aliasEnum := garage.BucketAliasEnumOneOfAsBucketAliasEnum(garage.NewBucketAliasEnumOneOf(bucketID, alias))

	_, resp, err := client.Client.BucketAliasAPI.RemoveBucketAlias(client.WithAuth(ctx)).BucketAliasEnum(aliasEnum).Execute()
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()
	status := 0
	if resp != nil {
		status = resp.StatusCode
	}
	if err == nil {
		return status, nil
	}
	err = garageAPIError(err)

	// Garage fails to remove an alias that is missing or belongs to another
	// bucket, so the alias list decides whether the error matters.
	aliases, probeErr := bucketGlobalAliases(ctx, client, bucketID)
	if probeErr == nil && !slices.Contains(aliases, alias) {
		return status, nil
	}

	return status, err
}

// bucketGlobalAliases returns the global aliases of a bucket.
func bucketGlobalAliases(ctx context.Context, client *GarageClient, bucketID string) ([]string, error) {
	bucket, resp, err := client.Client.BucketAPI.GetBucketInfo(client.WithAuth(ctx)).Id(bucketID).Execute()
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	return bucket.GetGlobalAliases(), nil
}

// buildWebsiteAccess converts the website_access schema block into an
// UpdateBucketWebsiteAccess payload. Garage requires indexDocument when
// website access is enabled, and rejects any documents when it is disabled,
// so documents are validated and sent accordingly.
func buildWebsiteAccess(config []interface{}) (*garage.UpdateBucketWebsiteAccess, error) {
	websiteAccess := garage.NewUpdateBucketWebsiteAccess(false)

	if len(config) > 0 && config[0] != nil {
		cfg := config[0].(map[string]interface{})
		enabled := cfg["enabled"].(bool)
		websiteAccess.SetEnabled(enabled)

		if enabled {
			index, _ := cfg["index_document"].(string)
			if index == "" {
				return nil, fmt.Errorf("index_document is required when website access is enabled")
			}
			websiteAccess.SetIndexDocument(index)
			if v, ok := cfg["error_document"].(string); ok && v != "" {
				websiteAccess.SetErrorDocument(v)
			}
		}
		// When disabled, index_document/error_document are intentionally not
		// sent: Garage rejects any documents alongside enabled=false.
	}

	return websiteAccess, nil
}

// setBucketWebsiteAccess applies the website hosting configuration for a bucket
// via the admin API UpdateBucket endpoint. An empty configuration list disables
// website access.
func setBucketWebsiteAccess(ctx context.Context, client *GarageClient, bucketID string, config []interface{}) error {
	websiteAccess, err := buildWebsiteAccess(config)
	if err != nil {
		return err
	}

	updateReq := garage.NewUpdateBucketRequestBody()
	updateReq.SetWebsiteAccess(*websiteAccess)

	_, resp, err := client.Client.BucketAPI.UpdateBucket(client.WithAuth(ctx)).Id(bucketID).UpdateBucketRequestBody(*updateReq).Execute()
	if err != nil {
		return err
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	return nil
}

// S3 Lifecycle Configuration structures
type LifecycleConfiguration struct {
	XMLName xml.Name `xml:"LifecycleConfiguration"`
	Rules   []Rule   `xml:"Rule"`
}

type Rule struct {
	ID         string      `xml:"ID"`
	Status     string      `xml:"Status"`
	Filter     *Filter     `xml:"Filter,omitempty"`
	Expiration *Expiration `xml:"Expiration,omitempty"`
}

type Filter struct {
	Prefix string `xml:"Prefix"`
}

type Expiration struct {
	Days int `xml:"Days"`
}

// setBucketLifecyclePolicy sets the lifecycle expiration policy for a bucket using S3-compatible API
func setBucketLifecyclePolicy(ctx context.Context, client *GarageClient, bucketID string, expirationDays int) error {
	// Get bucket info to find the bucket alias/name
	bucket, resp, err := client.Client.BucketAPI.GetBucketInfo(client.WithAuth(ctx)).Id(bucketID).Execute()
	if err != nil {
		return fmt.Errorf("failed to get bucket info: %w", err)
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	// Get bucket name (alias)
	bucketName := bucketID
	if aliases := bucket.GetGlobalAliases(); len(aliases) > 0 {
		bucketName = aliases[0]
	}

	// Construct S3 endpoint URL (Garage S3 API typically uses port 3900)
	// Replace admin port (3903) with S3 port (3900)
	s3URL := fmt.Sprintf("%s://%s/%s?lifecycle",
		client.Scheme,
		replacePort(client.Host, 3900),
		bucketName)

	// Create lifecycle configuration XML
	lifecycleConfig := LifecycleConfiguration{
		Rules: []Rule{
			{
				ID:     "expire-after-days",
				Status: "Enabled",
				Expiration: &Expiration{
					Days: expirationDays,
				},
			},
		},
	}

	xmlData, err := xml.MarshalIndent(lifecycleConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal lifecycle config: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "PUT", s3URL, bytes.NewReader(xmlData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", client.Token))

	// Execute request
	httpClient := &http.Client{}
	// Use a distinct variable: reassigning `resp` here would clobber the
	// GetBucketInfo response that the deferred Close above closes over —
	// nil-dereference panic when this request fails, and a leaked/double-closed
	// body when it succeeds.
	s3Resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if s3Resp != nil && s3Resp.Body != nil {
			_ = s3Resp.Body.Close()
		}
	}()

	if s3Resp.StatusCode != http.StatusOK && s3Resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", s3Resp.StatusCode)
	}

	return nil
}

// getBucketLifecyclePolicy retrieves the lifecycle expiration policy for a bucket
func getBucketLifecyclePolicy(ctx context.Context, client *GarageClient, bucketID string) (int, error) {
	// Get bucket info to find the bucket alias/name
	bucket, resp, err := client.Client.BucketAPI.GetBucketInfo(client.WithAuth(ctx)).Id(bucketID).Execute()
	if err != nil {
		return 0, fmt.Errorf("failed to get bucket info: %w", err)
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	// Get bucket name (alias)
	bucketName := bucketID
	if aliases := bucket.GetGlobalAliases(); len(aliases) > 0 {
		bucketName = aliases[0]
	}

	// Construct S3 endpoint URL
	s3URL := fmt.Sprintf("%s://%s/%s?lifecycle",
		client.Scheme,
		replacePort(client.Host, 3900),
		bucketName)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", s3URL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	// Use admin token for authorization
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", client.Token))

	// Execute request
	httpClient := &http.Client{}
	// Use a distinct variable: reassigning `resp` here would clobber the
	// GetBucketInfo response that the deferred Close above closes over —
	// nil-dereference panic when this request fails, and a leaked/double-closed
	// body when it succeeds.
	s3Resp, err := httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if s3Resp != nil && s3Resp.Body != nil {
			_ = s3Resp.Body.Close()
		}
	}()

	if s3Resp.StatusCode == http.StatusNotFound {
		return 0, nil // No lifecycle policy set
	}

	if s3Resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %d", s3Resp.StatusCode)
	}

	// Parse XML response
	var lifecycleConfig LifecycleConfiguration
	if err := xml.NewDecoder(s3Resp.Body).Decode(&lifecycleConfig); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract expiration days from first rule
	if len(lifecycleConfig.Rules) > 0 && lifecycleConfig.Rules[0].Expiration != nil {
		return lifecycleConfig.Rules[0].Expiration.Days, nil
	}

	return 0, nil
}

// deleteBucketLifecyclePolicy removes the lifecycle policy from a bucket
func deleteBucketLifecyclePolicy(ctx context.Context, client *GarageClient, bucketID string) error {
	// Get bucket info to find the bucket alias/name
	bucket, resp, err := client.Client.BucketAPI.GetBucketInfo(client.WithAuth(ctx)).Id(bucketID).Execute()
	if err != nil {
		return fmt.Errorf("failed to get bucket info: %w", err)
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	// Get bucket name (alias)
	bucketName := bucketID
	if aliases := bucket.GetGlobalAliases(); len(aliases) > 0 {
		bucketName = aliases[0]
	}

	// Construct S3 endpoint URL
	s3URL := fmt.Sprintf("%s://%s/%s?lifecycle",
		client.Scheme,
		replacePort(client.Host, 3900),
		bucketName)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "DELETE", s3URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Use admin token for authorization
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", client.Token))

	// Execute request
	httpClient := &http.Client{}
	// Use a distinct variable: reassigning `resp` here would clobber the
	// GetBucketInfo response that the deferred Close above closes over —
	// nil-dereference panic when this request fails, and a leaked/double-closed
	// body when it succeeds.
	s3Resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if s3Resp != nil && s3Resp.Body != nil {
			_ = s3Resp.Body.Close()
		}
	}()

	if s3Resp.StatusCode != http.StatusOK && s3Resp.StatusCode != http.StatusNoContent && s3Resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("unexpected status code: %d", s3Resp.StatusCode)
	}

	return nil
}

// replacePort replaces the port in a host string (e.g., "127.0.0.1:3903" -> "127.0.0.1:3900")
func replacePort(host string, newPort int) string {
	// Simple implementation - if host contains a port, replace it
	// Otherwise, append the new port
	for i := len(host) - 1; i >= 0; i-- {
		if host[i] == ':' {
			return host[:i+1] + strconv.Itoa(newPort)
		}
	}
	return host + ":" + strconv.Itoa(newPort)
}
