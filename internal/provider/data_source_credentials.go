package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mariadb-corporation/skysql-api-go"
)

func dataSourceCredentials() *schema.Resource {
	s := make(map[string]*schema.Schema)
	s["id"] = &schema.Schema{
		Type:     schema.TypeString,
		Computed: false,
		Required: true,
	}
	for _, field := range credentialsFields() {
		s[reservedNamesAtoT(field.Name)] = &schema.Schema{
			Type:      schema.TypeString,
			Computed:  true,
			Required:  false,
			Sensitive: field.Name == "password",
		}
	}
	return &schema.Resource{
		Description: "Default credentials for connecting to a MariaDB service deployed by SkySQL",
		ReadContext: dataSourceCredentialsRead,
		Schema:      s,
	}
}

func dataSourceCredentialsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*skysql.Client)
	var diags diag.Diagnostics

	id := d.Get("id").(string)

	credentials, err := readCredentials(ctx, client, id)
	if err != nil {
		return err
	}

	for _, field := range credentialsFields() {
		d.Set(reservedNamesAtoT(field.Name), credentials[field.Name])
	}

	d.SetId(id)

	return diags
}

func credentialsFields() []fieldInfo {
	return fields(skysql.DefaultCredentials{})
}

func readCredentials(ctx context.Context, client *skysql.Client, id string) (map[string]interface{}, diag.Diagnostics) {
	res, err := client.RetrieveDefaultCredentials(ctx, id)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	credentials, errDiag := decodeAPIResponseBody(res)
	if errDiag != nil {
		return nil, errDiag
	}

	username := credentials["username"].(string)
	password := credentials["password"].(string)
	if username == "" || password == "" {
		return nil, diag.FromErr(fmt.Errorf("bad response from SkySQL: %v", credentials))
	}

	return credentials, nil
}
