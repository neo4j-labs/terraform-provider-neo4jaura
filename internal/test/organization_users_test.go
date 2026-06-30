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
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

const testAccUsersDataSourceConfig = defaultProviderConfig + `
data "neo4jaura_organization_users" "this" {
  organization_id = "test-org-id-001"
}
`

const testAccUsersFilteredDataSourceConfig = defaultProviderConfig + `
data "neo4jaura_organization_users" "this" {
  organization_id = "test-org-id-001"
  project_id      = "proj-001"
}
`

func TestAcc_can_read_users_datasource(t *testing.T) {
	testMockServer.Reset()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUsersDataSourceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					// Two users returned
					statecheck.ExpectKnownValue(
						"data.neo4jaura_organization_users.this",
						tfjsonpath.New("users").AtSliceIndex(0).AtMapKey("id"),
						knownvalue.StringExact("user-001"),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_organization_users.this",
						tfjsonpath.New("users").AtSliceIndex(0).AtMapKey("email"),
						knownvalue.StringExact("alice@example.com"),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_organization_users.this",
						tfjsonpath.New("users").AtSliceIndex(0).AtMapKey("exempt_from_automatic_removal"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_organization_users.this",
						tfjsonpath.New("users").AtSliceIndex(0).AtMapKey("mfa_enrollment").AtMapKey("status"),
						knownvalue.StringExact("enrolled"),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_organization_users.this",
						tfjsonpath.New("users").AtSliceIndex(0).AtMapKey("mfa_enrollment").AtMapKey("methods").AtSliceIndex(0).AtMapKey("id"),
						knownvalue.StringExact("totp"),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_organization_users.this",
						tfjsonpath.New("users").AtSliceIndex(0).AtMapKey("organization_roles").AtSliceIndex(0),
						knownvalue.StringExact("admin"),
					),
					// Alice has 2 projects
					statecheck.ExpectKnownValue(
						"data.neo4jaura_organization_users.this",
						tfjsonpath.New("users").AtSliceIndex(0).AtMapKey("projects").AtSliceIndex(0).AtMapKey("id"),
						knownvalue.StringExact("proj-001"),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_organization_users.this",
						tfjsonpath.New("users").AtSliceIndex(0).AtMapKey("projects").AtSliceIndex(1).AtMapKey("id"),
						knownvalue.StringExact("proj-002"),
					),
					// Bob — second user
					statecheck.ExpectKnownValue(
						"data.neo4jaura_organization_users.this",
						tfjsonpath.New("users").AtSliceIndex(1).AtMapKey("id"),
						knownvalue.StringExact("user-002"),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_organization_users.this",
						tfjsonpath.New("users").AtSliceIndex(1).AtMapKey("exempt_from_automatic_removal"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_organization_users.this",
						tfjsonpath.New("users").AtSliceIndex(1).AtMapKey("mfa_enrollment").AtMapKey("status"),
						knownvalue.StringExact("not_enrolled"),
					),
				},
			},
		},
	})
}

func TestAcc_can_read_users_filtered_by_project(t *testing.T) {
	testMockServer.Reset()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUsersFilteredDataSourceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					// Only alice belongs to proj-001; bob does not
					statecheck.ExpectKnownValue(
						"data.neo4jaura_organization_users.this",
						tfjsonpath.New("users"),
						knownvalue.ListSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_organization_users.this",
						tfjsonpath.New("users").AtSliceIndex(0).AtMapKey("id"),
						knownvalue.StringExact("user-001"),
					),
					// Alice's full project list is preserved — both proj-001 and proj-002
					statecheck.ExpectKnownValue(
						"data.neo4jaura_organization_users.this",
						tfjsonpath.New("users").AtSliceIndex(0).AtMapKey("projects"),
						knownvalue.ListSizeExact(2),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_organization_users.this",
						tfjsonpath.New("users").AtSliceIndex(0).AtMapKey("projects").AtSliceIndex(0).AtMapKey("id"),
						knownvalue.StringExact("proj-001"),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_organization_users.this",
						tfjsonpath.New("users").AtSliceIndex(0).AtMapKey("projects").AtSliceIndex(1).AtMapKey("id"),
						knownvalue.StringExact("proj-002"),
					),
				},
			},
		},
	})
}
