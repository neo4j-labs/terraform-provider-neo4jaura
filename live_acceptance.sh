#!/usr/bin/env bash
# live_acceptance.sh — run acceptance tests against the real Aura API.
#
# Required environment variables:
#   AURA_CLIENT_ID      — Aura API client ID
#   AURA_CLIENT_SECRET  — Aura API client secret
#   AURA_PROJECT_ID     — Aura project (tenant) ID to create test instances in
#
# Optional:
#   LIVE_TEST_RUN       — -run filter, e.g. TestLive_instance_lifecycle
#                         (defaults to all TestLive_* tests)
#   LIVE_TEST_TIMEOUT   — go test timeout (default: 30m)
#
# Usage:
#   export AURA_CLIENT_ID=...
#   export AURA_CLIENT_SECRET=...
#   export AURA_PROJECT_ID=...
#   ./live_acceptance.sh
#
#   # Run a single test:
#   LIVE_TEST_RUN=TestLive_instance_lifecycle ./live_acceptance.sh

set -euo pipefail

# --- Validate required env vars -------------------------------------------

MISSING=()
for VAR in AURA_CLIENT_ID AURA_CLIENT_SECRET AURA_PROJECT_ID; do
  if [[ -z "${!VAR:-}" ]]; then
    MISSING+=("$VAR")
  fi
done

if [[ ${#MISSING[@]} -gt 0 ]]; then
  echo "ERROR: The following environment variables are required but not set:"
  for VAR in "${MISSING[@]}"; do
    echo "  $VAR"
  done
  echo ""
  echo "Set them and re-run:"
  echo "  export AURA_CLIENT_ID=<your-client-id>"
  echo "  export AURA_CLIENT_SECRET=<your-client-secret>"
  echo "  export AURA_PROJECT_ID=<your-project-id>"
  exit 1
fi

# --- Run tests ----------------------------------------------------------------

RUN_FILTER="${LIVE_TEST_RUN:-TestLive_}"
TIMEOUT="${LIVE_TEST_TIMEOUT:-30m}"

echo "Running live acceptance tests against the Aura API..."
echo "  Project:  $AURA_PROJECT_ID"
echo "  Filter:   -run $RUN_FILTER"
echo "  Timeout:  $TIMEOUT"
echo ""
echo "WARNING: These tests create and destroy real Aura instances."
echo "         Costs may be incurred. Ctrl-C to abort."
echo ""

TF_ACC=1 go test ./internal/livetest/... \
  -v \
  -timeout "$TIMEOUT" \
  -run "$RUN_FILTER" \
  "$@"
