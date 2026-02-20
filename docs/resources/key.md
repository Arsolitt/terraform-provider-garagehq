---
page_title: "garage_key Resource - terraform-provider-garage"
description: |-
  Manages an access key for Garage S3 API authentication.
---

# garage_key

Manages an access key in Garage. Access keys are used to authenticate with the S3-compatible API.

## Example Usage

### Basic Key

```hcl
resource "garage_key" "app_key" {
  name = "my-application"
}
```

### Key with Bucket Permissions

```hcl
resource "garage_key" "loki" {
  name = "loki-storage-key"
}

resource "garage_bucket" "loki_data" {
  global_alias = "loki-chunks"
}

resource "garage_bucket_key" "loki_access" {
  bucket_id     = garage_bucket.loki_data.id
  access_key_id = garage_key.loki.access_key_id
  read          = true
  write         = true
  owner         = false
}
```

## Import

Access keys can be imported using the access key ID:

```bash
terraform import garage_key.my_key GK1234567890ABCDEF
```

## Schema

### Required

- `name` (String) - The name of the access key

### Read-Only

- `access_key_id` (String) - The access key ID (used to identify the key)
- `secret_access_key` (String, Sensitive) - The secret access key (only available on initial creation)

-> **Important** The `secret_access_key` is only available immediately after creation. Store it securely in your secrets manager or Terraform state will be the only record of it.
