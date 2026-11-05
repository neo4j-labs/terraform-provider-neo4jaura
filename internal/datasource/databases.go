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
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package datasource

import (
	"context"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/client"
)

var (
	_ datasource.DataSource              = &DatabasesDataSource{}
	_ datasource.DataSourceWithConfigure = &DatabasesDataSource{}
)

func NewDatabasesDataSource() datasource.DataSource {
	return &DatabasesDataSource{}
}

type DatabasesDataSource struct {
	auraApi *client.AuraApi
}

type DatabasesModel struct {
	Id             types.String `tfsdk:"id"`
	OrganizationId types.String `tfsdk:"organization_id"`
	ProjectId      types.String `tfsdk:"project_id"`
	InstanceId     types.String `tfsdk:"instance_id"`
	Databases      types.List   `tfsdk:"databases"`
}

type ShortDatabaseModel struct {
	Id   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func (ds *DatabasesDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	auraApi, ok := request.ProviderData.(*client.AuraApi)
	if !ok {
		response.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.AuraApi, got %T. Please report this issue to the provider developers.", request.ProviderData),
		)
		return
	}
	ds.auraApi = auraApi
}

func (ds *DatabasesDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_databases"
}

func (ds *DatabasesDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "The databases of a Neo4j Aura instance (multi-database).",
		Description:         "The databases of a Neo4j Aura instance (multi-database).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Placeholder identifier for the databases data source",
				Description:         "Placeholder identifier for the databases data source",
				Computed:            true,
			},
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the organization that contains the project and instance.",
				Description:         "The ID of the organization that contains the project and instance.",
				Required:            true,
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the project that contains the instance.",
				Description:         "The ID of the project that contains the instance.",
				Required:            true,
			},
			"instance_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the instance to list databases for.",
				Description:         "The ID of the instance to list databases for.",
				Required:            true,
			},
			"databases": schema.ListNestedAttribute{
				MarkdownDescription: "The list of databases on the instance.",
				Description:         "The list of databases on the instance.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "The ID of the database.",
							Description:         "The ID of the database.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the database.",
							Description:         "The name of the database.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (ds *DatabasesDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var data DatabasesModel

	response.Diagnostics.Append(request.Config.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	databasesResponse, err := ds.auraApi.GetDatabasesByInstanceId(
		ctx,
		data.OrganizationId.ValueString(),
		data.ProjectId.ValueString(),
		data.InstanceId.ValueString(),
	)
	if err != nil {
		response.Diagnostics.AddError("Error while reading databases", err.Error())
		return
	}

	databases := make([]ShortDatabaseModel, len(databasesResponse.Data))
	for i := 0; i < len(databasesResponse.Data); i++ {
		d := databasesResponse.Data[i]
		databases[i] = ShortDatabaseModel{
			Id:   types.StringValue(d.Id),
			Name: types.StringValue(d.Name),
		}
	}

	sort.Slice(databases, func(i, j int) bool {
		return databases[i].Id.ValueString() < databases[j].Id.ValueString()
	})

	databasesValue, diags := types.ListValueFrom(ctx, data.Databases.ElementType(ctx), databases)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	data.Databases = databasesValue
	data.Id = types.StringValue("databases")

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}
}
