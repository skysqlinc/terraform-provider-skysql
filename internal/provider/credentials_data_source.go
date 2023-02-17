package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mariadb-corporation/terraform-provider-skysql/internal/skysql"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ datasource.DataSource = &CredentialsDataSource{}

func NewCredentialsDataSource() datasource.DataSource {
	return &CredentialsDataSource{}
}

// CredentialsDataSource defines the data source implementation.
type CredentialsDataSource struct {
	client *skysql.Client
}

type CredentialsDataSourceDataSourceModel struct {
	ID       types.String `tfsdk:"service_id"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	Host     types.String `tfsdk:"host"`
}

func (d *CredentialsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_credentials"
}

func (d *CredentialsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Returnes a default credentials for a SkySQL service",
		Attributes: map[string]schema.Attribute{
			"service_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the SkySQL service",
			},
			"username": schema.StringAttribute{
				Computed:    true,
				Description: "The database root username",
			},
			"password": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The database root user password",
			},
			"host": schema.StringAttribute{
				Computed:    true,
				Description: "The database root user host",
			},
		},
	}
}

func (d *CredentialsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *CredentialsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data CredentialsDataSourceDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.ValueString() == "" {
		resp.Diagnostics.AddError(
			"Missing Service ID",
			"service_id is required",
		)
		return
	}

	credentials, err := d.client.GetServiceCredentialsByID(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read SkySQL service", err.Error())
		return
	}

	data.Username = types.StringValue(credentials.Username)
	data.Password = types.StringValue(credentials.Password)
	data.Host = types.StringValue(credentials.Host)
	// Set state
	diags := resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
