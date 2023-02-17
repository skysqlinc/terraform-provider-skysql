package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/mariadb-corporation/terraform-provider-skysql-beta/internal/skysql"
	"net/url"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ datasource.DataSource = &VersionsDataSource{}

func NewVersionsDataSource() datasource.DataSource {
	return &VersionsDataSource{}
}

// VersionsDataSource defines the data source implementation.
type VersionsDataSource struct {
	client *skysql.Client
}

type VersionDataSourceDataSourceModel struct {
	Topology types.String   `tfsdk:"topology"`
	Versions []VersionModel `tfsdk:"versions"`
}

type VersionModel struct {
	Id              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Version         types.String `tfsdk:"version"`
	Topology        types.String `tfsdk:"topology"`
	Product         types.String `tfsdk:"product"`
	DisplayName     types.String `tfsdk:"display_name"`
	IsMajor         types.Bool   `tfsdk:"is_major"`
	ReleaseDate     types.String `tfsdk:"release_date"`
	ReleaseNotesUrl types.String `tfsdk:"release_notes_url"`
}

func (d *VersionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_versions"
}

func (d *VersionsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "SkySQL server versions",
		Attributes: map[string]schema.Attribute{
			"topology": schema.StringAttribute{
				Optional: true,
			},
			"versions": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "The ID of the version",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The name of the version",
						},
						"version": schema.StringAttribute{
							Computed:    true,
							Description: "The version display name",
						},
						"topology": schema.StringAttribute{
							Computed:    true,
							Description: "The topology that uses the version",
						},
						"product": schema.StringAttribute{
							Computed:    true,
							Description: "The product that uses the version",
						},
						"display_name": schema.StringAttribute{
							Computed:    true,
							Description: "The display name of the version",
						},
						"is_major": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether the version is a major version",
						},
						"release_date": schema.StringAttribute{
							Computed:    true,
							Description: "The release date of the version",
						},
						"release_notes_url": schema.StringAttribute{
							Computed:    true,
							Description: "The URL to the release notes of the version",
						},
					},
				},
			},
		},
	}
}

func (d *VersionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *VersionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state VersionDataSourceDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	versions, err := d.client.GetVersions(ctx, func(values url.Values) {
		if !state.Topology.IsNull() && len(state.Topology.String()) > 0 {
			tflog.Info(ctx, "Filtering versions by topology", map[string]interface{}{
				"topology": state.Topology.String(),
			})
			values.Set("topology", state.Topology.ValueString())
		}
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read SkySQL versions", err.Error())
		return
	}

	for _, version := range versions {
		versionState := VersionModel{
			Id:              types.StringValue(version.Id),
			Name:            types.StringValue(version.Name),
			Version:         types.StringValue(version.Version),
			Topology:        types.StringValue(version.Topology),
			Product:         types.StringValue(version.Product),
			DisplayName:     types.StringValue(version.DisplayName),
			IsMajor:         types.BoolValue(version.IsMajor),
			ReleaseDate:     types.StringValue(version.ReleaseDate.String()),
			ReleaseNotesUrl: types.StringValue(version.ReleaseNotesUrl),
		}
		state.Versions = append(state.Versions, versionState)
	}

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
