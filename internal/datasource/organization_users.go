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
	_ datasource.DataSource              = &OrganizationUsersDataSource{}
	_ datasource.DataSourceWithConfigure = &OrganizationUsersDataSource{}
)

func NewUsersDataSource() datasource.DataSource {
	return &OrganizationUsersDataSource{}
}

type OrganizationUsersDataSource struct {
	auraApi *client.AuraApi
}

type OrganizationUsersModel struct {
	Id             types.String            `tfsdk:"id"`
	OrganizationId types.String            `tfsdk:"organization_id"`
	ProjectId      types.String            `tfsdk:"project_id"`
	Users          []OrganizationUserModel `tfsdk:"users"`
}

type OrganizationUserModel struct {
	Id                         types.String       `tfsdk:"id"`
	Email                      types.String       `tfsdk:"email"`
	ExemptFromAutomaticRemoval types.Bool         `tfsdk:"exempt_from_automatic_removal"`
	LastActivityAt             types.String       `tfsdk:"last_activity_at"`
	MfaEnrollment              MfaEnrollmentModel `tfsdk:"mfa_enrollment"`
	OrganizationRoles          []types.String     `tfsdk:"organization_roles"`
	Projects                   []UserProjectModel `tfsdk:"projects"`
}

type MfaEnrollmentModel struct {
	Status  types.String     `tfsdk:"status"`
	Methods []MfaMethodModel `tfsdk:"methods"`
}

type MfaMethodModel struct {
	Id         types.String `tfsdk:"id"`
	EnrolledAt types.String `tfsdk:"enrolled_at"`
}

type UserProjectModel struct {
	Id           types.String   `tfsdk:"id"`
	Name         types.String   `tfsdk:"name"`
	ProjectRoles []types.String `tfsdk:"project_roles"`
}

func (ds *OrganizationUsersDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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

func (ds *OrganizationUsersDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_organization_users"
}

func (ds *OrganizationUsersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Aura Organization Users.",
		Description:         "Aura Organization Users.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Placeholder identifier for the users data source.",
				Description:         "Placeholder identifier for the users data source.",
			},
			"organization_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the organization to fetch users from.",
				Description:         "The ID of the organization to fetch users from.",
			},
			"project_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "When set, only users belonging to this project are returned.",
				Description:         "When set, only users belonging to this project are returned.",
			},
			"users": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "The list of users in the organization.",
				Description:         "The list of users in the organization.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The unique identifier of the user.",
							Description:         "The unique identifier of the user.",
						},
						"email": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The email address of the user.",
							Description:         "The email address of the user.",
						},
						"exempt_from_automatic_removal": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Whether the user is exempt from automatic removal due to inactivity.",
							Description:         "Whether the user is exempt from automatic removal due to inactivity.",
						},
						"last_activity_at": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The timestamp of the user's last activity.",
							Description:         "The timestamp of the user's last activity.",
						},
						"mfa_enrollment": schema.SingleNestedAttribute{
							Computed:            true,
							MarkdownDescription: "The MFA enrollment details of the user.",
							Description:         "The MFA enrollment details of the user.",
							Attributes: map[string]schema.Attribute{
								"status": schema.StringAttribute{
									Computed:            true,
									MarkdownDescription: "The MFA enrollment status.",
									Description:         "The MFA enrollment status.",
								},
								"methods": schema.ListNestedAttribute{
									Computed:            true,
									MarkdownDescription: "The MFA methods the user has enrolled in.",
									Description:         "The MFA methods the user has enrolled in.",
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"id": schema.StringAttribute{
												Computed:            true,
												MarkdownDescription: "The MFA method identifier.",
												Description:         "The MFA method identifier.",
											},
											"enrolled_at": schema.StringAttribute{
												Computed:            true,
												MarkdownDescription: "The timestamp when the user enrolled in this MFA method.",
												Description:         "The timestamp when the user enrolled in this MFA method.",
											},
										},
									},
								},
							},
						},
						"organization_roles": schema.ListAttribute{
							Computed:            true,
							ElementType:         types.StringType,
							MarkdownDescription: "The user's roles within the organization.",
							Description:         "The user's roles within the organization.",
						},
						"projects": schema.ListNestedAttribute{
							Computed:            true,
							MarkdownDescription: "The projects the user has access to within the organization.",
							Description:         "The projects the user has access to within the organization.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id": schema.StringAttribute{
										Computed:            true,
										MarkdownDescription: "The unique identifier of the project.",
										Description:         "The unique identifier of the project.",
									},
									"name": schema.StringAttribute{
										Computed:            true,
										MarkdownDescription: "The name of the project.",
										Description:         "The name of the project.",
									},
									"project_roles": schema.ListAttribute{
										Computed:            true,
										ElementType:         types.StringType,
										MarkdownDescription: "The user's roles within the project.",
										Description:         "The user's roles within the project.",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (ds *OrganizationUsersDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var data OrganizationUsersModel

	response.Diagnostics.Append(request.Config.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	orgId := data.OrganizationId.ValueString()

	usersResp, err := ds.auraApi.GetOrganizationUsers(ctx, orgId)
	if err != nil {
		response.Diagnostics.AddError("Error reading users", err.Error())
		return
	}

	projectId := ""
	if !data.ProjectId.IsNull() && !data.ProjectId.IsUnknown() {
		projectId = data.ProjectId.ValueString()
	}

	users := make([]OrganizationUserModel, 0)
	for _, u := range usersResp.Data {
		detailsResp, err := ds.auraApi.GetOrganizationUserDetails(ctx, orgId, u.UserId)
		if err != nil {
			response.Diagnostics.AddError("Error reading user details", fmt.Sprintf("user %s: %s", u.UserId, err.Error()))
			return
		}

		projects := detailsResp.Data.Projects
		if projectId != "" {
			belongsToProject := false
			for _, p := range projects {
				if p.Id == projectId {
					belongsToProject = true
					break
				}
			}
			if !belongsToProject {
				continue
			}
		}

		lastActivityAt := types.StringNull()
		if u.LastActivityAt != nil {
			lastActivityAt = types.StringValue(*u.LastActivityAt)
		}

		methods := make([]MfaMethodModel, len(u.MfaEnrolledMethods))
		for i, m := range u.MfaEnrolledMethods {
			methods[i] = MfaMethodModel{
				Id:         types.StringValue(m.Id),
				EnrolledAt: types.StringValue(m.EnrolledAt),
			}
		}

		orgRoles := make([]types.String, len(u.OrganizationRoles))
		for i, r := range u.OrganizationRoles {
			orgRoles[i] = types.StringValue(r)
		}

		projectModels := make([]UserProjectModel, len(projects))
		for i, p := range projects {
			roles := make([]types.String, len(p.ProjectRoles))
			for j, r := range p.ProjectRoles {
				roles[j] = types.StringValue(r)
			}
			projectModels[i] = UserProjectModel{
				Id:           types.StringValue(p.Id),
				Name:         types.StringValue(p.Name),
				ProjectRoles: roles,
			}
		}

		users = append(users, OrganizationUserModel{
			Id:                         types.StringValue(u.UserId),
			Email:                      types.StringValue(u.Email),
			ExemptFromAutomaticRemoval: types.BoolValue(u.ExemptFromAutomaticRemoval),
			LastActivityAt:             lastActivityAt,
			MfaEnrollment: MfaEnrollmentModel{
				Status:  types.StringValue(u.MfaEnrollmentStatus),
				Methods: methods,
			},
			OrganizationRoles: orgRoles,
			Projects:          projectModels,
		})
	}

	data.Users = users
	data.Id = types.StringValue(orgId)

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}
