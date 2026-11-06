---
page_title: "garage_node Data Source - terraform-provider-garage"
description: |-
  Reads a node of the Garage cluster.
---

# garage_node (Data Source)

Reads a node of the Garage cluster: its ID, its status, and its role in the current layout. Without `id`, it reads the node that serves the admin API.

Use it to get the node ID for a [`garage_cluster_layout`](../resources/cluster_layout.md). Garage makes the node ID from the node key on the first start, so you cannot know it before.

## Example Usage

### Node That Serves the Admin API

```hcl
data "garage_node" "self" {}

resource "garage_cluster_layout" "dev" {
  roles {
    id       = data.garage_node.self.id
    zone     = "local"
    capacity = "10G"
  }
}
```

### Node by ID

```hcl
data "garage_node" "storage_2" {
  id = "4a6ae5a1d0d33bf895f5bb4f0a418b7dc94c47c0dd2eb108d3a9d87f5ed77126"
}

output "storage_2_is_up" {
  value = data.garage_node.storage_2.is_up
}
```

## Schema

### Optional

- `id` (String) - Node ID (64 lowercase hexadecimal characters). Defaults to the ID of the node that serves the admin API.

### Read-Only

- `hostname` (String) - Hostname of the node
- `address` (String) - RPC address of the node
- `garage_version` (String) - Garage version of the node
- `is_up` (Boolean) - Whether the node is connected to the cluster
- `draining` (Boolean) - Whether the node has no role in the newest layout, but still holds data of an older layout version
- `zone` (String) - Zone of the node in the current layout. Empty if the node has no role.
- `capacity` (String) - Storage capacity of the node in the current layout, in the format of `garage_cluster_layout`. Empty for a gateway node or a node without a role.
- `tags` (List of String) - Tags of the node in the current layout

The node must be known to the cluster: connected, or in the layout. Else the read fails.

`capacity` uses a unit suffix only when it gives the exact value (`1G` is 1073741824 bytes). Else it is a plain number of bytes, for example `1000000000` for a capacity that `garage layout assign -c 1G` set.
