package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceDatabase(t *testing.T) {
	originalName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	updatedName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceDatabase(originalName),
				Check:  resource.ComposeAggregateTestCheckFunc(dbChecks(originalName)...),
			},
			{
				Config: testAccResourceDatabase(updatedName),
				Check:  resource.ComposeAggregateTestCheckFunc(dbChecks(updatedName)...),
			},
		},
	})
}

func testAccResourceDatabase(nameSeed string) string {
	return fmt.Sprintf(`
	resource "skysql_database" "wat" {
		release_version = "MariaDB Enterprise Server 10.4.18-11"
		topology        = "Standalone"
		size            = "Sky-2x4"
		tx_storage      = "100"
		maxscale_config = ""
		name            = "tf-db-test-%v"
		region          = "ca-central-1"
		repl_region     = ""
		cloud_provider  = "Amazon AWS"
		replicas        = "0"
		monitor         = "false"
		volume_iops     = "100"
		volume_type     = "io1"
		maxscale_proxy  = "false"
		tier            = "Premium"
	}`, nameSeed)
}

func dbChecks(nameSeed string) []resource.TestCheckFunc {
	return []resource.TestCheckFunc{
		resource.TestMatchResourceAttr("skysql_database.wat", "id", regexp.MustCompile("^db")),
		resource.TestMatchResourceAttr("skysql_database.wat", "cluster", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_database.wat", "operational_status", regexp.MustCompile(".+")),
		resource.TestMatchResourceAttr("skysql_database.wat", "sys_updated_on", regexp.MustCompile(`\d+`)),
		resource.TestMatchResourceAttr("skysql_database.wat", "number", regexp.MustCompile("^DB")),
		resource.TestMatchResourceAttr("skysql_database.wat", "instance_state", regexp.MustCompile(".+")),
		resource.TestMatchResourceAttr("skysql_database.wat", "read_only_port", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_database.wat", "read_write_port", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_database.wat", "release_version", regexp.MustCompile("MariaDB Enterprise Server")),
		resource.TestMatchResourceAttr("skysql_database.wat", "gl_account", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_database.wat", "sys_created_by", regexp.MustCompile("svc_skysql_api")),
		resource.TestMatchResourceAttr("skysql_database.wat", "ssl_certificate", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_database.wat", "columnstore_bucket", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_database.wat", "topology", regexp.MustCompile("Standalone")),
		resource.TestMatchResourceAttr("skysql_database.wat", "owned_by", regexp.MustCompile(".+")),
		resource.TestMatchResourceAttr("skysql_database.wat", "proxy", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_database.wat", "size", regexp.MustCompile("Sky-2x4")),
		resource.TestMatchResourceAttr("skysql_database.wat", "dns_domain", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_database.wat", "tx_storage", regexp.MustCompile("100")),
		resource.TestMatchResourceAttr("skysql_database.wat", "ssl_expires_on", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_database.wat", "repl_master_host_ext", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_database.wat", "maxscale_config", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_database.wat", "volume_iops", regexp.MustCompile("100")),
		resource.TestMatchResourceAttr("skysql_database.wat", "attributes", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_database.wat", "replication_status", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_database.wat", "skip_sync", regexp.MustCompile("false")),
		resource.TestMatchResourceAttr("skysql_database.wat", "replication_type", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_database.wat", "repl_master", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_database.wat", "sys_updated_by", regexp.MustCompile(".+")),
		resource.TestMatchResourceAttr("skysql_database.wat", "bulkdata_port_2", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_database.wat", "sys_created_on", regexp.MustCompile(".+")),
		resource.TestMatchResourceAttr("skysql_database.wat", "bulkdata_port_1", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_database.wat", "active_replicas", regexp.MustCompile("0")),
		resource.TestMatchResourceAttr("skysql_database.wat", "fqdn", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_database.wat", "ssl_serial", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_database.wat", "install_status", regexp.MustCompile(".+")),
		resource.TestMatchResourceAttr("skysql_database.wat", "name", regexp.MustCompile(nameSeed)),
		resource.TestMatchResourceAttr("skysql_database.wat", "region", regexp.MustCompile("ca-central-1")),
		resource.TestMatchResourceAttr("skysql_database.wat", "repl_region", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_database.wat", "custom_config", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_database.wat", "sys_id", regexp.MustCompile(".+")),
		resource.TestMatchResourceAttr("skysql_database.wat", "cloud_provider", regexp.MustCompile("Amazon AWS")),
		resource.TestMatchResourceAttr("skysql_database.wat", "mac_address", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_database.wat", "replicas", regexp.MustCompile("0")),
		resource.TestMatchResourceAttr("skysql_database.wat", "sys_mod_count", regexp.MustCompile(".+")),
		resource.TestMatchResourceAttr("skysql_database.wat", "monitor", regexp.MustCompile("false")),
		resource.TestMatchResourceAttr("skysql_database.wat", "ip_address", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_database.wat", "maxscale_proxy", regexp.MustCompile("false")),
		resource.TestMatchResourceAttr("skysql_database.wat", "fault_count", regexp.MustCompile("0")),
	}
}
