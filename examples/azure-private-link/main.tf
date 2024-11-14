data "azurerm_subscription" "current" {}

data "azurerm_resource_group" "this" {
  name       = var.resource_group_name
  depends_on = [azurerm_resource_group.this]
}

data "skysql_versions" "this" {
  topology = var.topology
}

data "skysql_service" "this" {
  service_id = skysql_service.this.id
}

###
# Create the SkySQL service
###
resource "skysql_service" "this" {
  service_type              = "transactional"
  topology                  = var.topology
  cloud_provider            = "azure"
  region                    = var.location
  name                      = var.skysql_service_name
  architecture              = "amd64"
  nodes                     = 1
  size                      = "sky-2x8"
  storage                   = 100
  ssl_enabled               = true
  version                   = data.skysql_versions.this.versions[0].name
  endpoint_mechanism        = "privateconnect"
  endpoint_allowed_accounts = [data.azurerm_subscription.current.subscription_id]
  wait_for_creation         = true
  # The following line will be required when tearing down the skysql service
  # deletion_protection = false
}

resource "azurerm_resource_group" "this" {
  count    = var.create_resource_group ? 1 : 0
  name     = var.resource_group_name
  location = var.location
}

resource "azurerm_private_dns_zone" "this" {
  name                = local.dns_domain
  resource_group_name = data.azurerm_resource_group.this.name
}

resource "azurerm_private_dns_zone_virtual_network_link" "this" {
  name                  = local.dns_link_name
  resource_group_name   = data.azurerm_resource_group.this.name
  private_dns_zone_name = azurerm_private_dns_zone.this.name
  virtual_network_id    = var.virtual_network_id
}

resource "azurerm_private_endpoint" "this" {
  name                = var.skysql_service_name
  location            = data.azurerm_resource_group.this.location
  resource_group_name = data.azurerm_resource_group.this.name
  subnet_id           = var.subnet_id

  private_service_connection {
    name                              = var.database_name
    private_connection_resource_alias = data.skysql_service.this.endpoints[0].endpoint_service
    is_manual_connection              = true
    request_message                   = "PL"

  }
}

resource "azurerm_private_dns_a_record" "this" {
  name                = skysql_service.this.id
  zone_name           = azurerm_private_dns_zone.this.name
  resource_group_name = data.azurerm_resource_group.this.name
  ttl                 = 300
  records             = [azurerm_private_endpoint.this.private_service_connection[0].private_ip_address]
}
