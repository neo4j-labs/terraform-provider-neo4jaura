# PRD: Defects & Go Best Practices

## Overview

A comprehensive audit and remediation of `terraform-provider-neo4jaura` covering correctness defects, alignment to Go idioms and Terraform provider best practices, and production-readiness gaps. All changes are delivered on the single branch `feat/defects-and-go-best-practices`.

Linear project: [Terraform Provider: Defects & Go Best Practices](https://linear.app/neo4j/project/terraform-provider-defects-and-go-best-practices-34b64d244f53)

## Goals

- Fix all correctness defects that produce wrong state, silent drift, or incorrect API behaviour.
- Align Go code to idiomatic conventions (`gofmt`, `go vet`, standard patterns).
- Close production-readiness gaps that block a public registry release.

## Non-Goals

- Adding new resources, data sources, or Aura API endpoints.
- Changing public schema attribute names (breaking changes, e.g. `instance_id` → `id`).
- Migrating from terraform-plugin-framework to SDK v2.

---

## Requirements

### Functional Requirements — Correctness Defects

- **REQ-F-001 (DEF-001):** `util.WaitUntil` must respect context cancellation. Replace `time.Sleep(delay)` with a `select` on `time.After(delay)` and `ctx.Done()`. The function signature already receives a context via closure; `WaitUntil` itself must accept `ctx context.Context` as its first parameter and propagate cancellation.

- **REQ-F-002 (DEF-002):** Remove the dead `requestedStatus = data.Status` reassignment at `instance.go:511`. Remove the no-op `data.Status = types.StringValue(data.Status.ValueString())` at line 512 that silently converts a null/unknown status to an empty string.

- **REQ-F-003 (DEF-003):** In `Create`, set `data.Status` from the API response after the instance reaches its final state (set to `instance.Data.Status` after the wait). In `Read`, set `stateData.Status` from `instance.Data.Status` (it is already set on line 555, but verify it is not overwritten downstream). The goal is that state always reflects the actual API status after Create.

- **REQ-F-004 (DEF-004):** In `Update`, handle `cdc_enrichment_mode` changes. If `plan.CdcEnrichmentMode != state.CdcEnrichmentMode`, include `CdcEnrichmentMode` in the PATCH request. After patching, wait for `running` status before writing state.

- **REQ-F-005 (DEF-005):** In `Update`, split the PATCH conditions: only send the name/memory PATCH if name or memory changed; only send the secondaries PATCH if secondaries changed. Never send a PATCH for unchanged fields.

- **REQ-F-006 (DEF-006):** In `InstanceResource.Read`, detect HTTP 404 from `GetInstanceById` and call `response.State.RemoveResource(ctx)` instead of returning an error. This unblocks `terraform plan` when an instance is deleted externally.

- **REQ-F-007 (DEF-007):** In `datasource/snapshot.go:128`, replace `fmt.Errorf("missing required attribute: snapshot_id or most_recent").Error()` with the plain string literal `"missing required attribute: snapshot_id or most_recent"`.

- **REQ-F-008 (DEF-008):** In `instance.go:662`, remove the redundant `string(...)` cast around `domain.InstanceStatusPaused` — it is already a `string`.

- **REQ-F-009 (DEF-009):** Add `stringplanmodifier.RequiresReplace()` to the `source` nested attribute schema so that changes to the source block force replacement rather than an unsupported in-place update.

### Functional Requirements — Go & Terraform Best Practices

- **REQ-F-010 (BP-001):** Merge the two `import` blocks in `main.go` into one.

- **REQ-F-011 (BP-002):** Change the provider address in `main.go` from `terraform.local/local/aura` to `registry.terraform.io/neo4j-labs/neo4jaura`.

- **REQ-F-012 (BP-003):** Merge the two `const` blocks in `aura_client.go` into one.

- **REQ-F-013 (BP-004):** Rewrite the indexed `for` loop in `datasource/projects.go:116` as `for i, t := range tenantsResponse.Data`.

- **REQ-F-014 (BP-005):** In `aura_api.go:WaitUntilInstanceIsDeleted`, remove the named return and bare `return`; use an explicit `return err` pattern.

- **REQ-F-015 (BP-006):** Change `mutex *sync.Mutex` in `AuraAuth` to a value `sync.Mutex` and update `NewAuraClient` accordingly (remove `&sync.Mutex{}`).

- **REQ-F-016 (BP-007):** Add a space after `//` in `aura_auth.go:69` (`//Check` → `// Check`).

- **REQ-F-017 (BP-008):** Extract a shared helper (e.g. `isStatusRunning`, `isStatusPaused`) to eliminate the duplicated `CanBePaused`/`CanBeResumed` logic between `InstanceResourceModel` and `GetInstanceData`.

### Functional Requirements — Production Readiness

- **REQ-F-018 (PR-001):** In `provider.go`, make `client_id` and `client_secret` `Optional` (not `Required`) and add env var fallback in `Configure`: if the config value is empty, fall back to `os.Getenv("AURA_CLIENT_ID")` / `os.Getenv("AURA_CLIENT_SECRET")`. If both are still empty after fallback, add a `Diagnostics.AddError`.

- **REQ-F-019 (PR-003):** In `datasource/snapshot.go:readSnapshotById`, replace the `GetSnapshotsByInstanceId` call (which fetches all snapshots) with a direct `GetSnapshotById(ctx, instanceId, snapshotId)` call. Return a `*client.GetSnapshotData` constructed from the response data.

### Non-Functional Requirements

- **REQ-NF-001:** All changes pass `go build ./...` and `go vet ./...` with no errors.
- **REQ-NF-002:** `TF_ACC=1 go test ./internal/test/... -timeout 120s` passes with no failures (mock server path).
- **REQ-NF-003:** No public schema attribute names are changed (no breaking changes).

---

## Technical Considerations

### DEF-001 — WaitUntil signature change
Adding `ctx context.Context` as the first parameter of `WaitUntil` is a call-site change. All callers are internal (`aura_api.go`). The change is mechanical but touches several call sites — update all at once to keep `go build` green.

### DEF-006 — 404 in Read
`GetInstanceById` returns `fmt.Errorf("aura error: Status: 404...")` — a string error, not a typed sentinel. The 404 check must parse the status code from the error message or, better, `AuraClient.Get` should return the HTTP status code alongside the error (it already does — `GetInstanceById` wraps it). Refactor `GetInstanceById` to return a distinguishable not-found error (e.g. a sentinel `ErrNotFound` or by returning status code directly to the caller).

### DEF-009 — `source` RequiresReplace
`source` is a `SingleNestedAttribute`. Plan modifiers on nested attributes are set via `PlanModifiers` on the `schema.SingleNestedAttribute` itself. Verify the correct import path for `objectplanmodifier.RequiresReplace()`.

### PR-001 — Optional credentials
Making `client_id`/`client_secret` `Optional` means the schema validator won't enforce their presence. The `Configure` method must explicitly validate that at least one path (config or env var) provided a value.

### DEF-005 — Separate PATCH calls
The current code sends name+memory together and secondaries separately in all cases. Split into: (a) PATCH name/memory only if changed, (b) PATCH secondaries only if changed, (c) wait once after all patches.

---

## Acceptance Criteria

- [ ] DEF-001: Ctrl+C during a `WaitUntil` poll exits immediately; timeout test passes
- [ ] DEF-002: Lines 511–512 removed; `go vet` clean
- [ ] DEF-003: After `Create`, `data.Status` in state equals `instance.Data.Status` from the API
- [ ] DEF-004: Changing `cdc_enrichment_mode` in config triggers a PATCH and the new value is confirmed in state
- [ ] DEF-005: Changing only `name` sends one PATCH (name+memory); secondaries PATCH is not sent
- [ ] DEF-006: `terraform plan` on a provider-managed instance that was deleted externally removes the resource from state without error
- [ ] DEF-007: `datasource/snapshot.go:128` uses a plain string, no `fmt.Errorf` wrapping
- [ ] DEF-008: No `string(...)` cast on `domain.InstanceStatusPaused`
- [ ] DEF-009: Changing the `source` block in config produces a "must replace" plan diff
- [ ] BP-001: Single import block in `main.go`; `goimports` clean
- [ ] BP-002: Provider address is `registry.terraform.io/neo4j-labs/neo4jaura`
- [ ] BP-003: Single const block in `aura_client.go`
- [ ] BP-004: `datasource/projects.go` uses range loop
- [ ] BP-005: `WaitUntilInstanceIsDeleted` uses explicit `return err`
- [ ] BP-006: `AuraAuth.mutex` is a value `sync.Mutex`
- [ ] BP-007: Space after `//` in `aura_auth.go`
- [ ] BP-008: Shared pause/resume status helpers with no duplication
- [ ] PR-001: Setting `AURA_CLIENT_ID`/`AURA_CLIENT_SECRET` env vars and omitting config block works end-to-end
- [ ] PR-003: `readSnapshotById` calls `GetSnapshotById` directly
- [ ] `go build ./...` and `go vet ./...` pass
- [ ] `TF_ACC=1 go test ./internal/test/... -timeout 120s` passes

---

## Out of Scope

- Renaming `instance_id` to `id` (breaking schema change — separate release)
- Adding new resources (Aura users, firewall rules, etc.)
- Changing the module path or provider name
- Performance testing or load testing

---

## Open Questions

1. **DEF-001 signature change**: Should `WaitUntil` accept `ctx` directly, or should callers pass it via closure? Direct parameter is cleaner for context propagation.
2. **DEF-006 error sentinel**: Should we define `var ErrNotFound = errors.New("not found")` in the `client` package, or parse the "Status: 404" string in `Read`? A typed sentinel is cleaner.
3. **PR-001 schema**: Should `Optional: true` + env var fallback be the approach, or use `schema.StringAttribute` with `DefaultFunc` (SDK v2 pattern)? Plugin Framework v1 doesn't have `DefaultFunc`; env var fallback in `Configure` is the correct pattern.
