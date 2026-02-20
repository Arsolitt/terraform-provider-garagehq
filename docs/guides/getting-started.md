---
page_title: "Getting Started - Guides - terraform-provider-garage"
description: |-
  Get started with the Garage Terraform Provider
---

# Getting Started

This guide walks you through setting up the Garage Terraform provider and creating your first bucket with access credentials.

## Prerequisites

- Garage v2.x running and accessible
- Admin token for the Garage Admin API
- Terraform >= 1.0 installed

## Provider Configuration

Create a new Terraform project and configure the provider:

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
  host   = "127.0.0.1:3903"  # Admin API endpoint
  scheme = "http"
  token  = var.garage_token
}
```

### Variables

Create a `variables.tf` file:

```hcl
variable "garage_token" {
  description = "Admin token for Garage API"
  type        = string
  sensitive   = true
}
```

## Create Your First Bucket

Create a bucket and an access key:

```hcl
# Create a bucket
resource "garage_bucket" "example" {
  global_alias = "my-first-bucket"
}

# Create an access key
resource "garage_key" "example" {
  name = "example-app-key"
}

# Grant read/write access to the key
resource "garage_bucket_key" "example" {
  bucket_id     = garage_bucket.example.id
  access_key_id = garage_key.example.access_key_id
  read          = true
  write         = true
  owner         = false
}
```

## Outputs

Add outputs to retrieve the credentials:

```hcl
output "bucket_id" {
  value = garage_bucket.example.id
}

output "access_key_id" {
  value = garage_key.example.access_key_id
}

output "secret_access_key" {
  value     = garage_key.example.secret_access_key
  sensitive = true
}

output "s3_endpoint" {
  value = "http://127.0.0.1:3900"  # S3 API endpoint (different from admin)
}
```

## Apply Configuration

```bash
terraform init
terraform apply -var="garage_token=your-admin-token"
```

## Use with AWS CLI

Configure AWS CLI to use your new credentials:

```bash
aws --endpoint-url http://127.0.0.1:3900 \
    s3 ls \
    --profile garage

# Configure profile
aws configure --profile garage
# Access Key: [from terraform output]
# Secret Key: [from terraform output]
# Region: garage
# Output: json
```

## Use with Applications

### Python (boto3)

```python
import boto3

s3 = boto3.client(
    's3',
    endpoint_url='http://127.0.0.1:3900',
    aws_access_key_id='ACCESS_KEY',
    aws_secret_access_key='SECRET_KEY',
)

s3.put_object(Bucket='my-first-bucket', Key='test.txt', Body=b'Hello Garage!')
```

### Go

```go
import (
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/credentials"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/s3"
)

sess, _ := session.NewSession(&aws.Config{
    Endpoint:    aws.String("http://127.0.0.1:3900"),
    Credentials: credentials.NewStaticCredentials("ACCESS_KEY", "SECRET_KEY", ""),
})
client := s3.New(sess)
```

## Next Steps

- [Bucket Lifecycle Policies](bucket-lifecycle.md) - Configure automatic object expiration
- [Admin Tokens](admin-tokens.md) - Create scoped admin tokens for services
- [Cluster Management](cluster-management.md) - Set up multi-node clusters
