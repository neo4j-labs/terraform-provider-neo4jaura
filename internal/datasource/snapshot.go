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

	var snapshot *client.GetSnapshotData
	instanceId := data.InstanceId.ValueString()
	if !data.MostRecent.IsNull() && data.MostRecent.ValueBool() {
		snapshot = ds.readMostRecentSnapshot(ctx, instanceId, response)
	} else if !data.SnapshotId.IsNull() && data.SnapshotId.ValueString() != "" {
		snapshot = ds.readSnapshotById(ctx, data.InstanceId.ValueString(), data.SnapshotId.ValueString(), response)
	} else {
		response.Diagnostics.AddError("Provide either snapshot_id or most_recent",
			fmt.Errorf("missing required attribute: snapshot_id or most_recent").Error())
		return
	}

	if snapshot != nil {
		data.SnapshotId = types.StringValue(snapshot.SnapshotId)
		data.Status = types.StringValue(snapshot.Status)
		data.Profile = types.StringValue(snapshot.Profile)
		data.Timestamp = types.StringValue(snapshot.Timestamp)
	}

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

func (ds *SnapshotDataSource) readMostRecentSnapshot(ctx context.Context, instanceId string, response *datasource.ReadResponse) *client.GetSnapshotData {
	snapshots, err := ds.auraApi.GetSnapshotsByInstanceId(ctx, instanceId)
	if err != nil {
		response.Diagnostics.AddError("Error while reading instance snapshots", err.Error())
		return nil
	}
	if len(snapshots.Data) == 0 {
		isRecentlyCreated, err := ds.isInstanceRecentlyCreated(ctx, instanceId)
		if err != nil {
			response.Diagnostics.AddError("Cannot read instance "+instanceId, err.Error())
			return nil
		}
		if isRecentlyCreated {
			snapshots, err = ds.auraApi.WaitUntilSnapshotsMatchCondition(ctx, instanceId, func(data client.GetSnapshotsResponse) bool {
				return len(data.Data) > 0
			})
			if err != nil {
				response.Diagnostics.AddError("Cannot find snapshot for instance "+instanceId, err.Error())
				return nil
			}
		} else {
			response.Diagnostics.AddError("Cannot find snapshot", "There are no snapshots for instance "+instanceId)
			return nil
		}
	}
	tflog.Debug(ctx, fmt.Sprintf("Snapshots: %+v", snapshots.Data))
	sort.Slice(snapshots.Data, func(i1, i2 int) bool {
		timestamp1, err := snapshots.Data[i1].TimestampAsTime()
		if err != nil {
			tflog.Error(ctx, "Fail to parse timestamp: "+snapshots.Data[i1].Timestamp)
			return true
		}
		timestamp2, err := snapshots.Data[i2].TimestampAsTime()
		if err != nil {
			tflog.Error(ctx, "Fail to parse timestamp: "+snapshots.Data[i2].Timestamp)
			return true
		}
		return timestamp1.Before(timestamp2)
	})
	return &snapshots.Data[len(snapshots.Data)-1]
}

func (ds *SnapshotDataSource) isInstanceRecentlyCreated(ctx context.Context, instanceId string) (bool, error) {
	instance, err := ds.auraApi.GetInstanceById(ctx, instanceId)
	if err != nil {
		return false, err
	}
	createdAt, err := instance.Data.CreatedAtAsTime()
	if err != nil {
		return false, err
	}
	return createdAt.After(time.Now().Add(-time.Minute * 5)), nil
}

func (ds *SnapshotDataSource) readSnapshotById(ctx context.Context, instanceId, snapshotId string, response *datasource.ReadResponse) *client.GetSnapshotData {
	snapshots, err := ds.auraApi.GetSnapshotsByInstanceId(ctx, instanceId)
	if err != nil {
		response.Diagnostics.AddError("Error while reading instance snapshots", err.Error())
		return nil
	}
	if len(snapshots.Data) == 0 {
		response.Diagnostics.AddError("Cannot find snapshot", "There are no snapshots for instance "+instanceId)
		return nil
	}
	for _, s := range snapshots.Data {
		if s.SnapshotId == snapshotId {
			return &s
		}
	}
	response.Diagnostics.AddError("Cannot find snapshot",
		fmt.Sprintf("There is no snapshot for instance %s and snapshot id %s",
			instanceId, snapshotId))
	return nil
}
