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

package resource

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/client"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/domain"
)

var (
	_ resource.Resource                = &InstanceV2Resource{}
	_ resource.ResourceWithConfigure   = &InstanceV2Resource{}
	_ resource.ResourceWithImportState = &InstanceV2Resource{}
)

func NewInstanceV2Resource() resource.Resource {
	return &InstanceV2Resource{}
}

type InstanceV2Resource struct {
	auraApi *client.AuraApi
}

type InstanceV2ResourceModel struct {
	OrganizationId    types.String `tfsdk:"organization_id"`
	ProjectId         types.String `tfsdk:"project_id"`
	InstanceId        types.String `tfsdk:"instance_id"`
	Name              types.String `tfsdk:"name"`
	Region            types.String `tfsdk:"region"`
	Memory            types.String `tfsdk:"memory"`
	Type              types.String `tfsdk:"type"`
	CloudProvider     types.String `tfsdk:"cloud_provider"`
	Storage           types.String `tfsdk:"storage"`
	MultiDatabase     types.Bool   `tfsdk:"multi_database"`
	VectorOptimized   types.Bool   `tfsdk:"vector_optimized"`
	GraphAnalytics    types.String `tfsdk:"graph_analytics"`
	Status            types.String `tfsdk:"status"`
	ConnectionUrl     types.String `tfsdk:"connection_url"`
	Username          types.String `tfsdk:"username"`
	Password          types.String `tfsdk:"password"`
	DefaultDatabaseId types.String `tfsdk:"default_database_id"`
}

// supportedGraphAnalytics mirrors the v2beta1 graph_analytics enum (v1 exposed
// this as the boolean graph_analytics_plugin attribute).
var supportedGraphAnalytics = []string{
	domain.GraphAnalyticsUnavailable,
	domain.GraphAnalyticsServerless,
	domain.GraphAnalyticsPlugin,
}

func (r *InstanceV2Resource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_instance_v2"
}

func (r *InstanceV2Resource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

func (r *InstanceV2Resource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Manages a Neo4j Aura instance via the v2beta1 API. " +
			"Use `multi_database = true` to enable multi-database support (requires a `business-critical` or enterprise/VDC instance type).",
		Description: "Manages a Neo4j Aura instance via the v2beta1 API. " +
			"Use `multi_database = true` to enable multi-database support (requires a business-critical or enterprise/VDC instance type).",
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
				MarkdownDescription: "The ID of the project to create the instance in.",
				Description:         "The ID of the project to create the instance in.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"instance_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the instance, assigned by the Aura API.",
				Description:         "The ID of the instance, assigned by the Aura API.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the instance. The v2beta1 API has no rename operation, so changing the name forces recreation.",
				Description:         "The name of the instance. The v2beta1 API has no rename operation, so changing the name forces recreation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"region": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Region of the instance. One of the Aura-supported regions per cloud provider.",
				Description:         "Region of the instance. One of the Aura-supported regions per cloud provider.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"memory": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Memory allocated for the instance. One of: `%s`.", strings.Join(supportedMemory, "`, `")),
				Description:         fmt.Sprintf("Memory allocated for the instance. One of [%s]", strings.Join(supportedMemory, ", ")),
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(supportedMemory...),
				},
			},
			"storage": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("The storage allocated to the instance. One of: `%s`. Must be provided together with `memory` when changed in place.", strings.Join(supportedStorage, "`, `")),
				Description:         fmt.Sprintf("The storage allocated to the instance. One of [%s]. Must be provided together with memory when changed in place.", strings.Join(supportedStorage, ", ")),
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(supportedStorage...),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Type of the instance. One of: `%s`. Multi-database requires `%s` (or an enterprise/VDC type).", strings.Join(supportedTypes, "`, `"), domain.InstanceTypeBusinessCritical),
				Description:         fmt.Sprintf("Type of the instance. One of [%s]. Multi-database requires business-critical (or an enterprise/VDC type).", strings.Join(supportedTypes, ", ")),
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(domain.InstanceTypeBusinessCritical),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(supportedTypes...),
				},
			},
			"cloud_provider": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Cloud provider. One of: `%s`.", strings.Join(supportedCloudProviders, "`, `")),
				Description:         fmt.Sprintf("Cloud provider. One of [%s]", strings.Join(supportedCloudProviders, ", ")),
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(domain.CloudProviderGcp),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(supportedCloudProviders...),
				},
			},
			"multi_database": schema.BoolAttribute{
				MarkdownDescription: "Whether multi-database is enabled on the instance. Requires a `business-critical` or enterprise/VDC instance type. " +
					"The v2beta1 API does not support toggling this after creation, so changing it forces recreation.",
				Description: "Whether multi-database is enabled on the instance. Requires a business-critical or enterprise/VDC instance type. " +
					"The v2beta1 API does not support toggling this after creation, so changing it forces recreation.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"vector_optimized": schema.BoolAttribute{
				MarkdownDescription: "The vector optimization configuration of the instance.",
				Description:         "The vector optimization configuration of the instance.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"graph_analytics": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("The Graph Analytics capability of the instance. One of: `%s`.", strings.Join(supportedGraphAnalytics, "`, `")),
				Description:         fmt.Sprintf("The Graph Analytics capability of the instance. One of [%s]", strings.Join(supportedGraphAnalytics, ", ")),
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(supportedGraphAnalytics...),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("The status of the instance. One of: `%s`.", strings.Join(supportedStatuses, "`, `")),
				Description:         fmt.Sprintf("The status of the instance. One of [%s]", strings.Join(supportedStatuses, ", ")),
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"connection_url": schema.StringAttribute{
				MarkdownDescription: "The Bolt connection URL for the instance database. Only returned when the instance is created, so it is `null` for an imported instance.",
				Description:         "The Bolt connection URL for the instance database. Only returned when the instance is created, so it is null for an imported instance.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "The username for the instance database. Only returned when the instance is created, so it is `null` for an imported instance.",
				Description:         "The username for the instance database. Only returned when the instance is created, so it is null for an imported instance.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					UseStateForUnknownOrNull(),
				},
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "The password for the instance database. Only returned when the instance is created, so it is `null` for an imported instance.",
				Description:         "The password for the instance database. Only returned when the instance is created, so it is null for an imported instance.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					UseStateForUnknownOrNull(),
				},
			},
			"default_database_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the default database of the instance. Only returned when the instance is created, so it is `null` for an imported instance.",
				Description:         "The ID of the default database of the instance. Only returned when the instance is created, so it is null for an imported instance.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					UseStateForUnknownOrNull(),
				},
			},
		},
	}
}

// ConfigValidators returns a list of resource-level validators.
func (r *InstanceV2Resource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		&multiDatabaseTierValidator{},
	}
}

func (r *InstanceV2Resource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data InstanceV2ResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	orgId := data.OrganizationId.ValueString()
	projectId := data.ProjectId.ValueString()

	createReq := &client.PostInstanceV2Request{
		Name:          data.Name.ValueString(),
		Region:        data.Region.ValueString(),
		Memory:        data.Memory.ValueString(),
		Type:          data.Type.ValueString(),
		CloudProvider: data.CloudProvider.ValueString(),
	}
	if !data.Storage.IsUnknown() && !data.Storage.IsNull() {
		createReq.Storage = data.Storage.ValueStringPointer()
	}
	if !data.MultiDatabase.IsUnknown() {
		createReq.MultiDatabase = data.MultiDatabase.ValueBoolPointer()
	}
	if !data.VectorOptimized.IsUnknown() {
		createReq.VectorOptimized = data.VectorOptimized.ValueBoolPointer()
	}
	if !data.GraphAnalytics.IsUnknown() && !data.GraphAnalytics.IsNull() {
		createReq.GraphAnalytics = data.GraphAnalytics.ValueStringPointer()
	}

	createResp, err := r.auraApi.PostInstanceV2(ctx, orgId, projectId, *createReq)
	if err != nil {
		response.Diagnostics.AddError(
			"Error while creating an instance",
			fmt.Sprintf("organization_id=%s project_id=%s: %s", orgId, projectId, err.Error()),
		)
		return
	}

	data.InstanceId = types.StringValue(createResp.Data.Id)
	data.ConnectionUrl = types.StringValue(createResp.Data.ConnectionUrl)
	data.Username = types.StringValue(createResp.Data.Username)
	data.Password = types.StringValue(createResp.Data.Password)
	if createResp.Data.DefaultDatabaseId != nil {
		data.DefaultDatabaseId = types.StringValue(*createResp.Data.DefaultDatabaseId)
	}

	tflog.Debug(ctx, "Created an instance with id "+createResp.Data.Id)

	// Instance IDs are shared between the v1 and v2beta1 APIs, and the v2beta1
	// GET exposes the legacy status string, so the existing v1-based poller is
	// reused to wait for the instance to become running.
	_, err = r.auraApi.WaitUntilInstanceIsInState(ctx, createResp.Data.Id, func(r client.GetInstanceResponse) bool {
		return strings.ToLower(r.Data.Status) == domain.InstanceStatusRunning
	})
	if err != nil {
		response.Diagnostics.AddError(
			"Instance is not running in time",
			fmt.Sprintf("instance_id=%s: %s", createResp.Data.Id, err.Error()),
		)
		return
	}

	if err := r.refreshFromApi(ctx, &data); err != nil {
		response.Diagnostics.AddError("Error while reading the instance after creation", err.Error())
		return
	}
	data.Status = types.StringValue(domain.InstanceStatusRunning)

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}
}

func (r *InstanceV2Resource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data InstanceV2ResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	instance, err := r.auraApi.GetInstanceV2ById(
		ctx,
		data.OrganizationId.ValueString(),
		data.ProjectId.ValueString(),
		data.InstanceId.ValueString(),
	)
	if errors.Is(err, client.ErrNotFound) {
		response.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		response.Diagnostics.AddError(
			"Error while reading the instance",
			fmt.Sprintf("instance_id=%s: %s", data.InstanceId.ValueString(), err.Error()),
		)
		return
	}

	r.applyApiData(&data, instance.Data)

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}
}

// Update only supports in-place memory/storage changes: the v2beta1 PATCH
// requires memory and storage to be provided together. All other attributes
// are RequiresReplace.
func (r *InstanceV2Resource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var data InstanceV2ResourceModel
	var state InstanceV2ResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	orgId := data.OrganizationId.ValueString()
	projectId := data.ProjectId.ValueString()
	instanceId := data.InstanceId.ValueString()

	memoryChanged := data.Memory.ValueString() != state.Memory.ValueString()
	storageChanged := !data.Storage.Equal(state.Storage)

	if memoryChanged || storageChanged {
		patchReq := client.PatchInstanceV2Request{
			Memory:  data.Memory.ValueStringPointer(),
			Storage: data.Storage.ValueStringPointer(),
		}
		tflog.Debug(ctx, fmt.Sprintf("Updating instance %s: memory %s -> %s, storage %s -> %s",
			instanceId, state.Memory.ValueString(), data.Memory.ValueString(), state.Storage.ValueString(), data.Storage.ValueString()))
		if err := r.auraApi.PatchInstanceV2(ctx, orgId, projectId, instanceId, patchReq); err != nil {
			response.Diagnostics.AddError(
				"Error while updating the instance details",
				fmt.Sprintf("instance_id=%s: %s", instanceId, err.Error()),
			)
			return
		}

		_, err := r.auraApi.WaitUntilInstanceIsInState(ctx, instanceId, func(r client.GetInstanceResponse) bool {
			return strings.ToLower(r.Data.Status) == domain.InstanceStatusRunning
		})
		if err != nil {
			response.Diagnostics.AddError(
				"Instance is not running in time after update",
				fmt.Sprintf("instance_id=%s: %s", instanceId, err.Error()),
			)
			return
		}
	}

	// Refresh all computed fields from the API before writing state.
	if err := r.refreshFromApi(ctx, &data); err != nil {
		response.Diagnostics.AddError("Error while reading the instance after update", err.Error())
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}
}

func (r *InstanceV2Resource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data InstanceV2ResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	orgId := data.OrganizationId.ValueString()
	projectId := data.ProjectId.ValueString()
	instanceId := data.InstanceId.ValueString()

	if err := r.auraApi.DeleteInstanceV2ById(ctx, orgId, projectId, instanceId); err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return
		}
		response.Diagnostics.AddError(
			"Error while deleting an instance",
			fmt.Sprintf("instance_id=%s: %s", instanceId, err.Error()),
		)
		return
	}

	// Deletion is asynchronous (202 Accepted) — poll until the instance is gone.
	if err := r.auraApi.WaitUntilInstanceIsDeleted(ctx, instanceId); err != nil {
		response.Diagnostics.AddError(
			"Error while waiting for the instance to be deleted",
			fmt.Sprintf("instance_id=%s: %s", instanceId, err.Error()),
		)
		return
	}
}

func (r *InstanceV2Resource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	parts := strings.Split(request.ID, "/")
	if len(parts) != 3 {
		response.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected format: {organization_id}/{project_id}/{instance_id}. Got: %q", request.ID),
		)
		return
	}
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	if parts[0] == "" || parts[1] == "" || parts[2] == "" {
		response.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected format: {organization_id}/{project_id}/{instance_id}. Got: %q", request.ID),
		)
		return
	}
	orgId, projectId, instanceId := parts[0], parts[1], parts[2]

	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("organization_id"), orgId)...)
	if response.Diagnostics.HasError() {
		return
	}
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("project_id"), projectId)...)
	if response.Diagnostics.HasError() {
		return
	}
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("instance_id"), instanceId)...)
	if response.Diagnostics.HasError() {
		return
	}
}

// refreshFromApi reads the instance from the v2beta1 API and applies all fields
// the API returns. Fields that are only exposed at creation time (connection
// URL, credentials, default database id) keep their existing values.
func (r *InstanceV2Resource) refreshFromApi(ctx context.Context, data *InstanceV2ResourceModel) error {
	instance, err := r.auraApi.GetInstanceV2ById(
		ctx,
		data.OrganizationId.ValueString(),
		data.ProjectId.ValueString(),
		data.InstanceId.ValueString(),
	)
	if err != nil {
		return fmt.Errorf("instance_id=%s: %s", data.InstanceId.ValueString(), err.Error())
	}
	r.applyApiData(data, instance.Data)
	return nil
}

// applyApiData copies the API-returned fields onto the model, preserving
// create-only fields that the GET endpoint does not report. Optional fields the
// API omits are written as explicit nulls so no planned unknown survives into
// the final state.
func (r *InstanceV2Resource) applyApiData(data *InstanceV2ResourceModel, d client.GetInstanceV2Data) {
	data.Name = types.StringValue(d.Name)
	data.Region = types.StringValue(d.Region)
	data.CloudProvider = types.StringValue(d.CloudProvider)
	data.Type = types.StringValue(d.Type)
	data.Memory = types.StringValue(d.Memory)
	if d.Storage != nil {
		data.Storage = types.StringValue(*d.Storage)
	} else {
		data.Storage = types.StringNull()
	}
	if d.MultiDatabase != nil {
		data.MultiDatabase = types.BoolValue(*d.MultiDatabase)
	} else {
		data.MultiDatabase = types.BoolNull()
	}
	if d.VectorOptimized != nil {
		data.VectorOptimized = types.BoolValue(*d.VectorOptimized)
	} else {
		data.VectorOptimized = types.BoolNull()
	}
	if d.GraphAnalytics != nil {
		data.GraphAnalytics = types.StringValue(*d.GraphAnalytics)
	} else {
		data.GraphAnalytics = types.StringNull()
	}
	if d.LegacyStatus != "" {
		data.Status = types.StringValue(d.LegacyStatus)
	}
}

// multiDatabaseTierValidator verifies that multi_database is only requested on
// instance types that support it (business-critical and enterprise/VDC types).
type multiDatabaseTierValidator struct{}

func (v *multiDatabaseTierValidator) Description(_ context.Context) string {
	return "multi_database is only supported on business-critical and enterprise/VDC instance types"
}

func (v *multiDatabaseTierValidator) MarkdownDescription(_ context.Context) string {
	return "multi_database is only supported on `business-critical` and enterprise/VDC instance types"
}

func (v *multiDatabaseTierValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data InstanceV2ResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.MultiDatabase.IsNull() || data.MultiDatabase.IsUnknown() || !data.MultiDatabase.ValueBool() {
		return
	}
	if data.Type.IsNull() || data.Type.IsUnknown() {
		return
	}

	instanceType := data.Type.ValueString()
	switch instanceType {
	case domain.InstanceTypeBusinessCritical,
		domain.InstanceTypeEnterpriseDb,
		domain.InstanceTypeEnterpriseDs:
		return
	default:
		resp.Diagnostics.AddAttributeError(
			path.Root("multi_database"),
			"Unsupported instance type for multi_database",
			fmt.Sprintf("multi_database requires a %q or enterprise/VDC instance type, but the configured type is %q.", domain.InstanceTypeBusinessCritical, instanceType),
		)
	}
}
