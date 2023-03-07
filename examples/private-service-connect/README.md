<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | ~>1.3.7 |
| <a name="requirement_google"></a> [google](#requirement\_google) | ~> 4.22 |
| <a name="requirement_skysql"></a> [skysql](#requirement\_skysql) | ~>1.0.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_google"></a> [google](#provider\_google) | 4.54.0 |
| <a name="provider_skysql"></a> [skysql](#provider\_skysql) | 1.0.0 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [google_compute_address.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_address) | resource |
| [google_compute_forwarding_rule.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_forwarding_rule) | resource |
| [google_dns_managed_zone.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/dns_managed_zone) | resource |
| [google_dns_record_set.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/dns_record_set) | resource |
| skysql_service.this | resource |
| [google_compute_network.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/data-sources/compute_network) | data source |
| [google_project.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/data-sources/project) | data source |
| skysql_service.this | data source |
| skysql_versions.this | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_link_dns"></a> [link\_dns](#input\_link\_dns) | Flag to enable private dns resolution of the skysql domain name | `bool` | `true` | no |
| <a name="input_network"></a> [network](#input\_network) | VPC network to connect to skysql service | `string` | `"default"` | no |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | GCP project id | `string` | n/a | yes |
| <a name="input_region"></a> [region](#input\_region) | GCP region | `string` | `"us-central1"` | no |
| <a name="input_skysql_service_name"></a> [skysql\_service\_name](#input\_skysql\_service\_name) | Name of the skysql service being created | `string` | n/a | yes |
| <a name="input_subnetwork"></a> [subnetwork](#input\_subnetwork) | VPC subnetwork to connect to skysql service | `string` | `"default"` | no |
| <a name="input_topology"></a> [topology](#input\_topology) | SkySQL topology type to deploy | `string` | `"es-single"` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_psc_address"></a> [psc\_address](#output\_psc\_address) | Private IP address assigned to the PSC endpoint |
| <a name="output_skysql_endpoint_service_id"></a> [skysql\_endpoint\_service\_id](#output\_skysql\_endpoint\_service\_id) | SkySQL privatelink endpoint service id |
| <a name="output_skysql_host"></a> [skysql\_host](#output\_skysql\_host) | Hostname for private database connections |
<!-- END_TF_DOCS -->