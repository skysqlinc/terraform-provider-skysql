terraform {
  required_providers {
    skysql = {
      source = "registry.terraform.io/skysqlinc/skysql-beta"
    }
  }
}

provider "skysql" {}


data "skysql_versions" "default" {
  topology = "es-single"
}


resource "skysql_service" "default" {
  service_type      = "transactional"
  topology          = "es-single"
  cloud_provider    = "aws"
  region            = "us-east-2"
  name              = "myservice"
  architecture      = "amd64"
  nodes             = 1
  size              = "sky-2x8"
  storage           = 100
  ssl_enabled       = true
  version           = data.skysql_versions.default.versions[0].name
  volume_type       = "gp3"
  volume_iops       = 3000
  volume_throughput = 125
  allow_list = [
    {
      "ip" : "127.0.0.1/32",
      "comment" : "localhost"
    }
  ]
  wait_for_creation = true
}

data "skysql_credentials" "default" {
  service_id = skysql_service.default.id
}

data "skysql_service" "default" {
  service_id = skysql_service.default.id
}

output "skysql_service" {
  value = data.skysql_service.default
}

output "skysql_credentials" {
  value     = data.skysql_credentials.default
  sensitive = true
}
