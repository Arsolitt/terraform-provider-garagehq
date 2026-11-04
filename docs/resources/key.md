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

### Key with Known Credentials

Set `access_key_id` and `secret_access_key` together to create the key with these credentials instead of generated ones. Use this to configure a client before the key exists, or to move a key from another S3 store:

```hcl
resource "garage_key" "app" {
  name              = "app"
  access_key_id     = "GK31c2f218a2e44f485b94239e"
  secret_access_key = var.app_secret_access_key
}
```

Garage accepts an `access_key_id` of 8 or more characters from ASCII letters, digits, `-`, `_` and `.`, and a `secret_access_key` of 16 or more graphic ASCII characters (`U+0021` to `U+007E`). Keys that Garage generates have the form `GK` followed by 24 hex digits, and a secret of 64 hex digits.

-> **Note** When you destroy a key, Garage keeps a tombstone of it. Garage then refuses to import the same `access_key_id` again, also with the same secret, and returns `KeyAlreadyExists`. Thus:

- After you destroy a key with an imported `access_key_id`, you cannot create it again with that `access_key_id`. Use a new `access_key_id`.
- A change to `access_key_id` replaces the key. The new `access_key_id` must be one that Garage did not see before. The tombstone is not visible at plan time, so if you go back to a former `access_key_id`, the apply deletes the current key and then fails.
- A change to only `secret_access_key` would replace the key with the same `access_key_id`, and the import would fail after the old key is deleted. The provider rejects this change at plan time. To change the secret, also set a new `access_key_id`.
- A change to only `name` updates the key in place.
- If you remove both `access_key_id` and `secret_access_key` from the configuration, the key and its credentials stay as they are, and Terraform shows no change.

~> **Warning** Do not use `terraform apply -replace` or `terraform taint` on a key with a configured `access_key_id`. The provider cannot reject such a replacement at plan time. Terraform deletes the key, and then Garage refuses to import the same `access_key_id`. To replace the key, set a new `access_key_id`.

## Import

Access keys can be imported using the access key ID:

```bash
terraform import garage_key.my_key GK1234567890ABCDEF
```

## Schema

### Required

- `name` (String) - The name of the access key

### Optional

- `access_key_id` (String) - The access key ID (used to identify the key). Requires `secret_access_key`. If you do not set it, Garage generates it. A change forces a new resource, see [Key with Known Credentials](#key-with-known-credentials).
- `secret_access_key` (String, Sensitive) - The secret access key. Requires `access_key_id`. If you do not set it, Garage generates it. A change forces a new resource when `access_key_id` also changes. The provider rejects a change to only this value at plan time, see [Key with Known Credentials](#key-with-known-credentials).

-> **Important** A generated `secret_access_key` is only available immediately after creation. Store it securely in your secrets manager or Terraform state will be the only record of it.
