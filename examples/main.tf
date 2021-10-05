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
