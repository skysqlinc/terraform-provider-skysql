package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceServices(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceService,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("data.skysql_service.wat", "id", regexp.MustCompile("^db")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "cluster", regexp.MustCompile("sky0005572")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "operational_status", regexp.MustCompile("Non-Operational")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "updated_on", regexp.MustCompile(`\d+`)),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "number", regexp.MustCompile("DB00008965")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "instance_state", regexp.MustCompile("Pending")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "read_only_port", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "read_write_port", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "release_version", regexp.MustCompile("MariaDB Enterprise Server openssl1.1.1-1")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "gl_account", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "created_by", regexp.MustCompile("don.mayo@mariadb.com")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "ssl_certificate", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "columnstore_bucket", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "topology", regexp.MustCompile("Standalone")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "owned_by", regexp.MustCompile("Don Mayo")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "proxy", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "size", regexp.MustCompile("Sky-2x4")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "dns_domain", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "tx_storage", regexp.MustCompile("100")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "ssl_expires_on", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "repl_master_host_ext", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "maxscale_config", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "volume_iops", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "volume_type", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "attributes", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "replication_status", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "skip_sync", regexp.MustCompile("false")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "replication_type", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "repl_master", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "updated_by", regexp.MustCompile(".+")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "bulkdata_port_2", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "created_on", regexp.MustCompile("2021-06-17 13:18:35")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "bulkdata_port_1", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "active_replicas", regexp.MustCompile("0")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "fqdn", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "ssl_serial", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "install_status", regexp.MustCompile("Pending Install")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "name", regexp.MustCompile("dmayo-api")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "region", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "repl_region", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "custom_config", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "cloud_provider", regexp.MustCompile("Amazon AWS")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "mac_address", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "replicas", regexp.MustCompile("0")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "mod_count", regexp.MustCompile(`\d+`)),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "monitor", regexp.MustCompile("false")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "ip_address", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "maxscale_proxy", regexp.MustCompile("false")),
					resource.TestMatchResourceAttr("data.skysql_service.wat", "fault_count", regexp.MustCompile("0")),
				),
			},
		},
	})
}

const testAccDataSourceService = `
data "skysql_service" "wat" {
  id = "db00008965"
}
`
