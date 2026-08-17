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
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
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
	_ resource.Resource                = &OrganizationUserResource{}
	_ resource.ResourceWithConfigure   = &OrganizationUserResource{}
	_ resource.ResourceWithImportState = &OrganizationUserResource{}
)

func NewOrganizationUserResource() resource.Resource {
	return &OrganizationUserResource{}
}

type OrganizationUserResource struct {
	auraApi *client.AuraApi
}

type OrganizationUserResourceModel struct {
	Id                         types.String   `tfsdk:"id"`
	OrganizationId             types.String   `tfsdk:"organization_id"`
	OrganizationRoles          []types.String `tfsdk:"organization_roles"`
	DeregisterOnDelete         types.Bool     `tfsdk:"deregister_on_delete"`
	Email                      types.String   `tfsdk:"email"`
	ExemptFromAutomaticRemoval types.Bool     `tfsdk:"exempt_from_automatic_removal"`
	LastActivityAt             types.String   `tfsdk:"last_activity_at"`
	// types.Object is used instead of a plain struct so it can hold Terraform's
	// "unknown" value during the plan phase for computed nested attributes.
	MfaEnrollment types.Object `tfsdk:"mfa_enrollment"`
}

var mfaMethodAttrTypes = map[string]attr.Type{
	"id":          types.StringType,
	"enrolled_at": types.StringType,
}

var mfaEnrollmentAttrTypes = map[string]attr.Type{
	"status":  types.StringType,
	"methods": types.ListType{ElemType: types.ObjectType{AttrTypes: mfaMethodAttrTypes}},
}

func (r *OrganizationUserResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_organization_user"
}

func (r *OrganizationUserResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

func (r *OrganizationUserResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Manages a user's organization-level roles within an Aura organization.",
		Description:         "Manages a user's organization-level roles within an Aura organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the user.",
				Description:         "The ID of the user.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"organization_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the organization the user belongs to.",
				Description:         "The ID of the organization the user belongs to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"organization_roles": schema.ListAttribute{
				Required:            true,
				ElementType:         types.StringType,
				MarkdownDescription: fmt.Sprintf("The roles assigned to the user within the organization. Possible values: `%s`.", strings.Join(domain.OrganizationRoles, "`, `")),
				Description:         fmt.Sprintf("The roles assigned to the user within the organization. Possible values: %s.", strings.Join(domain.OrganizationRoles, ", ")),
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.OneOf(domain.OrganizationRoles...)),
				},
			},
			"deregister_on_delete": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "When `true`, the user is removed from the organization on resource deletion. When `false` (default), only the Terraform state is removed.",
				Description:         "When true, the user is removed from the organization on resource deletion. When false (default), only the Terraform state is removed.",
			},
			"email": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The email address of the user.",
				Description:         "The email address of the user.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
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
		},
	}
}

func (r *OrganizationUserResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data OrganizationUserResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	orgId := data.OrganizationId.ValueString()
	userId := data.Id.ValueString()
	desiredRoles := util.ToStringSlice(data.OrganizationRoles)

	detailsResp, err := r.auraApi.GetOrganizationUserDetails(ctx, orgId, userId)
	if errors.Is(err, client.ErrNotFound) {
		response.Diagnostics.AddError(
			"User not found in organization",
			fmt.Sprintf("user %s does not exist in organization %s", userId, orgId),
		)
		return
	}
	if err != nil {
		response.Diagnostics.AddError("Error checking organization user", err.Error())
		return
	}

	if !util.SlicesEqualIgnoringOrder(detailsResp.Data.OrganizationRoles, desiredRoles) {
		_, err := r.auraApi.PatchOrganizationUser(ctx, orgId, userId, client.PatchOrganizationUserRequest{
			OrganizationRoles: desiredRoles,
		})
		if err != nil {
			response.Diagnostics.AddError("Error updating organization user roles", err.Error())
			return
		}
		// Re-fetch to get fresh state after the patch.
		detailsResp, err = r.auraApi.GetOrganizationUserDetails(ctx, orgId, userId)
		if err != nil {
			response.Diagnostics.AddError("Error reading organization user after role update", err.Error())
			return
		}
	}

	populateOrganizationUserModel(&data, detailsResp.Data)
	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *OrganizationUserResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data OrganizationUserResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	orgId := data.OrganizationId.ValueString()
	userId := data.Id.ValueString()

	detailsResp, err := r.auraApi.GetOrganizationUserDetails(ctx, orgId, userId)
	if errors.Is(err, client.ErrNotFound) {
		response.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		response.Diagnostics.AddError("Error reading organization user", err.Error())
		return
	}

	populateOrganizationUserModel(&data, detailsResp.Data)
	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *OrganizationUserResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var data OrganizationUserResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	orgId := data.OrganizationId.ValueString()
	userId := data.Id.ValueString()
	roles := util.ToStringSlice(data.OrganizationRoles)

	_, err := r.auraApi.PatchOrganizationUser(ctx, orgId, userId, client.PatchOrganizationUserRequest{
		OrganizationRoles: roles,
	})
	if err != nil {
		response.Diagnostics.AddError("Error updating organization user roles", err.Error())
		return
	}

	detailsResp, err := r.auraApi.GetOrganizationUserDetails(ctx, orgId, userId)
	if err != nil {
		response.Diagnostics.AddError("Error reading organization user after update", err.Error())
		return
	}

	populateOrganizationUserModel(&data, detailsResp.Data)
	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *OrganizationUserResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data OrganizationUserResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	if !data.DeregisterOnDelete.ValueBool() {
		tflog.Info(ctx, "deregister_on_delete is false — leaving user organization membership intact, removing only Terraform state")
		return
	}

	for _, role := range data.OrganizationRoles {
		if role.ValueString() == domain.OrganizationRoleOwner {
			tflog.Warn(ctx, fmt.Sprintf(
				"user %s has the %q role and will not be removed from organization %s — remove the role first, then destroy",
				data.Id.ValueString(), domain.OrganizationRoleOwner, data.OrganizationId.ValueString(),
			))
			return
		}
	}

	orgId := data.OrganizationId.ValueString()
	userId := data.Id.ValueString()

	if err := r.auraApi.DeleteOrganizationUser(ctx, orgId, userId); err != nil {
		response.Diagnostics.AddError("Error removing user from organization", err.Error())
		return
	}
}

func (r *OrganizationUserResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	parts := strings.SplitN(request.ID, ",", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		response.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected format: {organization_id},{user_id}. Got: %q", request.ID),
		)
		return
	}

	orgId, userId := parts[0], parts[1]

	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("organization_id"), orgId)...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("id"), userId)...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("organization_roles"), []string{})...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("deregister_on_delete"), false)...)
}

func populateOrganizationUserModel(data *OrganizationUserResourceModel, apiData client.OrganizationUserDetailsData) {
	data.OrganizationRoles = util.ToTypesStringSlice(apiData.OrganizationRoles)
	data.Email = types.StringValue(apiData.Email)
	data.ExemptFromAutomaticRemoval = types.BoolValue(apiData.ExemptFromAutomaticRemoval)

	if apiData.LastActivityAt != nil {
		data.LastActivityAt = types.StringValue(*apiData.LastActivityAt)
	} else {
		data.LastActivityAt = types.StringNull()
	}

	methodVals := make([]attr.Value, len(apiData.MfaEnrolledMethods))
	for i, m := range apiData.MfaEnrolledMethods {
		obj, _ := types.ObjectValue(mfaMethodAttrTypes, map[string]attr.Value{
			"id":          types.StringValue(m.Id),
			"enrolled_at": types.StringValue(m.EnrolledAt),
		})
		methodVals[i] = obj
	}
	methodsList, _ := types.ListValue(types.ObjectType{AttrTypes: mfaMethodAttrTypes}, methodVals)
	data.MfaEnrollment, _ = types.ObjectValue(mfaEnrollmentAttrTypes, map[string]attr.Value{
		"status":  types.StringValue(apiData.MfaEnrollmentStatus),
		"methods": methodsList,
	})
}
