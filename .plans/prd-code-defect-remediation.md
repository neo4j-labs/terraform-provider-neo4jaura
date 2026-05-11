# PRD: Code Defect Remediation

## Overview

A systematic review of the `terraform-provider-neo4jaura` codebase identified 9 confirmed defects across the resource, datasource, client, and test layers. Several are high-severity bugs that can leave Terraform state in an inconsistent or incorrect condition after failed API operations. This PRD covers the full remediation set, prioritised for sequential resolution.

## Goals

- Eliminate all confirmed bugs that can cause incorrect Terraform state, unexpected panics, or ignoring user cancellation signals.
- Improve reliability of the sort, wait, and update flows.
- Remove magic strings that diverge from domain constants.
- Make test infrastructure reliably surface encoding failures.

## Non-Goals

- New features or schema changes.
- Performance optimisations beyond fixing the context-cancellation issue.
- Refactoring that goes beyond the minimal fix required for each defect.

## Requirements

### Functional Requirements

- REQ-F-001: `InstanceResource.Create` must return early when `WaitUntilSnapshotIsInState` fails (before attempting instance creation).
- REQ-F-002: `InstanceResource.Create` must return early when the first `WaitUntilInstanceIsInState` fails (before using the zero-value response).
- REQ-F-003: `InstanceResource.Update` must not send the name/memory PATCH when only `secondaries_count` changed; and if the second PATCH fails the diagnostic error must clearly indicate a partial update.
- REQ-F-004: The `sort.Slice` comparator in `readMostRecentSnapshot` must handle timestamp parse errors without violating the antisymmetry invariant (return `false, error` and abort sorting).
- REQ-F-005: `WaitUntil` must select on `ctx.Done()` rather than calling `time.Sleep` so that Terraform plan cancellations interrupt the wait loop promptly.
- REQ-F-006: `InstanceResource.Create` must persist `instance.Data.Status` (the actual API status) into `data.Status` before writing state; the redundant lines 511–512 must be removed.
- REQ-F-007: All snapshot status comparisons must use `strings.ToLower(domain.SnapshotStatusCompleted)` or the constant directly, not the magic string `"completed"`.
- REQ-F-008: `writeJSON` in the mock server must log or panic on JSON encoding failure rather than silently discarding it.
- REQ-F-009: `SnapshotDataSource.Read` must return early when `snapshot == nil` rather than calling `State.Set` with zero values.

### Non-Functional Requirements

- REQ-NF-001: All existing mock-acceptance tests must continue to pass after each fix.
- REQ-NF-002: Each fix must be accompanied by a changelog entry under `.changes/unreleased/`.

## Technical Considerations

### Defect Details

| ID | Severity | File | Lines | Description |
|----|----------|------|-------|-------------|
| D-01 | High | `internal/resource/instance.go` | 379–387 | Missing `return` after `WaitUntilSnapshotIsInState` error; instance creation proceeds with a failed source snapshot |
| D-02 | High | `internal/resource/instance.go` | 418–423 | Missing `return` after first `WaitUntilInstanceIsInState` error; zero-value `GetInstanceResponse` used for state population |
| D-03 | High | `internal/resource/instance.go` | 639–656 | Update always issues two unconditional PATCH calls; second PATCH failure leaves partial state with no rollback indication |
| D-04 | Medium | `internal/datasource/snapshot.go` | 184–196 | `sort.Slice` comparator returns `true` for both `less(i,j)` and `less(j,i)` on parse error, violating antisymmetry and producing undefined sort order |
| D-05 | Medium | `internal/util/waiter.go` | 25–37 | `WaitUntil` calls `time.Sleep` with no `ctx.Done()` select; Terraform cancellation is not respected during wait intervals |
| D-06 | Medium | `internal/resource/instance.go` | 511–512 | `data.Status` never set from API response in Create path; redundant/no-op assignments overwrite planned value with itself |
| D-07 | Low | `internal/resource/snapshot.go:152`, `instance.go:380` | — | Magic string `"completed"` instead of `strings.ToLower(domain.SnapshotStatusCompleted)` |
| D-08 | Low | `internal/test/mock_server.go` | 452–456 | `writeJSON` silently discards JSON encoding errors, masking test failures |
| D-09 | Low | `internal/datasource/snapshot.go` | 132–139 | `State.Set` called with zero values when `snapshot == nil` instead of returning early |

### Recommended Resolution Order

**Group 1 — Missing returns (fix together, one PR):** D-01, D-02  
Both are missing `return` after error diagnostics in `InstanceResource.Create`. Fixing together avoids two separate diffs on adjacent code.

**Group 2 — Update partial-patch (standalone):** D-03  
Requires careful logic change: split PATCHes into conditional sends (name/memory only when changed, secondaries only when changed) and add a note in diagnostics if the second PATCH fails after the first succeeds.

**Group 3 — Sort correctness + Context cancel (fix together):** D-04, D-05  
Both affect correctness under failure/cancellation paths. D-05 requires adding `ctx context.Context` to `WaitUntil` signature; update all callers.

**Group 4 — Create status + Magic strings (fix together):** D-06, D-07  
Both touch status handling. Set `data.Status = types.StringValue(instance.Data.Status)` in Create, and replace magic strings with domain constants.

**Group 5 — Low severity (fix together):** D-08, D-09  
Test infrastructure and datasource early-return. Safe to batch.

## Acceptance Criteria

- [ ] D-01: If `WaitUntilSnapshotIsInState` returns an error, `Create` returns immediately; no `PostInstance` call is made.
- [ ] D-02: If `WaitUntilInstanceIsInState` returns an error on first call in `Create`, `Create` returns immediately; no nil/zero-value fields written to state.
- [ ] D-03: When only `name`/`memory` changes, exactly one PATCH is sent. When only `secondaries_count` changes, exactly one PATCH is sent. If one PATCH fails, the error message indicates which update succeeded.
- [ ] D-04: Snapshot data source with any unparseable timestamp returns a diagnostic error rather than silently mis-sorting.
- [ ] D-05: Cancelling a Terraform plan mid-wait exits `WaitUntil` within one tick (≤1 s for snapshots, ≤1 s for instances).
- [ ] D-06: After `Create`, `data.Status` in state equals `instance.Data.Status` from the API, not the planned value.
- [ ] D-07: No literal string `"completed"` used for snapshot status comparisons anywhere in `internal/`.
- [ ] D-08: A JSON encoding failure in `writeJSON` causes the test to fail (panic or `t.Fatal`).
- [ ] D-09: `SnapshotDataSource.Read` returns without calling `State.Set` when `snapshot == nil`.
- [ ] All mock-acceptance tests pass (`make mock-acceptance`).

## Out of Scope

- Auth token mutex contention under high concurrency (design choice, not a correctness bug).
- HTTP response body ordering pattern in `aura_client.go` (functionally correct with `go-retryablehttp`).
- New tests beyond the minimal regression coverage needed for each fix.

## Open Questions

- For D-05: should `WaitUntil` return `ctx.Err()` directly, or wrap it with the "waiting condition wasn't reached in time" message for consistency with existing error handling?
- For D-03: should the second PATCH be attempted at all when only `name`/`memory` changed and `secondaries_count` is unchanged? Current code always sends it — intended or oversight?
