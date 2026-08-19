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

package resource

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/client"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/domain"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/util"
)

var (
	_ resource.Resource                = &ProjectUserResource{}
	_ resource.ResourceWithConfigure   = &ProjectUserResource{}
	_ resource.ResourceWithImportState = &ProjectUserResource{}
)

func NewProjectUserResource() resource.Resource {
	return &ProjectUserResource{}
}

type ProjectUserResource struct {
	auraApi *client.AuraApi
}

type ProjectUserResourceModel struct {
	OrganizationId     types.String   `tfsdk:"organization_id"`
	ProjectId          types.String   `tfsdk:"project_id"`
	UserId             types.String   `tfsdk:"user_id"`
	ProjectRoles       []types.String `tfsdk:"project_roles"`
	DeregisterOnDelete types.Bool     `tfsdk:"deregister_on_delete"`
}

func (r *ProjectUserResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_project_user"
}

func (r *ProjectUserResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	auraApi, ok := request.ProviderData.(*client.AuraApi)
	if !ok {
		response.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.AuraApi, got: %T. Please report this issue to the provider developers.", request.ProviderData),
		)
		return
	}
	r.auraApi = auraApi
}

func (r *ProjectUserResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Manages a user's membership in an Aura project.",
		Description:         "Manages a user's membership in an Aura project.",
		Attributes: map[string]schema.Attribute{
			"organization_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the organization that contains the project.",
				Description:         "The ID of the organization that contains the project.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the project the user is being registered in.",
				Description:         "The ID of the project the user is being registered in.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the user to register in the project.",
				Description:         "The ID of the user to register in the project.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project_roles": schema.ListAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: fmt.Sprintf("The roles assigned to the user within the project. Possible values: `%s`.", strings.Join(domain.ProjectRoles, "`, `")),
				Description:         fmt.Sprintf("The roles assigned to the user within the project. Possible values: %s.", strings.Join(domain.ProjectRoles, ", ")),
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.OneOf(domain.ProjectRoles...)),
				},
				PlanModifiers: []planmodifier.List{
					util.SortedStringList(),
				},
			},
			"deregister_on_delete": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "When `true`, the user is removed from the project on resource deletion. When `false` (default), the user's membership is left intact and only the Terraform state is removed.",
				Description:         "When true, the user is removed from the project on resource deletion. When false (default), the user's membership is left intact and only the Terraform state is removed.",
			},
		},
	}
}

func (r *ProjectUserResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data ProjectUserResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	orgId := data.OrganizationId.ValueString()
	projectId := data.ProjectId.ValueString()
	userId := data.UserId.ValueString()
	roles := util.ToStringSlice(data.ProjectRoles)

	// Verify the user exists in the org and check if they're already in the project.
	detailsResp, err := r.auraApi.GetOrganizationUserDetails(ctx, orgId, userId)
	if errors.Is(err, client.ErrNotFound) {
		response.Diagnostics.AddError(
			"User not found in organization",
			fmt.Sprintf("user %s does not exist in organization %s", userId, orgId),
		)
		return
	}
	if err != nil {
		response.Diagnostics.AddError("Error checking user", err.Error())
		return
	}

	var existingProject *client.UserProjectData
	for i, p := range detailsResp.Data.Projects {
		if p.Id == projectId {
			existingProject = &detailsResp.Data.Projects[i]
			break
		}
	}

	if existingProject != nil {
		// User already in project — reconcile roles if they differ.
		if !util.SlicesEqualIgnoringOrder(existingProject.ProjectRoles, roles) {
			_, err := r.auraApi.PatchProjectUser(ctx, orgId, projectId, userId, client.PatchProjectUserRequest{
				ProjectRoles: roles,
			})
			if err != nil {
				response.Diagnostics.AddError("Error updating project user roles", err.Error())
				return
			}
		}
	} else {
		if err := r.auraApi.PostProjectUser(ctx, orgId, projectId, client.PostProjectUserRequest{
			ProjectRoles: roles,
			UserId:       userId,
		}); err != nil {
			response.Diagnostics.AddError("Error registering user in project", err.Error())
			return
		}
	}

	data.ProjectRoles = util.ToTypesStringSlice(roles)
	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *ProjectUserResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data ProjectUserResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	orgId := data.OrganizationId.ValueString()
	projectId := data.ProjectId.ValueString()
	userId := data.UserId.ValueString()

	detailsResp, err := r.auraApi.GetOrganizationUserDetails(ctx, orgId, userId)
	if errors.Is(err, client.ErrNotFound) {
		response.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		response.Diagnostics.AddError("Error reading user", err.Error())
		return
	}

	for _, p := range detailsResp.Data.Projects {
		if p.Id == projectId {
			data.ProjectRoles = util.ToTypesStringSlice(util.SortedStrings(p.ProjectRoles))
			response.Diagnostics.Append(response.State.Set(ctx, &data)...)
			return
		}
	}

	// Project no longer in user's project list — remove from state.
	response.State.RemoveResource(ctx)
}

func (r *ProjectUserResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var data ProjectUserResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	orgId := data.OrganizationId.ValueString()
	projectId := data.ProjectId.ValueString()
	userId := data.UserId.ValueString()
	roles := util.ToStringSlice(data.ProjectRoles)

	_, err := r.auraApi.PatchProjectUser(ctx, orgId, projectId, userId, client.PatchProjectUserRequest{
		ProjectRoles: roles,
	})
	if err != nil {
		response.Diagnostics.AddError("Error updating project user roles", err.Error())
		return
	}

	data.ProjectRoles = util.ToTypesStringSlice(roles)

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *ProjectUserResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data ProjectUserResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	if !data.DeregisterOnDelete.ValueBool() {
		tflog.Info(ctx, "deregister_on_delete is false — leaving user membership intact, removing only Terraform state")
		return
	}

	orgId := data.OrganizationId.ValueString()
	projectId := data.ProjectId.ValueString()
	userId := data.UserId.ValueString()

	if err := r.auraApi.DeleteProjectUser(ctx, orgId, projectId, userId); err != nil {
		response.Diagnostics.AddError("Error removing user from project", err.Error())
		return
	}
}

func (r *ProjectUserResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	parts := strings.SplitN(request.ID, ",", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		response.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected format: {organization_id},{project_id},{user_id}. Got: %q", request.ID),
		)
		return
	}

	orgId, projectId, userId := parts[0], parts[1], parts[2]

	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("organization_id"), orgId)...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("project_id"), projectId)...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("user_id"), userId)...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("project_roles"), []string{})...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("deregister_on_delete"), false)...)
}
