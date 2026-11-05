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

package test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/client"
)

const (
	databaseOrgId      = "test-org-id-001"
	databaseProjectId  = "database-test-project"
	databaseInstanceId = "instance-001"
)

// The mock server generates deterministic database IDs: db-<instanceId>-<counter>,
// where the counter is per-instance and reset together with the rest of the state.
var (
	databaseFirstId  = "db-" + databaseInstanceId + "-0001"
	databaseSecondId = "db-" + databaseInstanceId + "-0002"
)

const testAccDatabaseConfig = defaultProviderConfig + `
resource "neo4jaura_database" "this" {
  organization_id = "` + databaseOrgId + `"
  project_id      = "` + databaseProjectId + `"
  instance_id     = "` + databaseInstanceId + `"
  name            = "test-database"
}
`

const testAccDatabaseRenamedConfig = defaultProviderConfig + `
resource "neo4jaura_database" "this" {
  organization_id = "` + databaseOrgId + `"
  project_id      = "` + databaseProjectId + `"
  instance_id     = "` + databaseInstanceId + `"
  name            = "renamed-database"
}
`

const testAccDatabaseReplicatedConfig = defaultProviderConfig + `
resource "neo4jaura_database" "source" {
  organization_id = "` + databaseOrgId + `"
  project_id      = "` + databaseProjectId + `"
  instance_id     = "` + databaseInstanceId + `"
  name            = "source-database"
}

resource "neo4jaura_database" "replica" {
  organization_id            = "` + databaseOrgId + `"
  project_id                 = "` + databaseProjectId + `"
  instance_id                = "` + databaseInstanceId + `"
  name                       = "replica-database"
  replicate_from_database_id = neo4jaura_database.source.database_id

  depends_on = [neo4jaura_database.source]
}
`

const testAccDatabasesDataSourceConfig = defaultProviderConfig + `
data "neo4jaura_databases" "this" {
  organization_id = "` + databaseOrgId + `"
  project_id      = "` + databaseProjectId + `"
  instance_id     = "` + databaseInstanceId + `"
}
`

// TestAcc_database_lifecycle covers create (POST), read, replace on rename
// (no update endpoint exists for databases), and destroy (DELETE + poll).
func TestAcc_database_lifecycle(t *testing.T) {
	testMockServer.Reset()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: func(_ *terraform.State) error {
			if testMockServer.DatabaseExists(databaseOrgId, databaseProjectId, databaseInstanceId, databaseFirstId) {
				return fmt.Errorf("database %s still exists on instance %s after destroy", databaseFirstId, databaseInstanceId)
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: testAccDatabaseConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_database.this",
						tfjsonpath.New("organization_id"),
						knownvalue.StringExact(databaseOrgId),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_database.this",
						tfjsonpath.New("project_id"),
						knownvalue.StringExact(databaseProjectId),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_database.this",
						tfjsonpath.New("instance_id"),
						knownvalue.StringExact(databaseInstanceId),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_database.this",
						tfjsonpath.New("name"),
						knownvalue.StringExact("test-database"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_database.this",
						tfjsonpath.New("database_id"),
						knownvalue.StringExact(databaseFirstId),
					),
				},
			},
			{
				// Renaming forces replacement — the v2beta1 API has no rename endpoint.
				Config: testAccDatabaseRenamedConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_database.this",
						tfjsonpath.New("name"),
						knownvalue.StringExact("renamed-database"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_database.this",
						tfjsonpath.New("database_id"),
						knownvalue.StringExact(databaseSecondId),
					),
				},
			},
		},
	})
}

// TestAcc_database_replicate_from_database covers creating a database that
// replicates from an existing database on the same instance.
func TestAcc_database_replicate_from_database(t *testing.T) {
	testMockServer.Reset()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDatabaseReplicatedConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_database.replica",
						tfjsonpath.New("name"),
						knownvalue.StringExact("replica-database"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_database.replica",
						tfjsonpath.New("database_id"),
						knownvalue.StringExact(databaseSecondId),
					),
				},
			},
		},
	})
}

// TestAcc_database_invalid_name verifies that names violating the Aura database
// naming rules are rejected by the plan-time validator.
func TestAcc_database_invalid_name(t *testing.T) {
	testMockServer.Reset()

	invalidNameConfig := defaultProviderConfig + `
resource "neo4jaura_database" "this" {
  organization_id = "` + databaseOrgId + `"
  project_id      = "` + databaseProjectId + `"
  instance_id     = "` + databaseInstanceId + `"
  name            = "Invalid_Name"
}
`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      invalidNameConfig,
				ExpectError: regexp.MustCompile(`must be 3-63 characters of lowercase letters`),
			},
		},
	})
}

// TestAcc_database_import imports an existing database by composite ID
// ({organization_id}/{project_id}/{instance_id}/{database_name}) and verifies
// that the imported state round-trips without a diff and resolves the
// database ID from the API.
func TestAcc_database_import(t *testing.T) {
	const (
		importOrgId      = "test-org-id-001"
		importProjectId  = "database-import-project"
		importInstanceId = "instance-001"
		importDatabaseId = "db-seed-001"
		importDbName     = "imported-database"
	)
	importID := importOrgId + "/" + importProjectId + "/" + importInstanceId + "/" + importDbName

	testMockServer.Reset()
	testMockServer.SeedDatabase(importOrgId, importProjectId, importInstanceId, client.DatabaseSummaryData{
		Id:   importDatabaseId,
		Name: importDbName,
	})

	importConfig := defaultProviderConfig + `
resource "neo4jaura_database" "this" {
  organization_id = "` + importOrgId + `"
  project_id      = "` + importProjectId + `"
  instance_id     = "` + importInstanceId + `"
  name            = "` + importDbName + `"
}
`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             importConfig,
				ResourceName:       "neo4jaura_database.this",
				ImportState:        true,
				ImportStateId:      importID,
				ImportStatePersist: true,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_database.this",
						tfjsonpath.New("organization_id"),
						knownvalue.StringExact(importOrgId),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_database.this",
						tfjsonpath.New("project_id"),
						knownvalue.StringExact(importProjectId),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_database.this",
						tfjsonpath.New("instance_id"),
						knownvalue.StringExact(importInstanceId),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_database.this",
						tfjsonpath.New("database_id"),
						knownvalue.StringExact(importDatabaseId),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_database.this",
						tfjsonpath.New("name"),
						knownvalue.StringExact(importDbName),
					),
				},
			},
			{
				// Fails with "the plan was not empty" if the imported state does not
				// round-trip after Read refreshes it from the API.
				Config:   importConfig,
				PlanOnly: true,
			},
		},
	})
}

// TestAcc_database_reserved_name verifies that system/default database names
// are rejected by the plan-time validator.
func TestAcc_database_reserved_name(t *testing.T) {
	testMockServer.Reset()

	for _, reserved := range []string{"neo4j", "system"} {
		reservedNameConfig := defaultProviderConfig + `
resource "neo4jaura_database" "this" {
  organization_id = "` + databaseOrgId + `"
  project_id      = "` + databaseProjectId + `"
  instance_id     = "` + databaseInstanceId + `"
  name            = "` + reserved + `"
}
`

		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      reservedNameConfig,
					ExpectError: regexp.MustCompile(`Reserved database name`),
				},
			},
		})
	}
}

// TestAcc_database_drift verifies that a database deleted out-of-band is
// detected during refresh (Read removes it from state) and recreated.
func TestAcc_database_drift(t *testing.T) {
	testMockServer.Reset()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDatabaseConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_database.this",
						tfjsonpath.New("database_id"),
						knownvalue.StringExact(databaseFirstId),
					),
				},
			},
			{
				// Delete the database out-of-band before this step: refresh finds
				// nothing (Read removes the state entry) and Terraform recreates it.
				PreConfig: func() {
					testMockServer.DeleteDatabase(databaseOrgId, databaseProjectId, databaseInstanceId, databaseFirstId)
				},
				Config: testAccDatabaseConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_database.this",
						tfjsonpath.New("database_id"),
						knownvalue.StringExact(databaseSecondId),
					),
				},
			},
		},
	})
}

// TestAcc_databases_datasource verifies the list data source returns the
// databases seeded in the mock, sorted by database ID.
func TestAcc_databases_datasource(t *testing.T) {
	testMockServer.Reset()
	testMockServer.SeedDatabase(databaseOrgId, databaseProjectId, databaseInstanceId, client.DatabaseSummaryData{
		Id:   "db-seed-002",
		Name: "database-two",
	})
	testMockServer.SeedDatabase(databaseOrgId, databaseProjectId, databaseInstanceId, client.DatabaseSummaryData{
		Id:   "db-seed-001",
		Name: "database-one",
	})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDatabasesDataSourceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.neo4jaura_databases.this",
						tfjsonpath.New("organization_id"),
						knownvalue.StringExact(databaseOrgId),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_databases.this",
						tfjsonpath.New("instance_id"),
						knownvalue.StringExact(databaseInstanceId),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_databases.this",
						tfjsonpath.New("databases").AtSliceIndex(0).AtMapKey("id"),
						knownvalue.StringExact("db-seed-001"),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_databases.this",
						tfjsonpath.New("databases").AtSliceIndex(0).AtMapKey("name"),
						knownvalue.StringExact("database-one"),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_databases.this",
						tfjsonpath.New("databases").AtSliceIndex(1).AtMapKey("id"),
						knownvalue.StringExact("db-seed-002"),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_databases.this",
						tfjsonpath.New("databases").AtSliceIndex(1).AtMapKey("name"),
						knownvalue.StringExact("database-two"),
					),
				},
			},
		},
	})
}
