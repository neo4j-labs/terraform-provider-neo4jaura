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
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/client"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/util"
)

// Ensure resource defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &InstanceResource{}
	_ resource.ResourceWithConfigure   = &InstanceResource{}
	_ resource.ResourceWithImportState = &InstanceResource{}
)

func NewInstanceResource() resource.Resource {
	return &InstanceResource{}
}

var supportedStatuses = []string{
	"creating", "destroying", "running", "pausing", "paused", "suspending", "suspended", "resuming", "loading",
	"loading failed", "restoring", "updating", "overwriting",
}
var supportedMemory = []string{
	"1GB", "2GB", "4GB", "8GB", "16GB", "24GB", "32GB", "48GB", "64GB", "128GB", "192GB", "256GB", "384GB", "512GB",
}
var supportedTypes = []string{
	"enterprise-db", "enterprise-ds", "professional-db", "professional-ds", "free-db", "business-critical",
}
var supportedCloudProviders = []string{"gcp", "aws", "azure"}
var supportedVersions = []string{"5"}
var supportedStorage = []string{
	"2GB", "4GB", "8GB", "16GB", "32GB", "48GB", "64GB", "96GB", "128GB", "192GB", "256GB", "384GB", "512GB",
	"768GB", "1024GB", "1536GB", "2048GB",
}
var supportedCdcEnrichmentModes = []string{"OFF", "DIFF", "FULL"}

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
	GraphNodes            types.Int64  `tfsdk:"graph_nodes"`
	GraphRelationships    types.Int64  `tfsdk:"graph_relationships"`
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
		strings.ToLower(m.Status.ValueString()) == "running"
}

func (m InstanceResourceModel) CanBeResumed() bool {
	return !m.Status.IsUnknown() &&
		!m.Status.IsNull() &&
		strings.ToLower(m.Status.ValueString()) == "paused"
}

func (r *InstanceResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_instance"
}

func (r *InstanceResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

func (r *InstanceResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Aura instance",
		Description:         "Aura instance",
		Attributes: map[string]schema.Attribute{
			"instance_id": schema.StringAttribute{
				MarkdownDescription: "Id of the instance",
				Description:         "Id of the instance",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the instance",
				Description:         "Name of the instance",
				Required:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "Region of the instance",
				Description:         "Region of the instance",
				Required:            true,
			},
			"memory": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Memory allocated for the instance. One of [%s]", strings.Join(supportedMemory, ",")),
				Description:         fmt.Sprintf("Memory allocated for the instance. One of [%s]", strings.Join(supportedMemory, ",")),
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
				MarkdownDescription: fmt.Sprintf("Type of the instance. Depend on your project configuration. One of [%s]", strings.Join(supportedTypes, ", ")),
				Description:         fmt.Sprintf("Type of the instance. Depend on your project configuration. One of [%s]", strings.Join(supportedTypes, ", ")),
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("free-db"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(supportedTypes...),
				},
			},
			"cloud_provider": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Cloud provider. One of [%s]", strings.Join(supportedCloudProviders, ", ")),
				Description:         fmt.Sprintf("Cloud provider. One of [%s]", strings.Join(supportedCloudProviders, ", ")),
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("gcp"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(supportedCloudProviders...),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Id of the project",
				Description:         "Id of the project",
				Required:            true,
			},
			"connection_url": schema.StringAttribute{
				MarkdownDescription: "Bolt connection URL to the instance database",
				Description:         "Bolt connection URL to the instance database",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Username of the instance database",
				Description:         "Username of the instance database",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Password of the instance database",
				Description:         "Password of the instance database",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Version of Neo4j. One of [%s]", strings.Join(supportedVersions, ", ")),
				Description:         fmt.Sprintf("Version of Neo4j. One of [%s]", strings.Join(supportedVersions, ", ")),
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("5"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(supportedVersions...),
				},
			},
			"storage": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Storage allocated to the instance. One of [%s]", strings.Join(supportedStorage, ", ")),
				Description:         fmt.Sprintf("Storage allocated to the instance. One of [%s]", strings.Join(supportedStorage, ", ")),
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
				MarkdownDescription: fmt.Sprintf("Status of the instance. One of [%s]", strings.Join(supportedStatuses, ", ")),
				Description:         fmt.Sprintf("Status of the instance. One of [%s]", strings.Join(supportedStatuses, ", ")),
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
				MarkdownDescription: "The timestamp when the instance was created",
				Description:         "The timestamp when the instance was created",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"metrics_integration_url": schema.StringAttribute{
				MarkdownDescription: "Metrics integration endpoint URL",
				Description:         "Metrics integration endpoint URL",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"graph_nodes": schema.Int64Attribute{
				MarkdownDescription: "Number of nodes in the graph (free-db only)",
				Description:         "Number of nodes in the graph (free-db only)",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"graph_relationships": schema.Int64Attribute{
				MarkdownDescription: "Number of relationships in the graph (only for free-db)",
				Description:         "Number of relationships in the graph (only for free-db)",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"secondaries_count": schema.Int32Attribute{
				MarkdownDescription: "The number of secondaries in an Instance. (VDC only)",
				Description:         "The number of secondaries in an Instance. (VDC only)",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.UseStateForUnknown(),
				},
			},
			"cdc_enrichment_mode": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("CDC enrichment mode. One of [%s]", strings.Join(supportedCdcEnrichmentModes, ", ")),
				Description:         fmt.Sprintf("CDC enrichment mode. One of [%s]", strings.Join(supportedCdcEnrichmentModes, ", ")),
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
				MarkdownDescription: "Information about source for the instance",
				Description:         "Information about source for the instance",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"instance_id": schema.StringAttribute{
						MarkdownDescription: "Instance Id that contains the source database of the instance",
						Description:         "Instance Id that contains the source database of the instance",
						Required:            true,
					},
					"snapshot_id": schema.StringAttribute{
						MarkdownDescription: "Snapshot Id of the instance containing the source database of the instance",
						Description:         "Snapshot Id of the instance containing the source database of the instance",
						Optional:            true,
					},
				},
			},
		},
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
					return strings.ToLower(resp.Status) == "completed"
				})
			if err != nil {
				response.Diagnostics.AddError("Error while waiting snapshot to be completed", err.Error())
			}
			postInstanceRequest.SourceSnapshotId = sourceData.SnapshotId.ValueStringPointer()
		}
	}
	if !data.Storage.IsUnknown() {
		postInstanceRequest.Storage = data.Storage.ValueStringPointer()
	}
	if !data.SecondariesCount.IsUnknown() {
		postInstanceRequest.SecondariesCount = data.SecondariesCount.ValueInt32Pointer()
	}
	if !data.CdcEnrichmentMode.IsUnknown() {
		postInstanceRequest.CdcEnrichmentMode = data.CdcEnrichmentMode.ValueStringPointer()
	}
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

	requestedStatus := data.Status

	data.InstanceId = types.StringValue(postInstanceResp.Data.Id)
	data.ConnectionUrl = types.StringValue(postInstanceResp.Data.ConnectionUrl)
	data.Username = types.StringValue(postInstanceResp.Data.Username)
	data.Password = types.StringValue(postInstanceResp.Data.Password)

	tflog.Debug(ctx, "Created an instance with id "+postInstanceResp.Data.Id)

	instance, err := r.auraApi.WaitUntilInstanceIsInState(ctx, postInstanceResp.Data.Id, func(r client.GetInstanceResponse) bool {
		return strings.ToLower(r.Data.Status) == "running"
	})
	if err != nil {
		response.Diagnostics.AddError("Instance is not running in time", err.Error())
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
	if instance.Data.GraphNodes != nil {
		graphNodes, err := strconv.ParseInt(*instance.Data.GraphNodes, 10, 64)
		if err != nil {
			response.Diagnostics.AddWarning(
				"Error while parsing graph nodes",
				fmt.Sprintf("Cannot convert value to int: %s", *instance.Data.GraphNodes),
			)
			data.GraphNodes = types.Int64Null()
		} else {
			data.GraphNodes = types.Int64Value(graphNodes)
		}
	} else {
		data.GraphNodes = types.Int64Null()
	}
	if instance.Data.GraphRelationships != nil {
		graphRelationships, err := strconv.ParseInt(*instance.Data.GraphRelationships, 10, 64)
		if err != nil {
			response.Diagnostics.AddWarning(
				"Error while parsing graph relationships",
				fmt.Sprintf("Cannot convert value to int: %s", *instance.Data.GraphNodes),
			)
			data.GraphRelationships = types.Int64Null()
		} else {
			data.GraphRelationships = types.Int64Value(graphRelationships)
		}
	} else {
		data.GraphRelationships = types.Int64Null()
	}
	if instance.Data.SecondariesCount != nil {
		data.SecondariesCount = types.Int32Value(int32(*instance.Data.SecondariesCount))
	} else {
		data.SecondariesCount = types.Int32Null()
	}
	if instance.Data.CdcEnrichmentMode != nil {
		data.CdcEnrichmentMode = types.StringValue(*instance.Data.CdcEnrichmentMode)
	} else {
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

	requestedStatus = data.Status
	data.Status = types.StringValue(data.Status.ValueString())

	tflog.Debug(ctx, fmt.Sprintf("Instance %s is running", postInstanceResp.Data.Id))

	// Pausing new instance
	if strings.ToLower(requestedStatus.ValueString()) == "paused" {
		diagError := r.pauseInstance(ctx, data.InstanceId.ValueString())
		if diagError.IsNotEmpty() {
			response.Diagnostics.AddError(diagError.Message, diagError.Details)
			return
		}
		data.Status = requestedStatus
	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *InstanceResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var stateData InstanceResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &stateData)...)

	if response.Diagnostics.HasError() {
		return
	}

	instance, err := r.auraApi.GetInstanceById(ctx, stateData.InstanceId.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Error while getting instance details", err.Error())
		return
	}

	stateData.Name = types.StringValue(instance.Data.Name)
	stateData.Region = types.StringValue(instance.Data.Region)
	stateData.Memory = types.StringValue(instance.Data.Memory)
	stateData.Type = types.StringValue(instance.Data.Type)
	stateData.CloudProvider = types.StringValue(instance.Data.CloudProvider)
	stateData.ConnectionUrl = types.StringValue(instance.Data.ConnectionUrl)
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
	if instance.Data.GraphNodes != nil {
		graphNodes, err := strconv.ParseInt(*instance.Data.GraphNodes, 10, 64)
		if err != nil {
			response.Diagnostics.AddWarning(
				"Error while parsing graph nodes",
				fmt.Sprintf("Cannot convert value to int: %s", *instance.Data.GraphNodes),
			)
			stateData.GraphNodes = types.Int64Null()
		}
		stateData.GraphNodes = types.Int64Value(graphNodes)
	} else {
		stateData.GraphNodes = types.Int64Null()
	}
	if instance.Data.GraphRelationships != nil {
		graphRelationships, err := strconv.ParseInt(*instance.Data.GraphRelationships, 10, 64)
		if err != nil {
			response.Diagnostics.AddWarning(
				"Error while parsing graph relationships",
				fmt.Sprintf("Cannot convert value to int: %s", *instance.Data.GraphNodes),
			)
			stateData.GraphRelationships = types.Int64Null()
		}
		stateData.GraphRelationships = types.Int64Value(graphRelationships)
	} else {
		stateData.GraphRelationships = types.Int64Null()
	}
	if instance.Data.SecondariesCount != nil {
		stateData.SecondariesCount = types.Int32Value(int32(*instance.Data.SecondariesCount))
	} else {
		stateData.SecondariesCount = types.Int32Null()
	}
	if instance.Data.CdcEnrichmentMode != nil {
		stateData.CdcEnrichmentMode = types.StringValue(*instance.Data.CdcEnrichmentMode)
	} else {
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
	if strings.ToLower(plan.Status.ValueString()) == "running" && state.CanBeResumed() {
		diagError := r.resumeInstance(ctx, state.InstanceId.ValueString())
		if diagError.IsNotEmpty() {
			response.Diagnostics.AddError(diagError.Message, diagError.Details)
			return
		}
	}

	// Regular inplace update
	if !plan.Name.Equal(state.Name) || !plan.Memory.Equal(state.Memory) {
		tflog.Debug(ctx, fmt.Sprintf("Updating instance details: Name: %s -> %s. Memory: %s -> %s",
			state.Name.ValueString(), plan.Name.ValueString(), state.Memory.ValueString(), plan.Memory.ValueString()))

		_, err := r.auraApi.PatchInstanceById(ctx, state.InstanceId.ValueString(), client.PatchInstanceRequest{
			Name:   plan.Name.ValueStringPointer(),
			Memory: plan.Memory.ValueStringPointer(),
		})

		if err != nil {
			response.Diagnostics.AddError("Error while updating the instance details", err.Error())
			return
		}

		_, err = r.auraApi.WaitUntilInstanceIsInState(ctx, plan.InstanceId.ValueString(), func(resp client.GetInstanceResponse) bool {
			return resp.Data.Memory == plan.Memory.ValueString() &&
				resp.Data.Name == plan.Name.ValueString() &&
				(strings.ToLower(resp.Data.Status) == "running" || strings.ToLower(resp.Data.Status) == "paused")
		})

		if err != nil {
			response.Diagnostics.AddError("Error while waiting fro the instance details to be updated", err.Error())
			return
		}
	}

	// Pause
	if strings.ToLower(plan.Status.ValueString()) == "paused" && state.CanBePaused() {
		diagError := r.pauseInstance(ctx, state.InstanceId.ValueString())
		if diagError.IsNotEmpty() {
			response.Diagnostics.AddError(diagError.Message, diagError.Details)
			return
		}
	}

	response.Diagnostics.Append(response.State.Set(ctx, &plan)...)
}

func (r *InstanceResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data InstanceResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	_, err := r.auraApi.DeleteInstanceById(ctx, data.InstanceId.ValueString())
	// todo should we wait until instance is deleted
	if err != nil {
		response.Diagnostics.AddError("Error while deleting an instance", err.Error())
	}
	err = r.auraApi.WaitUntilInstanceIsDeleted(ctx, data.InstanceId.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Error while waiting for deleting an instance", err.Error())
	}
}

func (r *InstanceResource) resumeInstance(ctx context.Context, id string) util.DiagnosticsError {
	_, err := r.auraApi.ResumeInstanceById(ctx, id)
	if err != nil {
		return util.NewDiagnosticsError("Error while resume the instance", err.Error())
	}
	_, err = r.auraApi.WaitUntilInstanceIsInState(ctx, id, func(resp client.GetInstanceResponse) bool {
		return strings.ToLower(resp.Data.Status) == "running"
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
		return strings.ToLower(resp.Data.Status) == "paused"
	})
	if err != nil {
		return util.NewDiagnosticsError("Error while waiting for instance to be paused", err.Error())
	}
	return util.NoDiagnosticsError()
}

func (r *InstanceResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("instance_id"), request, response)
}
