variable "location" {
  description = "The Azure Region in which all resources will be created."
  type        = string
  default     = "eastus"
}

variable "resource_group_name" {
  description = "The name of the resource group in which all resources will be created."
  type        = string
  default     = "skysql-private-link"
}

variable "create_resource_group" {
  description = "Create a new resource group or use an existing one."
  type        = bool
  default     = true
}

variable "skysql_organization_id" {
  description = "The SkySQL Organization ID."
  type        = string
}

variable "skysql_base_domain" {
  description = "The base domain for SkySQL database endpoints."
  default     = "db3.skysql.com"
  type        = string
}

variable "virtual_network_id" {
  description = "The ID of the virtual network where the private endpoint will be created."
  type        = string
}

variable "subnet_id" {
  description = "The ID of the subnet where the private endpoint will be created."
  type        = string
}

variable "skysql_service_name" {
  description = "The name of the database to create."
  type        = string
  default     = "skysql-private-link"
}

variable "topology" {
  description = "The SkySQL topology to deploy."
  type        = string
  default     = "es-single"
}
