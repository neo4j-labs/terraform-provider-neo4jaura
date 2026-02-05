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
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/client"
)

var (
	_ resource.Resource                = &SnapshotResource{}
	_ resource.ResourceWithConfigure   = &SnapshotResource{}
	_ resource.ResourceWithImportState = &SnapshotResource{}
)

func NewSnapshotResource() resource.Resource {
	return &SnapshotResource{}
}

type SnapshotResource struct {
	auraApi *client.AuraApi
}

type SnapshotResourceModel struct {
	InstanceId types.String `tfsdk:"instance_id"`
	SnapshotId types.String `tfsdk:"snapshot_id"`
	Profile    types.String `tfsdk:"profile"`
	Status     types.String `tfsdk:"status"`
	Timestamp  types.String `tfsdk:"timestamp"`
}

func (r *SnapshotResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

func (r *SnapshotResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_snapshot"
}

func (r *SnapshotResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Resource for an instance snapshot",
		Description:         "Resource for an instance snapshot",
		Attributes: map[string]schema.Attribute{
			"instance_id": schema.StringAttribute{
				MarkdownDescription: "Id of the instance",
				Description:         "Id of the instance",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"snapshot_id": schema.StringAttribute{
				MarkdownDescription: "Id of the snapshot",
				Description:         "Id of the snapshot",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"profile": schema.StringAttribute{
				MarkdownDescription: "Profile of the snapshot. One of [AddHoc, Scheduled]",
				Description:         "Profile of the snapshot. One of [AddHoc, Scheduled]",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Status of the snapshot. One of [Completed, InProgress, Failed, Pending]",
				Description:         "Status of the snapshot. One of [Completed, InProgress, Failed, Pending]",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"timestamp": schema.StringAttribute{
				MarkdownDescription: "Timestamp of the snapshot",
				Description:         "Timestamp of the snapshot",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *SnapshotResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data SnapshotResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	postResponse, err := r.auraApi.PostSnapshot(ctx, data.InstanceId.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Error while creating a snapshot", err.Error())
		return
	}

	snapshot, err := r.auraApi.WaitUntilSnapshotIsInState(ctx, data.InstanceId.ValueString(), postResponse.Data.SnapshotId,
		func(resp client.GetSnapshotData) bool {
			return strings.ToLower(resp.Status) == "completed"
		})
	if err != nil {
		response.Diagnostics.AddError("Error while waiting for a snapshot", err.Error())
		return
	}

	data.SnapshotId = types.StringValue(snapshot.SnapshotId)
	data.Timestamp = types.StringValue(snapshot.Timestamp)
	data.Status = types.StringValue(snapshot.Status)
	data.Profile = types.StringValue(snapshot.Profile)

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *SnapshotResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data SnapshotResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	snapshotResponse, err := r.auraApi.GetSnapshotById(ctx, data.InstanceId.ValueString(), data.SnapshotId.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Error reading snapshot", err.Error())
		return
	}

	data.Timestamp = types.StringValue(snapshotResponse.Data.Timestamp)
	data.Status = types.StringValue(snapshotResponse.Data.Status)
	data.Profile = types.StringValue(snapshotResponse.Data.Profile)

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *SnapshotResource) Update(ctx context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	tflog.Info(ctx, "Snapshot resources are immutable and cannot be updated")
}

func (r *SnapshotResource) Delete(ctx context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	tflog.Info(ctx, "Snapshot resources are immutable and cannot be deleted")
}

func (r *SnapshotResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	idParts := strings.Split(request.ID, ",")

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		response.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: instance_id,snapshot_id. Got: %q", request.ID),
		)
		return
	}

	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("instance_id"), idParts[0])...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("snapshot_id"), idParts[1])...)
}
