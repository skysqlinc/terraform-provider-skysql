variable "cloud_provider" {
  type        = string
  default     = "gcp"
  nullable    = false
  description = "Specify the cloud provider. For additional information, see: https://mariadb.com/docs/skysql-new-release-dbaas/ref/skynr/selections/providers/"
}

data "skysql_availability_zones" "this" {
  region             = "us-central1"
  filter_by_provider = var.cloud_provider
}

output "availability_zones" {
  value = data.skysql_availability_zones.this
}