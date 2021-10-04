package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mariadb-corporation/skysql-api-go"
)

func dataSourceDatabase() *schema.Resource {
	s := make(map[string]*schema.Schema)
	for _, field := range databaseFields() {
		name := jsonFieldName(field)
		if name == "" {
			continue
		}
		s[reservedNamesAtoT(name)] = &schema.Schema{
			Type:     schema.TypeString,
			Computed: name != "id",
			Required: name == "id",
		}
	}
	return &schema.Resource{
		Description: "MariaDB database service deployed by SkySQL",
		ReadContext: dataSourceDatabaseRead,
		Schema:      s,
	}
}

func dataSourceDatabaseRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// use the meta value to retrieve your client from the provider configure method
	client := meta.(*skysql.Client)
	var diags diag.Diagnostics

	databaseID := d.Get("id").(string)

	res, err := client.ReadDatabase(ctx, databaseID)
	if err != nil {
		return diag.FromErr(err)
	}
	if res.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return diag.FromErr(fmt.Errorf("unable to retrieve database from SkySQL: Status: %v, Err: %v", res.StatusCode, err))
		}
		return diag.FromErr(fmt.Errorf("unable to retrieve database from SkySQL: Status: %v, Body: %v", res.StatusCode, string(body)))
	}

	defer res.Body.Close()

	var database map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&database); err != nil {
		return diag.FromErr(err)
	}

	id := database["id"].(string)
	if id == "" {
		return diag.FromErr(fmt.Errorf("unable to decode database response from SkySQL: %v", database))
	}

	for _, field := range databaseFields() {
		fieldName := jsonFieldName(field)
		if fieldName == "" {
			continue
		}
		d.Set(reservedNamesAtoT(fieldName), database[fieldName])
	}

	d.SetId(id)

	return diags
}

func databaseFields() []reflect.StructField {
	return reflect.VisibleFields(reflect.TypeOf(skysql.Database{}))
}

func jsonFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" || tag == "-" {
		return ""
	}
	parts := strings.Split(tag, ",")
	return parts[0]
}

// reservedNamesAtoT avoids name collisions with reserved words in terraform by
// swapping out the api name (A) with a placeholder used by the terraform client (T)
func reservedNamesAtoT(name string) string {
	if name == "provider" {
		return "cloud_provider"
	}
	return name
}
