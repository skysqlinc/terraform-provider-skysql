package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceService(t *testing.T) {
	originalName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	updatedName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceService(originalName),
				Check:  resource.ComposeAggregateTestCheckFunc(svcChecks(originalName)...),
			},
			{
				Config: testAccResourceService(updatedName),
				Check:  resource.ComposeAggregateTestCheckFunc(svcChecks(updatedName)...),
			},
		},
	})
}

func testAccResourceService(nameSeed string) string {
	return fmt.Sprintf(`
	resource "skysql_service" "wat" {
		release_version = "MariaDB Enterprise Server 10.6.4-1"
		topology        = "Single Node Transactions"
		size            = "Sky-2x4"
		tx_storage      = "100"
		maxscale_config = ""
		name            = "tf-svc-test-%v"
		region          = "ca-central-1"
		repl_region     = ""
		cloud_provider  = "Amazon AWS"
		replicas        = "0"
		monitor         = "false"
		volume_iops     = "100"
		volume_type     = "io1"
		maxscale_proxy  = "false"
		tier            = "Foundation"
		ssl_tls         = "Enabled"
	}`, nameSeed)
}

func svcChecks(nameSeed string) []resource.TestCheckFunc {
	return []resource.TestCheckFunc{
		resource.TestMatchResourceAttr("skysql_service.wat", "id", regexp.MustCompile("^db")),
		resource.TestMatchResourceAttr("skysql_service.wat", "cluster", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_service.wat", "operational_status", regexp.MustCompile(".+")),
		resource.TestMatchResourceAttr("skysql_service.wat", "updated_on", regexp.MustCompile(`\d+`)),
		resource.TestMatchResourceAttr("skysql_service.wat", "number", regexp.MustCompile("^DB")),
		resource.TestMatchResourceAttr("skysql_service.wat", "instance_state", regexp.MustCompile(".+")),
		resource.TestMatchResourceAttr("skysql_service.wat", "read_only_port", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_service.wat", "read_write_port", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_service.wat", "release_version", regexp.MustCompile("MariaDB Enterprise Server")),
		resource.TestMatchResourceAttr("skysql_service.wat", "gl_account", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_service.wat", "created_by", regexp.MustCompile("svc_skysql_api")),
		resource.TestMatchResourceAttr("skysql_service.wat", "ssl_certificate", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_service.wat", "columnstore_bucket", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_service.wat", "topology", regexp.MustCompile("Single Node Transactions")),
		resource.TestMatchResourceAttr("skysql_service.wat", "owned_by", regexp.MustCompile(".+")),
		resource.TestMatchResourceAttr("skysql_service.wat", "proxy", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_service.wat", "size", regexp.MustCompile("Sky-2x4")),
		resource.TestMatchResourceAttr("skysql_service.wat", "dns_domain", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_service.wat", "tx_storage", regexp.MustCompile("100")),
		resource.TestMatchResourceAttr("skysql_service.wat", "ssl_expires_on", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_service.wat", "repl_master_host_ext", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_service.wat", "maxscale_config", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_service.wat", "volume_iops", regexp.MustCompile("100")),
		resource.TestMatchResourceAttr("skysql_service.wat", "attributes", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_service.wat", "replication_status", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_service.wat", "skip_sync", regexp.MustCompile("false")),
		resource.TestMatchResourceAttr("skysql_service.wat", "replication_type", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_service.wat", "repl_master", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_service.wat", "updated_by", regexp.MustCompile(".+")),
		resource.TestMatchResourceAttr("skysql_service.wat", "bulkdata_port_2", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_service.wat", "created_on", regexp.MustCompile(".+")),
		resource.TestMatchResourceAttr("skysql_service.wat", "bulkdata_port_1", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_service.wat", "active_replicas", regexp.MustCompile("0")),
		resource.TestMatchResourceAttr("skysql_service.wat", "fqdn", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_service.wat", "ssl_serial", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_service.wat", "install_status", regexp.MustCompile(".+")),
		resource.TestMatchResourceAttr("skysql_service.wat", "name", regexp.MustCompile(nameSeed)),
		resource.TestMatchResourceAttr("skysql_service.wat", "region", regexp.MustCompile("ca-central-1")),
		resource.TestMatchResourceAttr("skysql_service.wat", "repl_region", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_service.wat", "custom_config", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_service.wat", "cloud_provider", regexp.MustCompile("Amazon AWS")),
		resource.TestMatchResourceAttr("skysql_service.wat", "mac_address", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_service.wat", "replicas", regexp.MustCompile("0")),
		resource.TestMatchResourceAttr("skysql_service.wat", "mod_count", regexp.MustCompile(".+")),
		resource.TestMatchResourceAttr("skysql_service.wat", "monitor", regexp.MustCompile("false")),
		resource.TestMatchResourceAttr("skysql_service.wat", "ip_address", regexp.MustCompile("")),
		resource.TestMatchResourceAttr("skysql_service.wat", "maxscale_proxy", regexp.MustCompile("false")),
		resource.TestMatchResourceAttr("skysql_service.wat", "fault_count", regexp.MustCompile("0")),
	}
}
