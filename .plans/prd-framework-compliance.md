# PRD: Framework Compliance & Best-Practice Remediation

## Overview

A comprehensive audit of `terraform-provider-neo4jaura` against `terraform-plugin-framework`
best practices, HashiCorp provider design principles, and the project's own skill guide
identified 15 gaps across correctness, schema design, provider configuration, test coverage,
and CI. This PRD covers the full remediation set, excluding new API features and excluding the
per-resource timeout refactor (tracked separately as it requires a state upgrade and semver bump).

## Goals

- Eliminate framework correctness issues that cause incorrect Terraform behaviour (wrong 404
  handling, plan drift from Update writing plan not API state, missing plan modifiers).
- Bring the provider into full compliance with HashiCorp provider design principles.
- Improve credential UX by supporting environment variable fallbacks.
- Complete acceptance test coverage with update, disappears, CheckDestroy, and PreCheck.
- Add CI linting so framework misuse is caught automatically on every PR.

## Non-Goals

- New resources or data sources (instance data source, list-instances data source, IP allowlist).
- Per-resource timeout refactor — tracked separately; requires `StateUpgrader` and semver bump.
- Performance optimisations.
- Refactoring beyond the minimal fix required for each gap.

## Requirements

### Functional Requirements

- REQ-F-001: `InstanceResource.Read` must call `resp.State.RemoveResource(ctx)` and return
  without error when the API returns 404 for the instance ID.
- REQ-F-002: `SnapshotResource.Read` must call `resp.State.RemoveResource(ctx)` and return
  without error when the API returns 404 for the snapshot ID.
- REQ-F-003: `AuraClient` (or `AuraApi`) must expose the HTTP status code alongside errors so
  resource Read methods can distinguish 404 from other failures.
- REQ-F-004: The `source` `SingleNestedAttribute` on `InstanceResource` must carry an
  `objectplanmodifier.RequiresReplace()` plan modifier — changing `source` after creation has
  no effect and must force resource replacement.
- REQ-F-005: `neo4jaura_projects` data source must expose a root `id` `Computed` attribute
  (set to a constant such as `"projects"`) to satisfy HashiCorp data source conventions.
- REQ-F-006: `provider.Configure` must fall back to `AURA_CLIENT_ID` and `AURA_CLIENT_SECRET`
  environment variables when `client_id` / `client_secret` are null or unknown in the config
  block (matching the existing `AURA_BASE_URL` pattern).
- REQ-F-007: `InstanceResource.Update` must re-read the instance from the API after waiting
  for the running state and write all fields from that response to state, rather than writing
  the plan directly — preventing computed-field drift for fields like `storage` and
  `metrics_integration_url`.
- REQ-F-008: Every `AddError` call in resource and datasource code must include the relevant
  resource ID (instance ID, snapshot ID) in the detail string.
- REQ-F-009: Every `resp.State.Set` / `resp.State.SetAttribute` call must be followed
  immediately by a `HasError()` guard and early return.
- REQ-F-010: `main.go` registry address must be `registry.terraform.io/neo4j-labs/neo4jaura`.
- REQ-F-011: `InstanceResource` acceptance tests must include a dedicated multi-step update
  test covering at least one in-place name/memory change.
- REQ-F-012: `InstanceResource` acceptance tests must include a disappears test: create the
  instance, delete it out-of-band via mock server state reset, verify the next plan proposes
  recreation rather than erroring.
- REQ-F-013: All `resource.TestCase` entries must include a `PreCheck` field that calls a
  `testAccPreCheck(t)` helper validating required environment variables.
- REQ-F-014: All `resource.TestCase` entries must include a `CheckDestroy` field that verifies
  via the Aura API (or mock) that the resource no longer exists after `terraform destroy`.
- REQ-F-015: `build.yml` CI workflow must include a lint job that runs `golangci-lint run`
  and `tfproviderlint ./...`, failing the build on any reported issues.

### Non-Functional Requirements

- REQ-NF-001: All existing mock-acceptance tests must continue to pass after each fix.
- REQ-NF-002: Each fix group must be accompanied by a changelog entry under
  `.changes/unreleased/`.
- REQ-NF-003: The `golangci-lint` configuration must include at minimum `staticcheck`,
  `errcheck`, `govet`, and `SA6005` (strings.EqualFold).

## Technical Considerations

### 404 Detection (REQ-F-001 to REQ-F-003)

`AuraClient.Get` uses `go-retryablehttp` and currently returns a plain `error` on non-200
responses with no way to inspect the status code at the resource layer. The minimal fix is to
inspect the HTTP response body or wrap the error with the status code. Options:
- Introduce a sentinel `ErrNotFound` error that `AuraClient` returns on 404, which the
  resource checks with `errors.Is`.
- Or return `(response, statusCode, error)` from `Get` — but this changes all call sites.
  The sentinel error approach is simpler and requires fewer call-site changes.

### `source` RequiresReplace (REQ-F-004)

`objectplanmodifier.RequiresReplace()` is available from `terraform-plugin-framework`. Because
`source` is only consumed during POST and ignored on all subsequent operations, any change to
`source` after creation should always trigger full replacement.

### Update → API refresh (REQ-F-007)

`InstanceResource.Update` currently ends with `resp.State.Set(ctx, &plan)`. Replace with a
`GetInstanceById` call after `WaitUntilInstanceIsInState` and populate all fields from the
response, matching the pattern in `Create`.

### Disappears test (REQ-F-012)

Because the mock server is reset between tests (not between steps), the disappears scenario
must be implemented by calling `testMockServer.DeleteInstance(id)` (or equivalent) inside the
`Check` function of the first step, then relying on the next step's refresh to observe the 404.
This requires REQ-F-001 to land first (otherwise the refresh errors instead of detecting drift).

### CI Linting (REQ-F-015)

`tfproviderlint` requires `go install github.com/bflad/tfproviderlint/cmd/tfproviderlint@latest`
and is invoked as `tfproviderlint ./...`. It checks for missing `id` attributes, missing
`UseStateForUnknown`, incorrect import handling, and other framework-specific patterns.
`golangci-lint` can be configured via `.golangci.yml` in the repo root.

## Acceptance Criteria

- [ ] REQ-F-001: `terraform refresh` on a manually deleted instance produces a plan to recreate it, not an error.
- [ ] REQ-F-002: `terraform refresh` on a manually deleted snapshot produces an empty plan, not an error.
- [ ] REQ-F-003: The resource layer can distinguish 404 from 5xx without parsing error strings.
- [ ] REQ-F-004: Changing the `source` block on an existing `neo4jaura_instance` resource causes `terraform plan` to show `forces replacement`.
- [ ] REQ-F-005: `neo4jaura_projects` data source has a root `id` attribute in its schema and sets it in `Read`.
- [ ] REQ-F-006: Setting only `AURA_CLIENT_ID` and `AURA_CLIENT_SECRET` env vars (no HCL provider block attributes) configures the provider successfully.
- [ ] REQ-F-007: After an in-place update, computed fields in state reflect the API response, not the plan.
- [ ] REQ-F-008: No `AddError` call in `internal/resource/` or `internal/datasource/` lacks the resource ID in its detail string.
- [ ] REQ-F-009: No `resp.State.Set` call is missing a following `HasError()` guard.
- [ ] REQ-F-010: `main.go` uses `registry.terraform.io/neo4j-labs/neo4jaura` as the provider address.
- [ ] REQ-F-011: `TestAcc_instance_update` exists and verifies at least one field is updated in-place.
- [ ] REQ-F-012: `TestAcc_instance_disappears` exists and verifies recreation is planned after out-of-band deletion.
- [ ] REQ-F-013: Every `TestCase` has `PreCheck: func() { testAccPreCheck(t) }`.
- [ ] REQ-F-014: Every `TestCase` has `CheckDestroy: testAccCheckInstanceDestroyed` (or equivalent).
- [ ] REQ-F-015: CI lint job runs and fails on `tfproviderlint` or `golangci-lint` violations.
- [ ] All mock-acceptance tests pass (`make mock-acceptance`).

## Recommended Resolution Order

**Group 1 — 404 infrastructure (REQ-F-003, REQ-F-001, REQ-F-002):** Must land first; disappears tests depend on it.

**Group 2 — Schema & plan modifiers (REQ-F-004, REQ-F-005):** Standalone schema changes; low risk.

**Group 3 — Provider configure & registry address (REQ-F-006, REQ-F-010):** Standalone; no dependencies.

**Group 4 — Update API refresh & error quality (REQ-F-007, REQ-F-008, REQ-F-009):** Fix Update correctness and harden all call sites.

**Group 5 — Test coverage (REQ-F-011, REQ-F-012, REQ-F-013, REQ-F-014):** Requires Group 1 for disappears test.

**Group 6 — CI linting (REQ-F-015):** Add last so the new lint gate doesn't block other groups mid-PR.

## Out of Scope

- New data sources (instance, list-instances).
- IP allowlist, export, or restore API coverage.
- Per-resource timeout refactor (breaking schema change — separate PRD/PR).
- `status` Optional+Computed design refactor (functional, not a compliance issue).
- Write-only `password` attribute (framework 1.11+ feature; low priority).

## Open Questions

- For REQ-F-003: sentinel `errors.Is(err, ErrNotFound)` vs. returning status code — which approach fits better with the existing `go-retryablehttp` error wrapping?
- For REQ-F-012: does `MockServer` need a new `DeleteInstance(id)` method, or can tests reset and re-seed without the existing instance?
- For REQ-F-015: should `tfproviderlint` run as a separate job or be folded into the existing `unit-tests` job?
