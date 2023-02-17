provider "skysql" {}

# Retrieve the list of available versions for each topology like standalone, masterslave, xpand-direct etc
data "skysql_versions" "default" {
  topology = "es-single"
}


# Retrieve the list of projects. Project is a way of grouping the services.
# Note: Next release will make project_id optional in the create service api
data "skysql_projects" "default" {}

output "skysql_projects" {
  value = data.skysql_projects.default
}

# Create a service
resource "skysql_service" "default" {
  service_type   = "transactional"
  topology       = "es-single"
  cloud_provider = "gcp"
  region         = "us-central1"
  name           = "myservice"
  architecture   = "amd64"
  nodes          = 1
  size           = "sky-2x8"
  storage        = 100
  ssl_enabled    = true
  version        = data.skysql_versions.default.versions[0].name
  # [Optional] Below you can find example with optional parameters how to configure a privatelink connection
  endpoint_mechanism        = "privatelink"
  endpoint_allowed_accounts = ["gcp-project-id"]
  # [/Optional]
  # The service create is an asynchronous operation.
  # if you want to wait for the service to be created set wait_for_creation to true
  wait_for_creation = true
  # You need to add your ip address in the CIRD format to allow list in order to connect to the service
  allow_list = [
    {
      "ip" : "104.28.203.45/32",
      "comment" : "homeoffice"
    }
  ]
}

# Retrieve the service default credentials.
# When the service is created please change the default credentials
data "skysql_credentials" "default" {
  service_id = skysql_service.default.id
}

# Retrieve the service details
data "skysql_service" "default" {
  service_id = skysql_service.default.id
}

# Show the service details
output "skysql_service" {
  value = data.skysql_service.default
}

# Show the service credentials
output "skysql_credentials" {
  value     = data.skysql_credentials.default
  sensitive = true
}


# Example how you can generate a command line for the database connection
output "skysql_cmd" {
  value = "mariadb --host ${data.skysql_service.default.fqdn} --port 3306 --user ${data.skysql_service.default.service_id} -p --ssl-ca ~/Downloads/skysql_chain_2022.pem"
}
