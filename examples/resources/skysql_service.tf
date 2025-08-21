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
    name        = "myservice"  # API will overwrite this with service name
    environment = "production" # Optional additional tags
  }
  # The service create is an asynchronous operation.
  # if you want to wait for the service to be created set wait_for_creation to true
  wait_for_creation = true
}