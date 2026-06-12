# Create a service
resource "skysql_service" "default" {
  project_id        = data.skysql_projects.default.projects[0].id
  service_type      = "transactional"
  topology          = "es-single"
  cloud_provider    = "aws"
  region            = "us-east-1"
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
  tags = {
    environment = "production"
    team        = "platform"
  }
  # The service create is an asynchronous operation.
  # if you want to wait for the service to be created set wait_for_creation to true
  wait_for_creation = true
  # Optional: apply a custom configuration object (requires wait_for_creation = true)
  # config_id = skysql_config.tuned.id
}

# Create a Galera cluster (multi-master, high availability)
resource "skysql_service" "galera" {
  project_id     = data.skysql_projects.default.projects[0].id
  service_type   = "transactional"
  topology       = "galera"
  cloud_provider = "gcp"
  region         = "us-central1"
  name           = "my-galera-cluster"
  architecture   = "amd64"
  # Galera replicates synchronously across all nodes.
  # Use an odd number of nodes (3 or 5) to keep quorum.
  nodes = 3
  # MaxScale fronts the cluster and load balances connections across nodes.
  maxscale_nodes = 1
  # sky-4x16 is the minimum size for Galera (sky-2x8 is not supported).
  size        = "sky-4x16"
  storage     = 100
  ssl_enabled = true
  # The version must be available for the galera topology
  # (see the skysql_versions data source).
  version           = "10.6.11-6-1"
  wait_for_creation = true
}