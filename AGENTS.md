# AGENTS.md — terraform-provider-neo4jaura

## Feedback Instructions

### BUILD COMMANDS
```
go build ./...
```

### LINT / VET COMMANDS
```
go vet ./...
```

### TEST COMMANDS
```
TF_ACC=1 go test ./internal/test/... -timeout 120s -v
```

Run acceptance tests only when a mock server is in place (task-005+). Until then, use build + vet as the feedback loop.

## Architecture

- `internal/client/` — HTTP client (`AuraClient`), auth (`AuraAuth`), and API wrapper (`AuraApi`).
- `internal/provider/` — Terraform provider wiring.
- `internal/resource/` — Terraform resource implementations (instance, snapshot).
- `internal/datasource/` — Terraform data source implementations (projects, snapshot).
- `internal/test/` — Acceptance tests and test helpers.

## Gotchas

- `NewAuraClient` takes a `baseURL string` as its 4th argument (added in task-001). Pass `""` to get the default `https://api.neo4j.io`, or pass `os.Getenv("AURA_BASE_URL")` so the mock server URL is picked up in tests.
- `AuraAuth.baseURL` is set at construction time from the same value — there is no separate default in `aura_auth.go`.
- The `auraBasePath` and `auraV1Path` constants were removed in task-001; use `defaultAuraBasePath` if you need the literal default URL string.
- `go vet` is strict about unused imports — always check after editing provider/test files.
- Test configs that depend on mock-seeded state must use `const` string concatenation (not `fmt.Sprintf`) when embedding known IDs — this avoids the `var` initialisation order issue and makes IDs available at compile time.
- The snapshot datasource config should hardcode `instance_id` as a string literal; do not reference a `neo4jaura_instance` resource, which would require live infra or complex multi-step test setup.
- `t.Parallel()` must be removed from tests that call `testMockServer.Reset()` — parallel tests sharing a single mock server will race on state.
- `sort.Slice` comparators cannot return errors. Use a closure-captured `var sortErr error` variable; set it on error, return `false`, then check after sort completes.
- `WaitUntil` takes `ctx context.Context` as its first argument (added in task-003). Always pass the request/operation context so cancellations are respected during polling.
- After `WaitUntilInstanceIsInState` in `Create`, write `data.Status = types.StringValue(instance.Data.Status)` — not `data.Status.ValueString()` which is the planned value, not the API value.
- Use `strings.ToLower(domain.SnapshotStatusCompleted)` (not the literal `"completed"`) for all snapshot status comparisons. The domain constant is `"Completed"`.
- `client.ErrNotFound` (defined in `aura_client.go`) is returned by `GetInstanceById` and `GetSnapshotById` on HTTP 404 via `fmt.Errorf("...: %w", ErrNotFound)`. Resource Read methods detect it with `errors.Is(err, client.ErrNotFound)` and call `response.State.RemoveResource(ctx)` for drift detection.
- `client_id` and `client_secret` are `Optional` in the provider schema. `Configure` falls back to `AURA_CLIENT_ID` / `AURA_CLIENT_SECRET` env vars when the HCL values are null, unknown, or empty — same triple-guard pattern as `AURA_BASE_URL`. The provider does not error on missing credentials at configure time; the error surfaces at the first API call.
- The provider registry address in `main.go` is `registry.terraform.io/neo4j-labs/neo4jaura`. Dev/test overrides use the `~/.terraformrc` `dev_overrides` block, not a different address in main.go.
