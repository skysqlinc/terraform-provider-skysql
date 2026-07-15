variable "topology" {
  description = "SkySQL topology type to deploy. Serverless topologies are not available for BYOA organizations."
  type        = string
  default     = "es-single"
}

variable "region" {
  description = "AWS region. Must be enabled for your BYOA account."
  type        = string
  default     = "us-east-1"
}

variable "skysql_service_name" {
  description = "Name of the skysql service being created"
  type        = string
}
