data "aws_caller_identity" "this" {}

data "skysql_versions" "this" {
  topology = var.topology
}

###
# Create the SkySQL service
###
resource "skysql_service" "this" {
  service_type              = "transactional"
  topology                  = var.topology
  cloud_provider            = "aws"
  region                    = var.region
  name                      = var.skysql_service_name
  architecture              = "amd64"
  nodes                     = 1
  size                      = "sky-2x8"
  storage                   = 100
  ssl_enabled               = true
  version                   = data.skysql_versions.this.versions[0].name
  endpoint_mechanism        = "privateconnect"
  endpoint_allowed_accounts = [data.aws_caller_identity.this.account_id]
  wait_for_creation         = true
  # The following line will be required when tearing down the skysql service
  # deletion_protection = false
}

data "skysql_service" "this" {
  service_id = skysql_service.this.id
}

data "aws_subnets" "this" {
  filter {
    name   = "vpc-id"
    values = [var.vpc_id]
  }
}

locals {
  # this should work for all topologies other than lakehouse
  readwrite_port = [for p in data.skysql_service.this.endpoints[0].ports : p.port if p.purpose == "readwrite"][0]
}

###
# Creates the security group that grants access to the privatelink endpoint
###
resource "aws_security_group" "this" {
  name        = var.skysql_service_name
  description = "Allow access to SkySQL Privatelink endpoint"
  vpc_id      = var.vpc_id

  ingress {
    from_port   = local.readwrite_port
    to_port     = local.readwrite_port
    protocol    = "tcp"
    cidr_blocks = var.security_group_cidr_blocks
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

###
# Create the AWS Privatelink VPC endpoint
###
resource "aws_vpc_endpoint" "this" {
  vpc_id            = var.vpc_id
  service_name      = data.skysql_service.this.endpoints[0].endpoint_service
  vpc_endpoint_type = "Interface"
  subnet_ids        = data.aws_subnets.this.ids

  security_group_ids = [
    aws_security_group.this.id,
  ]

  # this is disabled due to dns verification by AWS for privatelink being slow which causes the endpoint creation
  # to fail.  This can be enabled after a short wait for the dns verification process to complete.
  private_dns_enabled = false
}
