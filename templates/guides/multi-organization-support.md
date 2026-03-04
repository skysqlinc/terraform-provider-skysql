---
page_title: "Multi-Organization Support"
description: |-
  Managing services across multiple SkySQL organizations
---

# Multi-Organization Support

The SkySQL provider supports managing resources across multiple organizations using the `org_id` provider attribute. When set, all resources and data sources managed by that provider instance operate in the context of the specified organization.

When `org_id` is omitted, the API key's default (primary) organization is used.

## Single Organization

```hcl
provider "skysql" {}

resource "skysql_service" "default" {
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

## Targeting a Specific Organization

```hcl
provider "skysql" {
  org_id = "org-12345-production"
}
```

All resources — services, allow lists, configs, autonomous actions — and all data sources will use this organization.

## Managing Multiple Organizations

Since `org_id` is a provider-level setting, you need one provider instance per organization. There are two approaches:

### Option 1: Separate Workspaces (Recommended)

Use a separate Terraform workspace or root module per organization. This is the cleanest approach — each workspace has its own state, and there is no risk of accidentally modifying resources in the wrong org.

```
infra/
├── prod/
│   ├── main.tf          # provider "skysql" { org_id = var.org_id }
│   └── terraform.tfvars # org_id = "org-12345-production"
└── dev/
    ├── main.tf          # provider "skysql" { org_id = var.org_id }
    └── terraform.tfvars # org_id = "org-67890-development"
```

Each workspace uses the same module but with a different `org_id`:

```hcl
# main.tf (shared across workspaces)
variable "org_id" {
  description = "SkySQL organization ID"
  type        = string
}

provider "skysql" {
  org_id = var.org_id
}

resource "skysql_service" "db" {
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

Or use the `TF_SKYSQL_ORG_ID` environment variable in CI/CD pipelines:

```bash
# Deploy to production org
export TF_SKYSQL_ORG_ID="org-12345-production"
terraform apply

# Deploy to dev org
export TF_SKYSQL_ORG_ID="org-67890-development"
terraform apply
```

### Option 2: Provider Aliases

Use [provider aliases](https://developer.hashicorp.com/terraform/language/providers/configuration#alias-multiple-provider-configurations) to manage multiple organizations in a single Terraform configuration. Every resource and data source must specify which provider alias to use via the `provider` argument.

```hcl
provider "skysql" {
  alias  = "prod"
  org_id = "org-12345-production"
}

provider "skysql" {
  alias  = "dev"
  org_id = "org-67890-development"
}

# Service in production org
resource "skysql_service" "prod_db" {
  provider       = skysql.prod
  name           = "prod-database"
  service_type   = "transactional"
  topology       = "standalone"
  cloud_provider = "aws"
  region         = "us-east-1"
  size           = "sky-4x16"
  storage        = 500
  ssl_enabled    = true
}

# Service in dev org
resource "skysql_service" "dev_db" {
  provider       = skysql.dev
  name           = "dev-database"
  service_type   = "transactional"
  topology       = "standalone"
  cloud_provider = "gcp"
  region         = "us-central1"
  size           = "sky-2x8"
  storage        = 100
  ssl_enabled    = true
}

# Data sources also require the provider argument
data "skysql_credentials" "prod_creds" {
  provider   = skysql.prod
  service_id = skysql_service.prod_db.id
}

# Allow lists, configs, and autonomous actions work the same way
resource "skysql_allow_list" "prod_allowlist" {
  provider   = skysql.prod
  service_id = skysql_service.prod_db.id
  allow_list = [
    {
      ip      = "10.0.0.0/8"
      comment = "internal"
    }
  ]
}
```

-> **Note:** If you omit `provider = skysql.<alias>` on a resource, Terraform uses the default (un-aliased) provider, which may be a different organization or no organization at all.

## Migrating from Per-Resource `org_id`

In versions prior to 3.5.0, `org_id` was an attribute on the `skysql_service` resource. To migrate:

1. Remove `org_id` from all `skysql_service` resource blocks.
2. Add `org_id` to your `provider "skysql"` block (or set `TF_SKYSQL_ORG_ID`).
3. If you had services in different orgs, split them across provider aliases or separate workspaces.
4. Run `terraform plan` to verify no changes are detected.

## Important Notes

- Your API key must have access to the specified organization
- The `org_id` is optional — omitting it uses the API key's default organization
- All resource types (services, allow lists, configs, autonomous actions) and data sources inherit the org from their provider instance
