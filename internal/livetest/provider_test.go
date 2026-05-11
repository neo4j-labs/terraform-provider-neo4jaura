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

// Package livetest contains acceptance tests that run against the real Aura API.
// They require AURA_CLIENT_ID, AURA_CLIENT_SECRET, and AURA_PROJECT_ID to be set
// and are intentionally kept separate from the mock-based tests in internal/test/
// so that the mock TestMain never redirects traffic away from the live API.
//
// Run via: ./live_acceptance.sh  (or  TF_ACC=1 go test ./internal/livetest/... -v -timeout 30m)
package livetest

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/provider"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"neo4jaura": providerserver.NewProtocol6WithError(provider.New("live-test")()),
}

// liveProviderConfig relies on AURA_CLIENT_ID and AURA_CLIENT_SECRET env vars
// being picked up by provider.Configure's env-var fallback. No credentials are
// hardcoded so this config is safe to commit.
const liveProviderConfig = `
provider "neo4jaura" {}
`

// liveProjectID returns the Aura project (tenant) ID from the AURA_PROJECT_ID
// environment variable. Tests call this at runtime, not at package init time,
// so a missing value surfaces as a clear skip rather than a nil-deref.
func liveProjectID(t *testing.T) string {
	t.Helper()
	id := os.Getenv("AURA_PROJECT_ID")
	if id == "" {
		t.Skip("AURA_PROJECT_ID not set — skipping live test")
	}
	return id
}

// testAccPreCheck validates the three environment variables required by every
// live acceptance test. Tests are skipped (not failed) when vars are absent so
// that `go test ./...` in a dev environment without live credentials is safe.
func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv(resource.EnvTfAcc) == "" {
		t.Skipf("Acceptance tests skipped unless env '%s' set", resource.EnvTfAcc)
	}
	missing := []string{}
	for _, v := range []string{"AURA_CLIENT_ID", "AURA_CLIENT_SECRET", "AURA_PROJECT_ID"} {
		if os.Getenv(v) == "" {
			missing = append(missing, v)
		}
	}
	if len(missing) > 0 {
		t.Skipf("Live acceptance tests skipped — set %v to run them", missing)
	}
}

// liveInstanceConfig returns a Terraform config that creates a free-tier
// instance in the project identified by AURA_PROJECT_ID. The name includes
// the test name to make it easy to identify leaked resources in the console.
func liveInstanceConfig(t *testing.T, name string) string {
	t.Helper()
	return fmt.Sprintf(`
%s
resource "neo4jaura_instance" "this" {
  name           = "%s"
  cloud_provider = "gcp"
  region         = "europe-west1"
  memory         = "1GB"
  type           = "professional-db"
  project_id     = "%s"
}
`, liveProviderConfig, name, liveProjectID(t))
}
