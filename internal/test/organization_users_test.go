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

// default (include_projects=true), no project_id — all users with full project details.
const testAccUsersWithProjectsConfig = defaultProviderConfig + `
data "neo4jaura_organization_users" "this" {
  organization_id = "test-org-id-001"
}
`

// default (include_projects=true), project_id set — filtered users with full project details.
const testAccUsersFilteredWithProjectsConfig = defaultProviderConfig + `
data "neo4jaura_organization_users" "this" {
  organization_id = "test-org-id-001"
  project_id      = "proj-001"
}
`

// include_projects=false, no project_id — all users, projects empty.
const testAccUsersWithoutProjectsConfig = defaultProviderConfig + `
data "neo4jaura_organization_users" "this" {
  organization_id  = "test-org-id-001"
  include_projects = false
}
`

// include_projects=false, project_id set — filtered users, projects empty.
const testAccUsersFilteredWithoutProjectsConfig = defaultProviderConfig + `
data "neo4jaura_organization_users" "this" {
  organization_id  = "test-org-id-001"
  project_id       = "proj-001"
  include_projects = false
}
`

func TestAcc_can_read_users_datasource(t *testing.T) {
	testMockServer.Reset()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUsersWithProjectsConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.neo4jaura_organization_users.this",
						tfjsonpath.New("users"),
						knownvalue.ListSizeExact(2),
					),
					// Alice
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
					// Bob
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
					statecheck.ExpectKnownValue(
						"data.neo4jaura_organization_users.this",
						tfjsonpath.New("users").AtSliceIndex(1).AtMapKey("projects"),
						knownvalue.ListSizeExact(1),
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
				Config: testAccUsersFilteredWithProjectsConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					// proj-001 has only alice
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
					// Project entry built from project users response + name from projects list — no per-user call.
					statecheck.ExpectKnownValue(
						"data.neo4jaura_organization_users.this",
						tfjsonpath.New("users").AtSliceIndex(0).AtMapKey("projects"),
						knownvalue.ListSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_organization_users.this",
						tfjsonpath.New("users").AtSliceIndex(0).AtMapKey("projects").AtSliceIndex(0).AtMapKey("id"),
						knownvalue.StringExact("proj-001"),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_organization_users.this",
						tfjsonpath.New("users").AtSliceIndex(0).AtMapKey("projects").AtSliceIndex(0).AtMapKey("name"),
						knownvalue.StringExact("Alpha"),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_organization_users.this",
						tfjsonpath.New("users").AtSliceIndex(0).AtMapKey("projects").AtSliceIndex(0).AtMapKey("project_roles").AtSliceIndex(0),
						knownvalue.StringExact("owner"),
					),
				},
			},
		},
	})
}

func TestAcc_can_read_users_without_projects(t *testing.T) {
	testMockServer.Reset()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUsersWithoutProjectsConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.neo4jaura_organization_users.this",
						tfjsonpath.New("users"),
						knownvalue.ListSizeExact(2),
					),
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
					// projects is empty when include_projects is false
					statecheck.ExpectKnownValue(
						"data.neo4jaura_organization_users.this",
						tfjsonpath.New("users").AtSliceIndex(0).AtMapKey("projects"),
						knownvalue.ListSizeExact(0),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_organization_users.this",
						tfjsonpath.New("users").AtSliceIndex(1).AtMapKey("id"),
						knownvalue.StringExact("user-002"),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_organization_users.this",
						tfjsonpath.New("users").AtSliceIndex(1).AtMapKey("projects"),
						knownvalue.ListSizeExact(0),
					),
				},
			},
		},
	})
}

func TestAcc_can_filter_users_by_project_without_projects(t *testing.T) {
	testMockServer.Reset()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUsersFilteredWithoutProjectsConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					// Only alice belongs to proj-001 per project users endpoint
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
					// No project details fetched
					statecheck.ExpectKnownValue(
						"data.neo4jaura_organization_users.this",
						tfjsonpath.New("users").AtSliceIndex(0).AtMapKey("projects"),
						knownvalue.ListSizeExact(0),
					),
				},
			},
		},
	})
}
