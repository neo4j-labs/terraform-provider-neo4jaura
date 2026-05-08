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
