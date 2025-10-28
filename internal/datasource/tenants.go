package datasource

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/venikkin/neo4j-aura-terraform-provider/internal/client"
)

var (
	_ datasource.DataSource              = &TenantsDataSource{}
	_ datasource.DataSourceWithConfigure = &TenantsDataSource{}
)

func NewTenantsDataSource() datasource.DataSource {
	return &TenantsDataSource{}
}

type TenantsDataSource struct {
	auraApi *client.AuraApi
}

type TenantsModel struct {
	Tenants types.List `tfsdk:"tenants"`
}

type ShortTenantModel struct {
	Id   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func (ds *TenantsDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	auraClient, ok := request.ProviderData.(*client.AuraClient)
	if !ok {
		response.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.AuraClient, got %T. Please report this issue to the provider developers.", request.ProviderData),
		)
		return
	}
	ds.auraApi = client.NewAuraApi(auraClient)
}

func (ds *TenantsDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_tenants"
}

func (ds *TenantsDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Data Source containing all Aura Tenants",
		Attributes: map[string]schema.Attribute{
			"tenants": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
						},
						"name": schema.StringAttribute{
							Computed: true,
						},
					},
				},
				Computed: true,
			},
		},
	}
}

func (ds *TenantsDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var data TenantsModel

	response.Diagnostics.Append(request.Config.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	tenantsResponse, err := ds.auraApi.GetTenants()
	if err != nil {
		response.Diagnostics.AddError("Error while reading tenants", err.Error())
		return
	}

	tenants := make([]ShortTenantModel, len(tenantsResponse.Data))
	for i := 0; i < len(tenantsResponse.Data); i++ {
		t := tenantsResponse.Data[i]
		tenants[i] = ShortTenantModel{
			Id:   types.StringValue(t.Id),
			Name: types.StringValue(t.Name),
		}
	}

	tenantsValue, diags := types.ListValueFrom(ctx, data.Tenants.ElementType(ctx), tenants)
	response.Diagnostics.Append(diags...)

	data.Tenants = tenantsValue

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}
