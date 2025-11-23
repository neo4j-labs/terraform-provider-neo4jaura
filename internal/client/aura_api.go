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

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/util"
)

type AuraApi struct {
	auraClient      *AuraClient
	instanceTimeout time.Duration
	snapshotTimeout time.Duration
}

const (
	defaultInstanceTimeout = time.Duration(900) * time.Second
	defaultSnapshotTimeout = time.Duration(300) * time.Second
)

func NewAuraApi(client *AuraClient, instanceTimeoutInSecs *int64, snapshotTimeoutInSecs *int64) *AuraApi {
	instanceTimeout := defaultInstanceTimeout
	if instanceTimeoutInSecs != nil {
		instanceTimeout = time.Duration(*instanceTimeoutInSecs) * time.Second
	}

	snapshotTimeout := defaultSnapshotTimeout
	if snapshotTimeoutInSecs != nil {
		snapshotTimeout = time.Duration(*snapshotTimeoutInSecs) * time.Second
	}
	return &AuraApi{
		auraClient:      client,
		instanceTimeout: instanceTimeout,
		snapshotTimeout: snapshotTimeout,
	}
}

func (api *AuraApi) GetTenants(ctx context.Context) (GetProjectsResponse, error) {
	payload, status, err := api.auraClient.Get(ctx, "tenants")
	if err != nil {
		return GetProjectsResponse{}, err
	}

	if status != 200 {
		return GetProjectsResponse{}, fmt.Errorf("aura error: Status: %+v. Response: %+v", status, string(payload))
	}

	return util.Unmarshal[GetProjectsResponse](payload)
}

func (api *AuraApi) PostInstance(ctx context.Context, request PostInstanceRequest) (PostInstanceResponse, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		return PostInstanceResponse{}, err
	}

	body, status, err := api.auraClient.Post(ctx, "instances", payload)
	if err != nil {
		return PostInstanceResponse{}, err
	}

	if status != 202 {
		return PostInstanceResponse{}, fmt.Errorf("aura error: Status: %+v. Response: %+v", status, string(body))
	}

	return util.Unmarshal[PostInstanceResponse](body)
}

func (api *AuraApi) GetInstanceById(ctx context.Context, id string) (GetInstanceResponse, error) {
	payload, status, err := api.auraClient.Get(ctx, "instances/"+id)
	if err != nil {
		return GetInstanceResponse{}, err
	}
	if status != 200 {
		return GetInstanceResponse{}, fmt.Errorf("aura error: Status: %+v. Response: %+v", status, string(payload))
	}
	return util.Unmarshal[GetInstanceResponse](payload)
}

func (api *AuraApi) DeleteInstanceById(ctx context.Context, id string) (GetInstanceResponse, error) {
	payload, status, err := api.auraClient.Delete(ctx, "instances/"+id)
	if err != nil {
		return GetInstanceResponse{}, err
	}
	if status != 202 {
		return GetInstanceResponse{}, fmt.Errorf("aura error: Status: %+v. Response: %+v", status, string(payload))
	}
	return util.Unmarshal[GetInstanceResponse](payload)
}

func (api *AuraApi) PatchInstanceById(ctx context.Context, id string, request PatchInstanceRequest) (GetInstanceResponse, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		return GetInstanceResponse{}, err
	}

	body, status, err := api.auraClient.Patch(ctx, "instances/"+id, payload)
	if err != nil {
		return GetInstanceResponse{}, err
	}
	if status != 202 {
		return GetInstanceResponse{}, fmt.Errorf("aura error: Status: %+v. Response: %+v", status, string(body))
	}
	return util.Unmarshal[GetInstanceResponse](body)
}

func (api *AuraApi) PauseInstanceById(ctx context.Context, id string) (GetInstanceResponse, error) {
	body, status, err := api.auraClient.Post(ctx, fmt.Sprintf("instances/%s/pause", id), []byte("{}"))
	if err != nil {
		return GetInstanceResponse{}, err
	}
	if status != 202 {
		return GetInstanceResponse{}, fmt.Errorf("aura error: Status: %+v. Response: %+v", status, string(body))
	}
	return util.Unmarshal[GetInstanceResponse](body)
}

func (api *AuraApi) ResumeInstanceById(ctx context.Context, id string) (GetInstanceResponse, error) {
	body, status, err := api.auraClient.Post(ctx, fmt.Sprintf("instances/%s/resume", id), []byte("{}"))
	if err != nil {
		return GetInstanceResponse{}, err
	}
	if status != 202 {
		return GetInstanceResponse{}, fmt.Errorf("aura error: Status: %+v. Response: %+v", status, string(body))
	}
	return util.Unmarshal[GetInstanceResponse](body)
}

func (api *AuraApi) GetSnapshotsByInstanceId(ctx context.Context, instanceId string) (GetSnapshotsResponse, error) {
	body, status, err := api.auraClient.Get(ctx, fmt.Sprintf("instances/%s/snapshots", instanceId))
	if err != nil {
		return GetSnapshotsResponse{}, err
	}
	if status != 200 {
		return GetSnapshotsResponse{}, err
	}
	return util.Unmarshal[GetSnapshotsResponse](body)
}

func (api *AuraApi) GetSnapshotById(ctx context.Context, instanceId string, snapshotId string) (GetSnapshotResponse, error) {
	body, status, err := api.auraClient.Get(ctx, fmt.Sprintf("instances/%s/snapshots/%s", instanceId, snapshotId))
	if err != nil {
		return GetSnapshotResponse{}, err
	}
	if status != 200 {
		return GetSnapshotResponse{}, err
	}
	return util.Unmarshal[GetSnapshotResponse](body)
}

func (api *AuraApi) PostSnapshot(ctx context.Context, instanceId string) (PostSnapshotResponse, error) {
	body, status, err := api.auraClient.Post(ctx, fmt.Sprintf("instances/%s/snapshots", instanceId), nil)
	if err != nil {
		return PostSnapshotResponse{}, err
	}
	if status != 202 {
		return PostSnapshotResponse{}, fmt.Errorf("aura error: Status: %+v. Response: %+v", status, string(body))
	}
	return util.Unmarshal[PostSnapshotResponse](body)
}

func (api *AuraApi) WaitUntilSnapshotIsInState(
	ctx context.Context, instanceId string, snapshotId string,
	condition func(data GetSnapshotData) bool) (GetSnapshotData, error) {

	return util.WaitUntil(
		func() (GetSnapshotData, error) {
			r, e := api.GetSnapshotById(ctx, instanceId, snapshotId)
			tflog.Debug(ctx, fmt.Sprintf("Received response %+v and error %+v", r, e))
			if e != nil {
				return GetSnapshotData{}, e
			}
			return r.Data, e
		},
		func(resp GetSnapshotData, e error) bool {
			return e == nil && condition(resp)
		},
		time.Second,
		api.snapshotTimeout,
	)
}

func (api *AuraApi) WaitUntilInstanceIsInState(
	ctx context.Context,
	id string,
	condition func(GetInstanceResponse) bool) (GetInstanceResponse, error) {
	return util.WaitUntil(
		func() (GetInstanceResponse, error) {
			resp, err := api.GetInstanceById(ctx, id)
			tflog.Trace(ctx, fmt.Sprintf("Received response %+v and error %+v", resp, err))
			return resp, err
		},
		func(resp GetInstanceResponse, e error) bool {
			return e == nil && condition(resp)
		},
		time.Second,
		api.instanceTimeout,
	)
}

func (api *AuraApi) WaitUntilInstanceIsDeleted(ctx context.Context, id string) (err error) {
	_, err = util.WaitUntil(
		func() (status int, err error) {
			_, status, err = api.auraClient.Get(ctx, "instances/"+id)
			tflog.Trace(ctx, fmt.Sprintf("Received response status %+d and error %+v", status, err))
			return
		},
		func(status int, err error) bool {
			return err == nil && status == 404
		},
		time.Millisecond*500,
		api.instanceTimeout,
	)
	return
}
