locals {
  dns_domain = join(".", [var.skysql_organization_id, var.skysql_base_domain])
  dns_link_name = join(".", [var.skysql_organization_id, replace(var.skysql_base_domain, ".", "-")])
}
