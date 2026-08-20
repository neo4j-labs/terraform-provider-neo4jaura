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
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/client"
)

var (
	_ datasource.DataSource              = &OrganizationsDataSource{}
	_ datasource.DataSourceWithConfigure = &OrganizationsDataSource{}
)

func NewOrganizationsDataSource() datasource.DataSource {
	return &OrganizationsDataSource{}
}

type OrganizationsDataSource struct {
	auraApi *client.AuraApi
}

type OrganizationsModel struct {
	Id            types.String `tfsdk:"id"`
	Organizations types.List   `tfsdk:"organizations"`
}

type OrganizationModel struct {
	Id   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func (ds *OrganizationsDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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

func (ds *OrganizationsDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_organizations"
}

func (ds *OrganizationsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Aura Organizations.",
		Description:         "Aura Organizations.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Placeholder identifier for the organizations data source.",
				Description:         "Placeholder identifier for the organizations data source.",
				Computed:            true,
			},
			"organizations": schema.ListNestedAttribute{
				MarkdownDescription: "The list of all available organizations.",
				Description:         "The list of all available organizations.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "The unique identifier of the organization.",
							Description:         "The unique identifier of the organization.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the organization.",
							Description:         "The name of the organization.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (ds *OrganizationsDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var data OrganizationsModel

	response.Diagnostics.Append(request.Config.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	orgsResponse, err := ds.auraApi.GetOrganizations(ctx)
	if err != nil {
		response.Diagnostics.AddError("Error while reading organizations", err.Error())
		return
	}

	orgs := make([]OrganizationModel, len(orgsResponse.Data))
	for i, o := range orgsResponse.Data {
		orgs[i] = OrganizationModel{
			Id:   types.StringValue(o.Id),
			Name: types.StringValue(o.Name),
		}
	}

	sort.Slice(orgs, func(i, j int) bool {
		return orgs[i].Id.ValueString() < orgs[j].Id.ValueString()
	})

	orgsValue, diags := types.ListValueFrom(ctx, data.Organizations.ElementType(ctx), orgs)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	data.Organizations = orgsValue
	data.Id = types.StringValue("organizations")

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}
