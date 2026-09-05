---
page_title: "garage_bucket Resource - terraform-provider-garage"
description: |-
  Manages a bucket in Garage object storage.
---

# garage_bucket

Manages a bucket in Garage. Buckets are containers for objects and can have lifecycle policies for automatic expiration.

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

Buckets can be imported using the bucket ID:

```bash
terraform import garage_bucket.my_bucket abc123def456
```

## Schema

### Optional

- `global_alias` (String) - Global alias for the bucket. This appears as the bucket name in S3 API calls.
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
