---
page_title: "Cluster Management - Guides - terraform-provider-garage"
description: |-
  Manage Garage cluster layouts for development and production
---

# Cluster Management

This guide covers cluster layout management for both single-node development setups and multi-node production deployments.

## Understanding Cluster Layout

The cluster layout defines:
- **Which nodes** participate in the cluster
- **Where data** is stored (zones)
- **How much capacity** each node has
- **What role** each node plays (storage vs gateway)

## Single Node Setup (Development)

For local development or testing, a single-node layout is sufficient:

```hcl
resource "garage_cluster_layout" "dev" {
  roles {
    id       = var.node_id  # From garage node config
    zone     = "local"
    capacity = 1073741824   # 1GB in bytes
    tags     = ["dev", "local"]
  }
}
```

### Complete Development Example

```hcl
variable "node_id" {
  description = "Garage node ID from configuration"
  type        = string
}

resource "garage_cluster_layout" "development" {
  roles {
    id       = var.node_id
    zone     = "dev"
    capacity = 10737418240  # 10GB
    tags     = ["development", "standalone"]
  }
}

# Create buckets for development
resource "garage_bucket" "dev_data" {
  global_alias = "dev-bucket"
}

resource "garage_key" "dev_key" {
  name = "dev-access"
}

resource "garage_bucket_key" "dev_access" {
  bucket_id     = garage_bucket.dev_data.id
  access_key_id = garage_key.dev_key.access_key_id
  read          = true
  write         = true
  owner         = true
}
```

### Finding Your Node ID

The node ID is set in your Garage configuration:

```toml
# garage.toml
metadata_dir = "/var/lib/garage/meta"
data_dir = "/var/lib/garage/data"

[consul_discovery]
# or use static configuration
```

Or get it from the cluster status:

```bash
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
    http://localhost:3903/v2/GetClusterStatus
```

## Multi-Node Production Setup

For production, deploy at least 3 nodes across different zones for redundancy:

### Basic 3-Node Cluster

```hcl
resource "garage_cluster_layout" "production" {
  roles {
    id       = "node-1-id"
    zone     = "zone-a"
    capacity = 107374182400  # 100GB
    tags     = ["storage", "primary"]
  }

  roles {
    id       = "node-2-id"
    zone     = "zone-b"
    capacity = 107374182400  # 100GB
    tags     = ["storage", "primary"]
  }

  roles {
    id       = "node-3-id"
    zone     = "zone-c"
    capacity = 107374182400  # 100GB
    tags     = ["storage", "primary"]
  }
}
```

### Multi-Zone with Gateway Nodes

Use gateway nodes to handle traffic without storing data:

```hcl
resource "garage_cluster_layout" "with_gateways" {
  # Storage nodes (hold data)
  roles {
    id       = "storage-1"
    zone     = "datacenter-1"
    capacity = 536870912000  # 500GB
    tags     = ["storage", "ssd"]
  }

  roles {
    id       = "storage-2"
    zone     = "datacenter-2"
    capacity = 536870912000
    tags     = ["storage", "ssd"]
  }

  roles {
    id       = "storage-3"
    zone     = "datacenter-3"
    capacity = 536870912000
    tags     = ["storage", "hdd"]
  }

  # Gateway nodes (route traffic, no storage)
  roles {
    id   = "gateway-1"
    zone = "datacenter-1"
    tags = ["gateway", "edge"]
    # No capacity = gateway mode
  }

  roles {
    id   = "gateway-2"
    zone = "datacenter-2"
    tags = ["gateway", "edge"]
  }
}
```

## Capacity Planning

### Calculating Capacity

Capacity is specified in bytes. Use locals for readability:

```hcl
locals {
  gb = 1073741824
  tb = 1099511627776
}

resource "garage_cluster_layout" "production" {
  roles {
    id       = "node-1"
    zone     = "dc1"
    capacity = 2 * local.tb  # 2TB
  }
}
```

### Usable Capacity

Account for replication. With 3-way replication (default):

```hcl
# For 1TB of usable storage with 3-way replication:
# Each node needs 1TB (total 3TB raw)
resource "garage_cluster_layout" "example" {
  roles {
    id       = "node-1"
    zone     = "dc1"
    capacity = local.tb  # 1TB per node
  }

  roles {
    id       = "node-2"
    zone     = "dc2"
    capacity = local.tb
  }

  roles {
    id       = "node-3"
    zone     = "dc3"
    capacity = local.tb
  }
}
# Total: 3TB raw = ~1TB usable
```

## Zone Strategy

### Zone Naming Conventions

```hcl
# By datacenter
zones = ["us-east-1", "us-west-2", "eu-west-1"]

# By rack
zones = ["rack-a", "rack-b", "rack-c"]

# By availability zone
zones = ["az-1a", "az-1b", "az-1c"]
```

### Minimum Zones

- **Minimum**: 1 zone (not recommended for production)
- **Recommended**: 3+ zones for proper quorum
- **Optimal**: 5+ zones for large clusters

## Updating Layout

### Adding Nodes

```hcl
resource "garage_cluster_layout" "production" {
  # Existing nodes...
  roles {
    id       = "node-1"
    zone     = "dc1"
    capacity = local.tb
  }

  # New node
  roles {
    id       = "node-4"  # New node ID
    zone     = "dc4"     # New zone
    capacity = local.tb
  }
}
```

-> **Note** Adding nodes triggers data rebalancing. Plan for increased network traffic.

### Removing Nodes

Simply remove the node from the `roles` block. Data will be redistributed to remaining nodes.

### Changing Capacity

Update the `capacity` value. Garage will rebalance data accordingly.

## Monitoring Layout

Output layout version for tracking:

```hcl
output "layout_version" {
  value = garage_cluster_layout.production.version
}
```

Check cluster health after changes:

```bash
curl -H "Authorization: Bearer $TOKEN" \
    http://localhost:3903/v2/GetClusterHealth
```

## Best Practices

1. **Use at least 3 nodes** in different zones for production
2. **Start small** and add capacity as needed
3. **Use gateway nodes** at network edges for better latency
4. **Monitor rebalancing** after layout changes
5. **Test in dev** before applying to production
6. **Document node IDs** for disaster recovery
