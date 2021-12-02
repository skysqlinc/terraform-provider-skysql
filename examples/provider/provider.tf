terraform {
  required_providers {
    skysql = {
      source  = "registry.terraform.io/mariadb-corporation/skysql"
      version = "0.0.1"
    }
  }
}

provider "skysql" {}
