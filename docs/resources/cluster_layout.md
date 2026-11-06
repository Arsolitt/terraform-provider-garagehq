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

~> **Warning** Import writes all roles of the layout to the state. When the configuration lists only some of the nodes, the next apply removes the other nodes from the layout. Make sure that `roles` lists all nodes before you apply.

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

1. **Layout Changes**: Changes to the layout are applied immediately. Ensure you have proper change management procedures. The provider stages only the roles that are different from the current layout, and applies a new layout version only when it staged a change.

2. **Staged Changes**: When a role must change, the provider refuses to continue if the layout has staged changes that are not applied, for example from `garage layout assign`, because the apply would also apply them. Apply or revert them first (`garage layout apply` or `garage layout revert`). When no role must change, the provider applies nothing, thus staged changes do not stop it. When Garage refuses to apply the layout, the provider reverts the changes that it staged.

3. **Managed Nodes**: The resource manages only the nodes in `roles`. It does not change or remove the role of a node that is not in `roles`, and that node is not in the state. When you remove a node from `roles`, the provider removes the node from the layout. When a managed node leaves the layout outside Terraform, the next plan adds it again.

4. **Destroy**: `terraform destroy` removes the resource only from the state. The roles of the nodes stay in the layout, and the provider does not revert staged changes. To remove a node from the layout, remove it from `roles` and apply.

5. **Node IDs**: The node ID must match the ID configured in the Garage node's configuration file.

6. **Gateway Nodes**: Nodes without a `capacity` attribute operate as gateways, routing requests but not storing data.

7. **Zone Replication**: Data is replicated across zones. Ensure you have nodes in at least 3 zones for production deployments.

8. **Capacity Planning**: Total cluster capacity should account for replication factor and desired redundancy.
