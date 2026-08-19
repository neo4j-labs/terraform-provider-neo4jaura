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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
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
	_ resource.Resource                = &OrganizationInviteResource{}
	_ resource.ResourceWithConfigure   = &OrganizationInviteResource{}
	_ resource.ResourceWithImportState = &OrganizationInviteResource{}
)

func NewOrganizationInviteResource() resource.Resource {
	return &OrganizationInviteResource{}
}

type OrganizationInviteResource struct {
	auraApi *client.AuraApi
}

type OrganizationInviteResourceModel struct {
	Id                types.String         `tfsdk:"id"`
	OrganizationId    types.String         `tfsdk:"organization_id"`
	Email             types.String         `tfsdk:"email"`
	OrganizationRoles []types.String       `tfsdk:"organization_roles"`
	ProjectInvites    []ProjectInviteModel `tfsdk:"project_invites"`
	InvitedBy         types.String         `tfsdk:"invited_by"`
	ExpiresAt         types.String         `tfsdk:"expires_at"`
	Status            types.String         `tfsdk:"status"`
}

type ProjectInviteModel struct {
	ProjectId    types.String   `tfsdk:"project_id"`
	ProjectRoles []types.String `tfsdk:"project_roles"`
}

func (r *OrganizationInviteResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_organization_invite"
}

func (r *OrganizationInviteResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

func (r *OrganizationInviteResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Invites a user to an Aura organization by email, optionally granting project roles at the same time.\n\n" +
			"-> **Note:** The Aura API has no endpoint to edit a pending invite. Any change to `email`, `organization_roles`, or " +
			"`project_invites` will revoke the existing invite and create a new one.",
		Description: "Invites a user to an Aura organization by email, optionally granting project roles at the same time. " +
			"The Aura API has no endpoint to edit a pending invite, so any change to email, organization_roles, or project_invites " +
			"will revoke the existing invite and create a new one.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier of the invite.",
				Description:         "The unique identifier of the invite.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the organization to invite the user to.",
				Description:         "The ID of the organization to invite the user to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"email": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The email address of the invitee.",
				Description:         "The email address of the invitee.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"organization_roles": schema.ListAttribute{
				Required:            true,
				ElementType:         types.StringType,
				MarkdownDescription: fmt.Sprintf("The organization roles to grant the invitee. Possible values: `%s`.", strings.Join(domain.OrganizationRoles, "`, `")),
				Description:         fmt.Sprintf("The organization roles to grant the invitee. Possible values: %s.", strings.Join(domain.OrganizationRoles, ", ")),
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.OneOf(domain.OrganizationRoles...)),
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"project_invites": schema.ListNestedAttribute{
				Required:            true,
				MarkdownDescription: "Project roles to grant the invitee at the same time, alongside the organization role.",
				Description:         "Project roles to grant the invitee at the same time, alongside the organization role.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"project_id": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The ID of the project to grant roles in.",
							Description:         "The ID of the project to grant roles in.",
						},
						"project_roles": schema.ListAttribute{
							Required:            true,
							ElementType:         types.StringType,
							MarkdownDescription: fmt.Sprintf("The project roles to grant. Possible values: `%s`.", strings.Join(domain.InviteProjectRoles, "`, `")),
							Description:         fmt.Sprintf("The project roles to grant. Possible values: %s.", strings.Join(domain.InviteProjectRoles, ", ")),
							Validators: []validator.List{
								listvalidator.ValueStringsAre(stringvalidator.OneOf(domain.InviteProjectRoles...)),
							},
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
	}
}

func (r *OrganizationInviteResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data OrganizationInviteResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	orgId := data.OrganizationId.ValueString()

	req := client.PostOrganizationInviteRequest{
		Email:             data.Email.ValueString(),
		OrganizationRoles: util.ToStringSlice(data.OrganizationRoles),
		ProjectInvites:    toProjectInviteRequests(data.ProjectInvites),
	}

	inviteResp, err := r.auraApi.PostOrganizationInvite(ctx, orgId, req)
	if err != nil {
		response.Diagnostics.AddError("Error creating organization invite", err.Error())
		return
	}

	populateOrganizationInviteModel(&data, inviteResp.Data)
	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *OrganizationInviteResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data OrganizationInviteResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	orgId := data.OrganizationId.ValueString()
	inviteId := data.Id.ValueString()

	invitesResp, err := r.auraApi.GetOrganizationInvites(ctx, orgId)
	if err != nil {
		response.Diagnostics.AddError("Error reading organization invites", err.Error())
		return
	}

	for _, inv := range invitesResp.Data {
		if inv.Id == inviteId {
			populateOrganizationInviteModel(&data, inv)
			response.Diagnostics.Append(response.State.Set(ctx, &data)...)
			return
		}
	}

	response.State.RemoveResource(ctx)
}

// Update is unreachable in practice: every mutable attribute (email,
// organization_roles, project_invites) is marked RequiresReplace since the
// Aura API has no endpoint to edit a pending invite. The method still needs
// to satisfy the resource.Resource interface.
func (r *OrganizationInviteResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var data OrganizationInviteResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *OrganizationInviteResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data OrganizationInviteResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	orgId := data.OrganizationId.ValueString()
	inviteId := data.Id.ValueString()
	status := data.Status.ValueString()

	// the API 400s if you try to delete an invite that's already accepted/revoked/expired/declined.
	if status != domain.InviteStatusActive {
		tflog.Info(ctx, fmt.Sprintf("invite %s is already %q — nothing to revoke, removing only Terraform state", inviteId, status))
		return
	}

	if err := r.auraApi.DeleteOrganizationInvite(ctx, orgId, inviteId); err != nil && !errors.Is(err, client.ErrNotFound) {
		response.Diagnostics.AddError("Error revoking organization invite", err.Error())
		return
	}
}

func (r *OrganizationInviteResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	parts := strings.SplitN(request.ID, ",", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		response.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected format: {organization_id},{invite_id}. Got: %q", request.ID),
		)
		return
	}

	orgId, inviteId := parts[0], parts[1]

	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("organization_id"), orgId)...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("id"), inviteId)...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("email"), "")...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("organization_roles"), []string{})...)
}

func toProjectInviteRequests(models []ProjectInviteModel) []client.ProjectInviteRequest {
	if len(models) == 0 {
		return nil
	}
	requests := make([]client.ProjectInviteRequest, len(models))
	for i, m := range models {
		requests[i] = client.ProjectInviteRequest{
			ProjectId:    m.ProjectId.ValueString(),
			ProjectRoles: util.ToStringSlice(m.ProjectRoles),
		}
	}
	return requests
}

func populateOrganizationInviteModel(data *OrganizationInviteResourceModel, apiData client.OrganizationInviteData) {
	data.Id = types.StringValue(apiData.Id)
	data.OrganizationId = types.StringValue(apiData.OrganizationId)
	data.Email = types.StringValue(apiData.Email)
	data.OrganizationRoles = util.ToTypesStringSlice(apiData.OrganizationRoles)
	data.InvitedBy = types.StringValue(apiData.InvitedBy)
	data.ExpiresAt = types.StringValue(apiData.ExpiresAt)
	data.Status = types.StringValue(apiData.Status)

	// A nil (not just empty) slice is required so an unset `project_invites`
	// in config (null) round-trips through Create/Read as null rather than []
	// — otherwise Terraform reports an inconsistent result after apply.
	var projectInvites []ProjectInviteModel
	if len(apiData.ProjectInvites) > 0 {
		projectInvites = make([]ProjectInviteModel, len(apiData.ProjectInvites))
		for i, pi := range apiData.ProjectInvites {
			projectInvites[i] = ProjectInviteModel{
				ProjectId:    types.StringValue(pi.ProjectId),
				ProjectRoles: util.ToTypesStringSlice(pi.ProjectRoles),
			}
		}
	}
	data.ProjectInvites = projectInvites
}
