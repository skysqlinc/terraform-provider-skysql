variable "topology" {
  description = "SkySQL topology type to deploy"
  type        = string
  default     = "es-single"
}

variable "project_id" {
  description = "GCP project id"
  type        = string
}

variable "region" {
  description = "GCP region"
  type        = string
  default     = "us-central1"
}

variable "skysql_service_name" {
  description = "Name of the skysql service being created"
  type        = string
}

variable "network" {
  description = "VPC network to connect to skysql service"
  type        = string
  default     = "default"
}

variable "subnetwork" {
  description = "VPC subnetwork to connect to skysql service"
  type        = string
  default     = "default"
}

variable "link_dns" {
  description = "Flag to enable private dns resolution of the skysql domain name"
  type        = bool
  default     = true
}
