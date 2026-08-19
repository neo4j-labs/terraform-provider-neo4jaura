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
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/domain"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/util"
)

type AuraApi struct {
	v1auraClient    *AuraClient
	v2beta1Client   *AuraClient
	instanceTimeout time.Duration
	snapshotTimeout time.Duration
}

const (
	defaultInstanceTimeout = time.Duration(1200) * time.Second
	defaultSnapshotTimeout = time.Duration(300) * time.Second
)

func NewAuraApi(client *AuraClient, v2beta1Client *AuraClient, instanceTimeoutInSecs *int64, snapshotTimeoutInSecs *int64) *AuraApi {
	instanceTimeout := defaultInstanceTimeout
	if instanceTimeoutInSecs != nil {
		instanceTimeout = time.Duration(*instanceTimeoutInSecs) * time.Second
	}

	snapshotTimeout := defaultSnapshotTimeout
	if snapshotTimeoutInSecs != nil {
		snapshotTimeout = time.Duration(*snapshotTimeoutInSecs) * time.Second
	}
	return &AuraApi{
		v1auraClient:    client,
		v2beta1Client:   v2beta1Client,
		instanceTimeout: instanceTimeout,
		snapshotTimeout: snapshotTimeout,
	}
}

func validateResponseStatus(method string, status int, payload []byte) error {
	if isSuccessfulResponseStatus(method, status) {
		return nil
	}
	return fmt.Errorf("aura error: Status: %+v. Response: %+v", status, string(payload))
}

func unmarshalAuraResponse[T any](method string, status int, payload []byte) (T, error) {
	if err := validateResponseStatus(method, status, payload); err != nil {
		var resp T
		return resp, err
	}
	return util.Unmarshal[T](payload)
}

func isSuccessfulResponseStatus(method string, status int) bool {
	switch method {
	case http.MethodGet:
		return status == http.StatusOK

	default:

		return status == http.StatusOK || status == http.StatusAccepted
	}
}

func (api *AuraApi) GetTenants(ctx context.Context) (GetProjectsResponse, error) {
	payload, status, err := api.v1auraClient.Get(ctx, "tenants")
	if err != nil {
		return GetProjectsResponse{}, err
	}

	return unmarshalAuraResponse[GetProjectsResponse](http.MethodGet, status, payload)
}

func (api *AuraApi) PostInstance(ctx context.Context, request PostInstanceRequest) (PostInstanceResponse, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		return PostInstanceResponse{}, err
	}

	body, status, err := api.v1auraClient.Post(ctx, "instances", payload)
	if err != nil {
		return PostInstanceResponse{}, err
	}

	return unmarshalAuraResponse[PostInstanceResponse](http.MethodPost, status, body)
}

func (api *AuraApi) GetInstanceById(ctx context.Context, id string) (GetInstanceResponse, error) {
	payload, status, err := api.v1auraClient.Get(ctx, "instances/"+id)
	if err != nil {
		return GetInstanceResponse{}, err
	}
	if status == 404 {
		return GetInstanceResponse{}, fmt.Errorf("instance %s: %w", id, ErrNotFound)
	}
	return unmarshalAuraResponse[GetInstanceResponse](http.MethodGet, status, payload)
}

func (api *AuraApi) DeleteInstanceById(ctx context.Context, id string) (GetInstanceResponse, error) {
	payload, status, err := api.v1auraClient.Delete(ctx, "instances/"+id)
	if err != nil {
		return GetInstanceResponse{}, err
	}
	if status == 404 {
		return GetInstanceResponse{}, fmt.Errorf("instance %s not found: %w", id, ErrNotFound)
	}
	return unmarshalAuraResponse[GetInstanceResponse](http.MethodDelete, status, payload)
}

func (api *AuraApi) PatchInstanceById(ctx context.Context, id string, request PatchInstanceRequest) (GetInstanceResponse, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		return GetInstanceResponse{}, err
	}

	body, status, err := api.v1auraClient.Patch(ctx, "instances/"+id, payload)
	if err != nil {
		return GetInstanceResponse{}, err
	}
	return unmarshalAuraResponse[GetInstanceResponse](http.MethodPatch, status, body)
}

func (api *AuraApi) PauseInstanceById(ctx context.Context, id string) (GetInstanceResponse, error) {
	body, status, err := api.v1auraClient.Post(ctx, fmt.Sprintf("instances/%s/pause", id), []byte("{}"))
	if err != nil {
		return GetInstanceResponse{}, err
	}
	return unmarshalAuraResponse[GetInstanceResponse](http.MethodPost, status, body)
}

func (api *AuraApi) ResumeInstanceById(ctx context.Context, id string) (GetInstanceResponse, error) {
	body, status, err := api.v1auraClient.Post(ctx, fmt.Sprintf("instances/%s/resume", id), []byte("{}"))
	if err != nil {
		return GetInstanceResponse{}, err
	}
	return unmarshalAuraResponse[GetInstanceResponse](http.MethodPost, status, body)
}

func (api *AuraApi) GetSnapshotsByInstanceId(ctx context.Context, instanceId string) (GetSnapshotsResponse, error) {
	body, status, err := api.v1auraClient.Get(ctx, fmt.Sprintf("instances/%s/snapshots", instanceId))
	if err != nil {
		return GetSnapshotsResponse{}, err
	}
	return unmarshalAuraResponse[GetSnapshotsResponse](http.MethodGet, status, body)
}

func (api *AuraApi) GetSnapshotById(ctx context.Context, instanceId string, snapshotId string) (GetSnapshotResponse, error) {
	body, status, err := api.v1auraClient.Get(ctx, fmt.Sprintf("instances/%s/snapshots/%s", instanceId, snapshotId))
	if err != nil {
		return GetSnapshotResponse{}, err
	}
	if status == 404 {
		return GetSnapshotResponse{}, fmt.Errorf("snapshot %s (instance %s): %w", snapshotId, instanceId, ErrNotFound)
	}
	return unmarshalAuraResponse[GetSnapshotResponse](http.MethodGet, status, body)
}

func (api *AuraApi) PostSnapshot(ctx context.Context, instanceId string) (PostSnapshotResponse, error) {
	body, status, err := api.v1auraClient.Post(ctx, fmt.Sprintf("instances/%s/snapshots", instanceId), nil)
	if err != nil {
		return PostSnapshotResponse{}, err
	}
	return unmarshalAuraResponse[PostSnapshotResponse](http.MethodPost, status, body)
}

func (api *AuraApi) WaitUntilSnapshotIsInState(
	ctx context.Context, instanceId string, snapshotId string,
	condition func(data GetSnapshotData) bool) (GetSnapshotData, error) {

	return util.WaitUntil(
		ctx,
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

func (api *AuraApi) WaitUntilSnapshotsMatchCondition(
	ctx context.Context, instanceId string,
	condition func(data GetSnapshotsResponse) bool) (GetSnapshotsResponse, error) {

	return util.WaitUntil(
		ctx,
		func() (GetSnapshotsResponse, error) {
			r, e := api.GetSnapshotsByInstanceId(ctx, instanceId)
			tflog.Debug(ctx, fmt.Sprintf("Received response %+v and error %+v", r, e))
			if e != nil {
				return GetSnapshotsResponse{}, e
			}
			return r, e
		},
		func(resp GetSnapshotsResponse, e error) bool {
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
		ctx,
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

func (api *AuraApi) GetProjectUsers(ctx context.Context, organizationId, projectId string) (GetProjectUsersResponse, error) {
	payload, status, err := api.v2beta1Client.Get(ctx, fmt.Sprintf("organizations/%s/projects/%s/users", organizationId, projectId))
	if err != nil {
		return GetProjectUsersResponse{}, err
	}
	return unmarshalAuraResponse[GetProjectUsersResponse](http.MethodGet, status, payload)
}

func (api *AuraApi) GetOrganizationUsers(ctx context.Context, organizationId string) (GetOrganizationUsersResponse, error) {
	payload, status, err := api.v2beta1Client.Get(ctx, fmt.Sprintf("organizations/%s/users", organizationId))
	if err != nil {
		return GetOrganizationUsersResponse{}, err
	}
	return unmarshalAuraResponse[GetOrganizationUsersResponse](http.MethodGet, status, payload)
}

func (api *AuraApi) GetOrganizationUserDetails(ctx context.Context, organizationId, userId string) (GetOrganizationUserDetailsResponse, error) {
	payload, status, err := api.v2beta1Client.Get(ctx, fmt.Sprintf("organizations/%s/users/%s", organizationId, userId))
	if err != nil {
		return GetOrganizationUserDetailsResponse{}, err
	}
	if status == 404 {
		return GetOrganizationUserDetailsResponse{}, fmt.Errorf("user %s in organization %s: %w", userId, organizationId, ErrNotFound)
	}
	return unmarshalAuraResponse[GetOrganizationUserDetailsResponse](http.MethodGet, status, payload)
}

func (api *AuraApi) GetProjectsByOrganizationId(ctx context.Context, organizationId string) (GetProjectsResponse, error) {
	payload, status, err := api.v2beta1Client.Get(ctx, fmt.Sprintf("organizations/%s/projects", organizationId))
	if err != nil {
		return GetProjectsResponse{}, err
	}
	return unmarshalAuraResponse[GetProjectsResponse](http.MethodGet, status, payload)
}

func (api *AuraApi) GetOrganizations(ctx context.Context) (GetOrganizationsResponse, error) {
	payload, status, err := api.v2beta1Client.Get(ctx, "organizations")
	if err != nil {
		return GetOrganizationsResponse{}, err
	}
	return unmarshalAuraResponse[GetOrganizationsResponse](http.MethodGet, status, payload)
}

func (api *AuraApi) PostProjectUser(ctx context.Context, orgId, projectId string, req PostProjectUserRequest) error {
	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("organizations/%s/projects/%s/users", orgId, projectId)
	body, status, err := api.v2beta1Client.Post(ctx, path, payload)
	if err != nil {
		return err
	}
	if status == http.StatusCreated || status == http.StatusOK {
		return nil
	}
	return fmt.Errorf("aura error: Status: %d. Response: %s", status, string(body))
}

func (api *AuraApi) PatchProjectUser(ctx context.Context, orgId, projectId, userId string, req PatchProjectUserRequest) (PatchProjectUserResponse, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return PatchProjectUserResponse{}, err
	}
	path := fmt.Sprintf("organizations/%s/projects/%s/users/%s", orgId, projectId, userId)
	body, status, err := api.v2beta1Client.Patch(ctx, path, payload)
	if err != nil {
		return PatchProjectUserResponse{}, err
	}
	return unmarshalAuraResponse[PatchProjectUserResponse](http.MethodPatch, status, body)
}

func (api *AuraApi) DeleteProjectUser(ctx context.Context, orgId, projectId, userId string) error {
	path := fmt.Sprintf("organizations/%s/projects/%s/users/%s", orgId, projectId, userId)
	body, status, err := api.v2beta1Client.Delete(ctx, path)
	if err != nil {
		return err
	}
	if status == http.StatusNoContent || status == http.StatusOK || status == http.StatusAccepted {
		return nil
	}
	return fmt.Errorf("aura error: Status: %d. Response: %s", status, string(body))
}

func (api *AuraApi) PatchOrganizationUser(ctx context.Context, orgId, userId string, req PatchOrganizationUserRequest) (PatchOrganizationUserResponse, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return PatchOrganizationUserResponse{}, err
	}
	path := fmt.Sprintf("organizations/%s/users/%s", orgId, userId)
	body, status, err := api.v2beta1Client.Patch(ctx, path, payload)
	if err != nil {
		return PatchOrganizationUserResponse{}, err
	}
	return unmarshalAuraResponse[PatchOrganizationUserResponse](http.MethodPatch, status, body)
}

func (api *AuraApi) DeleteOrganizationUser(ctx context.Context, orgId, userId string) error {
	path := fmt.Sprintf("organizations/%s/users/%s", orgId, userId)
	body, status, err := api.v2beta1Client.Delete(ctx, path)
	if err != nil {
		return err
	}
	if status == http.StatusNoContent || status == http.StatusOK || status == http.StatusAccepted {
		return nil
	}
	return fmt.Errorf("aura error: Status: %d. Response: %s", status, string(body))
}

// projectInviteRolesToWire converts the "project-" prefixed roles used everywhere in the
// provider to the "namespace-" prefixed spelling the invite endpoints expect on the wire.
func projectInviteRolesToWire(invites []ProjectInviteRequest) []ProjectInviteRequest {
	converted := make([]ProjectInviteRequest, len(invites))
	for i, invite := range invites {
		roles := make([]string, len(invite.ProjectRoles))
		for j, role := range invite.ProjectRoles {
			roles[j] = domain.ProjectRoleToInviteRole(role)
		}
		converted[i] = ProjectInviteRequest{ProjectId: invite.ProjectId, ProjectRoles: roles}
	}
	return converted
}

// projectInviteRolesFromWire converts the "namespace-" prefixed roles returned by the invite
// endpoints back to the "project-" prefixed spelling used everywhere else in the provider.
func projectInviteRolesFromWire(invites []ProjectInviteData) []ProjectInviteData {
	converted := make([]ProjectInviteData, len(invites))
	for i, invite := range invites {
		roles := make([]string, len(invite.ProjectRoles))
		for j, role := range invite.ProjectRoles {
			roles[j] = domain.InviteRoleToProjectRole(role)
		}
		converted[i] = ProjectInviteData{ProjectId: invite.ProjectId, ProjectRoles: roles}
	}
	return converted
}

func (api *AuraApi) PostOrganizationInvite(ctx context.Context, orgId string, req PostOrganizationInviteRequest) (PostOrganizationInviteResponse, error) {
	req.ProjectInvites = projectInviteRolesToWire(req.ProjectInvites)

	payload, err := json.Marshal(req)
	if err != nil {
		return PostOrganizationInviteResponse{}, err
	}
	path := fmt.Sprintf("organizations/%s/invites", orgId)
	body, status, err := api.v2beta1Client.Post(ctx, path, payload)
	if err != nil {
		return PostOrganizationInviteResponse{}, err
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return PostOrganizationInviteResponse{}, fmt.Errorf("aura error: Status: %d. Response: %s", status, string(body))
	}
	resp, err := util.Unmarshal[PostOrganizationInviteResponse](body)
	if err != nil {
		return PostOrganizationInviteResponse{}, err
	}
	resp.Data.ProjectInvites = projectInviteRolesFromWire(resp.Data.ProjectInvites)
	return resp, nil
}

func (api *AuraApi) GetOrganizationInvites(ctx context.Context, orgId string) (GetOrganizationInvitesResponse, error) {
	payload, status, err := api.v2beta1Client.Get(ctx, fmt.Sprintf("organizations/%s/invites", orgId))
	if err != nil {
		return GetOrganizationInvitesResponse{}, err
	}
	resp, err := unmarshalAuraResponse[GetOrganizationInvitesResponse](http.MethodGet, status, payload)
	if err != nil {
		return GetOrganizationInvitesResponse{}, err
	}
	for i := range resp.Data {
		resp.Data[i].ProjectInvites = projectInviteRolesFromWire(resp.Data[i].ProjectInvites)
	}
	return resp, nil
}

func (api *AuraApi) DeleteOrganizationInvite(ctx context.Context, orgId, inviteId string) error {
	path := fmt.Sprintf("organizations/%s/invites/%s", orgId, inviteId)
	body, status, err := api.v2beta1Client.Delete(ctx, path)
	if err != nil {
		return err
	}
	if status == http.StatusNoContent || status == http.StatusOK || status == http.StatusAccepted {
		return nil
	}
	if status == http.StatusNotFound {
		return fmt.Errorf("invite %s in organization %s: %w", inviteId, orgId, ErrNotFound)
	}
	return fmt.Errorf("aura error: Status: %d. Response: %s", status, string(body))
}

func (api *AuraApi) WaitUntilInstanceIsDeleted(ctx context.Context, id string) (err error) {
	_, err = util.WaitUntil(
		ctx,
		func() (status int, err error) {
			_, status, err = api.v1auraClient.Get(ctx, "instances/"+id)
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
