---
page_title: "Bucket Lifecycle Policies - Guides - terraform-provider-garage"
description: |-
  Configure automatic object expiration with lifecycle policies
---

# Bucket Lifecycle Policies

Garage supports S3-compatible lifecycle policies for automatic object expiration. This guide shows how to configure buckets to automatically delete old objects.

## Use Cases

- **Log retention** - Delete logs after N days
- **Temporary uploads** - Clean up user uploads after processing
- **Backup rotation** - Implement backup retention policies
- **Cache management** - Expire cached objects periodically

## Basic Expiration

The simplest way to set expiration is using the `expiration_days` attribute:

```hcl
resource "garage_bucket" "logs" {
  global_alias    = "application-logs"
  expiration_days = 30  # Objects deleted after 30 days
}
```

## Common Patterns

### Log Aggregation (Loki, Prometheus)

```hcl
resource "garage_bucket" "loki_chunks" {
  global_alias    = "loki-chunks"
  expiration_days = 14  # Keep logs for 2 weeks
}

resource "garage_bucket" "loki_ruler" {
  global_alias    = "loki-ruler"
  expiration_days = 30  # Ruler state for 30 days
}

resource "garage_key" "loki" {
  name = "loki-storage"
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
```

### User Uploads

```hcl
resource "garage_bucket" "uploads" {
  global_alias    = "user-uploads"
  expiration_days = 7  # Clean up unprocessed uploads weekly
}
```

### Backup Retention

```hcl
resource "garage_bucket" "daily_backups" {
  global_alias    = "daily-backups"
  expiration_days = 7  # Keep 7 days of backups
}

resource "garage_bucket" "weekly_backups" {
  global_alias    = "weekly-backups"
  expiration_days = 30  # Keep weekly backups for a month
}

resource "garage_bucket" "monthly_backups" {
  global_alias    = "monthly-backups"
  expiration_days = 365  # Keep monthly backups for a year
}
```

## Disabling Expiration

Set `expiration_days` to 0 to disable:

```hcl
resource "garage_bucket" "permanent_storage" {
  global_alias    = "permanent-files"
  expiration_days = 0  # No automatic deletion
}
```

## How It Works

The provider configures an S3 lifecycle rule with the following XML structure:

```xml
<LifecycleConfiguration>
  <Rule>
    <ID>expire-after-days</ID>
    <Status>Enabled</Status>
    <Filter>
      <Prefix></Prefix>
    </Filter>
    <Expiration>
      <Days>N</Days>
    </Expiration>
  </Rule>
</LifecycleConfiguration>
```

-> **Note** Lifecycle rules use the S3 API endpoint (port 3900 by default), not the Admin API. The provider handles the port switching automatically.

## Considerations

1. **Timing**: Expiration is not exact. Garage processes lifecycle rules periodically, so objects may persist slightly longer than specified.

2. **Irreversible**: Once objects are expired and deleted, they cannot be recovered. Plan your retention policies carefully.

3. **Versioning**: If bucket versioning is enabled (future feature), expiration applies to current versions only.

4. **Performance**: Large numbers of expired objects may cause cleanup delays. Consider batch sizes when planning retention.

## Monitoring

Check bucket statistics to monitor lifecycle effects:

```hcl
output "bucket_stats" {
  value = {
    bytes   = garage_bucket.logs.bytes
    objects = garage_bucket.logs.objects
  }
}
```

## Updating Expiration

You can change `expiration_days` at any time:

```hcl
resource "garage_bucket" "logs" {
  global_alias    = "application-logs"
  expiration_days = var.log_retention_days  # Make it configurable
}
```

The lifecycle policy is updated on the next apply.
