package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceDatabases(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceDatabase,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("data.skysql_database.wat", "id", regexp.MustCompile("^db")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "cluster", regexp.MustCompile("sky0005572")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "operational_status", regexp.MustCompile("Non-Operational")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "sys_updated_on", regexp.MustCompile(`\d+`)),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "number", regexp.MustCompile("DB00008965")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "instance_state", regexp.MustCompile("Pending")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "read_only_port", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "read_write_port", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "release_version", regexp.MustCompile("MariaDB Enterprise Server openssl1.1.1-1")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "gl_account", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "sys_created_by", regexp.MustCompile("don.mayo@mariadb.com")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "ssl_certificate", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "columnstore_bucket", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "topology", regexp.MustCompile("Standalone")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "owned_by", regexp.MustCompile("Don Mayo")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "proxy", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "size", regexp.MustCompile("Sky-2x4")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "dns_domain", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "tx_storage", regexp.MustCompile("100")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "ssl_expires_on", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "repl_master_host_ext", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "maxscale_config", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "volume_iops", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "volume_type", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "attributes", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "replication_status", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "skip_sync", regexp.MustCompile("false")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "replication_type", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "repl_master", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "sys_updated_by", regexp.MustCompile(".+")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "bulkdata_port_2", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "sys_created_on", regexp.MustCompile("2021-06-17 13:18:35")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "bulkdata_port_1", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "active_replicas", regexp.MustCompile("0")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "fqdn", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "ssl_serial", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "install_status", regexp.MustCompile("Pending Install")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "name", regexp.MustCompile("dmayo-api")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "region", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "repl_region", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "custom_config", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "sys_id", regexp.MustCompile("13107475db38bc50687f0793e2961930")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "cloud_provider", regexp.MustCompile("Amazon AWS")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "mac_address", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "replicas", regexp.MustCompile("0")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "sys_mod_count", regexp.MustCompile(`\d+`)),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "monitor", regexp.MustCompile("false")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "ip_address", regexp.MustCompile("")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "maxscale_proxy", regexp.MustCompile("false")),
					resource.TestMatchResourceAttr("data.skysql_database.wat", "fault_count", regexp.MustCompile("0")),
				),
			},
		},
	})
}

const testAccDataSourceDatabase = `
data "skysql_database" "wat" {
  id = "db00008965"
}
`
