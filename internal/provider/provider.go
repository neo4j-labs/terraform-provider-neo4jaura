/*
 *  Copyright (c) "Neo4j"
 *  Neo4j Sweden AB [https://neo4j.com]
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/client"
	auradatasource "github.com/neo4j-labs/terraform-provider-neo4jaura/internal/datasource"
	auraresource "github.com/neo4j-labs/terraform-provider-neo4jaura/internal/resource"
)

type Neo4jAuraProvider struct {
	version string
}

type Neo4jAuraProviderModel struct {
	ClientId     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
}

func (n *Neo4jAuraProvider) Metadata(ctx context.Context, request provider.MetadataRequest, response *provider.MetadataResponse) {
	response.TypeName = "neo4jaura"
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
		auradatasource.NewProjectDataSource,
		auradatasource.NewSnapshotDataSource,
	}
}

func (n *Neo4jAuraProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		auraresource.NewInstanceResource,
		auraresource.NewSnapshotResource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &Neo4jAuraProvider{
			version: version,
		}
	}
}
