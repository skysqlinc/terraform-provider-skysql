<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_aws"></a> [aws](#requirement\_aws) | 4.55.0 |
| <a name="requirement_skysql"></a> [skysql](#requirement\_skysql) | 1.0.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_aws"></a> [aws](#provider\_aws) | 4.55.0 |
| <a name="provider_skysql"></a> [skysql](#provider\_skysql) | 1.0.0 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [aws_security_group.this](https://registry.terraform.io/providers/hashicorp/aws/4.55.0/docs/resources/security_group) | resource |
| [aws_vpc_endpoint.this](https://registry.terraform.io/providers/hashicorp/aws/4.55.0/docs/resources/vpc_endpoint) | resource |
| skysql_service.this | resource |
| [aws_caller_identity.this](https://registry.terraform.io/providers/hashicorp/aws/4.55.0/docs/data-sources/caller_identity) | data source |
| [aws_subnets.this](https://registry.terraform.io/providers/hashicorp/aws/4.55.0/docs/data-sources/subnets) | data source |
| skysql_service.this | data source |
| skysql_versions.this | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_region"></a> [region](#input\_region) | AWS region | `string` | `"us-east-2"` | no |
| <a name="input_security_group_cidr_blocks"></a> [security\_group\_cidr\_blocks](#input\_security\_group\_cidr\_blocks) | A list of sources to allow access to privatelink connection via security group | `list(string)` | `[]` | no |
| <a name="input_skysql_service_name"></a> [skysql\_service\_name](#input\_skysql\_service\_name) | Name of the skysql service being created | `string` | n/a | yes |
| <a name="input_topology"></a> [topology](#input\_topology) | SkySQL topology type to deploy | `string` | `"es-single"` | no |
| <a name="input_vpc_id"></a> [vpc\_id](#input\_vpc\_id) | ID of the AWS VPC that will host the privatelink endpoint | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_aws_security_group_id"></a> [aws\_security\_group\_id](#output\_aws\_security\_group\_id) | SkySQL privatelink endpoint service id |
| <a name="output_skysql_endpoint_service_id"></a> [skysql\_endpoint\_service\_id](#output\_skysql\_endpoint\_service\_id) | SkySQL privatelink endpoint service id |
| <a name="output_vpc_endpoint_id"></a> [vpc\_endpoint\_id](#output\_vpc\_endpoint\_id) | AWS privatelink endpoint id |
<!-- END_TF_DOCS -->