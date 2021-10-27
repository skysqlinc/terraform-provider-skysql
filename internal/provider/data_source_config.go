package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mariadb-corporation/skysql-api-go"
)

func dataSourceConfig() *schema.Resource {
	s := make(map[string]*schema.Schema)

	// "number" will be used for the resource id
	s["id"] = &schema.Schema{
		Type:     schema.TypeString,
		Computed: false,
		Required: true,
	}

	// add config fields to schema
	for _, field := range configFields() {
		s[reservedNamesAtoT(field.Name)] = &schema.Schema{
			Type:     schema.TypeString,
			Computed: true,
			Required: false,
		}
	}

	// add nested config version fields to schema
	vs := make(map[string]*schema.Schema)
	for _, field := range configVersionFields() {
		vs[reservedNamesAtoT(field.Name)] = &schema.Schema{
			Type:     schema.TypeString,
			Computed: true,
			Required: false,
		}
	}
	s["configuration_versions"] = &schema.Schema{
		Type:     schema.TypeList,
		Elem:     &schema.Resource{Schema: vs},
		Computed: true,
		Required: false,
	}
	return &schema.Resource{
		Description: "MariaDB config deployed by SkySQL",
		ReadContext: dataSourceConfigRead,
		Schema:      s,
	}
}

func dataSourceConfigRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// use the meta value to retrieve your client from the provider configure method
	client := meta.(*skysql.Client)
	var diags diag.Diagnostics

	id := d.Get("id").(string)

	config, err := readConfig(ctx, client, id)
	if err != nil {
		return err
	}

	d.Set("name", config.Name)
	d.Set("public", config.Public)
	d.Set("topology", config.Topology)

	cvs, err := flattenConfigurationVersions(config.ConfigurationVersions)
	if err != nil {
		return err
	}
	d.Set("configuration_versions", cvs)

	d.SetId(id)

	return diags
}

func configFields() []fieldInfo {
	exclude := map[string]bool{
		"number":                 true, // using this field for the id
		"configuration_versions": true, // nested list that needs to be handled explicitly
		"sys_id":                 true, // we're omitting these now, will be removed from api soon
	}
	return fields(skysql.ConfigurationResp{}, exclude)
}

func configVersionFields() []fieldInfo {
	return fields(skysql.ConfigurationVersionResp{})
}

// func configCreateFields() []fieldInfo {
// 	return fields(skysql.CreateConfigurationResp{})
// }

// func configUpdateFields() []fieldInfo {
// 	return fields(skysql.UpdateConfigurationRequest{})
// }

func readConfig(ctx context.Context, client *skysql.Client, id string) (*skysql.ConfigurationResp, diag.Diagnostics) {
	res, err := client.ReadConfiguration(ctx, id)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	config, errDiag := decodeConfigResponse(res)
	if errDiag != nil {
		return nil, errDiag
	}

	configID := config.Number
	if configID != id {
		return nil, diag.FromErr(fmt.Errorf("bad response from SkySQL: %v", config))
	}

	return config, nil
}

func decodeConfigResponse(res *http.Response) (*skysql.ConfigurationResp, diag.Diagnostics) {
	defer res.Body.Close()

	err := checkAPIStatus(res.StatusCode, res.Body)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	var decodedBody skysql.ConfigurationResp
	if err := json.NewDecoder(res.Body).Decode(&decodedBody); err != nil {
		return nil, diag.FromErr(err)
	}
	return &decodedBody, nil
}

func flattenConfigurationVersions(cvs *[]skysql.ConfigurationVersionResp) ([]map[string]interface{}, diag.Diagnostics) {
	empty := make([]map[string]interface{}, 0)
	if cvs == nil {
		return empty, nil
	}
	flat := make([]map[string]interface{}, len(*cvs))
	for i, cv := range *cvs {
		cfgJson, err := json.Marshal(cv.ConfigJson)
		if err != nil {
			return empty, diag.FromErr(fmt.Errorf("unable to marshal config versions json: %v", err))
		}

		fcv := make(map[string]interface{})
		fcv["config_json"] = string(cfgJson)
		fcv["current_version"] = cv.CurrentVersion
		fcv["version"] = cv.Version
		flat[i] = fcv
	}
	return flat, nil
}
