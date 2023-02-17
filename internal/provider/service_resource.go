package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	sdkresource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/mariadb-corporation/terraform-provider-skysql-beta/internal/skysql"
	"github.com/mariadb-corporation/terraform-provider-skysql-beta/internal/skysql/provisioning"
	"reflect"
	"regexp"
	"time"
)

const defaultCreateTimeout = 60 * time.Minute
const defaultDeleteTimeout = 60 * time.Minute
const defaultUpdateTimeout = 60 * time.Minute

var rxServiceName = regexp.MustCompile("(^[a-z][a-z0-9-]+$)")

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &ServiceResource{}
var _ resource.ResourceWithImportState = &ServiceResource{}
var _ resource.ResourceWithConfigure = &ServiceResource{}
var _ resource.ResourceWithModifyPlan = &ServiceResource{}
var _ resource.ResourceWithValidateConfig = &ServiceResource{}

var allowListElementType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"ip":      types.StringType,
		"comment": types.StringType,
	},
}

func NewServiceResource() resource.Resource {
	return &ServiceResource{}
}

// ServiceResource defines the resource implementation.
type ServiceResource struct {
	client *skysql.Client
}

// ServiceResourceModel describes the resource data model.
type ServiceResourceModel struct {
	ID                 types.String   `tfsdk:"id"`
	Name               types.String   `tfsdk:"name"`
	ProjectID          types.String   `tfsdk:"project_id"`
	ServiceType        types.String   `tfsdk:"service_type"`
	Provider           types.String   `tfsdk:"cloud_provider"`
	Region             types.String   `tfsdk:"region"`
	Version            types.String   `tfsdk:"version"`
	Nodes              types.Int64    `tfsdk:"nodes"`
	Architecture       types.String   `tfsdk:"architecture"`
	Size               types.String   `tfsdk:"size"`
	Topology           types.String   `tfsdk:"topology"`
	Storage            types.Int64    `tfsdk:"storage"`
	VolumeIOPS         types.Int64    `tfsdk:"volume_iops"`
	SSLEnabled         types.Bool     `tfsdk:"ssl_enabled"`
	NoSQLEnabled       types.Bool     `tfsdk:"nosql_enabled"`
	VolumeType         types.String   `tfsdk:"volume_type"`
	WaitForCreation    types.Bool     `tfsdk:"wait_for_creation"`
	Timeouts           timeouts.Value `tfsdk:"timeouts"`
	Mechanism          types.String   `tfsdk:"endpoint_mechanism"`
	AllowedAccounts    types.List     `tfsdk:"endpoint_allowed_accounts"`
	WaitForDeletion    types.Bool     `tfsdk:"wait_for_deletion"`
	ReplicationEnabled types.Bool     `tfsdk:"replication_enabled"`
	PrimaryHost        types.String   `tfsdk:"primary_host"`
	IsActive           types.Bool     `tfsdk:"is_active"`
	WaitForUpdate      types.Bool     `tfsdk:"wait_for_update"`
	DeletionProtection types.Bool     `tfsdk:"deletion_protection"`
	AllowList          types.List     `tfsdk:"allow_list"`
	MaxscaleNodes      types.Int64    `tfsdk:"maxscale_nodes"`
	MaxscaleSize       types.String   `tfsdk:"maxscale_size"`
}

// ServiceResourceNamedPortModel is an endpoint port
type ServiceResourceNamedPortModel struct {
	Name types.String `tfsdk:"name"`
	Port types.Int64  `tfsdk:"port"`
}

func (r *ServiceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (r *ServiceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages a service in SkySQL",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required: false,
				Optional: false,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "The ID of the service",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the service",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 24),
					stringvalidator.RegexMatches(
						rxServiceName,
						"must start from a lowercase letter and contain only lowercase letters, numbers and hyphens",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project_id": schema.StringAttribute{
				Required:    false,
				Optional:    true,
				Description: "The ID of the project to create the service in",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"service_type": schema.StringAttribute{
				Required:    true,
				Description: "The type of service to create. Valid values are: analytical or transactional",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cloud_provider": schema.StringAttribute{
				Required:    true,
				Description: "The cloud provider to create the service in. Valid values are: aws or gcp",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"region": schema.StringAttribute{
				Required:    true,
				Description: "The region to create the service in. Value should be valid for a specific cloud provider",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"version": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The software version",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"nodes": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "The number of nodes",
			},
			"architecture": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The architecture of the service. Valid values are: amd64 or arm64",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"size": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The size of the service. Valid values are: sky-2x4, sky-2x8 etc",
			},
			"topology": schema.StringAttribute{
				Required:    true,
				Description: "The topology of the service. Valid values are: es-single, es-replica, xpand, csdw and sa",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"storage": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "The storage size in GB. Valid values are: 100, 200, 300, 400, 500, 600, 700, 800, 900, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000",
			},
			"volume_iops": schema.Int64Attribute{
				Optional:    true,
				Description: "The volume IOPS. This is only applicable for AWS",
			},
			"ssl_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to enable SSL. Valid values are: true or false",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"nosql_enabled": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether to enable NoSQL. Valid values are: true or false",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"volume_type": schema.StringAttribute{
				Optional:    true,
				Description: "The volume type. Valid values are: gp2 and io1. This is only applicable for AWS",
			},
			"wait_for_creation": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether to wait for the service to be created. Valid values are: true or false",
			},
			"endpoint_mechanism": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The endpoint mechanism to use. Valid values are: privatelink or nlb",
			},
			"endpoint_allowed_accounts": schema.ListAttribute{
				Optional:    true,
				Description: "The list of cloud accounts (aws account ids or gcp projects) that are allowed to access the service",
				ElementType: types.StringType,
			},
			"wait_for_deletion": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether to wait for the service to be deleted. Valid values are: true or false",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"replication_enabled": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether to enable global replication. Valid values are: true or false. Works for xpand-direct topology only",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"primary_host": schema.StringAttribute{
				Optional:    true,
				Description: "The primary host of the service",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"is_active": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the service is active",
			},
			"wait_for_update": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether to wait for the service to be updated. Valid values are: true or false",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"deletion_protection": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to enable deletion protection. Valid values are: true or false. Default is true",
				PlanModifiers: []planmodifier.Bool{
					boolDefault(true),
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"allow_list": schema.ListNestedAttribute{
				Required:    false,
				Computed:    true,
				Optional:    true,
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
			"maxscale_nodes": schema.Int64Attribute{
				Optional:    true,
				Description: "The number of MaxScale nodes",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"maxscale_size": schema.StringAttribute{
				Optional:    true,
				Description: "The size of the MaxScale nodes. Valid values are: sky-2x4, sky-2x8 etc",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Delete: true,
				Update: true,
			}),
		},
	}
}

func (r *ServiceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *ServiceResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	createServiceRequest := &provisioning.CreateServiceRequest{
		Name:               data.Name.ValueString(),
		ProjectID:          data.ProjectID.ValueString(),
		ServiceType:        data.ServiceType.ValueString(),
		Provider:           data.Provider.ValueString(),
		Region:             data.Region.ValueString(),
		Version:            data.Version.ValueString(),
		Nodes:              uint(data.Nodes.ValueInt64()),
		Architecture:       data.Architecture.ValueString(),
		Size:               data.Size.ValueString(),
		Topology:           data.Topology.ValueString(),
		Storage:            uint(data.Storage.ValueInt64()),
		VolumeIOPS:         uint(data.VolumeIOPS.ValueInt64()),
		SSLEnabled:         data.SSLEnabled.ValueBool(),
		NoSQLEnabled:       data.NoSQLEnabled.ValueBool(),
		VolumeType:         data.VolumeType.ValueString(),
		Mechanism:          data.Mechanism.ValueString(),
		ReplicationEnabled: data.ReplicationEnabled.ValueBool(),
		PrimaryHost:        data.PrimaryHost.ValueString(),
		MaxscaleNodes:      uint(data.MaxscaleNodes.ValueInt64()),
	}

	if !data.MaxscaleSize.IsUnknown() && !data.MaxscaleSize.IsNull() && len(data.MaxscaleSize.ValueString()) > 0 {
		createServiceRequest.MaxscaleSize = toPtr[string](data.MaxscaleSize.ValueString())
	} else {
		createServiceRequest.MaxscaleSize = nil
	}

	b, err := json.Marshal(createServiceRequest)
	if err != nil {
		resp.Diagnostics.AddError("Failed to marshal create service request", err.Error())
		return
	}
	tflog.Debug(ctx, string(b))

	if !data.AllowList.IsUnknown() {
		var allowList []AllowListModel
		diags := data.AllowList.ElementsAs(ctx, &allowList, false)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		for _, allowListItem := range allowList {
			createServiceRequest.AllowList = append(createServiceRequest.AllowList, provisioning.AllowListItem{
				IPAddress: allowListItem.IPAddress.ValueString(),
				Comment:   allowListItem.Comment.ValueString(),
			})
		}
	}

	diags := data.AllowedAccounts.ElementsAs(ctx, &createServiceRequest.AllowedAccounts, false)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	service, err := r.client.CreateService(ctx, createServiceRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error creating service", err.Error())
		return
	}

	// save into the Terraform state.
	data.ID = types.StringValue(service.ID)
	data.Name = types.StringValue(service.Name)

	tflog.Trace(ctx, "created a resource")

	if service.StorageVolume.IOPS > 0 {
		data.VolumeIOPS = types.Int64Value(int64(service.StorageVolume.IOPS))
	} else {
		data.VolumeIOPS = types.Int64Null()
	}
	if service.StorageVolume.VolumeType != "" {
		data.VolumeType = types.StringValue(service.StorageVolume.VolumeType)
	} else {
		data.VolumeType = types.StringNull()
	}

	data.IsActive = types.BoolValue(service.IsActive)
	data.Architecture = types.StringValue(service.Architecture)
	data.Nodes = types.Int64Value(int64(service.Nodes))
	data.Size = types.StringValue(service.Size)
	data.Version = types.StringValue(service.Version)
	data.Storage = types.Int64Value(int64(service.StorageVolume.Size))
	data.SSLEnabled = types.BoolValue(service.SSLEnabled)
	if len(service.Endpoints) > 0 {
		data.Mechanism = types.StringValue(service.Endpoints[0].Mechanism)
		data.AllowedAccounts, _ = types.ListValueFrom(ctx, types.StringType, service.Endpoints[0].AllowedAccounts)
		data.AllowList, _ = r.allowListToListType(ctx, service.Endpoints[0].AllowList)
	}
	if !(data.MaxscaleSize.IsUnknown() || data.MaxscaleSize.IsNull()) && service.MaxscaleSize != nil {
		data.MaxscaleSize = types.StringValue(*service.MaxscaleSize)
	}
	if !(data.MaxscaleNodes.IsUnknown() || data.MaxscaleNodes.IsNull()) {
		data.MaxscaleNodes = types.Int64Value(int64(service.MaxscaleNodes))
	}
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

			service, err := r.client.GetServiceByID(ctx, service.ID)
			if err != nil {
				return sdkresource.NonRetryableError(fmt.Errorf("error retrieving service details: %v", err))
			}

			if service.Status != "ready" && service.Status != "failed" {
				return sdkresource.RetryableError(fmt.Errorf("expected instance to be ready or failed state but was in state %s", service.Status))
			}

			if service.Status == "failed" {
				return sdkresource.NonRetryableError(errors.New("service creation failed"))
			}

			return nil
		})

		if err != nil {
			resp.Diagnostics.AddError("Error creating service", fmt.Sprintf("Unable to create service, got error: %s", err))
		}
	}
}

func (r *ServiceResource) allowListToListType(ctx context.Context, allowList []provisioning.AllowListItem) (types.List, diag.Diagnostics) {

	allowListModels := make([]AllowListModel, 0, len(allowList))
	for _, allowListItem := range allowList {
		allowListModels = append(allowListModels, AllowListModel{
			IPAddress: types.StringValue(allowListItem.IPAddress),
			Comment:   types.StringValue(allowListItem.Comment),
		})
	}

	list, diags := types.ListValueFrom(ctx, allowListElementType, allowListModels)
	if diags.HasError() {
		return types.ListNull(allowListElementType), diags
	}
	return list, diags
}

func (r *ServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *ServiceResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.readServiceState(ctx, data)
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
	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *ServiceResource) readServiceState(ctx context.Context, data *ServiceResourceModel) error {
	service, err := r.client.GetServiceByID(ctx, data.ID.ValueString())
	if err != nil {
		return err
	}
	data.ID = types.StringValue(service.ID)
	data.Name = types.StringValue(service.Name)
	data.SSLEnabled = types.BoolValue(service.SSLEnabled)
	data.ServiceType = types.StringValue(service.ServiceType)
	data.Provider = types.StringValue(service.Provider)
	data.Region = types.StringValue(service.Region)
	data.Version = types.StringValue(service.Version)
	data.Nodes = types.Int64Value(int64(service.Nodes))
	data.Architecture = types.StringValue(service.Architecture)
	data.Size = types.StringValue(service.Size)
	data.Topology = types.StringValue(service.Topology)
	data.Storage = types.Int64Value(int64(service.StorageVolume.Size))
	if !data.VolumeIOPS.IsNull() && service.StorageVolume.IOPS > 0 {
		data.VolumeIOPS = types.Int64Value(int64(service.StorageVolume.IOPS))
	} else {
		data.VolumeIOPS = types.Int64Null()
	}
	if !data.VolumeType.IsNull() {
		data.VolumeType = types.StringValue(service.StorageVolume.VolumeType)
	}
	if !data.ReplicationEnabled.IsNull() {
		data.ReplicationEnabled = types.BoolValue(service.ReplicationEnabled)
	}
	if !data.PrimaryHost.IsNull() {
		data.PrimaryHost = types.StringValue(service.PrimaryHost)
	}
	data.IsActive = types.BoolValue(service.IsActive)
	data.SSLEnabled = types.BoolValue(service.SSLEnabled)
	if len(service.Endpoints) > 0 {
		data.Mechanism = types.StringValue(service.Endpoints[0].Mechanism)
		data.AllowedAccounts, _ = types.ListValueFrom(ctx, types.StringType, service.Endpoints[0].AllowedAccounts)
		data.AllowList, _ = r.allowListToListType(ctx, service.Endpoints[0].AllowList)
	}
	if !(data.MaxscaleSize.IsUnknown() || data.MaxscaleSize.IsNull()) && service.MaxscaleSize != nil {
		data.MaxscaleSize = types.StringValue(*service.MaxscaleSize)
	} else {
		data.MaxscaleSize = types.StringNull()
	}
	if !(data.MaxscaleNodes.IsUnknown() || data.MaxscaleSize.IsNull()) {
		data.MaxscaleNodes = types.Int64Value(int64(service.MaxscaleNodes))
	}
	return nil
}

func (r *ServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan *ServiceResourceModel
	var state *ServiceResourceModel

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

	err := r.readServiceState(ctx, state)
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

	state.WaitForUpdate = plan.WaitForUpdate
	state.WaitForCreation = plan.WaitForCreation
	state.WaitForDeletion = plan.WaitForDeletion
	state.Timeouts = plan.Timeouts
	state.DeletionProtection = plan.DeletionProtection
	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.updateServicePowerState(ctx, plan, state, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	r.updateServiceEndpoints(ctx, plan, state, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	r.updateServiceSize(ctx, plan, state, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	r.updateNumberOfNodeForService(ctx, plan, state, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	r.updateServiceStorage(ctx, plan, state, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	r.updateAllowList(ctx, plan, state, resp)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *ServiceResource) updateServiceStorage(ctx context.Context, plan *ServiceResourceModel, state *ServiceResourceModel, resp *resource.UpdateResponse) {
	if plan.Storage.ValueInt64() != state.Storage.ValueInt64() || plan.VolumeIOPS.ValueInt64() != state.VolumeIOPS.ValueInt64() {
		tflog.Info(ctx, "Updating storage size for the service", map[string]interface{}{
			"id":        state.ID.ValueString(),
			"from":      state.Storage.ValueInt64(),
			"to":        plan.Storage.ValueInt64(),
			"iops_from": state.VolumeIOPS.ValueInt64(),
			"iops_to":   plan.VolumeIOPS.ValueInt64(),
		})

		err := r.client.ModifyServiceStorage(ctx, state.ID.ValueString(), plan.Storage.ValueInt64(), plan.VolumeIOPS.ValueInt64())
		if err != nil {
			resp.Diagnostics.AddError("Error updating a storage for the service",
				fmt.Sprintf("Unable to update a storage size for the service, got error: %s", err))
			return
		}

		state.Storage = plan.Storage
		state.VolumeIOPS = plan.VolumeIOPS
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		r.waitForUpdate(ctx, state, err, resp)
	}
}

func (r *ServiceResource) updateNumberOfNodeForService(ctx context.Context, plan *ServiceResourceModel, state *ServiceResourceModel, resp *resource.UpdateResponse) {
	if plan.Nodes.ValueInt64() != state.Nodes.ValueInt64() {
		tflog.Info(ctx, "Updating number of nodes for the service", map[string]interface{}{
			"id":   state.ID.ValueString(),
			"from": state.Nodes.ValueInt64(),
			"to":   plan.Nodes.ValueInt64(),
		})

		err := r.client.ModifyServiceNodeNumber(ctx, state.ID.ValueString(), plan.Nodes.ValueInt64())
		if err != nil {
			resp.Diagnostics.AddError("Error updating a number of nodes for the service", fmt.Sprintf("Unable to update a nodes number for the service, got error: %s", err))
			return
		}

		state.Nodes = plan.Nodes
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		r.waitForUpdate(ctx, state, err, resp)
	}
}

func (r *ServiceResource) updateServiceSize(ctx context.Context, plan *ServiceResourceModel, state *ServiceResourceModel, resp *resource.UpdateResponse) {
	if plan.Size.ValueString() != state.Size.ValueString() {
		tflog.Info(ctx, "Updating service size", map[string]interface{}{
			"id":   state.ID.ValueString(),
			"from": state.Size.ValueString(),
			"to":   plan.Size.ValueString(),
		})

		err := r.client.ModifyServiceSize(ctx, state.ID.ValueString(), plan.Size.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error updating service size", fmt.Sprintf("Unable to update service size, got error: %s", err))
			return
		}

		state.Size = plan.Size
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		r.waitForUpdate(ctx, state, err, resp)
	}
}

func (r *ServiceResource) updateServiceEndpoints(ctx context.Context, plan *ServiceResourceModel, state *ServiceResourceModel, resp *resource.UpdateResponse) {
	var planAllowedAccounts []string
	diag := plan.AllowedAccounts.ElementsAs(ctx, &planAllowedAccounts, false)
	if diag.HasError() {
		return
	}

	var stateAllowedAccounts []string
	diag = state.AllowedAccounts.ElementsAs(ctx, &stateAllowedAccounts, false)
	if diag.HasError() {
		return
	}

	isMechanismChanged := plan.Mechanism.ValueString() != state.Mechanism.ValueString()

	isAllowedAccountsChanged := !reflect.DeepEqual(planAllowedAccounts, stateAllowedAccounts)

	if isMechanismChanged || isAllowedAccountsChanged {
		tflog.Info(ctx, "Updating service allowed accounts", map[string]interface{}{
			"id": state.ID.ValueString(),
		})

		visibility := "public"
		if plan.Mechanism.ValueString() == "privatelink" {
			visibility = "private"
		} else {
			planAllowedAccounts = []string{}
		}

		if planAllowedAccounts == nil {
			planAllowedAccounts = []string{}
		}

		endpoint, err := r.client.ModifyServiceEndpoints(ctx,
			state.ID.ValueString(),
			plan.Mechanism.ValueString(),
			planAllowedAccounts,
			visibility)
		if err != nil {
			resp.Diagnostics.AddError("Can not update service", err.Error())
			return
		}

		state.Mechanism = types.StringValue(endpoint.Mechanism)
		state.AllowedAccounts, _ = types.ListValueFrom(ctx, types.StringType, endpoint.AllowedAccounts)

		// Save updated data into Terraform state
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		r.waitForUpdate(ctx, state, err, resp)
		return
	}
}

func (r *ServiceResource) updateAllowList(ctx context.Context, plan *ServiceResourceModel, state *ServiceResourceModel, resp *resource.UpdateResponse) {
	if !plan.AllowList.IsUnknown() {
		var planAllowList []AllowListModel
		diags := plan.AllowList.ElementsAs(ctx, &planAllowList, false)
		if diags.HasError() {
			return
		}

		var stateAllowList []AllowListModel
		diags = state.AllowList.ElementsAs(ctx, &stateAllowList, false)
		if diags.HasError() {
			return
		}

		if !reflect.DeepEqual(planAllowList, stateAllowList) {
			tflog.Info(ctx, "Updating service allow list", map[string]interface{}{
				"id": state.ID.ValueString(),
			})

			allowListUpdateRequest := make([]provisioning.AllowListItem, 0)
			for i := range planAllowList {
				allowListUpdateRequest = append(allowListUpdateRequest, provisioning.AllowListItem{
					IPAddress: planAllowList[i].IPAddress.ValueString(),
					Comment:   planAllowList[i].Comment.ValueString(),
				})
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

			var cdiags diag.Diagnostics
			state.AllowList, cdiags = r.allowListToListType(ctx, allowListResp)
			if cdiags.HasError() {
				resp.Diagnostics.Append(cdiags...)
				return
			}

			state.AllowList = plan.AllowList

			// Save updated data into Terraform state
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			if resp.Diagnostics.HasError() {
				return
			}
			r.waitForUpdate(ctx, state, err, resp)
			return
		}
	}
}

func (r *ServiceResource) updateServicePowerState(ctx context.Context, plan *ServiceResourceModel, state *ServiceResourceModel, resp *resource.UpdateResponse) {
	if !plan.IsActive.IsUnknown() && plan.IsActive.ValueBool() != state.IsActive.ValueBool() {
		tflog.Info(ctx, "Updating service active state", map[string]interface{}{
			"id":        state.ID.ValueString(),
			"is_active": plan.IsActive.ValueBool(),
		})
		err := r.client.SetServicePowerState(ctx, state.ID.ValueString(), plan.IsActive.ValueBool())
		if err != nil {
			resp.Diagnostics.AddError("Can not update service", err.Error())
			return
		}
		state.IsActive = plan.IsActive
		// Save updated data into Terraform state
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		r.waitForUpdate(ctx, state, err, resp)
	}
	return
}

var serviceUpdateWaitStates = []string{"ready", "failed", "stopped"}

func (r *ServiceResource) waitForUpdate(ctx context.Context, state *ServiceResourceModel, err error, resp *resource.UpdateResponse) {
	if state.WaitForUpdate.ValueBool() {
		err = sdkresource.RetryContext(ctx, defaultUpdateTimeout, func() *sdkresource.RetryError {
			service, err := r.client.GetServiceByID(ctx, state.ID.ValueString())
			if err != nil {
				return sdkresource.NonRetryableError(fmt.Errorf("error retrieving service details: %v", err))
			}

			if Contains[string](serviceUpdateWaitStates, service.Status) {
				return nil
			}

			if service.Status == "failed" {
				return sdkresource.NonRetryableError(errors.New("service creation failed"))
			}

			return sdkresource.RetryableError(fmt.Errorf("expected instance to be ready or failed or stopped state but was in state %s", service.Status))
		})

		if err != nil {
			resp.Diagnostics.AddError("Error updating service", fmt.Sprintf("Unable to update service, got error: %s", err))
		}
	}
}

func (r *ServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state *ServiceResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if state.DeletionProtection.ValueBool() {
		resp.Diagnostics.AddError("Can not delete service", "Deletion protection is enabled")
		return
	}

	err := r.client.DeleteServiceByID(ctx, state.ID.ValueString())
	if err != nil {
		if errors.Is(err, skysql.ErrorServiceNotFound) {
			tflog.Warn(ctx, "SkySQL service not found, removing from state", map[string]interface{}{
				"id": state.ID.ValueString(),
			})
			resp.State.RemoveResource(ctx)

			return
		}
		return
	}
	if state.WaitForDeletion.ValueBool() {

		deleteTimeout, diagsErr := state.Timeouts.Delete(ctx, defaultDeleteTimeout)
		if diagsErr != nil {
			diagsErr.AddError("Error deleting service", fmt.Sprintf("Unable to delete service, got error: %s", err))
			resp.Diagnostics.Append(diagsErr...)
		}

		err = sdkresource.RetryContext(ctx, deleteTimeout, func() *sdkresource.RetryError {

			service, err := r.client.GetServiceByID(ctx, state.ID.ValueString())
			if err != nil {
				if errors.Is(err, skysql.ErrorServiceNotFound) {
					return nil
				}
				return sdkresource.NonRetryableError(fmt.Errorf("error retrieving service details: %v", err))
			}

			return sdkresource.RetryableError(fmt.Errorf("expected that the instance was deleted, but it was in state %s", service.Status))
		})

		if err != nil {
			resp.Diagnostics.AddError("Error delete service", fmt.Sprintf("Unable to delete service, got error: %s", err))
		}
	}
}

func (r *ServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *ServiceResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Plan does not need to be modified when the resource is being destroyed.
	if req.Plan.Raw.IsNull() {
		return
	}
}

func (r *ServiceResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {

	var config *ServiceResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !Contains[string]([]string{"gcp", "aws"}, config.Provider.ValueString()) {
		resp.Diagnostics.AddAttributeError(path.Root("provider"),
			"Invalid provider value",
			fmt.Sprintf("The %q is an invalid value. Allowed values: aws or gcp", config.Provider.ValueString()))
	}

	if config.Provider.ValueString() == "aws" {
		if !config.VolumeIOPS.IsNull() && config.VolumeType.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("volume_type"),
				"volume_type is require",
				"volume_type is required when volume_iops is set. "+
					"Use: io1 for volume_type if volume_iops is set")
			return
		}
		if !config.VolumeIOPS.IsNull() &&
			config.VolumeType.ValueString() != "io1" {
			resp.Diagnostics.AddAttributeError(path.Root("volume_type"),
				"volume_type must be io1 when you want to set IOPS",
				"Use: io1 for volume_type if volume_iops is set")
			return
		}
	} else {
		if !config.VolumeType.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("volume_type"),
				fmt.Sprintf("volume_type is not supported for %q provider", config.Provider.ValueString()),
				fmt.Sprintf("Volume type is not supported for %q provider", config.Provider.ValueString()))
			return
		}
		if !config.VolumeIOPS.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("volume_iops"),
				fmt.Sprintf("volume_iops is not supported for %q provider", config.Provider.ValueString()),
				fmt.Sprintf("Volume IOPS are not supported for %q provider", config.Provider.ValueString()))
			return
		}
	}

	if !Contains[string]([]string{"lakehouse", "sa"}, config.Topology.ValueString()) {
		if config.SSLEnabled.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("ssl_enabled"),
				"Missing required argument",
				fmt.Sprintf("The argument %q is required, but no definition was found.", "ssl_enabled"))
			return
		}
		if config.Storage.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("storage"),
				"Missing required argument",
				fmt.Sprintf("The argument %q is required, but no definition was found.", "storage"))
			return
		}
		if config.Size.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("size"),
				"Missing required argument",
				fmt.Sprintf("The argument %q is required, but no definition was found.", "size"))
			return
		}
		if config.Nodes.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("size"),
				"Missing required argument",
				fmt.Sprintf("The argument %q is required, but no definition was found.", "size"))
			return
		}
		if config.Version.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("version"),
				"Missing required argument",
				fmt.Sprintf("The argument %q is required, but no definition was found.", "version"))
			return
		}
	} else {
		if !config.Architecture.IsUnknown() && !config.Architecture.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("architecture"),
				"Attempt to modify read-only attribute",
				fmt.Sprintf("The argument %q is read only for the %q topology", "architecture", config.Topology.ValueString()))
		}
		if !config.Nodes.IsUnknown() && !config.Nodes.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("nodes"),
				"Attempt to modify read-only attribute",
				fmt.Sprintf("The argument %q is read only for the %q topology", "nodes", config.Topology.ValueString()))
		}

		if !config.Size.IsUnknown() && !config.Size.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("size"),
				"Attempt to modify read-only attribute",
				fmt.Sprintf("The argument %q is read only for the %q topology", "size", config.Topology.ValueString()))
		}

		if !config.SSLEnabled.IsUnknown() && !config.SSLEnabled.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("ssl_enabled"),
				"Attempt to modify read-only attribute",
				fmt.Sprintf("The argument %q is read only for the %q topology", "ssl_enabled", config.Topology.ValueString()))
		}

		if !config.Version.IsUnknown() && !config.Version.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("version"),
				"Attempt to modify read-only attribute",
				fmt.Sprintf("The argument %q is read only for the %q topology", "version", config.Topology.ValueString()))
		}
	}
}
