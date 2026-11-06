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

~> **Warning** `terraform plan -destroy` and `terraform destroy` fail during the pre-destroy refresh plan with `cannot change secret_access_key ...` when the configuration sets a `secret_access_key` that differs from the value in state. Set `secret_access_key` back to the state value, or remove `secret_access_key` and `access_key_id` from the configuration, and run the destroy again; alternatively use `terraform destroy -refresh=false`.

## Import

Access keys can be imported using the access key ID:

```bash
terraform import garage_key.my_key GK31c2f218a2e44f485b94239e
```

Or with an `import` block:

```hcl
import {
  to = garage_key.my_key
  id = "GK31c2f218a2e44f485b94239e"
}
```

The import reads the `secret_access_key` from Garage, so the admin token must be allowed to call `GetKeyInfo`.

## Schema

### Required

- `name` (String) - The name of the access key

### Optional

- `access_key_id` (String) - The access key ID (used to identify the key). Requires `secret_access_key`. If you do not set it, Garage generates it. A change forces a new resource, see [Key with Known Credentials](#key-with-known-credentials).
- `secret_access_key` (String, Sensitive) - The secret access key. Requires `access_key_id`. If you do not set it, Garage generates it. A change forces a new resource when `access_key_id` also changes. The provider rejects a change to only this value at plan time, see [Key with Known Credentials](#key-with-known-credentials). The provider records this value at create and import time, not on later reads; Garage keeps it, see the note below.

-> **Important** The provider records `secret_access_key` at create and import time and does not update it on later reads. To record the secret in the state again, remove the key from the state with `terraform state rm`, then [import](#import) the key. Garage itself keeps the secret and can show it, for example with `garage key info --show-secret <access_key_id>` or the admin API `GetKeyInfo` with `showSecretKey=true`. Do not destroy and recreate a key only to recover its secret: Garage refuses to import a configured `access_key_id` again after deletion. Store the secret securely in your secrets manager when the key is created.
