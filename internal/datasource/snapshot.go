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
	"sort"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/client"
)

var (
	_ datasource.DataSource                     = &SnapshotDataSource{}
	_ datasource.DataSourceWithConfigure        = &SnapshotDataSource{}
	_ datasource.DataSourceWithConfigValidators = &SnapshotDataSource{}
)

func NewSnapshotDataSource() datasource.DataSource {
	return &SnapshotDataSource{}
}

type SnapshotDataSource struct {
	auraApi *client.AuraApi
}

type SnapshotDataSourceModel struct {
	InstanceId types.String `tfsdk:"instance_id"`
	SnapshotId types.String `tfsdk:"snapshot_id"`
	Profile    types.String `tfsdk:"profile"`
	Status     types.String `tfsdk:"status"`
	Timestamp  types.String `tfsdk:"timestamp"`
	MostRecent types.Bool   `tfsdk:"most_recent"`
}

func (ds *SnapshotDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_snapshot"
}

func (ds *SnapshotDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Snapshot of an instance",
		Description:         "Snapshot of an instance",
		Attributes: map[string]schema.Attribute{
			"instance_id": schema.StringAttribute{
				MarkdownDescription: "Id of the instance",
				Description:         "Id of the instance",
				Required:            true,
			},
			"snapshot_id": schema.StringAttribute{
				MarkdownDescription: "Id of the snapshot",
				Description:         "Id of the snapshot",
				Optional:            true,
				Computed:            true,
			},
			"profile": schema.StringAttribute{
				MarkdownDescription: "Profile of the snapshot. One of [AddHoc, Scheduled]",
				Description:         "Profile of the snapshot. One of [AddHoc, Scheduled]",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Status of the snapshot. One of [Completed, InProgress, Failed, Pending]",
				Description:         "Status of the snapshot. One of [Completed, InProgress, Failed, Pending]",
				Computed:            true,
			},
			"timestamp": schema.StringAttribute{
				MarkdownDescription: "Timestamp of the snapshot",
				Description:         "Timestamp of the snapshot",
				Computed:            true,
			},
			"most_recent": schema.BoolAttribute{
				MarkdownDescription: "Flag indicated if the most recent snapshot should be returned",
				Description:         "Flag indicated if the most recent snapshot should be returned",
				Optional:            true,
			},
		},
	}
}

func (ds *SnapshotDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.Conflicting(
			path.MatchRoot("snapshot_id"),
			path.MatchRoot("most_recent"),
		),
	}
}

func (ds *SnapshotDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var data SnapshotDataSourceModel

	response.Diagnostics.Append(request.Config.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	instanceId := data.InstanceId.ValueString()
	snapshots, err := ds.auraApi.GetSnapshotsByInstanceId(ctx, instanceId)
	if err != nil {
		response.Diagnostics.AddError("Error while reading instance snapshots", err.Error())
		return
	}

	if len(snapshots.Data) == 0 {
		response.Diagnostics.AddError("Cannot find snapshot", "There are no snapshots for instance "+instanceId)
	}

	var selected client.GetSnapshotData
	if !data.MostRecent.IsNull() && data.MostRecent.ValueBool() {
		sort.Slice(snapshots.Data, func(i1, i2 int) bool {
			layout := "2006-01-02T03:04:05Z"
			timestamp1, err := time.Parse(layout, snapshots.Data[i1].Timestamp)
			if err != nil {
				tflog.Error(ctx, "Fail to parse timestamp: "+snapshots.Data[i1].Timestamp)
				return true
			}
			timestamp2, err := time.Parse(layout, snapshots.Data[i2].Timestamp)
			if err != nil {
				tflog.Error(ctx, "Fail to parse timestamp: "+snapshots.Data[i2].Timestamp)
				return true
			}
			return timestamp1.Before(timestamp2)
		})
		selected = snapshots.Data[len(snapshots.Data)-1]
		data.SnapshotId = types.StringValue(selected.SnapshotId)
	} else {
		found := false
		for _, snapshot := range snapshots.Data {
			if snapshot.SnapshotId == data.SnapshotId.ValueString() {
				selected = snapshot
				found = true
				break
			}
		}
		if !found {
			response.Diagnostics.AddError("Cannot find snapshot",
				fmt.Sprintf("There is no snapshot for instance %s and snapshot id %s",
					instanceId, data.SnapshotId.ValueString()))
			return
		}
	}

	data.Status = types.StringValue(selected.Status)
	data.Profile = types.StringValue(selected.Profile)
	data.Timestamp = types.StringValue(selected.Timestamp)

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (ds *SnapshotDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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
