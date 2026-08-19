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
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/client"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/domain"
)

var (
	_ datasource.DataSource              = &OrganizationInvitesDataSource{}
	_ datasource.DataSourceWithConfigure = &OrganizationInvitesDataSource{}
)

func NewOrganizationInvitesDataSource() datasource.DataSource {
	return &OrganizationInvitesDataSource{}
}

type OrganizationInvitesDataSource struct {
	auraApi *client.AuraApi
}

type OrganizationInvitesModel struct {
	Id             types.String              `tfsdk:"id"`
	OrganizationId types.String              `tfsdk:"organization_id"`
	Invites        []OrganizationInviteModel `tfsdk:"invites"`
}

type OrganizationInviteModel struct {
	Id                types.String                   `tfsdk:"id"`
	Email             types.String                   `tfsdk:"email"`
	OrganizationRoles []types.String                 `tfsdk:"organization_roles"`
	ProjectInvites    []DataSourceProjectInviteModel `tfsdk:"project_invites"`
	InvitedBy         types.String                   `tfsdk:"invited_by"`
	ExpiresAt         types.String                   `tfsdk:"expires_at"`
	Status            types.String                   `tfsdk:"status"`
}

type DataSourceProjectInviteModel struct {
	ProjectId    types.String   `tfsdk:"project_id"`
	ProjectRoles []types.String `tfsdk:"project_roles"`
}

func (ds *OrganizationInvitesDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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

func (ds *OrganizationInvitesDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_organization_invites"
}

func (ds *OrganizationInvitesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Lists pending and historical invites for an Aura organization.",
		Description:         "Lists pending and historical invites for an Aura organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Placeholder identifier for the invites data source.",
				Description:         "Placeholder identifier for the invites data source.",
			},
			"organization_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the organization the invites belong to.",
				Description:         "The ID of the organization the invites belong to.",
			},
			"invites": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "The list of invites for the organization.",
				Description:         "The list of invites for the organization.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The unique identifier of the invite.",
							Description:         "The unique identifier of the invite.",
						},
						"email": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The email address of the invitee.",
							Description:         "The email address of the invitee.",
						},
						"organization_roles": schema.ListAttribute{
							Computed:            true,
							ElementType:         types.StringType,
							MarkdownDescription: fmt.Sprintf("The organization roles granted to the invitee. Possible values: `%s`.", strings.Join(domain.OrganizationRoles, "`, `")),
							Description:         fmt.Sprintf("The organization roles granted to the invitee. Possible values: %s.", strings.Join(domain.OrganizationRoles, ", ")),
						},
						"project_invites": schema.ListNestedAttribute{
							Computed:            true,
							MarkdownDescription: "Project roles granted to the invitee alongside the organization role.",
							Description:         "Project roles granted to the invitee alongside the organization role.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"project_id": schema.StringAttribute{
										Computed:            true,
										MarkdownDescription: "The ID of the project.",
										Description:         "The ID of the project.",
									},
									"project_roles": schema.ListAttribute{
										Computed:            true,
										ElementType:         types.StringType,
										MarkdownDescription: fmt.Sprintf("The project roles granted. Possible values: `%s`.", strings.Join(domain.InviteProjectRoles, "`, `")),
										Description:         fmt.Sprintf("The project roles granted. Possible values: %s.", strings.Join(domain.InviteProjectRoles, ", ")),
									},
								},
							},
						},
						"invited_by": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The ID of the user who created the invite.",
							Description:         "The ID of the user who created the invite.",
						},
						"expires_at": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The timestamp after which the invite can no longer be accepted.",
							Description:         "The timestamp after which the invite can no longer be accepted.",
						},
						"status": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: fmt.Sprintf("The status of the invite. Possible values: `%s`.", strings.Join(domain.InviteStatuses, "`, `")),
							Description:         fmt.Sprintf("The status of the invite. Possible values: %s.", strings.Join(domain.InviteStatuses, ", ")),
						},
					},
				},
			},
		},
	}
}

func (ds *OrganizationInvitesDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var data OrganizationInvitesModel

	response.Diagnostics.Append(request.Config.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	orgId := data.OrganizationId.ValueString()

	invitesResp, err := ds.auraApi.GetOrganizationInvites(ctx, orgId)
	if err != nil {
		response.Diagnostics.AddError("Error reading organization invites", err.Error())
		return
	}

	invites := make([]OrganizationInviteModel, 0, len(invitesResp.Data))
	for _, inv := range invitesResp.Data {
		orgRoles := make([]types.String, len(inv.OrganizationRoles))
		for i, r := range inv.OrganizationRoles {
			orgRoles[i] = types.StringValue(r)
		}

		projectInvites := make([]DataSourceProjectInviteModel, len(inv.ProjectInvites))
		for i, pi := range inv.ProjectInvites {
			roles := make([]types.String, len(pi.ProjectRoles))
			for j, r := range pi.ProjectRoles {
				roles[j] = types.StringValue(r)
			}
			projectInvites[i] = DataSourceProjectInviteModel{
				ProjectId:    types.StringValue(pi.ProjectId),
				ProjectRoles: roles,
			}
		}

		invites = append(invites, OrganizationInviteModel{
			Id:                types.StringValue(inv.Id),
			Email:             types.StringValue(inv.Email),
			OrganizationRoles: orgRoles,
			ProjectInvites:    projectInvites,
			InvitedBy:         types.StringValue(inv.InvitedBy),
			ExpiresAt:         types.StringValue(inv.ExpiresAt),
			Status:            types.StringValue(inv.Status),
		})
	}

	sort.Slice(invites, func(i, j int) bool {
		return invites[i].Id.ValueString() < invites[j].Id.ValueString()
	})

	data.Invites = invites
	data.Id = types.StringValue(orgId)

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}
