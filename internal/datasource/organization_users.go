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
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/client"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/domain"
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
	Id              types.String            `tfsdk:"id"`
	OrganizationId  types.String            `tfsdk:"organization_id"`
	ProjectId       types.String            `tfsdk:"project_id"`
	IncludeProjects types.Bool              `tfsdk:"include_projects"`
	Users           []OrganizationUserModel `tfsdk:"users"`
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
				MarkdownDescription: "The ID of the organization the users are members of.",
				Description:         "The ID of the organization the users are members of.",
			},
			"project_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "When set, only users with membership in this project are returned.",
				Description:         "When set, only users with membership in this project are returned.",
			},
			"include_projects": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "When true, fetches project membership. Defaults to `true`.",
				Description:         "When true, fetches project membership. Defaults to true.",
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
									MarkdownDescription: fmt.Sprintf("The MFA enrollment status. Possible values: `%s`.", strings.Join(domain.MfaEnrollmentStatuses, "`, `")),
									Description:         fmt.Sprintf("The MFA enrollment status. Possible values: %s.", strings.Join(domain.MfaEnrollmentStatuses, ", ")),
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
							MarkdownDescription: fmt.Sprintf("The user's roles within the organization. Possible values: `%s`.", strings.Join(domain.OrganizationRoles, "`, `")),
							Description:         fmt.Sprintf("The user's roles within the organization. Possible values: %s.", strings.Join(domain.OrganizationRoles, ", ")),
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
										MarkdownDescription: fmt.Sprintf("The user's roles within the project. Possible values: `%s`.", strings.Join(domain.ProjectRoles, "`, `")),
										Description:         fmt.Sprintf("The user's roles within the project. Possible values: %s.", strings.Join(domain.ProjectRoles, ", ")),
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
	projectId := data.ProjectId.ValueString()                                            // "" when null
	includeProjects := data.IncludeProjects.IsNull() || data.IncludeProjects.ValueBool() // true by default

	usersResp, err := ds.auraApi.GetOrganizationUsers(ctx, orgId)
	if err != nil {
		response.Diagnostics.AddError("Error reading users", err.Error())
		return
	}

	// When project_id is set, fetch project members in one call.
	// Roles per user come from this response; no per-user detail call is needed for the filtered path.
	var projectMemberIDs map[string]struct{}
	var projectMemberRoles map[string][]string // userId → project roles
	var projectName string
	if projectId != "" {
		projectUsersResp, err := ds.auraApi.GetProjectUsers(ctx, orgId, projectId)
		if err != nil {
			response.Diagnostics.AddError("Error reading project users", err.Error())
			return
		}
		projectMemberIDs = make(map[string]struct{}, len(projectUsersResp.Data))
		projectMemberRoles = make(map[string][]string, len(projectUsersResp.Data))
		for _, pu := range projectUsersResp.Data {
			projectMemberIDs[pu.UserId] = struct{}{}
			projectMemberRoles[pu.UserId] = pu.ProjectRoles
		}

		// Fetch the project name once so we can build the project model without per-user calls.
		if includeProjects {
			projectsResp, err := ds.auraApi.GetProjectsByOrganizationId(ctx, orgId)
			if err != nil {
				response.Diagnostics.AddError("Error reading organization projects", err.Error())
				return
			}
			for _, p := range projectsResp.Data {
				if p.Id == projectId {
					projectName = p.Name
					break
				}
			}
		}
	}

	users := make([]OrganizationUserModel, 0, len(usersResp.Data))
	for _, u := range usersResp.Data {
		if projectMemberIDs != nil {
			if _, ok := projectMemberIDs[u.UserId]; !ok {
				continue
			}
		}

		var projectModels []UserProjectModel
		switch {
		case includeProjects && projectId != "":
			// Roles already fetched from the project users call; name from the projects list.
			roles := make([]types.String, len(projectMemberRoles[u.UserId]))
			for j, r := range projectMemberRoles[u.UserId] {
				roles[j] = types.StringValue(r)
			}
			projectModels = []UserProjectModel{{
				Id:           types.StringValue(projectId),
				Name:         types.StringValue(projectName),
				ProjectRoles: roles,
			}}
		case includeProjects:
			// No project filter — fetch full project list via per-user detail call.
			detailsResp, err := ds.auraApi.GetOrganizationUserDetails(ctx, orgId, u.UserId)
			if err != nil {
				response.Diagnostics.AddError("Error reading user details", fmt.Sprintf("user %s: %s", u.UserId, err.Error()))
				return
			}
			projectModels = make([]UserProjectModel, len(detailsResp.Data.Projects))
			for i, p := range detailsResp.Data.Projects {
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
		default:
			projectModels = []UserProjectModel{}
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
