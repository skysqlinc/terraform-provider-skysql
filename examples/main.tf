terraform {
  required_providers {
    skysql = {
      source  = "registry.terraform.io/mariadb-corporation/skysql"
      version = "0.0.1"
    }
  }
}
provider "skysql" {
  mdbid_url = "https://id-test.mariadb.com"
  host      = "https://api.test.gcp.mariadb.net"
}
data "skysql_credentials" "wat" {
  id = skysql_service.wat.id
}
output "wat_credential" {
  value     = data.skysql_credentials.wat
  sensitive = true
}

resource "skysql_service" "wat" {
  release_version = "MariaDB Enterprise Server 10.6.4-1"
  topology        = "Single Node Transactions"
  size            = "Sky-2x4"
  tx_storage      = "100"
  maxscale_config = ""
  name            = "standalone-example"
  region          = "ca-central-1"
  cloud_provider  = "Amazon AWS"
  replicas        = "0"
  monitor         = "false"
  volume_iops     = "100"
  maxscale_proxy  = "false"
  tier            = "Foundation"
}
