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

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/client"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/domain"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/util"
)

// Ensure resource defines types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &InstanceResource{}
	_ resource.ResourceWithConfigure   = &InstanceResource{}
	_ resource.ResourceWithImportState = &InstanceResource{}
)

func NewInstanceResource() resource.Resource {
	return &InstanceResource{}
}

type InstanceResource struct {
	auraApi *client.AuraApi
}

type InstanceResourceModel struct {
	InstanceId            types.String `tfsdk:"instance_id"`
	Name                  types.String `tfsdk:"name"`
	Region                types.String `tfsdk:"region"`
	Memory                types.String `tfsdk:"memory"`
	Type                  types.String `tfsdk:"type"`
	CloudProvider         types.String `tfsdk:"cloud_provider"`
	ProjectId             types.String `tfsdk:"project_id"`
	ConnectionUrl         types.String `tfsdk:"connection_url"`
	Username              types.String `tfsdk:"username"`
	Password              types.String `tfsdk:"password"`
	Version               types.String `tfsdk:"version"`
	Storage               types.String `tfsdk:"storage"`
	Status                types.String `tfsdk:"status"`
	CreatedAt             types.String `tfsdk:"created_at"`
	MetricsIntegrationUrl types.String `tfsdk:"metrics_integration_url"`
	SecondariesCount      types.Int32  `tfsdk:"secondaries_count"`
	CdcEnrichmentMode     types.String `tfsdk:"cdc_enrichment_mode"`
	VectorOptimized       types.Bool   `tfsdk:"vector_optimized"`
	GraphAnalyticsPlugin  types.Bool   `tfsdk:"graph_analytics_plugin"`

	Source types.Object `tfsdk:"source"`
}

type InstanceResourceSourceModel struct {
	InstanceId types.String `tfsdk:"instance_id"`
	SnapshotId types.String `tfsdk:"snapshot_id"`
}

func (m InstanceResourceModel) CanBePaused() bool {
	return !m.Status.IsUnknown() &&
		!m.Status.IsNull() &&
		strings.ToLower(m.Status.ValueString()) == domain.InstanceStatusRunning
}

func (m InstanceResourceModel) CanBeResumed() bool {
	return !m.Status.IsUnknown() &&
		!m.Status.IsNull() &&
		strings.ToLower(m.Status.ValueString()) == domain.InstanceStatusPaused
}

func (r *InstanceResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_instance"
}

func (r *InstanceResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

func (r *InstanceResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Aura Instance",
		Description:         "Aura Instance",
		Attributes: map[string]schema.Attribute{
			"instance_id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the instance.",
				Description:         "The unique identifier of the instance.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the instance.",
				Description:         "The name of the instance.",
				Required:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "The region where the instance is located.",
				Description:         "The region where the instance is located.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"memory": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Memory allocated for the instance. One of: `%s`.", strings.Join(supportedMemory, "`, `")),
				Description:         fmt.Sprintf("Memory allocated for the instance. One of [%s]", strings.Join(supportedMemory, ", ")),
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("1GB"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(supportedMemory...),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Type of the instance. Depend on your project configuration. One of: `%s`.", strings.Join(supportedTypes, "`, `")),
				Description:         fmt.Sprintf("Type of the instance. Depend on your project configuration. One of [%s]", strings.Join(supportedTypes, ", ")),
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("free-db"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
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
				Default:             stringdefault.StaticString("gcp"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(supportedCloudProviders...),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the project.",
				Description:         "The unique identifier of the project.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"connection_url": schema.StringAttribute{
				MarkdownDescription: "The Bolt connection URL for the instance database.",
				Description:         "The Bolt connection URL for the instance database.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "The username for the instance database.",
				Description:         "The username for the instance database.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "The password for the instance database.",
				Description:         "The password for the instance database.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Version of Neo4j. One of: `%s`.", strings.Join(supportedVersions, "`, `")),
				Description:         fmt.Sprintf("Version of Neo4j. One of [%s]", strings.Join(supportedVersions, ", ")),
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("5"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(supportedVersions...),
				},
			},
			"storage": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("The storage allocated to the instance. One of: `%s`.", strings.Join(supportedStorage, "`, `")),
				Description:         fmt.Sprintf("The storage allocated to the instance. One of [%s]", strings.Join(supportedStorage, ", ")),
				Computed:            true,
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(supportedStorage...),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("The status of the instance. One of: `%s`.", strings.Join(supportedStatuses, "`, `")),
				Description:         fmt.Sprintf("The status of the instance. One of [%s]", strings.Join(supportedStatuses, ", ")),
				Computed:            true,
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(supportedStatuses...),
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The timestamp when the instance was created.",
				Description:         "The timestamp when the instance was created.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"metrics_integration_url": schema.StringAttribute{
				MarkdownDescription: "The endpoint URL for metrics integration.",
				Description:         "The endpoint URL for metrics integration.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"secondaries_count": schema.Int32Attribute{
				MarkdownDescription: "The number of secondaries in the instance (VDC only).",
				Description:         "The number of secondaries in the instance (VDC only).",
				Optional:            true,
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.UseStateForUnknown(),
				},
			},
			"cdc_enrichment_mode": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("CDC enrichment mode. One of: `%s`.", strings.Join(supportedCdcEnrichmentModes, "`, `")),
				Description:         fmt.Sprintf("CDC enrichment mode. One of [%s]", strings.Join(supportedCdcEnrichmentModes, ", ")),
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(supportedCdcEnrichmentModes...),
				},
			},
			"vector_optimized": schema.BoolAttribute{
				MarkdownDescription: "The vector optimization configuration of the instance",
				Description:         "The vector optimization configuration of the instance",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"graph_analytics_plugin": schema.BoolAttribute{
				MarkdownDescription: "The graph analytics plugin configuration of the instance.",
				Description:         "The graph analytics plugin configuration of the instance.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"source": schema.SingleNestedAttribute{
				MarkdownDescription: "The source from which the instance is created.",
				Description:         "The source from which the instance is created.",
				Optional:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"instance_id": schema.StringAttribute{
						MarkdownDescription: "The unique identifier of the source instance.",
						Description:         "The unique identifier of the source instance.",
						Required:            true,
					},
					"snapshot_id": schema.StringAttribute{
						MarkdownDescription: "The unique identifier of the snapshot from the source instance.",
						Description:         "The unique identifier of the snapshot from the source instance.",
						Optional:            true,
					},
				},
			},
		},
	}
}

// ConfigValidators returns a list of resource-level validators
func (r *InstanceResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		&cdcTierValidator{},
		&vectorOptimizedValidator{},
		&graphAnalyticsPluginValidator{},
	}
}

func (r *InstanceResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data InstanceResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	postInstanceRequest := &client.PostInstanceRequest{
		Version:       data.Version.ValueString(),
		Region:        data.Region.ValueString(),
		Memory:        data.Memory.ValueString(),
		Name:          data.Name.ValueString(),
		Type:          data.Type.ValueString(),
		TenantId:      data.ProjectId.ValueString(),
		CloudProvider: data.CloudProvider.ValueString(),
	}
	if !data.Source.IsNull() {
		var sourceData InstanceResourceSourceModel
		response.Diagnostics.Append(data.Source.As(ctx, &sourceData, basetypes.ObjectAsOptions{})...)
		if response.Diagnostics.HasError() {
			return
		}
		postInstanceRequest.SourceInstanceId = sourceData.InstanceId.ValueStringPointer()
		if !sourceData.SnapshotId.IsNull() {
			_, err := r.auraApi.WaitUntilSnapshotIsInState(ctx, sourceData.InstanceId.ValueString(), sourceData.SnapshotId.ValueString(),
				func(resp client.GetSnapshotData) bool {
					return strings.EqualFold(resp.Status, domain.SnapshotStatusCompleted)
				})
			if err != nil {
				response.Diagnostics.AddError("Error while waiting snapshot to be completed",
					fmt.Sprintf("instance_id=%s snapshot_id=%s: %s", sourceData.InstanceId.ValueString(), sourceData.SnapshotId.ValueString(), err.Error()))
				return
			}
			postInstanceRequest.SourceSnapshotId = sourceData.SnapshotId.ValueStringPointer()
		}
	}
	if !data.Storage.IsUnknown() {
		postInstanceRequest.Storage = data.Storage.ValueStringPointer()
	}
	// Note: secondaries count cannot be set during instance creation
	// It must be set via PATCH after the instance is running (see below)
	// Note: CDC enrichment mode cannot be set during instance creation
	// It must be set via PATCH after the instance is running (see below)
	if !data.VectorOptimized.IsUnknown() {
		postInstanceRequest.VectorOptimized = data.VectorOptimized.ValueBoolPointer()
	}
	if !data.GraphAnalyticsPlugin.IsUnknown() {
		postInstanceRequest.GraphAnalyticsPlugin = data.GraphAnalyticsPlugin.ValueBoolPointer()
	}

	postInstanceResp, err := r.auraApi.PostInstance(ctx, *postInstanceRequest)
	if err != nil {
		response.Diagnostics.AddError("Error while creating an instance", err.Error())
		return
	}

	data.InstanceId = types.StringValue(postInstanceResp.Data.Id)
	data.ConnectionUrl = types.StringValue(postInstanceResp.Data.ConnectionUrl)
	data.Username = types.StringValue(postInstanceResp.Data.Username)
	data.Password = types.StringValue(postInstanceResp.Data.Password)

	tflog.Debug(ctx, "Created an instance with id "+postInstanceResp.Data.Id)

	instance, err := r.auraApi.WaitUntilInstanceIsInState(ctx, postInstanceResp.Data.Id, func(r client.GetInstanceResponse) bool {
		return strings.ToLower(r.Data.Status) == domain.InstanceStatusRunning
	})
	if err != nil {
		response.Diagnostics.AddError("Instance is not running in time",
			fmt.Sprintf("instance_id=%s: %s", postInstanceResp.Data.Id, err.Error()))
		return
	}

	// CDC enrichment mode and secondaries_count must be set via PATCH after instance creation
	// (POST /instances may not apply or return these; PATCH after the instance is running applies them)
	instanceType := data.Type.ValueString()
	needsCdcPatch := !data.CdcEnrichmentMode.IsNull() && (instanceType == domain.InstanceTypeBusinessCritical || instanceType == domain.InstanceTypeEnterpriseDb || instanceType == domain.InstanceTypeEnterpriseDs)
	needsSecondariesPatch := !data.SecondariesCount.IsNull() && (instanceType == domain.InstanceTypeBusinessCritical || instanceType == domain.InstanceTypeEnterpriseDb)
	if needsCdcPatch || needsSecondariesPatch {
		patchRequest := client.PatchInstanceRequest{}
		if needsCdcPatch {
			cdcMode := data.CdcEnrichmentMode.ValueString()
			patchRequest.CdcEnrichmentMode = &cdcMode
			tflog.Debug(ctx, fmt.Sprintf("Patching instance %s with CDC enrichment mode: %s", postInstanceResp.Data.Id, cdcMode))
		}
		if needsSecondariesPatch {
			sc := data.SecondariesCount.ValueInt32()
			patchRequest.SecondariesCount = &sc
			tflog.Debug(ctx, fmt.Sprintf("Patching instance %s with secondaries_count: %d", postInstanceResp.Data.Id, sc))
		}
		instance, err = r.auraApi.PatchInstanceById(ctx, postInstanceResp.Data.Id, patchRequest)
		if err != nil {
			response.Diagnostics.AddError("Error while patching instance (CDC / secondaries_count)",
				fmt.Sprintf("instance_id=%s: %s", postInstanceResp.Data.Id, err.Error()))
			return
		}
		instance, err = r.auraApi.WaitUntilInstanceIsInState(ctx, postInstanceResp.Data.Id, func(r client.GetInstanceResponse) bool {
			return strings.ToLower(r.Data.Status) == domain.InstanceStatusRunning
		})
		if err != nil {
			response.Diagnostics.AddError("Instance is not running after PATCH",
				fmt.Sprintf("instance_id=%s: %s", postInstanceResp.Data.Id, err.Error()))
			return
		}
		tflog.Debug(ctx, fmt.Sprintf("Successfully patched instance %s", postInstanceResp.Data.Id))
	}

	if instance.Data.Storage != nil {
		data.Storage = types.StringValue(*instance.Data.Storage)
	} else {
		data.Storage = types.StringNull()
	}
	if instance.Data.CreatedAt != nil {
		data.CreatedAt = types.StringValue(*instance.Data.CreatedAt)
	} else {
		data.CreatedAt = types.StringNull()
	}
	if instance.Data.MetricsIntegrationUrl != nil {
		data.MetricsIntegrationUrl = types.StringValue(*instance.Data.MetricsIntegrationUrl)
	} else {
		data.MetricsIntegrationUrl = types.StringNull()
	}
	if instance.Data.SecondariesCount != nil {
		data.SecondariesCount = types.Int32Value(int32(*instance.Data.SecondariesCount))
	} else if !data.SecondariesCount.IsNull() {
		// API may omit secondaries_count in response; keep the value we set via PATCH
		// data.SecondariesCount already has the correct planned value
	} else {
		data.SecondariesCount = types.Int32Null()
	}
	if instance.Data.CdcEnrichmentMode != nil {
		data.CdcEnrichmentMode = types.StringValue(*instance.Data.CdcEnrichmentMode)
	} else if !data.CdcEnrichmentMode.IsNull() {
		// API returns null for CDC enrichment mode on business-critical tier
		// Keep the planned value since we applied it via PATCH
		// data.CdcEnrichmentMode already has the correct planned value
	} else {
		// No planned value and API returns null - leave as null
		data.CdcEnrichmentMode = types.StringNull()
	}
	if instance.Data.VectorOptimized != nil {
		data.VectorOptimized = types.BoolValue(*instance.Data.VectorOptimized)
	} else {
		data.VectorOptimized = types.BoolNull()
	}
	if instance.Data.GraphAnalyticsPlugin != nil {
		data.GraphAnalyticsPlugin = types.BoolValue(*instance.Data.GraphAnalyticsPlugin)
	} else {
		data.GraphAnalyticsPlugin = types.BoolNull()
	}

	requestedStatus := data.Status
	data.Status = types.StringValue(instance.Data.Status)

	tflog.Debug(ctx, fmt.Sprintf("Instance %s is running", postInstanceResp.Data.Id))

	// Pausing a new instance
	if strings.ToLower(requestedStatus.ValueString()) == domain.InstanceStatusPaused {
		diagError := r.pauseInstance(ctx, data.InstanceId.ValueString())
		if diagError.IsNotEmpty() {
			response.Diagnostics.AddError(diagError.Message, diagError.Details)
			return
		}
		data.Status = requestedStatus
	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}
}

func (r *InstanceResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var stateData InstanceResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &stateData)...)

	if response.Diagnostics.HasError() {
		return
	}

	instance, err := r.auraApi.GetInstanceById(ctx, stateData.InstanceId.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Error while getting instance details",
			fmt.Sprintf("instance_id=%s: %s", stateData.InstanceId.ValueString(), err.Error()))
		return
	}

	stateData.Name = types.StringValue(instance.Data.Name)
	stateData.Region = types.StringValue(instance.Data.Region)
	stateData.Memory = types.StringValue(instance.Data.Memory)
	stateData.Type = types.StringValue(instance.Data.Type)
	stateData.CloudProvider = types.StringValue(instance.Data.CloudProvider)

	if instance.Data.ConnectionUrl != "" {
		stateData.ConnectionUrl = types.StringValue(instance.Data.ConnectionUrl)
	}
	if instance.Data.Storage != nil {
		stateData.Storage = types.StringValue(*instance.Data.Storage)
	} else {
		stateData.Storage = types.StringNull()
	}
	stateData.Status = types.StringValue(instance.Data.Status)
	if instance.Data.CreatedAt != nil {
		stateData.CreatedAt = types.StringValue(*instance.Data.CreatedAt)
	} else {
		stateData.CreatedAt = types.StringNull()
	}
	if instance.Data.MetricsIntegrationUrl != nil {
		stateData.MetricsIntegrationUrl = types.StringValue(*instance.Data.MetricsIntegrationUrl)
	} else {
		stateData.MetricsIntegrationUrl = types.StringNull()
	}
	if instance.Data.SecondariesCount != nil {
		stateData.SecondariesCount = types.Int32Value(int32(*instance.Data.SecondariesCount))
	} else if !stateData.SecondariesCount.IsNull() {
		// API may omit secondaries_count in response; keep existing state value set via PATCH
		// stateData.SecondariesCount already has the correct state value
	} else {
		stateData.SecondariesCount = types.Int32Null()
	}
	if instance.Data.CdcEnrichmentMode != nil {
		stateData.CdcEnrichmentMode = types.StringValue(*instance.Data.CdcEnrichmentMode)
	} else if !stateData.CdcEnrichmentMode.IsNull() {
		// API returns null for CDC enrichment mode on business-critical tier
		// Keep the existing state value since it was set via PATCH
		// stateData.CdcEnrichmentMode already has the correct state value
	} else {
		// No state value and API returns null - leave as null
		stateData.CdcEnrichmentMode = types.StringNull()
	}
	if instance.Data.VectorOptimized != nil {
		stateData.VectorOptimized = types.BoolValue(*instance.Data.VectorOptimized)
	} else {
		stateData.VectorOptimized = types.BoolNull()
	}
	if instance.Data.GraphAnalyticsPlugin != nil {
		stateData.GraphAnalyticsPlugin = types.BoolValue(*instance.Data.GraphAnalyticsPlugin)
	} else {
		stateData.GraphAnalyticsPlugin = types.BoolNull()
	}

	response.Diagnostics.Append(response.State.Set(ctx, &stateData)...)
	if response.Diagnostics.HasError() {
		return
	}
}

func (r *InstanceResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan InstanceResourceModel
	var state InstanceResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	// Resume
	if strings.ToLower(plan.Status.ValueString()) == domain.InstanceStatusRunning && state.CanBeResumed() {
		diagError := r.resumeInstance(ctx, state.InstanceId.ValueString())
		if diagError.IsNotEmpty() {
			response.Diagnostics.AddError(diagError.Message, diagError.Details)
			return
		}
	}

	// Regular inplace update
	planNameChanged := !plan.Name.Equal(state.Name)
	planMemoryChanged := !plan.Memory.Equal(state.Memory)
	planStorageChanged := !plan.Storage.Equal(state.Storage)
	planCdcEnrichmentModeChanged := !plan.CdcEnrichmentMode.Equal(state.CdcEnrichmentMode)
	planSecondariesChanged := !plan.SecondariesCount.Equal(state.SecondariesCount)
	planVectorOptimizedChanged := !plan.VectorOptimized.Equal(state.VectorOptimized)
	planGraphAnalyticsPluginChanged := !plan.GraphAnalyticsPlugin.Equal(state.GraphAnalyticsPlugin)
	if planNameChanged ||
		planMemoryChanged ||
		planStorageChanged ||
		planCdcEnrichmentModeChanged ||
		planVectorOptimizedChanged ||
		planGraphAnalyticsPluginChanged ||
		planSecondariesChanged {

		tflog.Debug(ctx, fmt.Sprintf("Updating instance details: Name: %s -> %s. Memory: %s -> %s. Storage: %s -> %s. CdcEnrichmentMode: %s -> %s. SecondariesCount: %v -> %v. VectorOptimized: %t -> %t. GraphAnalyticsPlugin: %t -> %t",
			state.Name.ValueString(), plan.Name.ValueString(), state.Memory.ValueString(), plan.Memory.ValueString(),
			state.Storage.ValueString(), plan.Storage.ValueString(), state.CdcEnrichmentMode.ValueString(), plan.CdcEnrichmentMode.ValueString(),
			state.SecondariesCount.ValueInt32(), plan.SecondariesCount.ValueInt32(), state.VectorOptimized.ValueBool(), plan.VectorOptimized.ValueBool(),
			state.GraphAnalyticsPlugin.ValueBool(), plan.GraphAnalyticsPlugin.ValueBool()))

		patchRequest := client.PatchInstanceRequest{}
		if planNameChanged {
			patchRequest.Name = plan.Name.ValueStringPointer()
		}
		if planMemoryChanged {
			patchRequest.Memory = plan.Memory.ValueStringPointer()
		}
		if planStorageChanged {
			patchRequest.Storage = plan.Storage.ValueStringPointer()
		}
		if planCdcEnrichmentModeChanged {
			patchRequest.CdcEnrichmentMode = plan.CdcEnrichmentMode.ValueStringPointer()
		}
		if planVectorOptimizedChanged {
			patchRequest.VectorOptimized = plan.VectorOptimized.ValueBoolPointer()
		}
		if planGraphAnalyticsPluginChanged {
			patchRequest.GraphAnalyticsPlugin = plan.GraphAnalyticsPlugin.ValueBoolPointer()
		}
		if planSecondariesChanged {
			patchRequest.SecondariesCount = plan.SecondariesCount.ValueInt32Pointer()
		}
		_, err := r.auraApi.PatchInstanceById(ctx, state.InstanceId.ValueString(), patchRequest)
		if err != nil {
			response.Diagnostics.AddError("Error while updating the instance details", fmt.Sprintf("instance_id=%s: %s",
				state.InstanceId.ValueString(), err.Error()))
			return
		}

		_, err = r.auraApi.WaitUntilInstanceIsInState(ctx, plan.InstanceId.ValueString(), func(resp client.GetInstanceResponse) bool {
			status := strings.ToLower(resp.Data.Status)
			if status != domain.InstanceStatusRunning && status != domain.InstanceStatusPaused {
				return false
			}
			if resp.Data.Memory != plan.Memory.ValueString() || resp.Data.Name != plan.Name.ValueString() {
				return false
			}
			if planStorageChanged && !stringPtrMatchesValue(resp.Data.Storage, plan.Storage) {
				return false
			}
			if planCdcEnrichmentModeChanged && resp.Data.CdcEnrichmentMode != nil && !stringPtrMatchesValue(resp.Data.CdcEnrichmentMode, plan.CdcEnrichmentMode) {
				return false
			}
			if planVectorOptimizedChanged && !boolPtrMatchesValue(resp.Data.VectorOptimized, plan.VectorOptimized) {
				return false
			}
			if planGraphAnalyticsPluginChanged && !boolPtrMatchesValue(resp.Data.GraphAnalyticsPlugin, plan.GraphAnalyticsPlugin) {
				return false
			}
			return true
		})
		if err != nil {
			response.Diagnostics.AddError("Error while waiting for the instance details to be updated",
				fmt.Sprintf("instance_id=%s: %s", plan.InstanceId.ValueString(), err.Error()))
			return
		}
	}

	// Pause
	if strings.ToLower(plan.Status.ValueString()) == domain.InstanceStatusPaused && state.CanBePaused() {
		diagError := r.pauseInstance(ctx, state.InstanceId.ValueString())
		if diagError.IsNotEmpty() {
			response.Diagnostics.AddError(diagError.Message, diagError.Details)
			return
		}
	}

	// Refresh state from the API after all updates so computed fields reflect the actual API response.
	instanceId := plan.InstanceId.ValueString()
	updatedInstance, err := r.auraApi.GetInstanceById(ctx, instanceId)
	if err != nil {
		response.Diagnostics.AddError(
			"Error reading instance after update",
			fmt.Sprintf("instance_id=%s: %s", instanceId, err.Error()),
		)
		return
	}

	plan.Name = types.StringValue(updatedInstance.Data.Name)
	plan.Region = types.StringValue(updatedInstance.Data.Region)
	plan.Memory = types.StringValue(updatedInstance.Data.Memory)
	plan.Type = types.StringValue(updatedInstance.Data.Type)
	plan.CloudProvider = types.StringValue(updatedInstance.Data.CloudProvider)
	// Aura API returns connection_url="" for paused instances; preserve the prior state
	// value so Terraform does not see a spurious diff against the planned (UseStateForUnknown) value.
	if updatedInstance.Data.ConnectionUrl != "" {
		plan.ConnectionUrl = types.StringValue(updatedInstance.Data.ConnectionUrl)
	} else {
		plan.ConnectionUrl = state.ConnectionUrl
	}
	plan.Status = types.StringValue(updatedInstance.Data.Status)
	if updatedInstance.Data.Storage != nil {
		plan.Storage = types.StringValue(*updatedInstance.Data.Storage)
	} else {
		plan.Storage = types.StringNull()
	}
	if updatedInstance.Data.CreatedAt != nil {
		plan.CreatedAt = types.StringValue(*updatedInstance.Data.CreatedAt)
	} else {
		plan.CreatedAt = types.StringNull()
	}
	if updatedInstance.Data.MetricsIntegrationUrl != nil {
		plan.MetricsIntegrationUrl = types.StringValue(*updatedInstance.Data.MetricsIntegrationUrl)
	} else {
		plan.MetricsIntegrationUrl = types.StringNull()
	}
	if updatedInstance.Data.SecondariesCount != nil {
		plan.SecondariesCount = types.Int32Value(int32(*updatedInstance.Data.SecondariesCount))
	} else if !plan.SecondariesCount.IsNull() {
		// API may omit secondaries_count; keep the planned value
	} else {
		plan.SecondariesCount = types.Int32Null()
	}
	if updatedInstance.Data.CdcEnrichmentMode != nil {
		plan.CdcEnrichmentMode = types.StringValue(*updatedInstance.Data.CdcEnrichmentMode)
	} else if !plan.CdcEnrichmentMode.IsNull() {
		// API returns null for CDC enrichment mode on some tiers; keep the planned value
	} else {
		plan.CdcEnrichmentMode = types.StringNull()
	}
	if updatedInstance.Data.VectorOptimized != nil {
		plan.VectorOptimized = types.BoolValue(*updatedInstance.Data.VectorOptimized)
	} else {
		plan.VectorOptimized = types.BoolNull()
	}
	if updatedInstance.Data.GraphAnalyticsPlugin != nil {
		plan.GraphAnalyticsPlugin = types.BoolValue(*updatedInstance.Data.GraphAnalyticsPlugin)
	} else {
		plan.GraphAnalyticsPlugin = types.BoolNull()
	}

	response.Diagnostics.Append(response.State.Set(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}
}

func stringPtrMatchesValue(actual *string, expected types.String) bool {
	if expected.IsUnknown() {
		return true
	}
	if expected.IsNull() {
		return actual == nil
	}
	return actual != nil && *actual == expected.ValueString()
}

func boolPtrMatchesValue(actual *bool, expected types.Bool) bool {
	if expected.IsUnknown() {
		return true
	}
	if expected.IsNull() {
		return actual == nil
	}
	return actual != nil && *actual == expected.ValueBool()
}

func (r *InstanceResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data InstanceResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	_, err := r.auraApi.DeleteInstanceById(ctx, data.InstanceId.ValueString())

	if err != nil {
		// If the instance is already gone, treat as success (idempotent delete).
		if errors.Is(err, client.ErrNotFound) {
			return
		}
		response.Diagnostics.AddError("Error while deleting an instance",
			fmt.Sprintf("instance_id=%s: %s", data.InstanceId.ValueString(), err.Error()))
		return
	}
	err = r.auraApi.WaitUntilInstanceIsDeleted(ctx, data.InstanceId.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Error while waiting for deleting an instance",
			fmt.Sprintf("instance_id=%s: %s", data.InstanceId.ValueString(), err.Error()))
	}
}

func (r *InstanceResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("instance_id"), request, response)
}

func (r *InstanceResource) resumeInstance(ctx context.Context, id string) util.DiagnosticsError {
	_, err := r.auraApi.ResumeInstanceById(ctx, id)
	if err != nil {
		return util.NewDiagnosticsError("Error while resume the instance", err.Error())
	}
	_, err = r.auraApi.WaitUntilInstanceIsInState(ctx, id, func(resp client.GetInstanceResponse) bool {
		return strings.ToLower(resp.Data.Status) == domain.InstanceStatusRunning
	})
	if err != nil {
		return util.NewDiagnosticsError("Error while waiting instance to be resumed", err.Error())
	}
	return util.NoDiagnosticsError()
}

func (r *InstanceResource) pauseInstance(ctx context.Context, id string) util.DiagnosticsError {
	_, err := r.auraApi.PauseInstanceById(ctx, id)
	if err != nil {
		return util.NewDiagnosticsError("Error while pausing the instance", err.Error())
	}
	_, err = r.auraApi.WaitUntilInstanceIsInState(ctx, id, func(resp client.GetInstanceResponse) bool {
		return strings.ToLower(resp.Data.Status) == domain.InstanceStatusPaused
	})
	if err != nil {
		return util.NewDiagnosticsError("Error while waiting for instance to be paused", err.Error())
	}
	return util.NoDiagnosticsError()
}
