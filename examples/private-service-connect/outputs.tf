output "skysql_endpoint_service_id" {
  description = "SkySQL privatelink endpoint service id"
  value       = data.skysql_service.this.endpoints[0].endpoint_service
}

output "psc_address" {
  description = "Private IP address assigned to the PSC endpoint"
  value       = google_compute_address.this.address
}

output "skysql_host" {
  description = "Hostname for private database connections"
  value       = var.link_dns ? data.skysql_service.this.fqdn : google_compute_address.this.address
}
