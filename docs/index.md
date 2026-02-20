---
page_title: "Provider: Garage"
description: |-
  The Garage provider interacts with Garage object storage using the Admin API v2.
---

# Garage Provider

The Garage provider allows you to manage [Garage](https://garagehq.deuxfleurs.fr/) object storage resources using Terraform. It uses the Garage Admin API v2 and requires Garage v2.x.

## Example Usage

```hcl
terraform {
  required_providers {
    garage = {
      source  = "arsolitt/garagehq"
      version = ">= 0.0.1"
    }
  }
}

provider "garage" {
  host   = "127.0.0.1:3903"
  scheme = "http"
  token  = var.garage_admin_token
}
```

## Authentication

The Garage provider requires an admin token to authenticate with the Garage Admin API.

### Static Credentials

```hcl
provider "garage" {
  host   = "garage.example.com:3903"
  scheme = "https"
  token  = "your-admin-token"
}
```

### Environment Variables

Credentials can also be provided via environment variables:

- `GARAGE_HOST` - The host and port for the Garage admin API
- `GARAGE_SCHEME` - The scheme (http or https), defaults to http
- `GARAGE_TOKEN` - The admin token

```hcl
provider "garage" {
  host   = coalesce(var.garage_host, env.GARAGE_HOST)
  scheme = coalesce(var.garage_scheme, env.GARAGE_SCHEME, "http")
  token  = coalesce(var.garage_token, env.GARAGE_TOKEN)
}
```

## Resources

| Resource | Description |
|----------|-------------|
| [`garage_key`](resources/key.md) | Manage access keys for S3 API authentication |
| [`garage_bucket`](resources/bucket.md) | Create and manage buckets with lifecycle policies |
| [`garage_bucket_key`](resources/bucket_key.md) | Manage permissions between keys and buckets |
| [`garage_admin_token`](resources/admin_token.md) | Create admin API tokens with restricted scopes |
| [`garage_cluster_layout`](resources/cluster_layout.md) | Manage cluster node layout and capacity |

## Getting Started

See the [Getting Started Guide](guides/getting-started.md) for a complete walkthrough.

## Requirements

- **Garage v2.x** - This provider uses Garage Admin API v2
- **Terraform >= 1.0**
- **Go >= 1.24** (for building from source)

## Schema

### Required

- `host` (String) - The host and port for the Garage admin API (e.g., `127.0.0.1:3903`)
- `token` (String, Sensitive) - The admin token for the Garage admin API

### Optional

- `scheme` (String) - The scheme to use for the Garage admin API. Defaults to `http`.
