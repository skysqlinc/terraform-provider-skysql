variable "topology" {
  description = "SkySQL topology type to deploy"
  type        = string
  default     = "es-single"
}

variable "region" {
  description = "AWS region"
  type        = string
  default     = "us-east-2"
}

variable "skysql_service_name" {
  description = "Name of the skysql service being created"
  type        = string
}

variable "security_group_cidr_blocks" {
  description = "A list of sources to allow access to privatelink connection via security group"
  type        = list(string)
  default     = []
}

variable "vpc_id" {
  description = "ID of the AWS VPC that will host the privatelink endpoint"
  type        = string
}
