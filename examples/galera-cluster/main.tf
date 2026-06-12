terraform {
  required_providers {
    skysql = {
      source = "skysqlinc/skysql"
    }
  }
}

provider "skysql" {
  # API key can be set via TF_SKYSQL_API_KEY environment variable
}

# Galera Cluster requires PowerPlus tier
# Minimum configuration: 3 or 5 nodes for quorum
# Minimum size: sky-4x16 (sky-2x8 is not supported for Galera)
resource "skysql_service" "galera_cluster" {
  project_id          = var.project_id
  service_type        = "transactional"
  topology            = "galera"
  cloud_provider      = "gcp"
  region              = "us-central1"
  name                = "my-galera-cluster"
  architecture        = "amd64"
  nodes               = 3
  maxscale_nodes      = 1
  size                = "sky-4x16"
  storage             = 100
  ssl_enabled         = true
  nosql_enabled       = false
  version             = "10.6.11-6-1"
  wait_for_creation   = true
  wait_for_deletion   = true
  deletion_protection = true

  allow_list = [
    {
      ip      = "0.0.0.0/0"
      comment = "Allow all - replace with your specific IP ranges"
    }
  ]

  tags = {
    environment = "production"
    topology    = "galera"
  }
}

output "service_id" {
  value = skysql_service.galera_cluster.id
}

output "service_fqdn" {
  value = skysql_service.galera_cluster.fqdn
}
