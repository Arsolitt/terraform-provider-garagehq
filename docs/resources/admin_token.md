---
page_title: "garage_admin_token Resource - terraform-provider-garage"
description: |-
  Manages an admin API token with restricted scope.
---

# garage_admin_token

Manages an admin API token in Garage. Admin tokens can be scoped to specific API operations, allowing fine-grained access control for different services and users.

-> **New in Garage v2** This resource uses the Admin API v2 token management features.

## Example Usage

### Read-Only Monitoring Token

```hcl
resource "garage_admin_token" "monitoring" {
  name  = "prometheus-metrics"
  scope = [
    "GetClusterStatus",
    "GetClusterHealth"
  ]
}
```

### Bucket Management Token

```hcl
resource "garage_admin_token" "bucket_manager" {
  name  = "bucket-operations"
  scope = [
    "ListBuckets",
    "CreateBucket",
    "GetBucketInfo",
    "DeleteBucket"
  ]
}
```

### Key Management Token

```hcl
resource "garage_admin_token" "key_manager" {
  name  = "key-operations"
  scope = [
    "ListKeys",
    "CreateKey",
    "GetKeyInfo",
    "UpdateKey",
    "DeleteKey"
  ]
}
```

### Full Access Token

```hcl
resource "garage_admin_token" "full_access" {
  name  = "full-admin"
  scope = ["*"]  # All operations
}
```

### Token with Expiration

```hcl
resource "garage_admin_token" "temp_access" {
  name       = "temporary-access"
  scope      = ["ListBuckets", "GetBucketInfo"]
  expiration = "2025-12-31T23:59:59Z"
}
```

### Never-Expiring Token

```hcl
resource "garage_admin_token" "service_account" {
  name          = "service-account"
  scope         = ["*"]
  never_expires = true
}
```

## Import

Admin tokens can be imported using the token ID:

```bash
terraform import garage_admin_token.my_token tkn1234567890abcdef
```

## Schema

### Required

- `name` (String) - Name of the admin API token
- `scope` (List of String) - List of API operation names the token can access. Use `["*"]` for all operations.

### Optional

- `expiration` (String) - Expiration time in RFC3339 format (e.g., `2025-12-31T23:59:59Z`)
- `never_expires` (Boolean) - Set to true for tokens that should never expire

### Read-Only

- `id` (String) - Identifier of the admin token
- `secret_token` (String, Sensitive) - The secret bearer token (only available on initial creation)
- `expired` (Boolean) - Whether this admin token is expired
- `created` (String) - Creation date in RFC3339 format

## Available Scopes

Common API operations that can be included in the scope:

| Scope | Description |
|-------|-------------|
| `*` | All operations |
| `GetClusterStatus` | View cluster status |
| `GetClusterHealth` | View cluster health |
| `ListBuckets` | List all buckets |
| `CreateBucket` | Create new buckets |
| `GetBucketInfo` | View bucket details |
| `DeleteBucket` | Delete buckets |
| `ListKeys` | List all access keys |
| `CreateKey` | Create access keys |
| `GetKeyInfo` | View key details |
| `UpdateKey` | Update key properties |
| `DeleteKey` | Delete access keys |

-> **Security Tip** Always use the minimum required scope for your use case. Avoid using `["*"]` for service accounts unless absolutely necessary.
