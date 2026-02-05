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

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/domain"
)

var supportedStatuses = []string{
	domain.InstanceStatusCreating,
	domain.InstanceStatusDestroying,
	domain.InstanceStatusRunning,
	domain.InstanceStatusPausing,
	domain.InstanceStatusPaused,
	domain.InstanceStatusSuspending,
	domain.InstanceStatusSuspended,
	domain.InstanceStatusResuming,
	domain.InstanceStatusLoading,
	domain.InstanceStatusLoadingFailed,
	domain.InstanceStatusRestoring,
	domain.InstanceStatusUpdating,
	domain.InstanceStatusOverwriting,
}

var supportedMemory = []string{
	domain.InstanceMemory1GB, domain.InstanceMemory2GB, domain.InstanceMemory4GB, domain.InstanceMemory8GB,
	domain.InstanceMemory16GB, domain.InstanceMemory24GB, domain.InstanceMemory32GB, domain.InstanceMemory48GB,
	domain.InstanceMemory64GB, domain.InstanceMemory128GB, domain.InstanceMemory192GB, domain.InstanceMemory256GB,
	domain.InstanceMemory384GB, domain.InstanceMemory512GB,
}
var supportedTypes = []string{
	domain.InstanceTypeEnterpriseDb, domain.InstanceTypeEnterpriseDs, domain.InstanceTypeProfessionalDb,
	domain.InstanceTypeProfessionalDs, domain.InstanceTypeFreeDb, domain.InstanceTypeBusinessCritical,
}
var supportedCloudProviders = []string{domain.CloudProviderGcp, domain.CloudProviderAws, domain.CloudProviderAzure}
var supportedVersions = []string{domain.InstanceVersion5}
var supportedStorage = []string{
	domain.InstanceStorage2GB, domain.InstanceStorage4GB, domain.InstanceStorage8GB, domain.InstanceStorage16GB,
	domain.InstanceStorage32GB, domain.InstanceStorage48GB, domain.InstanceStorage64GB, domain.InstanceStorage96GB,
	domain.InstanceStorage128GB, domain.InstanceStorage192GB, domain.InstanceStorage256GB, domain.InstanceStorage384GB,
	domain.InstanceStorage512GB, domain.InstanceStorage768GB, domain.InstanceStorage1024GB, domain.InstanceStorage1536GB,
	domain.InstanceStorage2048GB,
}
var supportedCdcEnrichmentModes = []string{domain.CdcEnrichmentModeOff, domain.CdcEnrichmentModeDiff, domain.CdcEnrichmentModeFull}

var (
	_ resource.ConfigValidator = &cdcTierValidator{}
	_ resource.ConfigValidator = &vectorOptimizedValidator{}
	_ resource.ConfigValidator = &graphAnalyticsPluginValidator{}
)

// cdcTierValidator validates that CDC enrichment mode is only used with supported tiers
type cdcTierValidator struct{}

func (v *cdcTierValidator) Description(_ context.Context) string {
	return "CDC enrichment mode is only supported on business-critical and enterprise instance types"
}

func (v *cdcTierValidator) MarkdownDescription(_ context.Context) string {
	return "CDC enrichment mode is only supported on `business-critical` and `enterprise` instance types"
}

func (v *cdcTierValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data InstanceResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	v.validateCdcTier(data, resp)
}

func (v *cdcTierValidator) validateCdcTier(data InstanceResourceModel, resp *resource.ValidateConfigResponse) {
	// If CDC enrichment mode is not set, no validation needed
	if data.CdcEnrichmentMode.IsNull() || data.CdcEnrichmentMode.IsUnknown() {
		return
	}

	// If type is not set, we can't validate yet
	if data.Type.IsNull() || data.Type.IsUnknown() {
		return
	}

	instanceType := data.Type.ValueString()
	if instanceType != domain.InstanceTypeBusinessCritical && instanceType != domain.InstanceTypeEnterpriseDb && instanceType != domain.InstanceTypeEnterpriseDs {
		resp.Diagnostics.AddAttributeError(
			path.Root("cdc_enrichment_mode"),
			"Invalid Configuration",
			fmt.Sprintf("CDC enrichment mode is only supported on business-critical and enterprise instance types. Instance type '%s' does not support CDC.", instanceType),
		)
	}
}

// vectorOptimizedValidator validates that vector optimization can only be applied to instance of a certain size
type vectorOptimizedValidator struct{}

func (v *vectorOptimizedValidator) Description(_ context.Context) string {
	return "Vector optimization can only be applied to instances with 4GB memory or greater."
}

func (v *vectorOptimizedValidator) MarkdownDescription(_ context.Context) string {
	return "Vector optimization can only be applied to instances with `4GB` memory or greater."
}

func (v *vectorOptimizedValidator) ValidateResource(ctx context.Context, request resource.ValidateConfigRequest, response *resource.ValidateConfigResponse) {
	var data InstanceResourceModel
	response.Diagnostics.Append(request.Config.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	v.validateVectorOptimized(data, response)
}

func (v *vectorOptimizedValidator) validateVectorOptimized(data InstanceResourceModel, response *resource.ValidateConfigResponse) {
	if data.VectorOptimized.IsNull() || data.VectorOptimized.IsUnknown() || !data.VectorOptimized.ValueBool() {
		return
	}

	if data.Memory.IsNull() || data.Memory.IsUnknown() {
		return
	}

	memory := data.Memory.ValueString()
	if memory == domain.InstanceMemory1GB || memory == domain.InstanceMemory2GB {
		response.Diagnostics.AddAttributeError(
			path.Root("vector_optimized"),
			"Invalid Configuration",
			fmt.Sprintf("Vector optimization is not supported for instances with %s memory.", memory),
		)
	}
}

type graphAnalyticsPluginValidator struct{}

func (v *graphAnalyticsPluginValidator) Description(_ context.Context) string {
	return "Graph Analytics Plugin can only be applied to Professional instances."
}

func (v *graphAnalyticsPluginValidator) MarkdownDescription(_ context.Context) string {
	return "Graph Analytics Plugin can only be applied to `Professional` instances."
}

func (v *graphAnalyticsPluginValidator) ValidateResource(ctx context.Context, request resource.ValidateConfigRequest, response *resource.ValidateConfigResponse) {
	var data InstanceResourceModel
	response.Diagnostics.Append(request.Config.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	v.validateGraphAnalyticsPlugin(data, response)
}

func (v *graphAnalyticsPluginValidator) validateGraphAnalyticsPlugin(data InstanceResourceModel, response *resource.ValidateConfigResponse) {
	if data.GraphAnalyticsPlugin.IsNull() || data.GraphAnalyticsPlugin.IsUnknown() || !data.GraphAnalyticsPlugin.ValueBool() {
		return
	}

	if data.Type.IsNull() || data.Type.IsUnknown() {
		return
	}

	instanceType := data.Type.ValueString()
	if instanceType != domain.InstanceTypeProfessionalDb {
		response.Diagnostics.AddAttributeError(
			path.Root("graph_analytics_plugin"),
			"Invalid Configuration",
			fmt.Sprintf("Graph Analytics Plugin is only supported on professional-db instance type. Instance type '%s' does not support Graph Analytics Plugin.", instanceType),
		)
	}
}
