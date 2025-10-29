---
page_title: "Multi-Organization Support"
description: |-
  Managing services across multiple SkySQL organizations
---

# Multi-Organization Support

Version 3.2.0+ supports managing services across multiple organizations using the `org_id` parameter.

## Usage

### Single Organization (Default)

Omit `org_id` to use your default organization:

```hcl
resource "skysql_service" "my_service" {
  name           = "my-database"
  service_type   = "transactional"
  topology       = "standalone"
  cloud_provider = "aws"
  region         = "us-east-1"
  size           = "sky-2x8"
  storage        = 100
  ssl_enabled    = true
}
```

### Multiple Organizations

Specify `org_id` for each service to manage across different organizations:

```hcl
resource "skysql_service" "prod_db" {
  org_id         = "org-12345-production"
  name           = "prod-database"
  service_type   = "transactional"
  topology       = "standalone"
  cloud_provider = "aws"
  region         = "us-east-1"
  size           = "sky-4x16"
  storage        = 500
  ssl_enabled    = true
}

resource "skysql_service" "dev_db" {
  org_id         = "org-67890-development"
  name           = "dev-database"
  service_type   = "transactional"
  topology       = "standalone"
  cloud_provider = "gcp"
  region         = "us-central1"
  size           = "sky-2x8"
  storage        = 100
  ssl_enabled    = true
}
```

## Important Notes

- **Changing `org_id` destroys and recreates the service**
- The parameter is optional - existing configurations continue to work
