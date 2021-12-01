package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mariadb-corporation/skysql-api-go"
)

func resourceAllowlistEntry() *schema.Resource {
	s := make(map[string]*schema.Schema)

	// defaults for all fields
	for _, field := range allowlistEntryFields() {
		s[reservedNamesAtoT(field.Name)] = &schema.Schema{
			Type:     schema.TypeString,
			Computed: false,
			Required: !field.Optional,
			Optional: field.Optional,
			ForceNew: true,
		}
	}

	s["wait_for_install"] = &schema.Schema{
		Type:        schema.TypeString,
		Computed:    false,
		Optional:    true,
		Default:     "true",
		Description: "Set to false to skip waiting for the service to be deployed",
	}

	s["status"] = &schema.Schema{
		Type:        schema.TypeString,
		Computed:    true,
		Optional:    true,
		Default:     skysql.AllowlistStatusesEnforcing,
		Description: "Status indicating if the allowlist is still provisioning",
	}

	return &schema.Resource{
		Description: "Allowlist entry for a MariaDB service deployed by SkySQL. Please include the subnet mask (e.g. /32) on each entry.",

		CreateContext: resourceAllowlistEntryCreate,
		ReadContext:   resourceAllowlistEntryRead,
		DeleteContext: resourceAllowlistEntryDelete,

		Schema: s,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(60 * time.Minute),
		},
	}
}

func resourceAllowlistEntryCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*skysql.Client)

	// collect the attributes specified by the user
	attrs := make(map[string]interface{})
	for _, field := range allowlistEntryFields() {
		attrs[field.Name] = d.Get(reservedNamesAtoT(field.Name))
	}

	// marshal them into a json byte string to take advantage
	// of the json tags on the generated type in the skysql SDK
	attrsJson, err := json.Marshal(attrs)
	if err != nil {
		diag.FromErr(err)
	}

	// unmarshal the attrs into a valid request body type
	var body skysql.AddAllowedAddressJSONRequestBody
	if err = json.Unmarshal(attrsJson, &body); err != nil {
		diag.FromErr(err)
	}

	svc := attrs["service_id"].(string)
	res, err := client.AddAllowedAddress(ctx, svc, body)
	if err != nil {
		return diag.FromErr(err)
	}

	resBody, errDiag := decodeAPIResponseBody(res)
	if errDiag != nil {
		return errDiag
	}

	id := allowlistCompositeId(svc, resBody["ip_address"].(string))
	if id == "" {
		diag.FromErr(fmt.Errorf("bad response from SkySQL: %v", resBody))
	}
	d.SetId(id)

	err = resource.RetryContext(ctx, d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		err := resourceAllowlistEntryRead(ctx, d, meta)
		if err != nil {
			return resource.NonRetryableError(fmt.Errorf("error retrieving service details: %v", err))
		}

		// block until install is complete
		if d.Get("wait_for_install") != "true" {
			return nil
		}

		if d.Get("status") != skysql.AllowlistStatusesEnforcing {
			return resource.RetryableError(fmt.Errorf("expected allowlist to be Enforcing but was in state %s", d.Get("status")))
		}

		return nil
	})
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceAllowlistEntryRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*skysql.Client)
	var diags diag.Diagnostics

	addr := d.Get("ip_address").(string)
	svc := d.Get("service_id").(string)
	id := allowlistCompositeId(svc, addr)

	body := skysql.ListAllowedAddressesParams{}
	res, err := client.ListAllowedAddresses(ctx, svc, &body)
	if err != nil {
		return diag.FromErr(err)
	}

	var allowlist []skysql.AllowlistIPAddress
	allowlist, errDiag = decodeAPIResponseBody(res)
	if errDiag != nil {
		return errDiag
	}

	for _, entry := range allowlist {
		if entry["ip_address"] != addr {
			continue
		}
		for _, field := range allowlistEntryFields() {
			d.Set(reservedNamesAtoT(field.Name), entry[field.Name])
		}
	}

	return diags
}

func resourceAllowlistEntryDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

func allowlistEntryFields() []fieldInfo {
	return fields(skysql.AllowlistIPAddress{})
}

func allowlistCompositeId(svc, addr string) string {
	return fmt.Sprintf("%s-%s", svc, addr)
}
