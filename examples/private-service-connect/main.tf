data "google_project" "this" {}

data "skysql_versions" "this" {
  topology = var.topology
}

###
# Create the SkySQL service
###
resource "skysql_service" "this" {
  service_type              = "transactional"
  topology                  = var.topology
  cloud_provider            = "gcp"
  region                    = var.region
  name                      = var.skysql_service_name
  architecture              = "amd64"
  nodes                     = 1
  size                      = "sky-2x8"
  storage                   = 100
  ssl_enabled               = true
  version                   = data.skysql_versions.this.versions[0].name
  endpoint_mechanism        = "privateconnect"
  endpoint_allowed_accounts = [data.google_project.this.number]
  wait_for_creation         = true
  # The following line will be required when tearing down the skysql service
  # deletion_protection = false
}

data "skysql_service" "this" {
  service_id = skysql_service.this.id
}

locals {
  # this should work for all topologies other than lakehouse
  readwrite_port = [for p in data.skysql_service.this.endpoints[0].ports : p.port if p.purpose == "readwrite"][0]
  skysql_domain  = "db.skysql.net"
}


###
# Creates the private address and forwarding rule for the PSC endpoint
###
resource "google_compute_address" "this" {
  name         = var.skysql_service_name
  address_type = "INTERNAL"
  subnetwork   = var.subnetwork
  project      = var.project_id
  region       = var.region
}

resource "google_compute_forwarding_rule" "this" {
  name                  = "psc-${var.skysql_service_name}"
  load_balancing_scheme = ""
  region                = var.region
  project               = var.project_id
  ip_address            = google_compute_address.this.id
  target                = data.skysql_service.this.endpoints[0].endpoint_service
  network               = var.network
}


###
# Create the private DNS zone and record for skysql dns resolution within the private network
###
data "google_compute_network" "this" {
  name = var.network
}

resource "google_dns_managed_zone" "this" {
  count       = var.link_dns ? 1 : 0
  name        = "skysql-psc"
  dns_name    = "${local.skysql_domain}."
  description = "SkySQL PSC forwarding zone"
  visibility  = "private"

  private_visibility_config {
    networks {
      network_url = data.google_compute_network.this.id
    }
  }
}

resource "google_dns_record_set" "this" {
  count        = var.link_dns ? 1 : 0
  managed_zone = google_dns_managed_zone.this[0].name
  name         = "${data.skysql_service.this.fqdn}."
  type         = "A"
  ttl          = 300
  rrdatas      = [google_compute_address.this.address]
}
