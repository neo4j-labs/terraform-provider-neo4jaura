# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
make build

# Format
make fmt          # gofmt -s -w -e .

# Vet / lint
make vet

# Unit tests (no live infra; TF_ACC unset)
make test

# Acceptance tests against the in-process mock server
make acceptance

# Acceptance tests against live Aura API (requires credentials)
make live-acceptance   # needs AURA_CLIENT_ID / AURA_CLIENT_SECRET / TF_VAR_* set

# Run a single acceptance test by name
TF_ACC=1 go test ./internal/test/... -run TestInstanceResource_basic -v -timeout 2m

# Install the provider binary locally
make install

# Re-generate docs (runs go generate in tools/)
make generate
```

Every PR must include a changelog entry under `.changes/unreleased/<short-description>.yaml`:
```yaml
kind: Fixed   # Added | Changed | Deprecated | Removed | Fixed | Security
body: 'Description of the change'
time: 2026-05-09T00:00:00Z
```
Run `changie new` to create one interactively if changie is installed.

## Architecture

The provider wraps the [Neo4j Aura management API](https://neo4j.com/docs/aura/platform/api/specification/) using `terraform-plugin-framework`.

### Layer overview

```
main.go
  └─ internal/provider/      — wires AuraApi into provider DataSourceData + ResourceData
       ├─ internal/resource/  — InstanceResource, SnapshotResource (CRUD + import)
       ├─ internal/datasource/ — ProjectDataSource, SnapshotDataSource (read-only)
       └─ internal/client/
            ├─ AuraClient     — HTTP transport (go-retryablehttp, token injection)
            ├─ AuraAuth       — OAuth client_credentials with mutex-protected token cache
            ├─ AuraApi        — Business methods + async pollers (WaitUntil*)
            ├─ aura_requests.go / aura_responses.go — typed API structs
       ├─ internal/domain/    — string constants (statuses, types, cloud providers, CDC modes)
       └─ internal/util/      — WaitUntil[T] generic poller, DiagnosticsError, JSON helper
```

### Key design points

**`AURA_BASE_URL`** — `NewAuraClient` accepts a `baseURL string` as its 4th argument. Empty string falls back to `https://api.neo4j.io`. In tests, `TestMain` sets `AURA_BASE_URL` to the mock server's URL; `provider.Configure` reads it via `os.Getenv("AURA_BASE_URL")`.

**Async operations** — The Aura API is asynchronous: POST/PATCH/DELETE return 202. Every mutating operation is followed by a `WaitUntil*` call that polls until the instance or snapshot reaches the expected state. Timeouts are configurable in the provider block (`instance_timeout`, `snapshot_timeout`).

**Two-phase create for `cdc_enrichment_mode` / `secondaries_count`** — The POST /instances endpoint does not accept these fields. They are applied via a follow-up PATCH after the instance reaches `running`. The API may then return `null` for `cdc_enrichment_mode` on business-critical tiers even after the PATCH succeeds; the resource preserves the planned value in state rather than overwriting with null.

**`source` block** — When set, `InstanceResource.Create` waits for the source snapshot to reach `completed` before posting, so that the clone starts from a stable snapshot.

### Mock server (`internal/test/mock_server.go`)

`MockServer` is an `httptest.Server`-backed in-process mock that handles the full Aura v1 API surface. It implements lightweight state machines using per-entity `getCount` counters:
- Instance: `getCount==1` → `creating`; `getCount>=2` → `running`
- Snapshot: `getCount==1` → `InProgress`; `getCount>=2` → `Completed`

`TestMain` starts one server for the whole test package. Individual tests call `testMockServer.Reset()` to wipe state, then `SeedInstance` / `SeedSnapshot` to pre-populate it.

**Do not use `t.Parallel()` in tests that call `testMockServer.Reset()`** — parallel tests sharing the single server will race on state. Use `t.Parallel()` only in tests that do not touch global server state or that seed distinct IDs.

Test configs that embed known IDs should use `const` string concatenation rather than `fmt.Sprintf` to avoid variable initialisation-order issues.
