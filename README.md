# Terraform Provider for Garage

A Terraform provider for Garage storage using the Admin API v2.

[![CI](https://github.com/d0ugal/terraform-provider-garage/actions/workflows/ci.yml/badge.svg)](https://github.com/d0ugal/terraform-provider-garage/actions/workflows/ci.yml)

## Requirements

- **Garage v2.x** - This provider uses Garage Admin API v2
- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.24 (to build the provider)

## Features

### Resources

- **garage_key**: Create, update, and delete access keys
- **garage_bucket**: Create and delete buckets with lifecycle policies
- **garage_bucket_key**: Manage key permissions on buckets
- **garage_admin_token**: Manage admin API tokens with restricted scopes
- **garage_cluster_layout**: Manage cluster node layout

## Building

```bash
make build
make install
```

## Usage

```hcl
terraform {
  required_providers {
    garage = {
      source  = "d0ugal/garage"
      version = "0.1.0"
    }
  }
}

provider "garage" {
  scheme = "http"
  host   = "127.0.0.1:3903"
  token  = "your-admin-token"
}

# Create an access key
resource "garage_key" "loki_key" {
  name = "loki-access-key"
}

# Create a bucket
resource "garage_bucket" "loki" {
  global_alias = "loki"
}

# Grant key permissions on bucket
resource "garage_bucket_key" "loki_access" {
  bucket_id     = garage_bucket.loki.id
  access_key_id = garage_key.loki_key.access_key_id
  read          = true
  write         = true
  owner         = false
}

# Create an admin token with restricted scope
resource "garage_admin_token" "readonly" {
  name  = "readonly-token"
  scope = ["GetClusterStatus", "GetClusterHealth", "ListBuckets", "ListKeys"]
}

# Manage cluster layout
resource "garage_cluster_layout" "main" {
  roles {
    id       = "node-id-1"
    zone     = "dc1"
    capacity = 1073741824  # 1GB in bytes
    tags     = ["storage", "fast"]
  }
}
```

## Installation

After building, install to your local Terraform plugins directory:

```bash
make install
```

## Development

```bash
# Build
make build

# Test
make test

# Lint
make lint

# Clean
make clean
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
