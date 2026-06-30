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
	Id             types.String `tfsdk:"id"`
	OrganizationId types.String `tfsdk:"organization_id"`
	Projects       types.List   `tfsdk:"projects"`
}

type ShortProjectModel struct {
	Id   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func (ds *ProjectsDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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

func (ds *ProjectsDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_projects"
}

func (ds *ProjectsDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Aura Projects.",
		Description:         "Aura Projects.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Placeholder identifier for the projects data source",
				Description:         "Placeholder identifier for the projects data source",
				Computed:            true,
			},
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "When set, fetches only projects belonging to this organization.",
				Description:         "When set, fetches only projects belonging to this organization.",
				Optional:            true,
			},
			"projects": schema.ListNestedAttribute{
				MarkdownDescription: "The list of all available projects.",
				Description:         "The list of all available projects.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "The unique identifier of the project.",
							Description:         "The unique identifier of the project.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the project.",
							Description:         "The name of the project.",
							Computed:            true,
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

	var tenantsResponse client.GetProjectsResponse
	var err error
	if !data.OrganizationId.IsNull() && !data.OrganizationId.IsUnknown() && data.OrganizationId.ValueString() != "" {
		tenantsResponse, err = ds.auraApi.GetProjectsByOrganizationId(ctx, data.OrganizationId.ValueString())
	} else {
		tenantsResponse, err = ds.auraApi.GetTenants(ctx)
	}
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
	if response.Diagnostics.HasError() {
		return
	}

	data.Projects = tenantsValue
	data.Id = types.StringValue("projects")

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}
}
