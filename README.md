# Terraform Provider for Garage

A Terraform provider for [Garage](https://garagehq.deuxfleurs.fr/) object storage using the Admin API v2.

[![CI](https://github.com/arsolitt/terraform-provider-garagehq/actions/workflows/ci.yml/badge.svg)](https://github.com/arsolitt/terraform-provider-garagehq/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/arsolitt/terraform-provider-garagehq)](https://goreportcard.com/report/github.com/arsolitt/terraform-provider-garagehq)

## Requirements

- **Garage v2.x** - This provider uses Garage Admin API v2
- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.24 (to build from source)

## Resources

| Resource | Description |
|----------|-------------|
| `garage_key` | Manage S3 access keys |
| `garage_bucket` | Create buckets with lifecycle policies |
| `garage_bucket_key` | Manage bucket permissions |
| `garage_admin_token` | Scoped admin API tokens |
| `garage_cluster_layout` | Cluster topology management |

## Quick Start

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
  token  = var.garage_token
}

resource "garage_bucket" "example" {
  global_alias = "my-bucket"
}

resource "garage_key" "example" {
  name = "my-app-key"
}

resource "garage_bucket_key" "example" {
  bucket_id     = garage_bucket.example.id
  access_key_id = garage_key.example.access_key_id
  read          = true
  write         = true
  owner         = false
}
```

## Examples

### Loki Stack

Complete setup for Grafana Loki:

```hcl
resource "garage_key" "loki" {
  name = "loki-storage"
}

resource "garage_bucket" "loki_chunks" {
  global_alias    = "loki-chunks"
  expiration_days = 14
}

resource "garage_bucket" "loki_ruler" {
  global_alias    = "loki-ruler"
  expiration_days = 30
}

resource "garage_bucket" "loki_admin" {
  global_alias = "loki-admin"
}

resource "garage_bucket_key" "loki_chunks" {
  bucket_id     = garage_bucket.loki_chunks.id
  access_key_id = garage_key.loki.access_key_id
  read          = true
  write         = true
  owner         = false
}

resource "garage_bucket_key" "loki_ruler" {
  bucket_id     = garage_bucket.loki_ruler.id
  access_key_id = garage_key.loki.access_key_id
  read          = true
  write         = true
  owner         = false
}

resource "garage_bucket_key" "loki_admin" {
  bucket_id     = garage_bucket.loki_admin.id
  access_key_id = garage_key.loki.access_key_id
  read          = true
  write         = true
  owner         = true
}
```

### Backup Retention Policy

Different retention for different backup types:

```hcl
resource "garage_key" "backup" {
  name = "backup-service"
}

resource "garage_bucket" "daily" {
  global_alias    = "daily-backups"
  expiration_days = 7
}

resource "garage_bucket" "weekly" {
  global_alias    = "weekly-backups"
  expiration_days = 30
}

resource "garage_bucket" "monthly" {
  global_alias    = "monthly-backups"
  expiration_days = 365
}

resource "garage_bucket_key" "backup" {
  for_each = {
    daily   = garage_bucket.daily.id
    weekly  = garage_bucket.weekly.id
    monthly = garage_bucket.monthly.id
  }

  bucket_id     = each.value
  access_key_id = garage_key.backup.access_key_id
  read          = true
  write         = true
  owner         = false
}
```

### Read-Only Access for CDN

```hcl
resource "garage_bucket" "cdn_assets" {
  global_alias = "cdn-assets"
}

resource "garage_key" "cdn_reader" {
  name = "cdn-pull-only"
}

resource "garage_bucket_key" "cdn_readonly" {
  bucket_id     = garage_bucket.cdn_assets.id
  access_key_id = garage_key.cdn_reader.access_key_id
  read          = true
  write         = false
  owner         = false
}
```

### Scoped Admin Token for Monitoring

```hcl
resource "garage_admin_token" "prometheus" {
  name  = "prometheus-monitoring"
  scope = [
    "GetClusterStatus",
    "GetClusterHealth"
  ]
  never_expires = true
}
```

### Single Node Development Cluster

```hcl
variable "node_id" {
  type = string
}

resource "garage_cluster_layout" "dev" {
  roles {
    id       = var.node_id
    zone     = "local"
    capacity = "10G"
    tags     = ["dev", "standalone"]
  }
}
```

### Multi-Node Production Cluster

```hcl
resource "garage_cluster_layout" "production" {
  roles {
    id       = "node-dc1-a"
    zone     = "datacenter-1"
    capacity = "2T"
    tags     = ["storage", "ssd", "primary"]
  }

  roles {
    id       = "node-dc1-b"
    zone     = "datacenter-1"
    capacity = "2T"
    tags     = ["storage", "ssd", "primary"]
  }

  roles {
    id       = "node-dc2-a"
    zone     = "datacenter-2"
    capacity = "2T"
    tags     = ["storage", "ssd", "secondary"]
  }

  # Gateway for edge traffic
  roles {
    id   = "gateway-edge"
    zone = "datacenter-1"
    tags = ["gateway", "edge"]
    # No capacity = gateway mode
  }
}
```

## Provider Configuration

### Static Credentials

```hcl
provider "garage" {
  host   = "garage.example.com:3903"
  scheme = "https"
  token  = "your-admin-token"
}
```

### Environment Variables

```hcl
variable "garage_host" {
  default     = ""
  description = "Override Garage host"
}

variable "garage_token" {
  default     = ""
  sensitive   = true
  description = "Override Garage admin token"
}

provider "garage" {
  host   = coalesce(var.garage_host, env.GARAGE_HOST, "127.0.0.1:3903")
  scheme = coalesce(env.GARAGE_SCHEME, "http")
  token  = coalesce(var.garage_token, env.GARAGE_TOKEN)
}
```

Supported environment variables:
- `GARAGE_HOST` - Admin API host:port
- `GARAGE_SCHEME` - http or https
- `GARAGE_TOKEN` - Admin token

## Building from Source

```bash
git clone https://github.com/arsolitt/terraform-provider-garagehq
cd terraform-provider-garage
make build
```

## Local Installation

For local development and testing:

```bash
make install
```

This installs the provider to your local Terraform plugin directory.

## Development

```bash
make build     # Build the provider
make test      # Run tests with coverage
make lint      # Format and lint code
make clean     # Clean build artifacts
```

## Documentation

- [Provider Documentation](docs/index.md)
- [Getting Started Guide](docs/guides/getting-started.md)
- [Bucket Lifecycle Policies](docs/guides/bucket-lifecycle.md)
- [Admin Token Management](docs/guides/admin-tokens.md)
- [Cluster Management](docs/guides/cluster-management.md)

## Releasing

This project uses automated releases via GitHub Actions:

1. Merge PRs with conventional commit messages
2. Release Please creates a release PR
3. Merge the release PR to trigger a new release
4. GoReleaser builds and publishes binaries

See [PUBLISHING.md](PUBLISHING.md) for details.

## Troubleshooting

### Connection Refused

Ensure Garage is running and the admin API is accessible:

```bash
curl -H "Authorization: Bearer $TOKEN" http://localhost:3903/v2/GetClusterHealth
```

### Permission Denied

Verify your admin token has the required permissions. Some operations require specific scopes.

### Secret Key Not Available

The `secret_access_key` is only available on initial creation. If you need to recover it, you'll need to recreate the key.

### Lifecycle Policy Not Working

Ensure the S3 API is accessible (default port 3900). The provider uses the S3 API for lifecycle configuration, not the admin API.

## License

MIT License - see [LICENSE](LICENSE) for details.
