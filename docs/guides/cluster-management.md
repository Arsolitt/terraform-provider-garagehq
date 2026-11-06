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

For local development or testing, a single-node layout is sufficient. The [`garage_node`](../data-sources/node.md) data source gives the ID of the node that serves the admin API:

```hcl
data "garage_node" "self" {}

resource "garage_cluster_layout" "dev" {
  roles {
    id       = data.garage_node.self.id
    zone     = "local"
    capacity = "1G"
    tags     = ["dev", "local"]
  }
}
```

### Complete Development Example

```hcl
data "garage_node" "self" {}

resource "garage_cluster_layout" "development" {
  roles {
    id       = data.garage_node.self.id
    zone     = "dev"
    capacity = "10G"
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

The node ID is the public key of the node. Garage makes it on the first start and keeps it in the metadata directory (`node_key.pub`), so you cannot set it in `garage.toml`.

The `garage_node` data source reads it from the cluster. Without `id`, it reads the node that serves the admin API; with `id`, it reads another node:

```hcl
data "garage_node" "self" {}

output "node" {
  value = {
    id       = data.garage_node.self.id
    hostname = data.garage_node.self.hostname
    is_up    = data.garage_node.self.is_up
    zone     = data.garage_node.self.zone
  }
}
```

On the node, `garage node id` also shows it. Or get the IDs of all nodes from the cluster status:

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
    capacity = "100G"
    tags     = ["storage", "primary"]
  }

  roles {
    id       = "node-2-id"
    zone     = "zone-b"
    capacity = "100G"
    tags     = ["storage", "primary"]
  }

  roles {
    id       = "node-3-id"
    zone     = "zone-c"
    capacity = "100G"
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
    capacity = "500G"
    tags     = ["storage", "ssd"]
  }

  roles {
    id       = "storage-2"
    zone     = "datacenter-2"
    capacity = "500G"
    tags     = ["storage", "ssd"]
  }

  roles {
    id       = "storage-3"
    zone     = "datacenter-3"
    capacity = "500G"
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

### Specifying Capacity

Capacity can be specified with binary unit suffixes for readability:

```hcl
resource "garage_cluster_layout" "production" {
  roles {
    id       = "node-1"
    zone     = "dc1"
    capacity = "2T"
  }
}
```

Supported suffixes:

| Suffix | Meaning | Example |
|--------|---------|---------|
| K, KiB | 1024 bytes | `512K` |
| M, MiB | 1024² bytes | `500M` |
| G, GiB | 1024³ bytes | `1G` |
| T, TiB | 1024⁴ bytes | `2T` |

Plain numbers (bytes) are also accepted: `1073741824`
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
    capacity = "1T"
  }

  roles {
    id       = "node-2"
    zone     = "dc2"
    capacity = "1T"
  }

  roles {
    id       = "node-3"
    zone     = "dc3"
    capacity = "1T"
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
    capacity = "1T"
  }

  # New node
  roles {
    id       = "node-4"  # New node ID
    zone     = "dc4"     # New zone
    capacity = "1T"
  }
}
```

-> **Note** Adding nodes triggers data rebalancing. Plan for increased network traffic.

### Removing Nodes

Remove the node from the `roles` block. The provider removes the node from the layout and applies a new layout version. Garage then moves the data of the node to the remaining nodes.

The provider removes only the nodes that were in `roles` before. It does not remove a node that the configuration never managed.

~> **Warning** `terraform import` writes all roles of the layout to the state. Thus all nodes count as managed after an import. When the configuration lists only some of the nodes, the next apply removes the other nodes from the layout.

### Changing Capacity

Update the `capacity` value. Garage will rebalance data accordingly.

### Staged Changes

The provider stages only the roles that are different from the current layout, and applies a new layout version only when it staged a change. When a role must change, it refuses to continue if the layout has staged changes that are not applied, for example from `garage layout assign`, because the apply would also apply them. Apply or revert them first with `garage layout apply` or `garage layout revert`. When no role must change, the provider applies nothing, thus staged changes do not stop it.

### Destroying the Layout Resource

`terraform destroy` removes the `garage_cluster_layout` resource only from the state. The layout does not change, thus the cluster continues to operate. The provider also keeps the staged changes, because they can come from other tools.

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
