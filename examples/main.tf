terraform {
  required_providers {
    skysql = {
      source  = "registry.terraform.io/mariadb-corporation/skysql"
      version = "0.0.1"
    }
  }
}
provider "skysql" {}
data "skysql_service" "existing-service" {
  id = "db00008965"
}
data "skysql_config" "existing-config" {
  id = "CFG0001355"
}
output "existing-service-details" {
  value = data.skysql_service.existing-service
}
output "existing-config-details" {
  value = data.skysql_config.existing-config
}
resource "skysql_service" "dmayo-example-resource" {
  release_version = "MariaDB Enterprise Server 10.6.4-1"
  topology        = "Standalone"
  size            = "Sky-2x4"
  tx_storage      = "100"
  maxscale_config = ""
  name            = "dmayo-example"
  region          = "ca-central-1"
  repl_region     = ""
  cloud_provider  = "Amazon AWS"
  replicas        = "0"
  monitor         = "false"
  volume_iops     = "100"
  volume_type     = "io1"
  maxscale_proxy  = "false"
  tier            = "Foundation"
}
