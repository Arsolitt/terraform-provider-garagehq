---
page_title: "Admin Token Management - Guides - terraform-provider-garage"
description: |-
  Create scoped admin tokens for different services and use cases
---

# Admin Token Management

Admin tokens allow fine-grained access control to the Garage Admin API. This guide shows how to create tokens with specific scopes for different services.

## Why Scoped Tokens?

- **Security**: Limit exposure if a token is compromised
- **Separation of concerns**: Different services get only the permissions they need
- **Audit trail**: Track which service performed which action
- **Rotation**: Easier to rotate tokens for specific services

## Common Token Patterns

### Monitoring Token (Read-Only)

For Prometheus or other monitoring systems:

```hcl
resource "garage_admin_token" "prometheus" {
  name  = "prometheus-monitoring"
  scope = [
    "GetClusterStatus",
    "GetClusterHealth"
  ]
}
```

Use with Prometheus `http_sd_config` or custom exporters.

### CI/CD Token (Bucket Operations)

For pipelines that manage buckets:

```hcl
resource "garage_admin_token" "ci_pipeline" {
  name  = "ci-bucket-management"
  scope = [
    "ListBuckets",
    "CreateBucket",
    "GetBucketInfo",
    "DeleteBucket"
  ]
}
```

### Key Management Service

For services that need to create/manage access keys:

```hcl
resource "garage_admin_token" "key_service" {
  name  = "key-management-service"
  scope = [
    "ListKeys",
    "CreateKey",
    "GetKeyInfo",
    "UpdateKey",
    "DeleteKey"
  ]
}
```

### Backup Service Token

For automated backup systems:

```hcl
resource "garage_admin_token" "backup" {
  name  = "backup-service"
  scope = [
    "GetClusterStatus",
    "ListBuckets",
    "GetBucketInfo"
  ]
  # Backup tokens should not expire unexpectedly
  never_expires = true
}
```

## Token Expiration

### Time-Limited Tokens

For temporary access (contractors, debugging):

```hcl
resource "garage_admin_token" "temp_debug" {
  name       = "debug-access-2024"
  scope      = ["ListBuckets", "GetBucketInfo", "ListKeys"]
  expiration = "2024-12-31T23:59:59Z"
}
```

### Never-Expiring Tokens

For long-running services:

```hcl
resource "garage_admin_token" "service_account" {
  name          = "automation-service"
  scope         = ["*"]
  never_expires = true
}
```

## Multi-Environment Setup

Create different tokens for different environments:

```hcl
# Development - full access
resource "garage_admin_token" "dev_admin" {
  name          = "dev-full-access"
  scope         = ["*"]
  never_expires = true
}

# Staging - limited access
resource "garage_admin_token" "staging_admin" {
  name  = "staging-operations"
  scope = ["ListBuckets", "GetBucketInfo", "CreateBucket", "ListKeys", "CreateKey"]
}

# Production - read-only
resource "garage_admin_token" "prod_readonly" {
  name  = "production-readonly"
  scope = ["GetClusterStatus", "ListBuckets", "GetBucketInfo", "ListKeys"]
}
```

## Available Scopes

### Cluster Operations

| Scope | Description |
|-------|-------------|
| `GetClusterStatus` | View cluster status and statistics |
| `GetClusterHealth` | View cluster health status |

### Bucket Operations

| Scope | Description |
|-------|-------------|
| `ListBuckets` | List all buckets |
| `CreateBucket` | Create new buckets |
| `GetBucketInfo` | View bucket details and statistics |
| `UpdateBucket` | Update bucket properties |
| `DeleteBucket` | Delete buckets |

### Key Operations

| Scope | Description |
|-------|-------------|
| `ListKeys` | List all access keys |
| `CreateKey` | Create new access keys |
| `GetKeyInfo` | View key details |
| `UpdateKey` | Update key properties |
| `DeleteKey` | Delete access keys |

### Layout Operations

| Scope | Description |
|-------|-------------|
| `GetClusterLayout` | View cluster layout |
| `UpdateClusterLayout` | Modify cluster layout |
| `ApplyClusterLayout` | Apply layout changes |

### Admin Token Operations

| Scope | Description |
|-------|-------------|
| `ListAdminTokens` | List admin tokens |
| `CreateAdminToken` | Create new admin tokens |
| `GetAdminTokenInfo` | View token details |
| `UpdateAdminToken` | Update token properties |
| `DeleteAdminToken` | Delete admin tokens |

## Outputs

Export tokens for use by other systems:

```hcl
output "prometheus_token" {
  value     = garage_admin_token.prometheus.secret_token
  sensitive = true
}

output "ci_token" {
  value     = garage_admin_token.ci_pipeline.secret_token
  sensitive = true
}
```

## Security Best Practices

1. **Principle of least privilege**: Only grant scopes that are actually needed
2. **Use expiration**: Set expiration for temporary access
3. **Rotate tokens**: Periodically recreate tokens
4. **Audit scope**: Regularly review what each token can do
5. **Avoid `*` scope**: Only use full access when absolutely necessary
