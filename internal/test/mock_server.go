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

package test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/client"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/domain"
)

const (
	mockBearerToken = "mock-bearer-token-for-testing"
)

// mockInstanceState holds the in-memory state for a single instance,
// including a hit counter used to drive the state machine.
type mockInstanceState struct {
	instance client.GetInstanceData
	getCount int
}

// mockSnapshotState holds the in-memory state for a single snapshot,
// including a hit counter used to drive the state machine.
type mockSnapshotState struct {
	snapshot client.GetSnapshotData
	getCount int
}

// MockServer is an httptest-backed mock for the Aura v1 API.
// All state access is protected by mu so it is safe for parallel sub-tests.
type MockServer struct {
	mu     sync.Mutex
	server *httptest.Server
	// instances stores mock instance state; all access must hold mu.
	instances map[string]*mockInstanceState
	// snapshots is keyed by "<instanceId>/<snapshotId>"; all access must hold mu.
	snapshots map[string]*mockSnapshotState
	// snapshotListEmptyUntil tracks, per instance, how many more GET /snapshots
	// calls should return an empty list before returning seeded snapshots.
	// Used to simulate a recently-created instance with no snapshots yet.
	snapshotListEmptyUntil map[string]int
	// projectUsers is keyed by "<orgId>/<projectId>/<userId>" and stores project roles.
	projectUsers map[string][]string
}

// NewMockServer creates a new MockServer, registers all handlers, and starts
// the underlying httptest.Server. Call Close() when finished.
func NewMockServer() *MockServer {
	ms := &MockServer{
		instances:              make(map[string]*mockInstanceState),
		snapshots:              make(map[string]*mockSnapshotState),
		snapshotListEmptyUntil: make(map[string]int),
		projectUsers:           make(map[string][]string),
	}

	mux := http.NewServeMux()

	// Auth endpoint — no Bearer check required.
	mux.HandleFunc("POST /oauth/token", ms.handlePostToken)

	// All other endpoints go through the auth middleware.
	mux.HandleFunc("/v1/", ms.withAuth(ms.routeV1))
	mux.HandleFunc("/v2beta1/", ms.withAuth(ms.routeV2Beta1))

	ms.server = httptest.NewServer(mux)
	return ms
}

// URL returns the base URL of the running mock server (e.g. "http://127.0.0.1:PORT").
func (ms *MockServer) URL() string {
	return ms.server.URL
}

// Close shuts down the underlying httptest.Server.
func (ms *MockServer) Close() {
	ms.server.Close()
}

// Reset clears all in-memory instance and snapshot state so that each test
// starts from a clean slate.
func (ms *MockServer) Reset() {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.instances = make(map[string]*mockInstanceState)
	ms.snapshots = make(map[string]*mockSnapshotState)
	ms.snapshotListEmptyUntil = make(map[string]int)
	ms.projectUsers = make(map[string][]string)
}

// HoldSnapshotList makes the next n calls to GET /v1/instances/{instanceId}/snapshots
// return an empty list, regardless of seeded snapshots. After n calls the seeded
// snapshots are returned normally. This simulates a recently-created instance that
// does not yet have any completed snapshots, allowing isInstanceRecentlyCreated to
// be exercised.
func (ms *MockServer) HoldSnapshotList(instanceId string, n int) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.snapshotListEmptyUntil[instanceId] = n
}

// SeedInstance inserts an instance into the mock's state store. The instance
// is available immediately via GET /v1/instances/{id}. The getCount is
// initialised to 1 so that the first GET returns the seeded status directly
// (consistent with the state-machine used by task-003 handlers).
func (ms *MockServer) SeedInstance(instance client.GetInstanceData) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.instances[instance.Id] = &mockInstanceState{
		instance: instance,
		getCount: 1,
	}
}

// InstanceExists reports whether an instance with the given ID is currently
// present in the mock server's state store. Used by testAccCheckInstanceDestroyed
// to verify the instance was actually deleted.
func (ms *MockServer) InstanceExists(id string) bool {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	_, ok := ms.instances[id]
	return ok
}

// DeleteInstance removes an instance from the mock's state store out-of-band,
// simulating an external deletion without going through the Terraform delete path.
// After calling this, GET /v1/instances/{id} returns 404, which triggers
// resp.State.RemoveResource in InstanceResource.Read and causes Terraform to
// plan recreation on the next apply.
func (ms *MockServer) DeleteInstance(id string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	delete(ms.instances, id)
}

// SnapshotExists reports whether a snapshot with the given instance and snapshot
// IDs is currently present in the mock server's state store. Used by
// testAccCheckSnapshotNotInState to verify that no API-level delete was issued
// (since the Aura API has no snapshot delete endpoint and SnapshotResource.Delete
// is intentionally a no-op).
func (ms *MockServer) SnapshotExists(instanceId, snapshotId string) bool {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	_, ok := ms.snapshots[snapshotKey(instanceId, snapshotId)]
	return ok
}

// SeedProjectUser inserts a project user membership into the mock's state store.
func (ms *MockServer) SeedProjectUser(orgId, projectId, userId string, roles []string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.projectUsers[projectUserKey(orgId, projectId, userId)] = roles
}

// ProjectUserExists reports whether a project user with the given IDs is in the mock's state store.
func (ms *MockServer) ProjectUserExists(orgId, projectId, userId string) bool {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	_, ok := ms.projectUsers[projectUserKey(orgId, projectId, userId)]
	return ok
}

// SeedSnapshot inserts a snapshot into the mock's state store. The snapshot
// is available immediately via GET /v1/instances/{instanceId}/snapshots/{snapshotId}.
// The getCount is initialised to 1 so that the first GET returns the seeded
// status directly (consistent with the state machine used by task-004 handlers).
func (ms *MockServer) SeedSnapshot(snapshot client.GetSnapshotData) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	key := snapshotKey(snapshot.InstanceId, snapshot.SnapshotId)
	ms.snapshots[key] = &mockSnapshotState{
		snapshot: snapshot,
		getCount: 1,
	}
}

// ---------------------------------------------------------------------------
// Auth handler
// ---------------------------------------------------------------------------

func (ms *MockServer) handlePostToken(w http.ResponseWriter, r *http.Request) {
	resp := client.TokenResponse{
		AccessToken: mockBearerToken,
		ExpiresIn:   3600,
	}
	writeJSON(w, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// Auth middleware
// ---------------------------------------------------------------------------

// withAuth returns an http.HandlerFunc that validates the Authorization header
// before delegating to next. Any request with a missing or incorrect Bearer
// token receives HTTP 401.
func (ms *MockServer) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		expected := fmt.Sprintf("Bearer %s", mockBearerToken)
		if authHeader != expected {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// ---------------------------------------------------------------------------
// V1 router — dispatches based on path pattern.
// task-004 will replace the snapshot/tenant stubs below with real handlers.
// ---------------------------------------------------------------------------

func (ms *MockServer) routeV1(w http.ResponseWriter, r *http.Request) {
	// Strip the leading "/v1/" prefix for easier matching.
	path := strings.TrimPrefix(r.URL.Path, "/v1/")
	parts := strings.Split(strings.Trim(path, "/"), "/")

	switch {
	// GET /v1/tenants
	case len(parts) == 1 && parts[0] == "tenants" && r.Method == http.MethodGet:
		ms.handleGetTenants(w, r)

	// POST /v1/instances
	case len(parts) == 1 && parts[0] == "instances" && r.Method == http.MethodPost:
		ms.handlePostInstance(w, r)

	// GET /v1/instances/{id}
	case len(parts) == 2 && parts[0] == "instances" && r.Method == http.MethodGet:
		ms.handleGetInstance(w, r, parts[1])

	// DELETE /v1/instances/{id}
	case len(parts) == 2 && parts[0] == "instances" && r.Method == http.MethodDelete:
		ms.handleDeleteInstance(w, r, parts[1])

	// PATCH /v1/instances/{id}
	case len(parts) == 2 && parts[0] == "instances" && r.Method == http.MethodPatch:
		ms.handlePatchInstance(w, r, parts[1])

	// POST /v1/instances/{id}/pause
	case len(parts) == 3 && parts[0] == "instances" && parts[2] == "pause" && r.Method == http.MethodPost:
		ms.handlePauseInstance(w, r, parts[1])

	// POST /v1/instances/{id}/resume
	case len(parts) == 3 && parts[0] == "instances" && parts[2] == "resume" && r.Method == http.MethodPost:
		ms.handleResumeInstance(w, r, parts[1])

	// POST /v1/instances/{id}/snapshots
	case len(parts) == 3 && parts[0] == "instances" && parts[2] == "snapshots" && r.Method == http.MethodPost:
		ms.handlePostSnapshot(w, r, parts[1])

	// GET /v1/instances/{id}/snapshots
	case len(parts) == 3 && parts[0] == "instances" && parts[2] == "snapshots" && r.Method == http.MethodGet:
		ms.handleGetSnapshots(w, r, parts[1])

	// GET /v1/instances/{id}/snapshots/{snapshotId}
	case len(parts) == 4 && parts[0] == "instances" && parts[2] == "snapshots" && r.Method == http.MethodGet:
		ms.handleGetSnapshot(w, r, parts[1], parts[3])

	default:
		http.NotFound(w, r)
	}
}

// ---------------------------------------------------------------------------
// Stub handlers — will be replaced in task-003 and task-004.
// ---------------------------------------------------------------------------

func (ms *MockServer) handleGetTenants(w http.ResponseWriter, _ *http.Request) {
	resp := client.GetProjectsResponse{
		Data: []client.ProjectResponseData{
			{Id: "test-project-id-001", Name: "Test Project"},
		},
	}
	writeJSON(w, http.StatusOK, resp)
}

// handlePostInstance creates a new instance in "creating" state and returns 202
// with a PostInstanceResponse. A unique ID is generated from the current time.
func (ms *MockServer) handlePostInstance(w http.ResponseWriter, r *http.Request) {
	var req client.PostInstanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	id := fmt.Sprintf("inst-%d", time.Now().UnixNano())

	ms.mu.Lock()
	ms.instances[id] = &mockInstanceState{
		instance: client.GetInstanceData{
			Id:                   id,
			Name:                 req.Name,
			Status:               domain.InstanceStatusCreating,
			TenantId:             req.TenantId,
			CloudProvider:        req.CloudProvider,
			Region:               req.Region,
			Type:                 req.Type,
			Memory:               req.Memory,
			Storage:              req.Storage,
			VectorOptimized:      req.VectorOptimized,
			GraphAnalyticsPlugin: req.GraphAnalyticsPlugin,
		},
		getCount: 0,
	}
	ms.mu.Unlock()

	resp := client.PostInstanceResponse{
		Data: client.PostInstanceData{
			Id:            id,
			Name:          req.Name,
			TenantId:      req.TenantId,
			CloudProvider: req.CloudProvider,
			Region:        req.Region,
			Type:          req.Type,
		},
	}
	writeJSON(w, http.StatusAccepted, resp)
}

// handleGetInstance returns the instance state, driving a state machine:
//   - getCount == 1 (first real GET after creation): status="creating"
//   - getCount >= 2 and status is still "creating": status="running"
//   - getCount >= 2 and status has been explicitly set (e.g. via pause/resume): status is preserved
func (ms *MockServer) handleGetInstance(w http.ResponseWriter, _ *http.Request, id string) {
	ms.mu.Lock()
	state, ok := ms.instances[id]
	if !ok {
		ms.mu.Unlock()
		http.NotFound(w, nil)
		return
	}
	state.getCount++
	count := state.getCount

	if count >= 2 && state.instance.Status == domain.InstanceStatusCreating {
		// Drive the initial creating → running transition only.
		// If the status has been explicitly changed by pause/resume, preserve it.
		state.instance.Status = domain.InstanceStatusRunning
	} else if count < 2 {
		state.instance.Status = domain.InstanceStatusCreating
	}
	snap := state.instance
	ms.mu.Unlock()

	writeJSON(w, http.StatusOK, client.GetInstanceResponse{Data: snap})
}

// handleDeleteInstance removes the instance from state and returns 202.
// Subsequent GETs for the same ID will return 404 (handled by the not-found
// branch in handleGetInstance), satisfying WaitUntilInstanceIsDeleted.
func (ms *MockServer) handleDeleteInstance(w http.ResponseWriter, _ *http.Request, id string) {
	ms.mu.Lock()
	state, ok := ms.instances[id]
	if !ok {
		ms.mu.Unlock()
		http.NotFound(w, nil)
		return
	}
	snap := state.instance
	delete(ms.instances, id)
	ms.mu.Unlock()

	writeJSON(w, http.StatusAccepted, client.GetInstanceResponse{Data: snap})
}

// handlePatchInstance updates mutable instance fields and returns 202 with the
// updated GetInstanceResponse.
func (ms *MockServer) handlePatchInstance(w http.ResponseWriter, r *http.Request, id string) {
	var req client.PatchInstanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	ms.mu.Lock()
	state, ok := ms.instances[id]
	if !ok {
		ms.mu.Unlock()
		http.NotFound(w, nil)
		return
	}
	if req.Name != nil {
		state.instance.Name = *req.Name
	}
	if req.Memory != nil {
		state.instance.Memory = *req.Memory
	}
	if req.Storage != nil {
		state.instance.Storage = req.Storage
	}
	if req.CdcEnrichmentMode != nil {
		state.instance.CdcEnrichmentMode = req.CdcEnrichmentMode
	}
	if req.SecondariesCount != nil {
		v := int(*req.SecondariesCount)
		state.instance.SecondariesCount = &v
	}
	if req.VectorOptimized != nil {
		state.instance.VectorOptimized = req.VectorOptimized
	}
	if req.GraphAnalyticsPlugin != nil {
		state.instance.GraphAnalyticsPlugin = req.GraphAnalyticsPlugin
	}
	snap := state.instance
	ms.mu.Unlock()

	writeJSON(w, http.StatusAccepted, client.GetInstanceResponse{Data: snap})
}

// handlePauseInstance transitions the instance status to "paused" and returns 202.
func (ms *MockServer) handlePauseInstance(w http.ResponseWriter, _ *http.Request, id string) {
	ms.mu.Lock()
	state, ok := ms.instances[id]
	if !ok {
		ms.mu.Unlock()
		http.NotFound(w, nil)
		return
	}
	state.instance.Status = domain.InstanceStatusPaused
	snap := state.instance
	ms.mu.Unlock()

	writeJSON(w, http.StatusAccepted, client.GetInstanceResponse{Data: snap})
}

// handleResumeInstance transitions the instance status back to "running" and returns 202.
func (ms *MockServer) handleResumeInstance(w http.ResponseWriter, _ *http.Request, id string) {
	ms.mu.Lock()
	state, ok := ms.instances[id]
	if !ok {
		ms.mu.Unlock()
		http.NotFound(w, nil)
		return
	}
	state.instance.Status = domain.InstanceStatusRunning
	snap := state.instance
	ms.mu.Unlock()

	writeJSON(w, http.StatusAccepted, client.GetInstanceResponse{Data: snap})
}

func (ms *MockServer) handlePostSnapshot(w http.ResponseWriter, _ *http.Request, instanceId string) {
	snapshotId := fmt.Sprintf("snap-%d", time.Now().UnixNano())
	snap := client.GetSnapshotData{
		InstanceId: instanceId,
		SnapshotId: snapshotId,
		Profile:    domain.SnapshotProfileScheduled,
		Status:     domain.SnapshotStatusInProgress,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}

	ms.mu.Lock()
	ms.snapshots[snapshotKey(instanceId, snapshotId)] = &mockSnapshotState{
		snapshot: snap,
		getCount: 0,
	}
	ms.mu.Unlock()

	writeJSON(w, http.StatusAccepted, client.PostSnapshotResponse{
		Data: client.PostSnapshotData{SnapshotId: snapshotId},
	})
}

func (ms *MockServer) handleGetSnapshots(w http.ResponseWriter, _ *http.Request, instanceId string) {
	ms.mu.Lock()
	// If there are remaining hold counts for this instance, decrement and return empty.
	if count := ms.snapshotListEmptyUntil[instanceId]; count > 0 {
		ms.snapshotListEmptyUntil[instanceId] = count - 1
		ms.mu.Unlock()
		writeJSON(w, http.StatusOK, client.GetSnapshotsResponse{Data: []client.GetSnapshotData{}})
		return
	}
	var snapshots []client.GetSnapshotData
	for _, state := range ms.snapshots {
		if state.snapshot.InstanceId == instanceId {
			snapshots = append(snapshots, state.snapshot)
		}
	}
	ms.mu.Unlock()

	if snapshots == nil {
		snapshots = []client.GetSnapshotData{}
	}
	writeJSON(w, http.StatusOK, client.GetSnapshotsResponse{Data: snapshots})
}

func (ms *MockServer) handleGetSnapshot(w http.ResponseWriter, _ *http.Request, instanceId, snapshotId string) {
	key := snapshotKey(instanceId, snapshotId)

	ms.mu.Lock()
	state, ok := ms.snapshots[key]
	if !ok {
		ms.mu.Unlock()
		http.NotFound(w, nil)
		return
	}
	state.getCount++
	if state.getCount >= 2 {
		state.snapshot.Status = domain.SnapshotStatusCompleted
	} else {
		state.snapshot.Status = domain.SnapshotStatusInProgress
	}
	snap := state.snapshot
	ms.mu.Unlock()

	writeJSON(w, http.StatusOK, client.GetSnapshotResponse{Data: snap})
}

// ---------------------------------------------------------------------------
// V2Beta1 router
// ---------------------------------------------------------------------------

func (ms *MockServer) routeV2Beta1(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v2beta1/")
	parts := strings.Split(strings.Trim(path, "/"), "/")

	switch {
	// GET /v2beta1/organizations
	case len(parts) == 1 && parts[0] == "organizations" && r.Method == http.MethodGet:
		ms.handleGetOrganizations(w, r)

	// GET /v2beta1/organizations/{id}/projects
	case len(parts) == 3 && parts[0] == "organizations" && parts[2] == "projects" && r.Method == http.MethodGet:
		ms.handleGetProjectsByOrganization(w, r, parts[1])

	// GET /v2beta1/organizations/{id}/users
	case len(parts) == 3 && parts[0] == "organizations" && parts[2] == "users" && r.Method == http.MethodGet:
		ms.handleGetOrgUsers(w, r)

	// GET /v2beta1/organizations/{orgId}/users/{userId}
	case len(parts) == 4 && parts[0] == "organizations" && parts[2] == "users" && r.Method == http.MethodGet:
		ms.handleGetOrgUser(w, r, parts[1], parts[3])

	// GET /v2beta1/organizations/{orgId}/projects/{projectId}/users
	case len(parts) == 5 && parts[0] == "organizations" && parts[2] == "projects" && parts[4] == "users" && r.Method == http.MethodGet:
		ms.handleGetProjectUsers(w, r, parts[1], parts[3])

	// POST /v2beta1/organizations/{orgId}/projects/{projectId}/users/{userId}
	case len(parts) == 6 && parts[0] == "organizations" && parts[2] == "projects" && parts[4] == "users" && r.Method == http.MethodPost:
		ms.handlePostProjectUser(w, r, parts[1], parts[3], parts[5])

	// PATCH /v2beta1/organizations/{orgId}/projects/{projectId}/users/{userId}
	case len(parts) == 6 && parts[0] == "organizations" && parts[2] == "projects" && parts[4] == "users" && r.Method == http.MethodPatch:
		ms.handlePatchProjectUser(w, r, parts[1], parts[3], parts[5])

	// DELETE /v2beta1/organizations/{orgId}/projects/{projectId}/users/{userId}
	case len(parts) == 6 && parts[0] == "organizations" && parts[2] == "projects" && parts[4] == "users" && r.Method == http.MethodDelete:
		ms.handleDeleteProjectUser(w, r, parts[1], parts[3], parts[5])

	default:
		http.NotFound(w, r)
	}
}

func (ms *MockServer) handleGetProjectUsers(w http.ResponseWriter, _ *http.Request, orgId, projectId string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Build the list from the mutable projectUsers store first, then fall back to seeded data.
	var data []client.ProjectUserData
	for key, roles := range ms.projectUsers {
		parts := strings.SplitN(key, "/", 3)
		if len(parts) == 3 && parts[0] == orgId && parts[1] == projectId {
			data = append(data, client.ProjectUserData{
				UserId:       parts[2],
				ProjectRoles: roles,
			})
		}
	}

	// If no dynamic state exists for this project, return static seed data.
	if len(data) == 0 {
		seeded := map[string][]client.ProjectUserData{
			"proj-001": {
				{UserId: "user-001", Email: "alice@example.com", ProjectRoles: []string{"owner"}},
			},
			"proj-002": {
				{UserId: "user-001", Email: "alice@example.com", ProjectRoles: []string{"viewer"}},
				{UserId: "user-002", Email: "bob@example.com", ProjectRoles: []string{"owner"}},
			},
		}
		if seededData, ok := seeded[projectId]; ok {
			data = seededData
		}
	}

	if data == nil {
		data = []client.ProjectUserData{}
	}
	writeJSON(w, http.StatusOK, client.GetProjectUsersResponse{Data: data})
}

func (ms *MockServer) handlePostProjectUser(w http.ResponseWriter, r *http.Request, orgId, projectId, userId string) {
	var req client.PostProjectUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	key := projectUserKey(orgId, projectId, userId)
	ms.mu.Lock()
	ms.projectUsers[key] = req.ProjectRoles
	ms.mu.Unlock()

	w.WriteHeader(http.StatusCreated)
}

func (ms *MockServer) handlePatchProjectUser(w http.ResponseWriter, r *http.Request, orgId, projectId, userId string) {
	var req client.PatchProjectUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	key := projectUserKey(orgId, projectId, userId)
	ms.mu.Lock()
	_, exists := ms.projectUsers[key]
	if !exists {
		ms.mu.Unlock()
		http.NotFound(w, r)
		return
	}
	ms.projectUsers[key] = req.ProjectRoles
	ms.mu.Unlock()

	writeJSON(w, http.StatusOK, client.PatchProjectUserResponse{
		Data: client.ProjectUserData{
			UserId:       userId,
			ProjectRoles: req.ProjectRoles,
		},
	})
}

func (ms *MockServer) handleDeleteProjectUser(w http.ResponseWriter, _ *http.Request, orgId, projectId, userId string) {
	key := projectUserKey(orgId, projectId, userId)
	ms.mu.Lock()
	_, exists := ms.projectUsers[key]
	if !exists {
		ms.mu.Unlock()
		http.NotFound(w, nil)
		return
	}
	delete(ms.projectUsers, key)
	ms.mu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

func (ms *MockServer) handleGetOrgUsers(w http.ResponseWriter, _ *http.Request) {
	lastActivity := "2024-06-01T12:00:00Z"
	resp := client.GetOrganizationUsersResponse{
		Data: []client.OrganizationUserData{
			{
				UserId:                     "user-001",
				Email:                      "alice@example.com",
				ExemptFromAutomaticRemoval: false,
				LastActivityAt:             &lastActivity,
				MfaEnrollmentStatus:        "enrolled",
				MfaEnrolledMethods:         []client.MfaMethodData{{Id: "totp", EnrolledAt: "2024-01-01T00:00:00Z"}},
				OrganizationRoles:          []string{"admin"},
			},
			{
				UserId:                     "user-002",
				Email:                      "bob@example.com",
				ExemptFromAutomaticRemoval: true,
				LastActivityAt:             nil,
				MfaEnrollmentStatus:        "not_enrolled",
				MfaEnrolledMethods:         []client.MfaMethodData{},
				OrganizationRoles:          []string{"member"},
			},
		},
	}
	writeJSON(w, http.StatusOK, resp)
}

func (ms *MockServer) handleGetOrgUser(w http.ResponseWriter, _ *http.Request, orgId, userId string) {
	lastActivity := "2024-06-01T12:00:00Z"

	staticUsers := map[string]client.OrganizationUserData{
		"user-001": {
			UserId:                     "user-001",
			Email:                      "alice@example.com",
			ExemptFromAutomaticRemoval: false,
			LastActivityAt:             &lastActivity,
			MfaEnrollmentStatus:        "enrolled",
			MfaEnrolledMethods:         []client.MfaMethodData{{Id: "totp", EnrolledAt: "2024-01-01T00:00:00Z"}},
			OrganizationRoles:          []string{"admin"},
		},
		"user-002": {
			UserId:                     "user-002",
			Email:                      "bob@example.com",
			ExemptFromAutomaticRemoval: true,
			LastActivityAt:             nil,
			MfaEnrollmentStatus:        "not_enrolled",
			MfaEnrolledMethods:         []client.MfaMethodData{},
			OrganizationRoles:          []string{"member"},
		},
	}
	staticProjects := map[string][]client.UserProjectData{
		"user-001": {
			{Id: "proj-001", Name: "Alpha", ProjectRoles: []string{"owner"}},
			{Id: "proj-002", Name: "Beta", ProjectRoles: []string{"viewer"}},
		},
		"user-002": {
			{Id: "proj-002", Name: "Beta", ProjectRoles: []string{"owner"}},
		},
	}

	userData, ok := staticUsers[userId]
	if !ok {
		http.NotFound(w, nil)
		return
	}

	projects := make([]client.UserProjectData, len(staticProjects[userId]))
	copy(projects, staticProjects[userId])

	staticProjectIds := make(map[string]struct{}, len(staticProjects[userId]))
	for _, p := range staticProjects[userId] {
		staticProjectIds[p.Id] = struct{}{}
	}

	// Append dynamic project memberships from projectUsers store (not already in static list).
	ms.mu.Lock()
	for key, roles := range ms.projectUsers {
		parts := strings.SplitN(key, "/", 3)
		if len(parts) == 3 && parts[0] == orgId && parts[2] == userId {
			projectId := parts[1]
			if _, exists := staticProjectIds[projectId]; !exists {
				projects = append(projects, client.UserProjectData{
					Id:           projectId,
					ProjectRoles: roles,
				})
			}
		}
	}
	ms.mu.Unlock()

	writeJSON(w, http.StatusOK, client.GetOrganizationUserDetailsResponse{
		Data: client.OrganizationUserDetailsData{
			OrganizationUserData: userData,
			Projects:             projects,
		},
	})
}

func (ms *MockServer) handleGetProjectsByOrganization(w http.ResponseWriter, _ *http.Request, _ string) {
	resp := client.GetProjectsResponse{
		Data: []client.ProjectResponseData{
			{Id: "test-org-project-id-001", Name: "Org Project One"},
			{Id: "test-org-project-id-002", Name: "Org Project Two"},
			{Id: "proj-001", Name: "Alpha"},
			{Id: "proj-002", Name: "Beta"},
		},
	}
	writeJSON(w, http.StatusOK, resp)
}

func (ms *MockServer) handleGetOrganizations(w http.ResponseWriter, _ *http.Request) {
	resp := client.GetOrganizationsResponse{
		Data: []client.OrganizationData{
			{Id: "test-org-id-001", Name: "Test Organization"},
		},
	}
	writeJSON(w, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func snapshotKey(instanceId, snapshotId string) string {
	return instanceId + "/" + snapshotId
}

func projectUserKey(orgId, projectId, userId string) string {
	return orgId + "/" + projectId + "/" + userId
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		panic(fmt.Sprintf("writeJSON: failed to encode response: %v", err))
	}
}
