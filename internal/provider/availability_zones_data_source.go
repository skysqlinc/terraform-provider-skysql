package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mariadb-corporation/terraform-provider-skysql/internal/skysql"
	"net/url"
	"sort"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ datasource.DataSource = &AvailabilityZonesDataSource{}

func NewAvailabilityZonesDataSource() datasource.DataSource {
	return &AvailabilityZonesDataSource{}
}

// AvailabilityZonesDataSource defines the data source implementation.
type AvailabilityZonesDataSource struct {
	client *skysql.Client
}

// AvailabilityZonesDataSourceModel describes the data source data model.
type AvailabilityZonesDataSourceModel struct {
	Region   types.String             `tfsdk:"region"`
	Provider types.String             `tfsdk:"filter_by_provider"`
	Zones    []AvailabilityZonesModel `tfsdk:"zones"`
}

type AvailabilityZonesModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Region   types.String `tfsdk:"region"`
	Provider types.String `tfsdk:"provider"`
}

func (d *AvailabilityZonesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_availability_zones"
}

func (d *AvailabilityZonesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieve the list of availability_zones.",
		Attributes: map[string]schema.Attribute{
			"filter_by_provider": schema.StringAttribute{
				Optional:    true,
				Computed:    false,
				Required:    false,
				Description: "Filter availability zones by provider.",
			},
			"region": schema.StringAttribute{
				Optional: false,
				Computed: false,
				Required: true,
			},
			"zones": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "The ID of the availability zone.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The name of the availability zone.",
						},
						"region": schema.StringAttribute{
							Computed:    true,
							Description: "The region name of the availability zone.",
						},
						"provider": schema.StringAttribute{
							Computed:    true,
							Description: "The provider for the availability zone.",
						},
					},
				},
			},
		},
	}
}

func (d *AvailabilityZonesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = client
}

func (d *AvailabilityZonesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state AvailabilityZonesDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	zones, err := d.client.GetAvailabilityZones(ctx, state.Region.ValueString(), func(values url.Values) {
		if state.Provider.ValueString() != "" {
			values.Set("provider", state.Provider.ValueString())
		}
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read SkySQL availability zones", err.Error())
		return
	}

	sort.Slice(zones, func(i, j int) bool {
		return zones[i].Name < zones[j].Name
	})

	for _, zone := range zones {
		az := AvailabilityZonesModel{
			ID:       types.StringValue(zone.ID),
			Name:     types.StringValue(zone.Name),
			Region:   types.StringValue(zone.Region),
			Provider: types.StringValue(zone.Provider),
		}
		state.Zones = append(state.Zones, az)
	}

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
