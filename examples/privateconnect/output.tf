output "skysql_endpoint_service_id" {
  description = "SkySQL privatelink endpoint service id"
  value       = data.skysql_service.this.endpoints[0].endpoint_service
}

output "aws_security_group_id" {
  description = "SkySQL privatelink endpoint service id"
  value       = aws_security_group.this.id
}

output "vpc_endpoint_id" {
  description = "AWS privatelink endpoint id"
  value       = aws_vpc_endpoint.this.id
}
