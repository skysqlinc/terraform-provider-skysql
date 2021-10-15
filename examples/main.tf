terraform {
  required_providers {
    skysql = {
      source  = "registry.terraform.io/mariadb-corporation/skysql"
      version = "0.0.1"
    }
  }
}
provider "skysql" {}
data "skysql_database" "wat" {
  id = "db00008965"
}
output "wat" {
  value = data.skysql_database.wat
}
resource "skysql_database" "wat" {
  release_version = "MariaDB Enterprise Server 10.5.9-6"
  topology        = "Standalone"
  size            = "Sky-2x4"
  tx_storage      = "100"
  maxscale_config = ""
  name            = "standalone-example"
  region          = "ca-central-1"
  repl_region     = ""
  cloud_provider  = "Amazon AWS"
  replicas        = "0"
  monitor         = "false"
  volume_iops     = "100"
  volume_type     = "io1"
  maxscale_proxy  = "false"
}
