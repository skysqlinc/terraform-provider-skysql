terraform {
  required_providers {
    skysql = {
      source  = "registry.terraform.io/skysqlinc/skysql"
      version = "1.0.0"
    }
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "3.96.0"
    }
  }
}

provider "skysql" {}
provider "azurerm" {
  features {}
}
