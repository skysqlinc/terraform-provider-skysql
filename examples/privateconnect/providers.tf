terraform {
  required_providers {
    skysql = {
      source  = "registry.terraform.io/skysqlinc/skysql"
      version = "1.0.0"
    }
    aws = {
      source  = "hashicorp/aws"
      version = "4.55.0"
    }
  }
}

provider "skysql" {}
provider "aws" {
  region = var.region
}
