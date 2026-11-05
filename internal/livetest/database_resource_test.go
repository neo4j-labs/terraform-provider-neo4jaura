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
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package livetest

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// liveOrganizationId returns the Aura organization ID from AURA_ORGANIZATION_ID.
// When unset, the test config derives it from the organizations data source
// instead (index 0), so a credential that belongs to a single organization works
// without extra configuration.
func liveOrganizationId(t *testing.T) string {
	t.Helper()
	return os.Getenv("AURA_ORGANIZATION_ID")
}

// liveDatabaseOrgRef returns either the literal organization ID (from
// AURA_ORGANIZATION_ID) or a data-source reference resolving to the first
// organization the credential has access to.
func liveDatabaseOrgRef(t *testing.T) string {
	t.Helper()
	if id := liveOrganizationId(t); id != "" {
		return id
	}
	return `data.neo4jaura_organizations.orgs.organizations.0.id`
}

// TestLive_database_lifecycle creates a real business-critical Aura instance
// with multi_database enabled via the v2beta1 API, creates a database on it,
// verifies both the database resource and the databases list data source, and
// finally destroys everything.
//
// Requires: AURA_CLIENT_ID, AURA_CLIENT_SECRET, AURA_PROJECT_ID, TF_ACC=1
// Optional: AURA_ORGANIZATION_ID (defaults to the first organization of the
// credential), AURA_TEST_REGION (default europe-west1),
// AURA_TEST_CLOUD_PROVIDER (default gcp).
//
// Note: multi-database requires a business-critical (or enterprise/VDC)
// instance; the instance created by this test incurs real cost until it is
// destroyed at the end of the test. Typical runtime: 5–15 minutes.
func TestLive_database_lifecycle(t *testing.T) {
	t.Parallel()

	projectId := liveProjectID(t)

	region := os.Getenv("AURA_TEST_REGION")
	if region == "" {
		region = "europe-west1"
	}
	cloudProvider := os.Getenv("AURA_TEST_CLOUD_PROVIDER")
	if cloudProvider == "" {
		cloudProvider = "gcp"
	}

	orgBlock := ""
	orgRef := liveDatabaseOrgRef(t)
	if liveOrganizationId(t) == "" {
		orgBlock = "data \"neo4jaura_organizations\" \"orgs\" {}\n"
	}

	liveDatabaseConfig := fmt.Sprintf(`%s
%s
data "neo4jaura_databases" "on_instance" {
  organization_id = "%s"
  project_id      = "%s"
  instance_id     = neo4jaura_instance_v2.this.instance_id

  depends_on = [neo4jaura_database.this]
}

resource "neo4jaura_instance_v2" "this" {
  organization_id = "%s"
  project_id      = "%s"
  name            = "tf-live-test-multidb"
  cloud_provider  = "%s"
  region          = "%s"
  memory          = "16GB"
  type            = "business-critical"
  multi_database  = true
}

resource "neo4jaura_database" "this" {
  organization_id = "%s"
  project_id      = "%s"
  instance_id     = neo4jaura_instance_v2.this.instance_id
  name            = "live-test-database"
}
`, liveProviderConfig, orgBlock, orgRef, projectId, orgRef, projectId, cloudProvider, region, orgRef, projectId)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: liveDatabaseConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_instance_v2.this",
						tfjsonpath.New("status"),
						knownvalue.StringExact("running"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance_v2.this",
						tfjsonpath.New("multi_database"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance_v2.this",
						tfjsonpath.New("connection_url"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_database.this",
						tfjsonpath.New("name"),
						knownvalue.StringExact("live-test-database"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_database.this",
						tfjsonpath.New("database_id"),
						knownvalue.NotNull(),
					),
					// The list data source must see the newly created database
					// alongside the instance's default database.
					statecheck.ExpectKnownValue(
						"data.neo4jaura_databases.on_instance",
						tfjsonpath.New("databases"),
						knownvalue.ListSizeExact(2),
					),
				},
			},
		},
	})
}
