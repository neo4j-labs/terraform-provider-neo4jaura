# PRD: Acceptance Tests with Mock Aura Server

## Overview

Replace the live Neo4j Aura infrastructure dependency in the acceptance test suite with an in-process mock HTTP server that simulates the Aura v1 API. This eliminates the cost and runtime of provisioning real cloud instances during CI, while preserving end-to-end coverage of provider CRUD logic, state management, and async polling behaviour.

## Goals

- Run all acceptance tests without real Aura API credentials or live instances
- Preserve full coverage of provider CRUD operations: create, read, update, delete, import
- Exercise the provider's async polling logic (`WaitUntil`) against a simulated state machine
- Exercise the provider's OAuth2 token-acquisition and Bearer-token verification flow
- Remove the `neo4j-go-driver` dependency from the test package (no real Neo4j instances needed)
- Tests complete in seconds rather than minutes

## Non-Goals

- Replacing the live-infrastructure test path entirely — a separate CI job with real credentials can still be run on-demand or pre-release
- Mocking the Neo4j Bolt protocol or graph query responses
- Testing provider behaviour against Aura API edge cases not already exercised (e.g. 422 quota errors) — future work
- Changing any resource or data source schemas
- Building a standalone mock server binary or Docker image

## Requirements

### Functional Requirements

- REQ-F-001: `AuraClient` reads an optional `AURA_BASE_URL` environment variable to override `https://api.neo4j.io`. When the variable is unset the client behaves exactly as today. Both the API base path and the OAuth token URL must respect the override.
- REQ-F-002: An in-process mock server is implemented in `internal/test/mock_server.go` using Go's `net/http/httptest` package.
- REQ-F-003: The mock server handles `POST /oauth/token` and returns a valid `TokenResponse` JSON body containing a dummy access token and a non-zero `expires_in` value.
- REQ-F-004: The mock server verifies that all non-auth API calls carry an `Authorization: Bearer <token>` header matching the token it issued. Requests without a valid token receive HTTP 401.
- REQ-F-005: The mock server maintains in-memory state for instances (create, read, update, delete, pause, resume) and snapshots (create, read, list).
- REQ-F-006: The mock server implements a lightweight state machine for instance lifecycle: `POST /v1/instances` creates an instance in the `creating` state; the first `GET /v1/instances/{id}` returns `creating`; the second and subsequent GETs return `running`. This exercises the `WaitUntil` polling loop without any real sleep.
- REQ-F-007: The mock server implements a snapshot state machine: `POST /v1/instances/{id}/snapshots` creates a snapshot in `InProgress`; the first `GET /v1/instances/{id}/snapshots/{snapshotId}` returns `InProgress`; the second returns `Completed`.
- REQ-F-008: `GET /v1/instances/{id}` responses include preset non-nil `graph_nodes` and `graph_relationships` values once the instance reaches `running` state.
- REQ-F-009: `provider_test.go` starts the mock server in a `TestMain`, sets `AURA_BASE_URL` to the mock's address, and shuts it down after all tests complete. Provider config in tests uses fixed dummy values for `client_id` and `client_secret`.
- REQ-F-010: All existing test cases in `instance_resource_test.go`, `snapshot_resource_test.go`, `snapshot_test.go`, and `projects_test.go` are updated to run against the mock.
- REQ-F-011: The Cypher execution step and `graph_nodes`/`graph_relationships` count-assertion steps in `TestAcc_can_create_instance_resource` are removed. The test instead verifies that the provider correctly maps the preset values returned by the mock's `GET /v1/instances/{id}` response.
- REQ-F-012: `TestAcc_can_import_instance_resource` sub-cases that previously called `api.PostInstance` / `api.WaitUntilInstanceIsInState` directly are rewritten to pre-seed the mock's in-memory state instead.
- REQ-F-013: `neo4j_utils.go` and all Cypher-related test helpers are removed from the test package once no test references them.
- REQ-F-014: The `neo4j-go-driver/v5` dependency is removed from `go.mod` and `go.sum` (via `go mod tidy`) once no code outside the test package references it.

### Non-Functional Requirements

- REQ-NF-001: The full acceptance test suite completes in under 60 seconds on a standard developer machine.
- REQ-NF-002: Mock server code passes `go vet ./...` and `go build ./...` cleanly.
- REQ-NF-003: `TF_ACC=1 go test ./internal/test/...` passes with no real Aura credentials set.
- REQ-NF-004: The live-infrastructure path remains functional: unsetting `AURA_BASE_URL` and supplying real credentials must still work unchanged.

## Technical Considerations

- **`AuraClient` base URL override**: The constant `auraBasePath` in `aura_client.go` is used in `doOperation` and also referenced in `aura_auth.go` for the token URL. Making both respect `AURA_BASE_URL` requires plumbing the override through `NewAuraClient` so that `AuraAuth` receives the same base. The constant becomes the default; the override is passed at construction time.
- **Mock state and parallel tests**: `TestAcc_can_import_instance_resource` runs sub-tests in parallel. Each parallel sub-test must not interfere with others' mock state. The recommended approach is for each sub-test to generate a unique resource ID prefix, with the mock routing state by ID. Alternatively, each sub-test can start its own `httptest.Server` instance.
- **`WaitUntil` speed**: The waiter in `util/waiter.go` sleeps 1 second between polls. Because the mock transitions on the 2nd GET, each `WaitUntil` call incurs at most one 1-second sleep. Tests should pass a short timeout (e.g., 10 seconds) to `NewAuraApi` to fail fast if the mock misbehaves rather than waiting the default 900 seconds.
- **`TF_ACC` requirement**: `terraform-plugin-testing`'s `resource.Test` still requires `TF_ACC=1` to run. The `SkipIfNotAcceptance` guard can be removed from tests that previously skipped to avoid live costs, since mock-based tests are safe to run in any CI environment with `TF_ACC=1`.
- **Mock endpoints required**: All endpoints exercised by the existing provider code must be handled:
  - `POST /oauth/token`
  - `GET /v1/tenants`
  - `POST /v1/instances`
  - `GET /v1/instances/{id}`
  - `DELETE /v1/instances/{id}`
  - `PATCH /v1/instances/{id}`
  - `POST /v1/instances/{id}/pause`
  - `POST /v1/instances/{id}/resume`
  - `GET /v1/instances/{id}/snapshots`
  - `GET /v1/instances/{id}/snapshots/{snapshotId}`
  - `POST /v1/instances/{id}/snapshots`

## Acceptance Criteria

- [ ] `NewAuraClient` accepts a base URL parameter; `AURA_BASE_URL` env var is read and passed through, defaulting to `https://api.neo4j.io`
- [ ] Both the API base path and the OAuth token URL respect `AURA_BASE_URL`
- [ ] Mock server handles all eleven endpoints listed above
- [ ] Mock returns HTTP 401 for API calls that are missing a valid Bearer token
- [ ] Instance state machine: `creating` on first GET, `running` on second GET
- [ ] Snapshot state machine: `InProgress` on first GET, `Completed` on second GET
- [ ] `GET /v1/instances/{id}` in `running` state includes non-nil `graph_nodes` and `graph_relationships`
- [ ] `TF_ACC=1 go test ./internal/test/... -timeout 120s` completes without any Aura credentials
- [ ] Test suite completes in under 60 seconds
- [ ] `neo4j-go-driver` removed from `go.mod` and `go.sum`
- [ ] `go vet ./...` and `go build ./...` pass cleanly

## Out of Scope

- Aura API error-path coverage (quota exceeded, 422 validation errors, 5xx responses)
- Mocking `POST /v1/instances/{id}/restore` (not exercised by any existing test)
- A standalone mock binary or container for manual testing

## Open Questions

1. **Parallel test isolation strategy**: Should parallel sub-tests in `TestAcc_can_import_instance_resource` each start their own `httptest.Server`, or should a single shared server use resource-ID namespacing to prevent state collisions?
2. **`WaitUntil` timeout in tests**: Should tests construct `AuraApi` with a short timeout (e.g., 10 seconds) to fail fast against the mock, or is the existing 900-second default acceptable given the mock responds quickly?
3. **`SkipIfNotAcceptance` disposition**: Should all `SkipIfNotAcceptance` calls be removed (mock tests are cheap to run), or kept and replaced with a different guard that only skips when neither `TF_ACC` nor a mock is configured?
