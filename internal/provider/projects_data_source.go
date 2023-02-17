package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mariadb-corporation/terraform-provider-skysql-beta/internal/skysql"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ datasource.DataSource = &ProjectsDataSource{}

func NewProjectsDataSource() datasource.DataSource {
	return &ProjectsDataSource{}
}

// ProjectsDataSource defines the data source implementation.
type ProjectsDataSource struct {
	client *skysql.Client
}

// ProjectsDataSourceDataSourceModel describes the data source data model.
type ProjectsDataSourceDataSourceModel struct {
	Projects []ProjectModel `tfsdk:"projects"`
}

type ProjectModel struct {
	Id          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	IsDefault   types.Bool   `tfsdk:"is_default"`
}

func (d *ProjectsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_projects"
}

func (d *ProjectsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieve the list of projects. Project is a way of grouping the services.",
		Attributes: map[string]schema.Attribute{
			"projects": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "The ID of the project.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The name of the project.",
						},
						"description": schema.StringAttribute{
							Computed:    true,
							Description: "The description of the project.",
						},
						"is_default": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether the project is the default project. ",
						},
					},
				},
			},
		},
	}
}

func (d *ProjectsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ProjectsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state ProjectsDataSourceDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projects, err := d.client.GetProjects(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read SkySQL projects", err.Error())
		return
	}

	for _, project := range projects {
		projectState := ProjectModel{
			Id:          types.StringValue(project.Id),
			Name:        types.StringValue(project.Name),
			Description: types.StringValue(project.Description),
			IsDefault:   types.BoolValue(project.IsDefault),
		}
		state.Projects = append(state.Projects, projectState)
	}

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
