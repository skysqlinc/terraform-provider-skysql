package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceConfig(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.skysql_config.wat", "id", "CFG0001355"),
					resource.TestCheckResourceAttr("data.skysql_config.wat", "name", "interactive-timeout"),
					resource.TestCheckResourceAttr("data.skysql_config.wat", "public", "false"),
					resource.TestCheckResourceAttr("data.skysql_config.wat", "topology", "Standalone"),
					resource.TestCheckResourceAttr("data.skysql_config.wat", "configuration_versions.0.current_version", "true"),
					resource.TestCheckResourceAttr("data.skysql_config.wat", "configuration_versions.0.version", "1"),
					resource.TestMatchResourceAttr("data.skysql_config.wat", "configuration_versions.0.config_json", regexp.MustCompile("300")),
				),
			},
		},
	})
}

const testAccDataSourceConfig = `
data "skysql_config" "wat" {
  id = "CFG0001355"
}
`
