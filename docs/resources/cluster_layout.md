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
    capacity = 1073741824  # 1GB
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
    capacity = 107374182400  # 100GB
    tags     = ["storage", "ssd"]
  }

  roles {
    id       = "node-2"
    zone     = "dc1"
    capacity = 107374182400  # 100GB
    tags     = ["storage", "ssd"]
  }

  roles {
    id       = "node-3"
    zone     = "dc2"
    capacity = 107374182400  # 100GB
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
    capacity = 53687091200  # 50GB
    tags     = ["storage"]
  }

  roles {
    id       = "storage-2"
    zone     = "zone-b"
    capacity = 53687091200  # 50GB
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
    capacity = 21474836480  # 20GB
    tags     = ["primary", "fast"]
  }

  roles {
    id       = "eu-west-1b"
    zone     = "eu-west-1"
    capacity = 21474836480
    tags     = ["primary", "fast"]
  }

  roles {
    id       = "us-east-1a"
    zone     = "us-east-1"
    capacity = 21474836480
    tags     = ["secondary", "standard"]
  }

  roles {
    id       = "us-east-1b"
    zone     = "us-east-1"
    capacity = 21474836480
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
- `capacity` (Number, Optional) - Storage capacity in bytes. Omit for gateway nodes.
- `tags` (List of String, Optional) - Tags assigned to the node

### Read-Only

- `version` (Number) - Current layout version

## Capacity Calculation

Capacity is specified in bytes. Common values:

| Capacity | Bytes |
|----------|-------|
| 1 GB | 1073741824 |
| 10 GB | 10737418240 |
| 100 GB | 107374182400 |
| 1 TB | 1099511627776 |

```hcl
locals {
  gb = 1073741824
  tb = 1099511627776
}

resource "garage_cluster_layout" "example" {
  roles {
    id       = "node-1"
    zone     = "dc1"
    capacity = 100 * local.gb  # 100GB
  }
}
```

## Important Considerations

1. **Layout Changes**: Changes to the layout are applied immediately. Ensure you have proper change management procedures.

2. **Node IDs**: The node ID must match the ID configured in the Garage node's configuration file.

3. **Gateway Nodes**: Nodes without a `capacity` attribute operate as gateways, routing requests but not storing data.

4. **Zone Replication**: Data is replicated across zones. Ensure you have nodes in at least 3 zones for production deployments.

5. **Capacity Planning**: Total cluster capacity should account for replication factor and desired redundancy.
