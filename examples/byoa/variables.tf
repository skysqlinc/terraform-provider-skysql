variable "topology" {
  description = "Topology to deploy. Serverless topologies are not available for BYOA organizations."
  type        = string
  default     = "es-single"
}

variable "region" {
  description = "AWS region. Must be enabled for your BYOA account."
  type        = string
  default     = "us-east-1"
}

variable "skysql_service_name" {
  description = "Name of the service being created"
  type        = string
}
