package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/mariadb-corporation/terraform-provider-skysql/internal/skysql"
	"github.com/mariadb-corporation/terraform-provider-skysql/internal/skysql/autonomous"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &AutonomousResource{}
var _ resource.ResourceWithImportState = &AutonomousResource{}
var _ resource.ResourceWithConfigure = &AutonomousResource{}
var _ resource.ResourceWithModifyPlan = &AutonomousResource{}

func NewAutonomousResource() resource.Resource {
	return &AutonomousResource{}
}

// AutonomousResource defines the resource implementation.
type AutonomousResource struct {
	client *skysql.Client
}

// AutonomousResourceModel describes the data source data model.
type AutonomousResourceModel struct {
	ID                             types.String `tfsdk:"id"`
	ServiceID                      types.String `tfsdk:"service_id"`
	ServiceName                    types.String `tfsdk:"service_name"`
	AutoScaleDiskAction            types.Object `tfsdk:"auto_scale_disk"`
	AutoScaleNodesHorizontalAction types.Object `tfsdk:"auto_scale_nodes_horizontal"`
	AutoScaleNodesVerticalAction   types.Object `tfsdk:"auto_scale_nodes_vertical"`
}

func (r *AutonomousResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_autonomous"
}

func (r *AutonomousResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Autonomous features enable automatic scaling in response to changes in workload.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required: false,
				Optional: false,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"service_id": schema.StringAttribute{
				Required:    true,
				Optional:    false,
				Computed:    false,
				Description: "Service ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"service_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the service to manage the autonomous features for.",
			},
			"auto_scale_disk": schema.SingleNestedAttribute{
				Required: false,
				Optional: true,
				Computed: false,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Required: false,
						Optional: false,
						Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
							stringplanmodifier.RequiresReplace(),
						},
					},
					"max_storage_size_gbs": schema.Int64Attribute{
						Required: false,
						Optional: true,
						Computed: true,
					},
				},
			},
			"auto_scale_nodes_vertical": schema.SingleNestedAttribute{
				Required: false,
				Optional: true,
				Computed: false,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Required: false,
						Optional: false,
						Computed: true,
					},
					"max_node_size": schema.StringAttribute{
						Required: false,
						Optional: true,
						Computed: true,
					},
					"min_node_size": schema.StringAttribute{
						Required: false,
						Optional: true,
						Computed: true,
					},
				},
			},
			"auto_scale_nodes_horizontal": schema.SingleNestedAttribute{
				Required: false,
				Optional: true,
				Computed: false,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Required: false,
						Optional: false,
						Computed: true,
					},
					"min_nodes": schema.Int64Attribute{
						Required: false,
						Optional: true,
						Computed: true,
					},
					"max_nodes": schema.Int64Attribute{
						Required: false,
						Optional: true,
						Computed: true,
					},
				},
			},
		},
	}
}

var autoScaleDiskAttrTypes = map[string]attr.Type{
	"id":                   types.StringType,
	"max_storage_size_gbs": types.Int64Type,
}

type AutoScaleDiskAction struct {
	ID                types.String `tfsdk:"id"`
	MaxStorageSizeGBs types.Int64  `tfsdk:"max_storage_size_gbs"`
}

var autoScaleNodesVerticalAttrTypes = map[string]attr.Type{
	"id":            types.StringType,
	"max_node_size": types.StringType,
	"min_node_size": types.StringType,
}

type AutoScaleNodesVerticalAction struct {
	ID          types.String `tfsdk:"id"`
	MaxNodeSize types.String `tfsdk:"max_node_size"`
	MinNodeSize types.String `tfsdk:"min_node_size"`
}

var autoScaleNodesHorizontalAttrTypes = map[string]attr.Type{
	"id":        types.StringType,
	"min_nodes": types.Int64Type,
	"max_nodes": types.Int64Type,
}

type AutoScaleNodesHorizontalAction struct {
	ID       types.String `tfsdk:"id"`
	MinNodes types.Int64  `tfsdk:"min_nodes"`
	MaxNodes types.Int64  `tfsdk:"max_nodes"`
}

func (r *AutonomousResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AutonomousResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *AutonomousResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	data.ID = types.StringValue(data.ServiceID.ValueString())

	if resp.Diagnostics.HasError() {
		return
	}

	service, err := r.client.GetServiceByID(ctx, data.ServiceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Can not read service", err.Error())
		return
	}

	request := autonomous.SetAutonomousActionsRequest{
		ServiceID:   data.ID.ValueString(),
		ServiceName: data.ServiceName.ValueString(),
	}

	request.Actions = make([]autonomous.AutoScaleAction, 0, 3)

	if !data.AutoScaleDiskAction.IsUnknown() && !data.AutoScaleDiskAction.IsNull() {
		var action AutoScaleDiskAction
		resp.Diagnostics.Append(data.AutoScaleDiskAction.As(ctx, &action, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
		request.Actions = append(
			request.Actions,
			autonomous.NewAutoScaleDiskAction(
				action.MaxStorageSizeGBs.ValueInt64(),
			),
		)
	} else {
		data.AutoScaleDiskAction = types.ObjectNull(autoScaleDiskAttrTypes)
	}

	if !data.AutoScaleNodesVerticalAction.IsUnknown() && !data.AutoScaleNodesVerticalAction.IsNull() {
		var action *AutoScaleNodesVerticalAction
		resp.Diagnostics.Append(data.AutoScaleNodesVerticalAction.As(ctx, &action, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
		request.Actions = append(
			request.Actions,
			autonomous.NewAutoScaleNodesVerticalAction(
				action.MaxNodeSize.ValueString(),
				action.MinNodeSize.ValueString(),
			),
		)
	} else {
		data.AutoScaleNodesVerticalAction = types.ObjectNull(autoScaleNodesVerticalAttrTypes)
	}

	if !data.AutoScaleNodesHorizontalAction.IsUnknown() && !data.AutoScaleNodesHorizontalAction.IsNull() {

		if Contains([]string{"es-single", "standalone"}, service.Topology) {
			resp.Diagnostics.AddError("Can not set horizontal scaling for service with topology", service.Topology)
			return
		}

		var action *AutoScaleNodesHorizontalAction
		resp.Diagnostics.Append(data.AutoScaleNodesHorizontalAction.As(ctx, &action, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
		request.Actions = append(
			request.Actions,
			autonomous.NewAutoScaleNodesHorizontalAction(
				action.MinNodes.ValueInt64(),
				action.MaxNodes.ValueInt64(),
			),
		)
	} else {
		data.AutoScaleNodesHorizontalAction = types.ObjectNull(autoScaleNodesHorizontalAttrTypes)
	}

	if len(request.Actions) > 0 {
		actions, err := r.client.SetAutonomousActions(ctx, request)
		if err != nil {
			resp.Diagnostics.AddError("error creating skysql_autonomous resource", err.Error())
			return
		}
		resp.Diagnostics.Append(r.actionsResponseToData(actions, data)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *AutonomousResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *AutonomousResourceModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.GetServiceByID(ctx, data.ServiceID.ValueString())
	if err != nil {
		if errors.Is(err, skysql.ErrorServiceNotFound) {
			tflog.Warn(ctx, "SkySQL service not found, removing from state", map[string]interface{}{
				"id": data.ID.ValueString(),
			})
			resp.State.RemoveResource(ctx)

			return
		}
		resp.Diagnostics.AddError("Can not read service", err.Error())
		return
	}

	actions, err := r.client.GetAutonomousActions(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("can not read skysql_autonomous resource", err.Error())
		return
	}

	if len(actions) == 0 {
		data.AutoScaleDiskAction = types.ObjectNull(autoScaleDiskAttrTypes)
		data.AutoScaleNodesVerticalAction = types.ObjectNull(autoScaleNodesVerticalAttrTypes)
		data.AutoScaleNodesHorizontalAction = types.ObjectNull(autoScaleNodesHorizontalAttrTypes)
		return
	}

	resp.Diagnostics.Append(r.actionsResponseToData(actions, data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *AutonomousResource) actionsResponseToData(actions []autonomous.ActionResponse, data *AutonomousResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	for _, action := range actions {
		switch action.Group {
		case autonomous.AutoScaleDiskActionGroup:
			params := autonomous.AutoScaleDiskActionParams{}
			err := json.Unmarshal(action.Params, &params)
			if err != nil {
				diags.AddError("can not read skysql_autonomous resource", err.Error())
				return diags

			}
			data.AutoScaleDiskAction = types.ObjectValueMust(autoScaleDiskAttrTypes, map[string]attr.Value{
				"id":                   types.StringValue(action.ID),
				"max_storage_size_gbs": types.Int64Value(params.MaxStorageSizeGBs),
			})
		case autonomous.AutoScaleNodesVerticalActionGroup:
			params := autonomous.AutoScaleNodesVerticalActionParams{}
			err := json.Unmarshal(action.Params, &params)
			if err != nil {
				diags.AddError("can not read skysql_autonomous resource", err.Error())
				return diags
			}
			data.AutoScaleNodesVerticalAction = types.ObjectValueMust(autoScaleNodesVerticalAttrTypes, map[string]attr.Value{
				"id":            types.StringValue(action.ID),
				"max_node_size": types.StringValue(params.MaxNodeSize),
				"min_node_size": types.StringValue(params.MinNodeSize),
			})
		case autonomous.AutoScaleNodesHorizontalActionGroup:
			params := autonomous.AutoScaleNodesHorizontalActionParams{}
			err := json.Unmarshal(action.Params, &params)
			if err != nil {
				diags.AddError("can not read skysql_autonomous resource", err.Error())
				return diags
			}
			data.AutoScaleNodesHorizontalAction = types.ObjectValueMust(autoScaleNodesHorizontalAttrTypes, map[string]attr.Value{
				"id":        types.StringValue(action.ID),
				"min_nodes": types.Int64Value(params.MinNodes),
				"max_nodes": types.Int64Value(params.MaxNodes),
			})
		}
	}
	return diags
}

func (r *AutonomousResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan *AutonomousResourceModel
	var state *AutonomousResourceModel

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

	service, err := r.client.GetServiceByID(ctx, state.ServiceID.ValueString())
	if err != nil {
		if errors.Is(err, skysql.ErrorServiceNotFound) {
			tflog.Warn(ctx, "SkySQL service not found, removing from state", map[string]interface{}{
				"id": state.ID.ValueString(),
			})
			resp.State.RemoveResource(ctx)

			return
		}
		resp.Diagnostics.AddError("Can not read service", err.Error())
		return
	}

	request := autonomous.SetAutonomousActionsRequest{
		ServiceID:   plan.ID.ValueString(),
		ServiceName: plan.ServiceName.ValueString(),
		Actions:     make([]autonomous.AutoScaleAction, 0, 3),
	}

	if !state.AutoScaleNodesHorizontalAction.IsNull() && plan.AutoScaleNodesHorizontalAction.IsNull() {
		r.deleteAutoScaleDiskAction(ctx, state)
	}

	if !state.AutoScaleNodesVerticalAction.IsNull() && plan.AutoScaleNodesVerticalAction.IsNull() {
		r.deleteAutoScaleNodesVerticalAction(ctx, state)
	}

	if !state.AutoScaleDiskAction.IsNull() && plan.AutoScaleDiskAction.IsNull() {
		r.deleteAutoScaleDiskAction(ctx, state)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.AutoScaleDiskAction.IsNull() {
		var action AutoScaleDiskAction
		resp.Diagnostics.Append(plan.AutoScaleDiskAction.As(ctx, &action, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
		request.Actions = append(
			request.Actions,
			autonomous.NewAutoScaleDiskAction(
				action.MaxStorageSizeGBs.ValueInt64(),
			),
		)
	} else {
		state.AutoScaleDiskAction = types.ObjectNull(autoScaleDiskAttrTypes)
	}

	if !plan.AutoScaleNodesVerticalAction.IsNull() {
		var action *AutoScaleNodesVerticalAction
		resp.Diagnostics.Append(plan.AutoScaleNodesVerticalAction.As(ctx, &action, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
		request.Actions = append(
			request.Actions,
			autonomous.NewAutoScaleNodesVerticalAction(
				action.MaxNodeSize.ValueString(),
				action.MinNodeSize.ValueString(),
			),
		)
		var diags diag.Diagnostics
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		state.AutoScaleNodesVerticalAction = types.ObjectNull(autoScaleNodesVerticalAttrTypes)
	}

	if !plan.AutoScaleNodesHorizontalAction.IsNull() {
		if Contains([]string{"es-single", "standalone"}, service.Topology) {
			resp.Diagnostics.AddError("Can not set horizontal scaling for service with topology", service.Topology)
			return
		}
		var action *AutoScaleNodesHorizontalAction
		resp.Diagnostics.Append(plan.AutoScaleNodesHorizontalAction.As(ctx, &action, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
		request.Actions = append(
			request.Actions,
			autonomous.NewAutoScaleNodesHorizontalAction(
				action.MinNodes.ValueInt64(),
				action.MaxNodes.ValueInt64(),
			),
		)
	} else {
		state.AutoScaleNodesHorizontalAction = types.ObjectNull(autoScaleNodesHorizontalAttrTypes)
	}

	if len(request.Actions) > 0 {
		actions, err := r.client.SetAutonomousActions(ctx, request)
		if err != nil {
			resp.Diagnostics.AddError("error creating skysql_autonomous resource", err.Error())
			return
		}
		resp.Diagnostics.Append(r.actionsResponseToData(actions, state)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *AutonomousResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *AutonomousResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.AutoScaleDiskAction.IsUnknown() && !data.AutoScaleDiskAction.IsNull() {
		resp.Diagnostics.Append(r.deleteAutoScaleDiskAction(ctx, data)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if !data.AutoScaleNodesVerticalAction.IsUnknown() && !data.AutoScaleNodesVerticalAction.IsNull() {
		resp.Diagnostics.Append(r.deleteAutoScaleNodesVerticalAction(ctx, data)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if !data.AutoScaleNodesHorizontalAction.IsUnknown() && !data.AutoScaleNodesHorizontalAction.IsNull() {
		resp.Diagnostics.Append(r.deleteAutoScaleNodesHorizontalAction(ctx, data)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
}

func (r *AutonomousResource) deleteAutoScaleDiskAction(ctx context.Context, data *AutonomousResourceModel) diag.Diagnostics {
	var action AutoScaleDiskAction
	var diags diag.Diagnostics
	diags.Append(data.AutoScaleDiskAction.As(ctx, &action, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return diags
	}
	if !action.ID.IsNull() {
		err := r.client.DeleteAutonomousAction(ctx, action.ID.ValueString())
		if err != nil {
			diags.AddError("error deleting skysql_autonomous resource", err.Error())
		}
	}
	return diags
}

func (r *AutonomousResource) deleteAutoScaleNodesVerticalAction(ctx context.Context, data *AutonomousResourceModel) diag.Diagnostics {
	var action AutoScaleNodesVerticalAction
	var diags diag.Diagnostics
	diags.Append(data.AutoScaleNodesVerticalAction.As(ctx, &action, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return diags
	}
	if !action.ID.IsNull() {
		err := r.client.DeleteAutonomousAction(ctx, action.ID.ValueString())
		if err != nil {
			diags.AddError("error deleting skysql_autonomous resource", err.Error())
		}
	}
	return diags
}

func (r *AutonomousResource) deleteAutoScaleNodesHorizontalAction(ctx context.Context, data *AutonomousResourceModel) diag.Diagnostics {
	var action AutoScaleNodesHorizontalAction
	var diags diag.Diagnostics
	diags.Append(data.AutoScaleNodesHorizontalAction.As(ctx, &action, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return diags
	}
	if !action.ID.IsNull() {
		err := r.client.DeleteAutonomousAction(ctx, action.ID.ValueString())
		if err != nil {
			diags.AddError("error deleting skysql_autonomous resource", err.Error())
		}
	}
	return diags
}

func (r *AutonomousResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *AutonomousResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Plan does not need to be modified when the resource is being destroyed.
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan AutonomousResourceModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.AutoScaleNodesHorizontalAction.IsNull() && !plan.AutoScaleNodesVerticalAction.IsNull() {
		resp.Diagnostics.AddError("Can not set both horizontal and vertical scaling for service",
			"Cant set both horizontal and vertical scaling for service")
		return
	}
}
