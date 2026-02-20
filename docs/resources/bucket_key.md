---
page_title: "garage_bucket_key Resource - terraform-provider-garage"
description: |-
  Manages permissions between an access key and a bucket.
---

# garage_bucket_key

Manages the relationship between an access key and a bucket, controlling what operations the key can perform on the bucket.

## Example Usage

### Read-Write Access

```hcl
resource "garage_bucket" "data" {
  global_alias = "application-data"
}

resource "garage_key" "app" {
  name = "application-key"
}

resource "garage_bucket_key" "app_access" {
  bucket_id     = garage_bucket.data.id
  access_key_id = garage_key.app.access_key_id
  read          = true
  write         = true
  owner         = false
}
```

### Read-Only Access

```hcl
resource "garage_bucket" "public_assets" {
  global_alias = "cdn-assets"
}

resource "garage_key" "cdn_reader" {
  name = "cdn-pull-key"
}

resource "garage_bucket_key" "cdn_access" {
  bucket_id     = garage_bucket.public_assets.id
  access_key_id = garage_key.cdn_reader.access_key_id
  read          = true
  write         = false
  owner         = false
}
```

### Owner Access

```hcl
resource "garage_bucket" "admin" {
  global_alias = "admin-bucket"
}

resource "garage_key" "admin_key" {
  name = "admin-access"
}

resource "garage_bucket_key" "admin_access" {
  bucket_id     = garage_bucket.admin.id
  access_key_id = garage_key.admin_key.access_key_id
  read          = true
  write         = true
  owner         = true  # Full control including permission management
}
```

### Multi-Bucket Access Pattern

```hcl
resource "garage_key" "loki" {
  name = "loki-storage"
}

resource "garage_bucket" "chunks" {
  global_alias = "loki-chunks"
}

resource "garage_bucket" "ruler" {
  global_alias = "loki-ruler"
}

resource "garage_bucket" "admin" {
  global_alias = "loki-admin"
}

# Same key has different permissions on different buckets
resource "garage_bucket_key" "chunks_access" {
  bucket_id     = garage_bucket.chunks.id
  access_key_id = garage_key.loki.access_key_id
  read          = true
  write         = true
  owner         = false
}

resource "garage_bucket_key" "ruler_access" {
  bucket_id     = garage_bucket.ruler.id
  access_key_id = garage_key.loki.access_key_id
  read          = true
  write         = true
  owner         = false
}

resource "garage_bucket_key" "admin_access" {
  bucket_id     = garage_bucket.admin.id
  access_key_id = garage_key.loki.access_key_id
  read          = true
  write         = true
  owner         = true
}
```

## Import

Bucket key relationships can be imported using the format `bucket_id/access_key_id`:

```bash
terraform import garage_bucket_key.app_access "abc123def456/GK1234567890ABCDEF"
```

## Schema

### Required

- `bucket_id` (String) - The bucket ID
- `access_key_id` (String) - The access key ID
- `read` (Boolean) - Grant read permission
- `write` (Boolean) - Grant write permission
- `owner` (Boolean) - Grant owner permission (includes permission management)

### Important Notes

- Changing `bucket_id` or `access_key_id` will force recreation of the resource
- Setting all permissions to `false` is equivalent to no access
- Owner permission implies read and write access
