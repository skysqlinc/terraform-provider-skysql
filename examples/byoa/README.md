# BYOA (Bring Your Own Account) Example

This example creates a provisioned MariaDB Cloud service in a BYOA organization — one where services are deployed into your own cloud account rather than MariaDB Cloud's own infrastructure. BYOA is enabled for an organization by the MariaDB Cloud team, and the provider needs no special configuration: the API detects a BYOA organization from the API key.

Two things to know when writing Terraform for a BYOA organization. Serverless is not available — `serverless-standalone` is rejected, so use a provisioned topology such as `es-single` or `es-replica`. And endpoints are private by default: this example sets `endpoint_mechanism = "privateconnect"` and `endpoint_allowed_accounts` explicitly so the configuration matches what is actually created. Services run on dedicated tenancy in your account, and only the regions enabled for your BYOA account during onboarding can be used.

## Usage

```shell
export TF_SKYSQL_API_KEY="..."
terraform init
terraform apply -var skysql_service_name="my-byoa-database"
```

To connect through the private endpoint from your VPC, see the [`privateconnect`](../privateconnect) (AWS PrivateLink), [`azure-private-link`](../azure-private-link), and [`private-service-connect`](../private-service-connect) (GCP) examples. There is also a [BYOA guide](../../docs/guides/byoa.md) with more detail.
