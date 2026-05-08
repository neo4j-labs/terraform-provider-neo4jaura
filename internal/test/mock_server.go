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
}

// NewMockServer creates a new MockServer, registers all handlers, and starts
// the underlying httptest.Server. Call Close() when finished.
func NewMockServer() *MockServer {
	ms := &MockServer{
		instances: make(map[string]*mockInstanceState),
		snapshots: make(map[string]*mockSnapshotState),
	}

	mux := http.NewServeMux()

	// Auth endpoint — no Bearer check required.
	mux.HandleFunc("POST /oauth/token", ms.handlePostToken)

	// All other endpoints go through the auth middleware.
	mux.HandleFunc("/v1/", ms.withAuth(ms.routeV1))

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
			Id:            id,
			Name:          req.Name,
			Status:        domain.InstanceStatusCreating,
			TenantId:      req.TenantId,
			CloudProvider: req.CloudProvider,
			Region:        req.Region,
			Type:          req.Type,
			Memory:        req.Memory,
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
//   - getCount == 1 (first real GET after creation): status="creating", no graph metrics
//   - getCount >= 2: status="running", graph_nodes=10, graph_relationships=5
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

	if count >= 2 {
		state.instance.Status = domain.InstanceStatusRunning
		nodes := int64(10)
		rels := int64(5)
		state.instance.GraphNodes = &nodes
		state.instance.GraphRelationships = &rels
	} else {
		state.instance.Status = domain.InstanceStatusCreating
		state.instance.GraphNodes = nil
		state.instance.GraphRelationships = nil
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

// handlePatchInstance updates mutable fields (name, memory, cdc_enrichment_mode,
// secondaries_count) and returns 202 with the updated GetInstanceResponse.
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
	if req.CdcEnrichmentMode != nil {
		state.instance.CdcEnrichmentMode = req.CdcEnrichmentMode
	}
	if req.SecondariesCount != nil {
		v := int(*req.SecondariesCount)
		state.instance.SecondariesCount = &v
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
// Helpers
// ---------------------------------------------------------------------------

func snapshotKey(instanceId, snapshotId string) string {
	return instanceId + "/" + snapshotId
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
