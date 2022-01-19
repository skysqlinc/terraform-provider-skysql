package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceCredentials(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceCredentials,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("data.skysql_credentials.wat", "username", regexp.MustCompile(".+")),
					resource.TestMatchResourceAttr("data.skysql_credentials.wat", "password", regexp.MustCompile(".+")),
				),
			},
		},
	})
}

const testAccDataSourceCredentials = `
data "skysql_credentials" "wat" {
  id = "db00011100"
}
`
