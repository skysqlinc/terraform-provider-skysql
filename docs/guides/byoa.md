---
page_title: "Bring Your Own Account (BYOA)"
description: |-
  Using the provider with a BYOA organization
---

# Bring Your Own Account (BYOA)

In a BYOA (Bring Your Own Account) organization, SkySQL deploys database services into your own cloud account instead of SkySQL-managed infrastructure. BYOA is enabled per organization and per cloud provider by the SkySQL team — there is nothing to configure in the provider itself. The API recognizes a BYOA organization from your API key and applies the rules below automatically.

## What is different for a BYOA organization

### Topologies

Serverless is not available. Creating a service with `topology = "serverless-standalone"` fails with:

```
The "serverless-standalone" topology is not available for your organization.
```

Use provisioned topologies such as `es-single` or `es-replica` instead.

### Tenancy

BYOA services always run on dedicated tenancy in your cloud account. The API enforces this regardless of configuration.

### Endpoints

BYOA services default to private connectivity:

- Without an IP `allow_list`, the service is created with a private `privateconnect` endpoint. Set `endpoint_mechanism = "privateconnect"` and `endpoint_allowed_accounts` explicitly so your configuration matches what the API creates.
- With an IP `allow_list`, the service is created with a public endpoint restricted to those addresses. The API rejects switching a BYOA endpoint to a public `nlb` endpoint without a non-empty allow list.

### Regions

Only the regions enabled for your BYOA account during onboarding are available to your organization.

## Example

```hcl
provider "skysql" {}

data "aws_caller_identity" "this" {}

data "skysql_versions" "this" {
  topology = "es-single"
}

resource "skysql_service" "this" {
  service_type              = "transactional"
  topology                  = "es-single"
  cloud_provider            = "aws"
  region                    = "us-east-1"
  name                      = "my-byoa-database"
  architecture              = "amd64"
  nodes                     = 1
  size                      = "sky-2x8"
  storage                   = 100
  ssl_enabled               = true
  version                   = data.skysql_versions.this.versions[0].name
  endpoint_mechanism        = "privateconnect"
  endpoint_allowed_accounts = [data.aws_caller_identity.this.account_id]
  wait_for_creation         = true
}
```

A complete runnable example is available in [`examples/byoa`](https://github.com/skysqlinc/terraform-provider-skysql/tree/main/examples/byoa). For wiring up the private endpoint on the consumer side, see the [`privateconnect` (AWS PrivateLink)](https://github.com/skysqlinc/terraform-provider-skysql/tree/main/examples/privateconnect), [`azure-private-link`](https://github.com/skysqlinc/terraform-provider-skysql/tree/main/examples/azure-private-link), and [`private-service-connect` (GCP)](https://github.com/skysqlinc/terraform-provider-skysql/tree/main/examples/private-service-connect) examples.
