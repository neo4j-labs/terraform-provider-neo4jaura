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

package datasource

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/client"
)

var (
	_ datasource.DataSource              = &ProjectsDataSource{}
	_ datasource.DataSourceWithConfigure = &ProjectsDataSource{}
)

func NewProjectDataSource() datasource.DataSource {
	return &ProjectsDataSource{}
}

type ProjectsDataSource struct {
	auraApi *client.AuraApi
}

type ProjectsModel struct {
	Projects types.List `tfsdk:"projects"`
}

type ShortProjectModel struct {
	Id   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func (ds *ProjectsDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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

func (ds *ProjectsDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_projects"
}

func (ds *ProjectsDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Aura Projects",
		Description:         "Aura Projects",
		Attributes: map[string]schema.Attribute{
			"projects": schema.ListNestedAttribute{
				MarkdownDescription: "List of all projects",
				Description:         "List of all projects",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Id of the project",
							Description:         "Id of the project",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Name of the project",
							Description:         "Name of the project",
						},
					},
				},
			},
		},
	}
}

func (ds *ProjectsDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var data ProjectsModel

	response.Diagnostics.Append(request.Config.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	tenantsResponse, err := ds.auraApi.GetTenants()
	if err != nil {
		response.Diagnostics.AddError("Error while reading projects", err.Error())
		return
	}

	tenants := make([]ShortProjectModel, len(tenantsResponse.Data))
	for i := 0; i < len(tenantsResponse.Data); i++ {
		t := tenantsResponse.Data[i]
		tenants[i] = ShortProjectModel{
			Id:   types.StringValue(t.Id),
			Name: types.StringValue(t.Name),
		}
	}

	tenantsValue, diags := types.ListValueFrom(ctx, data.Projects.ElementType(ctx), tenants)
	response.Diagnostics.Append(diags...)

	data.Projects = tenantsValue

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}
