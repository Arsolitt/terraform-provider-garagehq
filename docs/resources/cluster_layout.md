---
page_title: "garage_cluster_layout Resource - terraform-provider-garage"
description: |-
  Manages the cluster layout and node roles in Garage.
---

# garage_cluster_layout

Manages the cluster layout in Garage, including node roles, zones, capacity, and tags. This resource allows you to define your cluster topology declaratively.

-> **Important** This resource applies layout changes immediately. In production, ensure you understand Garage's layout versioning system before making changes.

## Example Usage

### Single Node (Development)

```hcl
resource "garage_cluster_layout" "dev" {
  roles {
    id       = "node-dev-1"
    zone     = "local"
    capacity = "1G"
    tags     = ["dev", "storage"]
  }
}
```

### Multi-Node Production Cluster

```hcl
resource "garage_cluster_layout" "production" {
  roles {
    id       = "node-1"
    zone     = "dc1"
    capacity = "100G"
    tags     = ["storage", "ssd"]
  }

  roles {
    id       = "node-2"
    zone     = "dc1"
    capacity = "100G"
    tags     = ["storage", "ssd"]
  }

  roles {
    id       = "node-3"
    zone     = "dc2"
    capacity = "100G"
    tags     = ["storage", "hdd"]
  }
}
```

### Cluster with Gateway Nodes

```hcl
resource "garage_cluster_layout" "with_gateways" {
  # Storage nodes
  roles {
    id       = "storage-1"
    zone     = "zone-a"
    capacity = "50G"
    tags     = ["storage"]
  }

  roles {
    id       = "storage-2"
    zone     = "zone-b"
    capacity = "50G"
    tags     = ["storage"]
  }

  # Gateway nodes (no capacity, just routing)
  roles {
    id       = "gateway-1"
    zone     = "zone-a"
    tags     = ["gateway", "edge"]
    # No capacity = gateway mode
  }

  roles {
    id       = "gateway-2"
    zone     = "zone-b"
    tags     = ["gateway", "edge"]
  }
}
```

### Zone-Aware Layout

```hcl
resource "garage_cluster_layout" "multi_zone" {
  # 2 nodes per zone for redundancy
  roles {
    id       = "eu-west-1a"
    zone     = "eu-west-1"
    capacity = "20G"
    tags     = ["primary", "fast"]
  }

  roles {
    id       = "eu-west-1b"
    zone     = "eu-west-1"
    capacity = "20G"
    tags     = ["primary", "fast"]
  }

  roles {
    id       = "us-east-1a"
    zone     = "us-east-1"
    capacity = "20G"
    tags     = ["secondary", "standard"]
  }

  roles {
    id       = "us-east-1b"
    zone     = "us-east-1"
    capacity = "20G"
    tags     = ["secondary", "standard"]
  }
}
```

## Import

Cluster layout is a singleton resource and is imported as `cluster-layout`:

```bash
terraform import garage_cluster_layout.main cluster-layout
```

## Schema

### Required

- `roles` (Block List) - List of node roles in the cluster layout

### roles Block

- `id` (String, Required) - Node ID (must match the node's configured ID)
- `zone` (String, Required) - Zone assigned to the node for data replication
- `capacity` (String, Optional) - Storage capacity with unit suffix (e.g., `1G`, `500M`, `2TiB`). Omit for gateway nodes.
- `tags` (List of String, Optional) - Tags assigned to the node

### Read-Only

- `version` (Number) - Current layout version

## Capacity Format

Capacity can be specified with binary unit suffixes:

| Suffix | Meaning | Example |
|--------|---------|---------|
| K, KiB | 1024 bytes | `512K` |
| M, MiB | 1024² bytes | `500M` |
| G, GiB | 1024³ bytes | `1G` |
| T, TiB | 1024⁴ bytes | `2T` |

Plain numbers (bytes) are also accepted: `1073741824`

```hcl
resource "garage_cluster_layout" "example" {
  roles {
    id       = "node-1"
    zone     = "dc1"
    capacity = "100G"
  }
}
```

## Important Considerations

1. **Layout Changes**: Changes to the layout are applied immediately. Ensure you have proper change management procedures.

2. **Node IDs**: The node ID must match the ID configured in the Garage node's configuration file.

3. **Gateway Nodes**: Nodes without a `capacity` attribute operate as gateways, routing requests but not storing data.

4. **Zone Replication**: Data is replicated across zones. Ensure you have nodes in at least 3 zones for production deployments.

5. **Capacity Planning**: Total cluster capacity should account for replication factor and desired redundancy.
