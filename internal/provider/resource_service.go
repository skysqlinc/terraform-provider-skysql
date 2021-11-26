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

func resourceService() *schema.Resource {
	s := make(map[string]*schema.Schema)

	// defaults for all fields
	for _, field := range serviceFields() {
		s[reservedNamesAtoT(field.Name)] = &schema.Schema{
			Type:     schema.TypeString,
			Computed: true,
		}
	}

	// overrides for fields included in create requests
	for _, field := range serviceCreateFields() {
		s[reservedNamesAtoT(field.Name)].Computed = false
		s[reservedNamesAtoT(field.Name)].Required = !field.Optional
		s[reservedNamesAtoT(field.Name)].Optional = field.Optional
		s[reservedNamesAtoT(field.Name)].ForceNew = true
	}

	// overrides for fields that may be updated in place
	for _, field := range serviceUpdateFields() {
		s[reservedNamesAtoT(field.Name)].ForceNew = false
	}

	return &schema.Resource{
		Description: "MariaDB service deployed by SkySQL",

		CreateContext: resourceServiceCreate,
		ReadContext:   resourceServiceRead,
		UpdateContext: resourceServiceUpdate,
		DeleteContext: resourceServiceDelete,

		Schema: s,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceServiceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*skysql.Client)

	// collect the attributes specified by the user
	attrs := make(map[string]interface{})
	for _, field := range serviceCreateFields() {
		attrs[field.Name] = d.Get(reservedNamesAtoT(field.Name))
	}

	// marshal them into a json byte string to take advantage
	// of the json tags on the generated type in the skysql SDK
	attrsJson, err := json.Marshal(attrs)
	if err != nil {
		diag.FromErr(err)
	}

	// unmarshal the attrs into a valid request body type
	var body skysql.CreateServiceJSONRequestBody
	if err = json.Unmarshal(attrsJson, &body); err != nil {
		diag.FromErr(err)
	}

	res, err := client.CreateService(ctx, body)
	if err != nil {
		return diag.FromErr(err)
	}

	service, errDiag := decodeAPIResponseBody(res)
	if errDiag != nil {
		return errDiag
	}

	id := service["id"].(string)
	if id == "" {
		diag.FromErr(fmt.Errorf("bad response from SkySQL: %v", service))
	}
	d.SetId(id)

	return resourceServiceRead(ctx, d, meta)
}

func resourceServiceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*skysql.Client)
	var diags diag.Diagnostics

	id := d.Id()

	service, err := readService(ctx, client, id)
	if err != nil {
		return err
	}

	for _, field := range serviceFields() {
		d.Set(reservedNamesAtoT(field.Name), service[field.Name])
	}

	return diags
}

func resourceServiceUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*skysql.Client)
	id := d.Id()

	// collect the attributes specified by the user
	var updateNeeded bool
	attrs := make(map[string]interface{})
	for _, field := range serviceUpdateFields() {
		attrs[field.Name] = d.Get(reservedNamesAtoT(field.Name))
		updateNeeded = updateNeeded || d.HasChange(field.Name)
	}

	if updateNeeded {
		// marshal them into a json byte string to take advantage
		// of the json tags on the generated type in the skysql SDK
		attrsJson, err := json.Marshal(attrs)
		if err != nil {
			diag.FromErr(err)
		}

		// unmarshal the attrs into a valid request body type
		var body skysql.UpdateServiceJSONRequestBody
		if err = json.Unmarshal(attrsJson, &body); err != nil {
			diag.FromErr(err)
		}

		res, err := client.UpdateService(ctx, id, body)
		if err != nil {
			return diag.FromErr(err)
		}

		_, errDiag := decodeAPIResponseBody(res)
		if errDiag != nil {
			return errDiag
		}
	}

	return resourceServiceRead(ctx, d, meta)
}

func resourceServiceDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := meta.(*skysql.Client)
	id := d.Id()

	res, err := client.DeleteService(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}

	err = checkAPIStatus(res.StatusCode, res.Body, http.StatusNoContent)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return diags
}
