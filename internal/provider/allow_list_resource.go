package provider

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	sdkresource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/skysqlinc/terraform-provider-skysql/internal/skysql"
	"github.com/skysqlinc/terraform-provider-skysql/internal/skysql/provisioning"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &ServiceAllowListResource{}
var _ resource.ResourceWithImportState = &ServiceAllowListResource{}
var _ resource.ResourceWithConfigure = &ServiceAllowListResource{}

func NewServiceAllowListResource() resource.Resource {
	return &ServiceAllowListResource{}
}

// ServiceAllowListResource defines the resource implementation.
type ServiceAllowListResource struct {
	client *skysql.Client
}

// ServiceAllowListResourceModel describes the data source data model.
type ServiceAllowListResourceModel struct {
	ID              types.String     `tfsdk:"service_id"`
	AllowList       []AllowListModel `tfsdk:"allow_list"`
	WaitForCreation types.Bool       `tfsdk:"wait_for_creation"`
	Timeouts        timeouts.Value   `tfsdk:"timeouts"`
}

type AllowListModel struct {
	IPAddress types.String `tfsdk:"ip"`
	Comment   types.String `tfsdk:"comment"`
}

func (r *ServiceAllowListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_allow_list"
}

func (r *ServiceAllowListResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the allow list for a service",
		Attributes: map[string]schema.Attribute{
			"service_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the service to manage the allow list for",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				}},
			"allow_list": schema.ListNestedAttribute{
				Required:    true,
				Description: "The list of IP addresses with comments to allow access to the service",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ip": schema.StringAttribute{
							Required:    true,
							Description: "The IP address to allow access to the service. The IP must be in a valid CIDR format",
							Validators: []validator.String{
								allowListIPValidator{},
							},
						},
						"comment": schema.StringAttribute{
							Optional:    true,
							Description: "A comment to describe the IP address",
						},
					},
				},
			},
			"wait_for_creation": schema.BoolAttribute{
				Optional:    true,
				Description: "If true, the provider will wait for the service to be updated before returning. ",
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
			}),
		},
	}
}

func (r *ServiceAllowListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*skysql.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *ServiceAllowListResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *ServiceAllowListResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	allowListUpdateRequest := make([]provisioning.AllowListItem, len(data.AllowList))
	for i := range data.AllowList {
		allowListUpdateRequest[i].IPAddress = data.AllowList[i].IPAddress.ValueString()
		allowListUpdateRequest[i].Comment = data.AllowList[i].Comment.ValueString()
	}

	allowListResp, err := r.client.UpdateServiceAllowListByID(ctx, data.ID.ValueString(), allowListUpdateRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error updating service allow list", err.Error())
		return
	}

	data.AllowList = make([]AllowListModel, len(allowListResp))
	for i := range allowListResp {
		data.AllowList[i].IPAddress = types.StringValue(allowListResp[i].IPAddress)
		data.AllowList[i].Comment = types.StringValue(allowListResp[i].Comment)
	}

	// save into the Terraform state.
	tflog.Trace(ctx, "service allow list updated")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.WaitForCreation.ValueBool() {

		createTimeout, diagsErr := data.Timeouts.Create(ctx, defaultCreateTimeout)
		if diagsErr != nil {
			diagsErr.AddError("Error creating service", fmt.Sprintf("Unable to create service, got error: %s", err))
			resp.Diagnostics.Append(diagsErr...)
		}

		err = sdkresource.RetryContext(ctx, createTimeout, func() *sdkresource.RetryError {

			service, err := r.client.GetServiceByID(ctx, data.ID.ValueString())
			if err != nil {
				return sdkresource.NonRetryableError(fmt.Errorf("error retrieving service details: %v", err))
			}

			if service.Status != "ready" && service.Status != "failed" {
				return sdkresource.RetryableError(fmt.Errorf("expected instance to be ready or failed state but was in state %s", service.Status))
			}

			return nil
		})

		if err != nil {
			resp.Diagnostics.AddError("Error updating service", fmt.Sprintf("Unable to update service, got error: %s", err))
		}
	}
}

func (r *ServiceAllowListResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *ServiceAllowListResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	allowListResp, err := r.client.ReadServiceAllowListByID(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Can not find service", err.Error())
		return
	}

	if len(allowListResp) == 0 {
		data.AllowList = make([]AllowListModel, 0)
	} else {
		data.AllowList = make([]AllowListModel, len(allowListResp[0].AllowList))
		if len(allowListResp[0].AllowList) > 0 {
			for i := range allowListResp[0].AllowList {
				data.AllowList[i].IPAddress = types.StringValue(allowListResp[0].AllowList[i].IPAddress)
				data.AllowList[i].Comment = types.StringValue(allowListResp[0].AllowList[i].Comment)
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *ServiceAllowListResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan *ServiceAllowListResourceModel
	var state *ServiceAllowListResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Prevent panic if the provider has not been configured.
	if plan == nil {
		return
	}

	allowListUpdateRequest := make([]provisioning.AllowListItem, len(plan.AllowList))
	for i := range plan.AllowList {
		allowListUpdateRequest[i].IPAddress = plan.AllowList[i].IPAddress.ValueString()
		allowListUpdateRequest[i].Comment = plan.AllowList[i].Comment.ValueString()
	}

	allowListResp, err := r.client.UpdateServiceAllowListByID(ctx, plan.ID.ValueString(), allowListUpdateRequest)
	if err != nil {
		if errors.Is(err, skysql.ErrorServiceNotFound) {
			tflog.Warn(ctx, "SkySQL service not found, removing from state", map[string]interface{}{
				"id": state.ID.ValueString(),
			})
			resp.State.RemoveResource(ctx)

			return
		}
		resp.Diagnostics.AddError("Error updating service allow list", err.Error())
		return
	}

	state.AllowList = make([]AllowListModel, len(allowListResp))
	for i := range allowListResp {
		state.AllowList[i].IPAddress = types.StringValue(allowListResp[i].IPAddress)
		state.AllowList[i].Comment = types.StringValue(allowListResp[i].Comment)
	}

	// save into the Terraform state.
	tflog.Trace(ctx, "service allow list updated")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.WaitForCreation.ValueBool() {

		createTimeout, diagsErr := state.Timeouts.Create(ctx, defaultCreateTimeout)
		if diagsErr != nil {
			diagsErr.AddError("Error creating service", fmt.Sprintf("Unable to create service, got error: %s", err))
			resp.Diagnostics.Append(diagsErr...)
		}

		err = sdkresource.RetryContext(ctx, createTimeout, func() *sdkresource.RetryError {

			service, err := r.client.GetServiceByID(ctx, state.ID.ValueString())
			if err != nil {
				return sdkresource.NonRetryableError(fmt.Errorf("error retrieving service details: %v", err))
			}

			if service.Status != "ready" && service.Status != "failed" {
				return sdkresource.RetryableError(fmt.Errorf("expected instance to be ready or failed state but was in state %s", service.Status))
			}

			return nil
		})

		if err != nil {
			resp.Diagnostics.AddError("Error updating service", fmt.Sprintf("Unable to update service, got error: %s", err))
		}
	}
}

func (r *ServiceAllowListResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *ServiceAllowListResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	allowListUpdateRequest := make([]provisioning.AllowListItem, 0)

	_, err := r.client.UpdateServiceAllowListByID(ctx, data.ID.ValueString(), allowListUpdateRequest)
	if err != nil {
		if errors.Is(err, skysql.ErrorServiceNotFound) {
			tflog.Warn(ctx, "SkySQL service not found, removing from state", map[string]interface{}{
				"id": data.ID.ValueString(),
			})
			resp.State.RemoveResource(ctx)

			return
		}
		resp.Diagnostics.AddError("Error updating service allow list", err.Error())
		return
	}

	data.AllowList = make([]AllowListModel, 0)

	// save into the Terraform state.
	tflog.Trace(ctx, "service allow list updated")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.WaitForCreation.ValueBool() {

		createTimeout, diagsErr := data.Timeouts.Create(ctx, defaultCreateTimeout)
		if diagsErr != nil {
			diagsErr.AddError("Error deleting allowlist", fmt.Sprintf("Unable to delete service allowlist, got error: %s", err))
			resp.Diagnostics.Append(diagsErr...)
		}

		err = sdkresource.RetryContext(ctx, createTimeout, func() *sdkresource.RetryError {

			service, err := r.client.GetServiceByID(ctx, data.ID.ValueString())
			if err != nil {
				return sdkresource.NonRetryableError(fmt.Errorf("error retrieving service details: %v", err))
			}

			if service.Status != "ready" && service.Status != "failed" {
				return sdkresource.RetryableError(fmt.Errorf("expected instance to be ready or failed state but was in state %s", service.Status))
			}

			return nil
		})

		if err != nil {
			resp.Diagnostics.AddError("Error deleting allowlist", fmt.Sprintf("Unable to update allowlist, got error: %s", err))
		}
	}
}

func (r *ServiceAllowListResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
