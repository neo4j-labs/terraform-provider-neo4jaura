# PRD: Review of Terraform Provider for Neo4j Aura

## Overview

A comprehensive review of the `terraform-provider-neo4jaura` codebase against Go language standards and Terraform provider best practices. The review identifies defects (bugs affecting correctness), gaps (missing standard capabilities), and improvements (code quality and maintainability). Changes will be delivered on separate branches, with defects addressed first.

The provider manages Neo4j Aura infrastructure via the [Aura v1 API](https://neo4j.com/docs/aura/platform/api/specification/?urls.primaryName=Aura%20v1), exposing resources for instances and snapshots, and data sources for projects and snapshots.

---

## Goals

- Identify and document all defects that cause incorrect provider behaviour
- Identify gaps against Terraform provider standards and Go conventions
- Define clear, actionable remediation for each finding
- Prioritise defects over improvements; deliver each as a focused branch

## Non-Goals

- Adding new Aura API resources or endpoints beyond what is already partially modelled
- UI/UX changes to the Terraform Registry documentation
- Changing the public provider API (resource/attribute names) unless required for standards compliance

---

## Findings

### Defects

Defects are bugs that cause incorrect, unsafe, or undefined behaviour.

#### DEF-001 — `Read` method loses GraphNodes/GraphRelationships on parse error

**File:** [internal/resource/instance.go](internal/resource/instance.go) lines 551–576  
**Severity:** High

When `strconv.ParseInt` fails for `graph_nodes` or `graph_relationships` in the `Read` method, the code sets the state field to `null` and then **immediately overwrites it** with `types.Int64Value(graphNodes)` (which will be `0`), because there is no `else` guard. The `Create` method has this correct (with `else`). The `Read` method is missing the `else`.

Additionally, the error message for `graphRelationships` parse failure in `Create` references `*instance.Data.GraphNodes` instead of `*instance.Data.GraphRelationships`.

**Fix:** Add `else` guards to `Read` to match the pattern in `Create`. Fix the wrong field reference in the warning message.

---

#### DEF-002 — `GetSnapshotsByInstanceId` and `GetSnapshotById` swallow non-200 errors

**File:** [internal/client/aura_api.go](internal/client/aura_api.go) lines 150–168  
**Severity:** High

Both methods check `if status != 200` but return `err` (which is `nil` at that point) instead of a formatted error. The caller silently receives an empty response and no error signal, causing misleading or silent failures.

```go
// Current (wrong):
if status != 200 {
    return GetSnapshotsResponse{}, err  // err is nil here
}

// Required (consistent with rest of codebase):
if status != 200 {
    return GetSnapshotsResponse{}, fmt.Errorf("aura error: Status: %+v. Response: %+v", status, string(body))
}
```

**Fix:** Match the error-return pattern used by all other `AuraApi` methods.

---

#### DEF-003 — `SnapshotResource.Delete` does not remove state

**File:** [internal/resource/snapshot.go](internal/resource/snapshot.go) lines 172–174  
**Severity:** High

`Delete` logs a message but does not call `response.State.RemoveResource(ctx)`. Terraform will therefore continue tracking the "deleted" snapshot in state, causing persistent drift and broken plans.

**Fix:** Add `response.State.RemoveResource(ctx)` to `Delete`. Since snapshots cannot be deleted via the API, the resource should document this behaviour clearly and remove it from Terraform state when `terraform destroy` is called.

---

#### DEF-004 — Immutable fields missing `RequiresReplace` plan modifiers

**File:** [internal/resource/instance.go](internal/resource/instance.go)  
**Severity:** High

The following attributes cannot be changed in-place on an existing Aura instance (the API does not support it), but they use `UseStateForUnknown()` rather than `RequiresReplace()`. Changing them will cause the provider to send an unsupported PATCH request, resulting in an API error or silent state drift:

- `region`
- `type`
- `cloud_provider`
- `version`
- `project_id`

**Fix:** Replace `UseStateForUnknown()` with `stringplanmodifier.RequiresReplace()` (and remove `UseStateForUnknown`) on all five attributes. For `project_id`, remove the plan modifier entirely and just add `RequiresReplace`.

---

#### DEF-005 — `PatchInstanceRequest` JSON tags missing `omitempty`

**File:** [internal/client/aura_requests.go](internal/client/aura_requests.go) lines 37–40  
**Severity:** Medium

`Name` and `Memory` fields in `PatchInstanceRequest` have no `omitempty` tag. If either field is nil, the JSON body will contain `"name": null` or `"memory": null`, which may be rejected by the API or unintentionally nullify those fields.

**Fix:** Add `omitempty` to both fields:
```go
type PatchInstanceRequest struct {
    Name   *string `json:"name,omitempty"`
    Memory *string `json:"memory,omitempty"`
}
```

---

#### DEF-006 — Dead/misleading `requestedStatus` reassignment in `Create`

**File:** [internal/resource/instance.go](internal/resource/instance.go) lines 417 and 496–497  
**Severity:** Low

`requestedStatus` is assigned at line 417, then re-assigned from the same unchanged `data.Status` value at line 496. Line 497 then sets `data.Status` to itself as a no-op. This redundant code obscures intent and could hide future logic errors.

**Fix:** Remove the redundant second assignment (lines 496–497). Keep the original assignment at line 417.

---

### Gaps Against Standards

#### GAP-001 — No `ImportState` implementation

**File:** [internal/resource/instance.go](internal/resource/instance.go)  
**Standard:** Terraform provider best practice

Without implementing `resource.ResourceWithImportState`, users cannot import existing Aura instances into Terraform state with `terraform import`. This is a standard capability expected in production-grade providers.

**Fix:** Implement `ImportState` on `InstanceResource` using `resource.ImportStatePassthroughID` to map the imported ID to `instance_id`, then call `Read` to populate full state.

---

#### GAP-002 — No environment variable support for credentials

**File:** [internal/provider/provider.go](internal/provider/provider.go)  
**Standard:** Terraform provider best practice

The provider only reads `client_id` and `client_secret` from the provider block. Standard practice is to also check environment variables (e.g. `AURA_CLIENT_ID`, `AURA_CLIENT_SECRET`) so credentials do not need to be hardcoded in configuration.

**Fix:** In the `Configure` method, fall back to `os.Getenv("AURA_CLIENT_ID")` / `os.Getenv("AURA_CLIENT_SECRET")` when the config values are empty. Document this in the provider schema descriptions.

---

#### GAP-003 — No acceptance tests

**Standard:** Terraform provider best practice; Go testing conventions

The repository has zero test files. Terraform provider acceptance tests (`_test.go` files using `resource.Test`) are the standard mechanism for verifying end-to-end CRUD behaviour. The absence of tests makes defects difficult to detect and regressions likely.

**Fix:** Add acceptance tests for `InstanceResource` and `SnapshotResource` CRUD operations using the Terraform Plugin Framework testing helpers. These tests require a real Aura API key and should be gated behind a `TF_ACC=1` environment variable (standard convention).

---

#### GAP-004 — `instance_id` attribute deviates from Terraform convention

**File:** [internal/resource/instance.go](internal/resource/instance.go)  
**Standard:** Terraform provider convention

Terraform convention names the primary resource identifier `id`. Using `instance_id` breaks compatibility with Terraform's built-in `id` attribute and tooling that expects `id` to exist on all resources. Users referencing the instance ID must use `neo4jaura_instance.example.instance_id` rather than the conventional `.id`.

**Fix:** Rename `instance_id` to `id` and update all internal usages, examples, and documentation. This is a breaking schema change and should be considered in versioning.

---

#### GAP-005 — `WaitUntil` does not respect context cancellation

**File:** [internal/util/waiter.go](internal/util/waiter.go)  
**Standard:** Go context propagation conventions

`WaitUntil` uses `time.Sleep(delay)` without checking the passed context. If Terraform sends a cancellation signal (e.g. user presses Ctrl+C), the provider will continue polling until its timeout expires rather than stopping immediately.

**Fix:** Replace `time.Sleep` with a `select` on a `time.After(delay)` channel and `ctx.Done()`, so the waiter returns immediately on cancellation. The function signature should accept `ctx context.Context` directly (it is already available via closure but the pattern is fragile).

---

### Improvements

#### IMP-001 — Duplicate import blocks in `main.go`

**File:** [main.go](main.go) lines 3–13  
**Standard:** `goimports` / `gofmt` convention

Go convention is a single import block (optionally grouped with blank lines). Having two separate `import` declarations is non-idiomatic and rejected by `goimports`.

**Fix:** Merge into a single import block.

---

#### IMP-002 — Version hardcoded in two locations

**Files:** [main.go](main.go) line 16, [internal/client/aura_client.go](internal/client/aura_client.go) line 34  
**Standard:** Go build toolchain convention

The version string `"0.0.1-beta"` is duplicated. The `main.go` variable is already designed to be set via `-ldflags` at build time (see `.goreleaser.yml`), but `aura_client.go` has a hardcoded copy in the `userAgent` constant that will never update.

**Fix:** Pass `version` through to `NewAuraClient` so the `User-Agent` header stays in sync. Remove the hardcoded string from `aura_client.go`.

---

#### IMP-003 — Provider registry address is a local development address

**File:** [main.go](main.go) line 25  
**Standard:** Terraform provider convention

`Address: "terraform.local/local/aura"` is a local development override. The published registry address (`registry.terraform.io/neo4j-labs/neo4jaura`) should be the default, with local override achieved via `.terraformrc` (as documented in the README).

**Fix:** Set `Address` to `"registry.terraform.io/neo4j-labs/neo4jaura"` (or derive it from a build-time variable) in `providerserver.ServeOpts`.

---

#### IMP-004 — Duplicate response-to-model mapping between `Create` and `Read`

**File:** [internal/resource/instance.go](internal/resource/instance.go) lines 431–494 and 529–598  
**Standard:** Go DRY principle

The logic to map `GetInstanceData` fields onto `InstanceResourceModel` is duplicated between `Create` and `Read`. This caused DEF-001 (the bug existed in `Read` but not `Create` because they were not kept in sync).

**Fix:** Extract a helper function `populateInstanceModelFromData(data *InstanceResourceModel, instance GetInstanceData)` and call it from both methods.

---

#### IMP-005 — Naming inconsistency: `projects` vs `tenants`

**Files:** [internal/datasource/projects.go](internal/datasource/projects.go), [internal/client/aura_api.go](internal/client/aura_api.go), [internal/client/aura_responses.go](internal/client/aura_responses.go)  
**Standard:** Consistent naming

The data source is named `projects` (matching the Aura UI terminology), but the API endpoint and internal types use `tenants`. This mismatch confuses contributors. The response type `GetProjectsResponse` calls the API endpoint `tenants`.

**Fix:** Internally align naming to `tenants` (matching the API) or `projects` (matching the Terraform surface). Pick one and apply it consistently across internal types and method names. Since the public Terraform attribute is `projects`, internal naming should follow that convention.

---

#### IMP-006 — Snapshot `profile` description has a typo

**File:** [internal/resource/snapshot.go](internal/resource/snapshot.go) line 99, [internal/datasource/snapshot.go](internal/datasource/snapshot.go) line 79  
**Standard:** Documentation accuracy

The profile description says `"One of [AddHoc, Scheduled]"`. The correct term is `"AdHoc"` (or `"OnDemand"` per some API versions). This should be verified against the Aura API spec and corrected.

**Fix:** Verify the correct value from the API spec and update both schema descriptions.

---

## Requirements

### Functional Requirements

- REQ-F-001: Fix DEF-001 — `Read` method `GraphNodes`/`GraphRelationships` parse error handling
- REQ-F-002: Fix DEF-002 — `GetSnapshotsByInstanceId` and `GetSnapshotById` non-200 error responses
- REQ-F-003: Fix DEF-003 — `SnapshotResource.Delete` must remove resource from state
- REQ-F-004: Fix DEF-004 — Add `RequiresReplace` to immutable instance attributes
- REQ-F-005: Fix DEF-005 — Add `omitempty` to `PatchInstanceRequest` JSON tags
- REQ-F-006: Fix DEF-006 — Remove redundant `requestedStatus` reassignment
- REQ-F-007: Address GAP-001 — Implement `ImportState` for `InstanceResource`
- REQ-F-008: Address GAP-002 — Support env vars for `client_id` and `client_secret`
- REQ-F-009: Address GAP-003 — Add acceptance tests for CRUD operations
- REQ-F-010: Address GAP-004 — Rename `instance_id` to `id` (breaking change, semver bump required)
- REQ-F-011: Address GAP-005 — Make `WaitUntil` respect context cancellation

### Non-Functional Requirements

- REQ-NF-001: Each fix delivered on a separate branch; defects (DEF-*) before gaps (GAP-*) before improvements (IMP-*)
- REQ-NF-002: No regressions introduced to existing example configurations
- REQ-NF-003: All Go code passes `go vet` and `gofmt`

---

## Technical Considerations

- **Breaking changes:** GAP-004 (`id` rename) is a breaking schema change. Existing users will need to update their configurations and run `terraform state mv`. This should be co-ordinated with a major or minor version bump.
- **Test infrastructure:** GAP-003 acceptance tests require real Aura API credentials. A test project/tenant should be designated for CI use.
- **`WaitUntil` refactor (GAP-005):** The generic function signature change is backwards-compatible within Go but touches multiple call sites. Care should be taken to preserve existing polling semantics.
- **DEF-003 (SnapshotResource.Delete):** Snapshots in Aura cannot be deleted via the API. The correct Terraform pattern here is to remove the resource from state on destroy, acknowledging the snapshot persists in Aura. This should be clearly documented.

---

## Acceptance Criteria

- [ ] DEF-001: `Read` method correctly handles GraphNodes/GraphRelationships parse failure without overwriting null with 0
- [ ] DEF-001: Error message for GraphRelationships parse failure in `Create` references the correct field
- [ ] DEF-002: `GetSnapshotsByInstanceId` and `GetSnapshotById` return a formatted error on non-200 status
- [ ] DEF-003: `SnapshotResource.Delete` calls `response.State.RemoveResource(ctx)`
- [ ] DEF-004: `region`, `type`, `cloud_provider`, `version`, `project_id` all have `RequiresReplace` plan modifiers
- [ ] DEF-005: `PatchInstanceRequest.Name` and `.Memory` have `omitempty` JSON tags
- [ ] DEF-006: Redundant `requestedStatus` lines removed from `Create`
- [ ] GAP-001: `terraform import` works for `neo4jaura_instance`
- [ ] GAP-002: Provider accepts credentials from `AURA_CLIENT_ID` / `AURA_CLIENT_SECRET` env vars
- [ ] GAP-003: Acceptance tests exist and pass for instance and snapshot CRUD
- [ ] GAP-004: `instance_id` renamed to `id` with migration guidance documented
- [ ] GAP-005: `WaitUntil` returns immediately on context cancellation
- [ ] IMP-001 to IMP-006: All improvements applied
- [ ] `go vet ./...` and `go build ./...` pass cleanly

---

## Out of Scope

- Adding new resource types (e.g. Aura users, firewall rules, CDN)
- Migrating from terraform-plugin-framework to SDK v2
- Changing the provider name or module path

---

## Open Questions

1. **GAP-004 versioning:** Should `instance_id` → `id` be delivered in a patch alongside defect fixes, or held for a dedicated breaking-change release? Given this is still `0.0.1-beta`, a v0.1.0 release may be appropriate.
2. **Snapshot delete behaviour:** Should `SnapshotResource` remain a managed resource (with explicit destroy behaviour) or be converted to a data source? The current model allows snapshot creation via `terraform apply` but cannot clean up via `terraform destroy`.
3. **DEF-004 — `storage`:** Is `storage` also immutable post-creation? The PATCH endpoint only accepts `name` and `memory`. If so, it should also get `RequiresReplace`.
