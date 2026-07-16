---
page_title: "Bring Your Own Account (BYOA)"
description: |-
  Using the provider with a BYOA organization
---

# Bring Your Own Account (BYOA)

In a BYOA organization, MariaDB Cloud deploys database services into your own cloud account instead of MariaDB Cloud's own infrastructure. BYOA is enabled for an organization by the MariaDB Cloud team, per cloud provider. There is nothing to configure in the provider itself: the API recognizes a BYOA organization from your API key and applies BYOA behavior automatically.

Most things work exactly as they do in any other organization. A few don't, and they matter when writing Terraform.

Serverless is not available. Creating a service with `topology = "serverless-standalone"` fails with `The "serverless-standalone" topology is not available for your organization.` Use a provisioned topology such as `es-single` or `es-replica` instead.

Endpoints are private by default. If you don't configure an IP `allow_list`, the service comes up with a private `privateconnect` endpoint — so set `endpoint_mechanism = "privateconnect"` and `endpoint_allowed_accounts` explicitly, otherwise your configuration says one thing and the created service another. If you do configure an `allow_list`, the service gets a public endpoint restricted to those addresses. The API will not let a BYOA endpoint become a public `nlb` endpoint without a non-empty allow list.

Two smaller points: services always run on dedicated tenancy in your cloud account regardless of configuration, and only the regions enabled for your BYOA account during onboarding are available.

A typical BYOA service looks like this:

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

A complete runnable version of this is in [`examples/byoa`](https://github.com/skysqlinc/terraform-provider-skysql/tree/main/examples/byoa). For wiring up the private endpoint on the consumer side, see the [`privateconnect` (AWS PrivateLink)](https://github.com/skysqlinc/terraform-provider-skysql/tree/main/examples/privateconnect), [`azure-private-link`](https://github.com/skysqlinc/terraform-provider-skysql/tree/main/examples/azure-private-link), and [`private-service-connect` (GCP)](https://github.com/skysqlinc/terraform-provider-skysql/tree/main/examples/private-service-connect) examples.
