data "aws_caller_identity" "this" {}

data "skysql_versions" "this" {
  topology = var.topology
}

###
# Create a MariaDB Cloud service in a BYOA (Bring Your Own Account)
# organization. The API detects a BYOA organization automatically: the service
# is deployed into your own cloud account, runs on dedicated tenancy, and
# endpoints default to private connectivity.
#
# The serverless-standalone topology is not available for BYOA organizations;
# use a provisioned topology such as es-single or es-replica.
###
resource "skysql_service" "this" {
  service_type              = "transactional"
  topology                  = var.topology
  cloud_provider            = "aws"
  region                    = var.region
  name                      = var.skysql_service_name
  architecture              = "amd64"
  nodes                     = 1
  size                      = "sky-2x8"
  storage                   = 100
  ssl_enabled               = true
  version                   = data.skysql_versions.this.versions[0].name
  endpoint_mechanism        = "privateconnect"
  endpoint_allowed_accounts = [data.aws_caller_identity.this.account_id]
  wait_for_creation         = true
  volume_type               = "gp3"
  volume_iops               = 3000
  volume_throughput         = 125
  # The following line will be required when tearing down the skysql service
  # deletion_protection = false
}

data "skysql_service" "this" {
  service_id = skysql_service.this.id
}
