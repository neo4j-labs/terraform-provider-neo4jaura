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
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

const (
	projectUserOrgId     = "test-org-id-001"
	projectUserProjectId = "resource-test-project"
	projectUserUserId    = "user-001"
)

const testAccProjectUserConfig = defaultProviderConfig + `
resource "neo4jaura_project_user" "this" {
  organization_id = "` + projectUserOrgId + `"
  project_id      = "` + projectUserProjectId + `"
  user_id         = "` + projectUserUserId + `"
  project_roles   = ["project-member"]
}
`

const testAccProjectUserUpdatedConfig = defaultProviderConfig + `
resource "neo4jaura_project_user" "this" {
  organization_id = "` + projectUserOrgId + `"
  project_id      = "` + projectUserProjectId + `"
  user_id         = "` + projectUserUserId + `"
  project_roles   = ["project-admin", "project-viewer"]
}
`

const testAccProjectUserDeregisterConfig = defaultProviderConfig + `
resource "neo4jaura_project_user" "this" {
  organization_id     = "` + projectUserOrgId + `"
  project_id          = "` + projectUserProjectId + `"
  user_id             = "` + projectUserUserId + `"
  project_roles       = ["project-member"]
  deregister_on_delete = true
}
`

// TestAcc_project_user_lifecycle covers create (POST), read, update roles (PATCH),
// and destroy (no-op when deregister_on_delete=false).
func TestAcc_project_user_lifecycle(t *testing.T) {
	testMockServer.Reset()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectUserConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_project_user.this",
						tfjsonpath.New("organization_id"),
						knownvalue.StringExact(projectUserOrgId),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_project_user.this",
						tfjsonpath.New("project_id"),
						knownvalue.StringExact(projectUserProjectId),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_project_user.this",
						tfjsonpath.New("user_id"),
						knownvalue.StringExact(projectUserUserId),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_project_user.this",
						tfjsonpath.New("project_roles"),
						knownvalue.ListSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_project_user.this",
						tfjsonpath.New("project_roles").AtSliceIndex(0),
						knownvalue.StringExact("project-member"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_project_user.this",
						tfjsonpath.New("deregister_on_delete"),
						knownvalue.Bool(false),
					),
				},
			},
			{
				// Update project_roles — triggers PATCH.
				Config: testAccProjectUserUpdatedConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_project_user.this",
						tfjsonpath.New("project_roles"),
						knownvalue.ListSizeExact(2),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_project_user.this",
						tfjsonpath.New("project_roles").AtSliceIndex(0),
						knownvalue.StringExact("project-admin"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_project_user.this",
						tfjsonpath.New("project_roles").AtSliceIndex(1),
						knownvalue.StringExact("project-viewer"),
					),
				},
			},
		},
	})
}

// TestAcc_project_user_deregister_on_delete verifies that DELETE is called when
// deregister_on_delete=true, removing the user from the project on destroy.
func TestAcc_project_user_deregister_on_delete(t *testing.T) {
	testMockServer.Reset()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: func(_ *terraform.State) error {
			if testMockServer.ProjectUserExists(projectUserOrgId, projectUserProjectId, projectUserUserId) {
				return fmt.Errorf("project user %s still exists in project %s after destroy", projectUserUserId, projectUserProjectId)
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: testAccProjectUserDeregisterConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_project_user.this",
						tfjsonpath.New("deregister_on_delete"),
						knownvalue.Bool(true),
					),
				},
			},
		},
	})
}

// TestAcc_project_user_import imports an existing project user by composite ID and
// verifies that the imported state matches the seeded data.
func TestAcc_project_user_import(t *testing.T) {
	const (
		importOrgId     = "test-org-id-001"
		importProjectId = "import-test-project"
		importUserId    = "user-001"
	)
	importID := fmt.Sprintf("%s,%s,%s", importOrgId, importProjectId, importUserId)

	testMockServer.Reset()
	testMockServer.SeedProjectUser(importOrgId, importProjectId, importUserId, []string{"project-member"})

	importConfig := fmt.Sprintf(`%s
resource "neo4jaura_project_user" "this" {
  organization_id     = "%s"
  project_id          = "%s"
  user_id             = "%s"
  project_roles       = ["project-member"]
  deregister_on_delete = false
}
`, defaultProviderConfig, importOrgId, importProjectId, importUserId)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             importConfig,
				ResourceName:       "neo4jaura_project_user.this",
				ImportState:        true,
				ImportStateId:      importID,
				ImportStatePersist: true,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_project_user.this",
						tfjsonpath.New("organization_id"),
						knownvalue.StringExact(importOrgId),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_project_user.this",
						tfjsonpath.New("project_id"),
						knownvalue.StringExact(importProjectId),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_project_user.this",
						tfjsonpath.New("user_id"),
						knownvalue.StringExact(importUserId),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_project_user.this",
						tfjsonpath.New("project_roles").AtSliceIndex(0),
						knownvalue.StringExact("project-member"),
					),
				},
			},
		},
	})
}
