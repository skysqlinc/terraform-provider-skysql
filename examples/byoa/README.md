# BYOA (Bring Your Own Account) Example

This example creates a provisioned SkySQL service in a BYOA organization. BYOA is enabled per organization and per cloud provider by the SkySQL team; the provider needs no special configuration — the API detects a BYOA organization from the API key.

## What the API enforces for BYOA organizations

- The service is deployed into your own cloud account on dedicated tenancy.
- Endpoints default to private connectivity. This example sets `endpoint_mechanism = "privateconnect"` and `endpoint_allowed_accounts` explicitly so the configuration matches what is created.
- The `serverless-standalone` topology is not available. Use provisioned topologies such as `es-single` or `es-replica`.
- Only regions enabled for your BYOA account during onboarding can be used.

## Usage

```shell
export TF_SKYSQL_API_KEY="..."
terraform init
terraform apply -var skysql_service_name="my-byoa-database"
```

To connect through the private endpoint from your VPC, see the [`privateconnect`](../privateconnect) (AWS PrivateLink), [`azure-private-link`](../azure-private-link), and [`private-service-connect`](../private-service-connect) (GCP) examples.

See also the [BYOA guide](../../docs/guides/byoa.md).
