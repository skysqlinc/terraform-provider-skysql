package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
		s[reservedNamesAtoT(field)] = &schema.Schema{
			Type:     schema.TypeString,
			Computed: field != "id",
			Required: field == "id",
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

	id := d.Get("id").(string)

	database, err := readDatabase(ctx, client, id)
	if err != nil {
		return err
	}

	for _, field := range databaseFields() {
		d.Set(reservedNamesAtoT(field), database[field])
	}

	d.SetId(id)

	return diags
}

func databaseFields() []string {
	return fieldNames(skysql.Database{})
}

func databaseCreateFields() []string {
	return fieldNames(skysql.NewDatabase{})
}

func databaseUpdateFields() []string {
	return fieldNames(skysql.DatabaseUpdate{})
}

func fieldNames(val interface{}) []string {
	var names []string
	for _, field := range reflect.VisibleFields(reflect.TypeOf(val)) {
		fieldName := jsonFieldName(field)
		if fieldName == "" {
			continue
		}
		names = append(names, fieldName)
	}
	return names
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

func readDatabase(ctx context.Context, client *skysql.Client, id string) (map[string]interface{}, diag.Diagnostics) {
	res, err := client.ReadDatabase(ctx, id)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	database, errDiag := decodeAPIResponseBody(res)
	if errDiag != nil {
		return nil, errDiag
	}

	databaseID := database["id"].(string)
	if databaseID != id {
		return nil, diag.FromErr(fmt.Errorf("bad response from SkySQL: %v", database))
	}

	return database, nil
}

func decodeAPIResponseBody(res *http.Response) (map[string]interface{}, diag.Diagnostics) {
	defer res.Body.Close()

	err := checkAPIStatus(res.StatusCode, res.Body)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	var decodedBody map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&decodedBody); err != nil {
		return nil, diag.FromErr(err)
	}
	return decodedBody, nil
}

func checkAPIStatus(code int, body io.ReadCloser) error {
	if code != http.StatusOK {
		body, err := ioutil.ReadAll(body)
		if err != nil {
			return fmt.Errorf("bad response from SkySQL: Status: %v, Err: %v", code, err)
		}
		return fmt.Errorf("bad response from from SkySQL: Status: %v, Body: %v", code, string(body))
	}
	return nil
}
