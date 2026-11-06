---
page_title: "garage_bucket Resource - terraform-provider-garage"
description: |-
  Manages a bucket in Garage object storage.
---

# garage_bucket

Manages a bucket in Garage. Buckets are containers for objects and can have lifecycle policies for automatic expiration and quotas.

## Example Usage

### Basic Bucket

```hcl
resource "garage_bucket" "static_assets" {
  global_alias = "my-app-assets"
}
```

### Bucket with Lifecycle Policy

```hcl
resource "garage_bucket" "logs" {
  global_alias    = "application-logs"
  expiration_days = 30  # Auto-delete objects after 30 days
}
```

### Temporary Storage Bucket

```hcl
resource "garage_bucket" "uploads" {
  global_alias    = "user-uploads"
  expiration_days = 7  # Delete uploaded files after 7 days
}
```

### Bucket with Quotas

```hcl
resource "garage_bucket" "tenant" {
  global_alias = "tenant-data"
  max_size     = 10737418240 # 10 GiB
  max_objects  = 100000
}
```

### Website Hosting Bucket

```hcl
resource "garage_bucket" "static_site" {
  global_alias = "static-content"

  website_access {
    enabled        = true
    index_document = "index.html"
    error_document = "error.html"
  }
}
```

### Backup Bucket with Key

```hcl
resource "garage_bucket" "backups" {
  global_alias = "daily-backups"
}

resource "garage_key" "backup_writer" {
  name = "backup-service"
}

resource "garage_bucket_key" "backup_access" {
  bucket_id     = garage_bucket.backups.id
  access_key_id = garage_key.backup_writer.access_key_id
  read          = true
  write         = true
  owner         = false
}
```

## Import

Buckets can be imported using the bucket ID (64 hex digits) or a global alias:

```bash
terraform import garage_bucket.my_bucket 6a8e3f0f1b2c4d5e6f708192a3b4c5d6e7f8091a2b3c4d5e6f708192a3b4c5d6
terraform import garage_bucket.my_bucket my-bucket
```

Or with an `import` block:

```hcl
import {
  to = garage_bucket.my_bucket
  id = "my-bucket"
}
```

An import by alias writes that alias to `global_alias`. An import by bucket ID writes the lexicographically first global alias of the bucket. If the bucket has no global alias, the import writes no alias. If `global_alias` in your configuration is a different alias of the bucket, the next apply removes the alias that the import wrote, so import the bucket with the alias in your configuration.

The import also writes `expiration_days` and `website_access` to the state, so add them to your configuration. If you omit an attribute that the import wrote, the next plan shows its removal:

- `global_alias`: the apply removes the alias. If it is the last alias of the bucket, the apply fails, see [Global Alias](#global-alias).
- `expiration_days`: the apply removes the lifecycle policy.
- `website_access`: the apply disables website hosting.

The provider resolves a global alias to the bucket ID, so the resource ID in the state is always the bucket ID.

## Schema

### Optional

- `global_alias` (String) - Global alias for the bucket. This appears as the bucket name in S3 API calls. A change adds the new alias, then removes the old alias. The provider manages only this alias. Other global aliases of the bucket stay and cause no diff. See [Global Alias](#global-alias).
- `max_size` (Number) - Maximum total size in bytes of the objects in this bucket. Unset or `0` means no quota. See [Quotas](#quotas) for the limit on 32-bit builds.
- `max_objects` (Number) - Maximum number of objects in this bucket. Unset or `0` means no quota. See [Quotas](#quotas) for the limit on 32-bit builds.
- `expiration_days` (Number) - Number of days after which objects will be automatically deleted. Set to 0 to disable expiration.
- `website_access` (Block List, Max: 1) - Website hosting configuration for the bucket (see [below](#nested-schema-for-website_access)).

<a id="nested-schema-for-website_access"></a>
### Nested Schema for `website_access`

Optional:

- `enabled` (Boolean) - Whether website access is enabled for this bucket. Defaults to `false`.
- `index_document` (String) - Name of the index document served at the root of the website (e.g. `index.html`). Required when `enabled` is `true`.
- `error_document` (String) - Name of the error document served on errors (e.g. `error.html`). Only used when `enabled` is `true`.

### Read-Only

- `id` (String) - The bucket ID
- `bytes` (Number) - Total bytes used by objects in this bucket
- `objects` (Number) - Number of objects in this bucket

~> **Note** Lifecycle policies use the S3-compatible API and require proper configuration of the S3 endpoint (typically port 3900).

## Lifecycle Configuration

When `expiration_days` is set, the provider automatically configures an S3 lifecycle rule that will delete objects after the specified number of days. This is useful for:

- Log rotation
- Temporary file storage
- Backup retention policies

## Global Alias

When `global_alias` changes, the provider adds the new alias to the bucket and then removes the old alias. The bucket ID and the objects do not change.

Garage does not remove the last alias of a bucket. If you remove `global_alias` and the bucket has no other alias, the apply fails and the bucket keeps its global alias. To remove the global alias, first give the bucket another global alias, or a local alias in the namespace of an access key via `garage bucket alias <bucket> <name> --local <access-key-id>` (Garage CLI) or the `AddBucketAlias` admin API call, or delete the bucket.

## Quotas

Garage enforces `max_size` and `max_objects` on upload. If you remove a quota attribute or set it to `0`, the provider removes the quota from the bucket.

The provider sends both quotas in each update. A change to one attribute does not change the other attribute. If a quota is set in Garage and is not in your configuration, the plan shows its removal and the apply removes it.

On 32-bit builds of the provider, such as `linux_arm`, the limit of each attribute is 2147483647. For `max_size`, this is just under 2 GiB. A higher value in the configuration causes an error before the apply. If a bucket has a higher quota in Garage, the provider reads an incorrect value. Use a 64-bit build for higher quotas.
