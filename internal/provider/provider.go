package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/venikkin/neo4j-aura-terraform-provider/internal/provider/client"
	auradatasource "github.com/venikkin/neo4j-aura-terraform-provider/internal/provider/datasource"
)

type Neo4jAuraProvider struct {
	version string
}

type Neo4jAuraProviderModel struct {
	ClientId     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
}

func (n *Neo4jAuraProvider) Metadata(ctx context.Context, request provider.MetadataRequest, response *provider.MetadataResponse) {
	response.TypeName = "aura"
	response.Version = n.version
}

func (n *Neo4jAuraProvider) Schema(ctx context.Context, request provider.SchemaRequest, response *provider.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"client_id": schema.StringAttribute{
				MarkdownDescription: "Aura Client ID",
				Required:            true,
				Sensitive:           true,
			},
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "Aura Client Secret",
				Required:            true,
				Sensitive:           true,
			},
		},
	}
	// todo diagnostics
}

func (n *Neo4jAuraProvider) Configure(ctx context.Context, request provider.ConfigureRequest, response *provider.ConfigureResponse) {
	var data Neo4jAuraProviderModel

	response.Diagnostics.Append(request.Config.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	auraClient := client.NewAuraClient(
		data.ClientId.ValueString(),
		data.ClientSecret.ValueString(),
	)
	response.DataSourceData = auraClient
	response.ResourceData = auraClient
}

func (n *Neo4jAuraProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		auradatasource.NewTenantsDataSource,
	}
}

func (n *Neo4jAuraProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		//auraresource.NewInstanceResource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &Neo4jAuraProvider{
			version: version,
		}
	}
}
