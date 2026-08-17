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
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/client"
)

const (
	orgUserOrgId  = "test-org-id-001"
	orgUserUserId = "user-org-test"
)

func seedTestOrgUser(orgId, userId string, roles []string) {
	lastActivity := "2024-06-01T12:00:00Z"
	testMockServer.SeedOrganizationUser(orgId, client.OrganizationUserData{
		UserId:                     userId,
		Email:                      "orguser@example.com",
		ExemptFromAutomaticRemoval: false,
		LastActivityAt:             &lastActivity,
		MfaEnrollmentStatus:        "enrolled",
		MfaEnrolledMethods:         []client.MfaMethodData{{Id: "totp", EnrolledAt: "2024-01-01T00:00:00Z"}},
		OrganizationRoles:          roles,
	})
}

const testAccOrgUserConfig = defaultProviderConfig + `
resource "neo4jaura_organization_user" "this" {
  id              = "` + orgUserUserId + `"
  organization_id = "` + orgUserOrgId + `"
  organization_roles = ["organization-member"]
}
`

const testAccOrgUserUpdatedConfig = defaultProviderConfig + `
resource "neo4jaura_organization_user" "this" {
  id              = "` + orgUserUserId + `"
  organization_id = "` + orgUserOrgId + `"
  organization_roles = ["organization-admin", "organization-member"]
}
`

const testAccOrgUserDeregisterConfig = defaultProviderConfig + `
resource "neo4jaura_organization_user" "this" {
  id                   = "` + orgUserUserId + `"
  organization_id      = "` + orgUserOrgId + `"
  organization_roles   = ["organization-member"]
  deregister_on_delete = true
}
`

// TestAcc_organization_user_lifecycle covers create (with role reconciliation via PATCH),
// update roles, and destroy (no-op when deregister_on_delete=false).
func TestAcc_organization_user_lifecycle(t *testing.T) {
	testMockServer.Reset()
	// Seed with a different role so Create triggers a PATCH to reconcile.
	seedTestOrgUser(orgUserOrgId, orgUserUserId, []string{"organization-admin"})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOrgUserConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_user.this",
						tfjsonpath.New("id"),
						knownvalue.StringExact(orgUserUserId),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_user.this",
						tfjsonpath.New("organization_id"),
						knownvalue.StringExact(orgUserOrgId),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_user.this",
						tfjsonpath.New("organization_roles"),
						knownvalue.ListSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_user.this",
						tfjsonpath.New("organization_roles").AtSliceIndex(0),
						knownvalue.StringExact("organization-member"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_user.this",
						tfjsonpath.New("deregister_on_delete"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_user.this",
						tfjsonpath.New("email"),
						knownvalue.StringExact("orguser@example.com"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_user.this",
						tfjsonpath.New("exempt_from_automatic_removal"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_user.this",
						tfjsonpath.New("last_activity_at"),
						knownvalue.StringExact("2024-06-01T12:00:00Z"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_user.this",
						tfjsonpath.New("mfa_enrollment").AtMapKey("status"),
						knownvalue.StringExact("enrolled"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_user.this",
						tfjsonpath.New("mfa_enrollment").AtMapKey("methods").AtSliceIndex(0).AtMapKey("id"),
						knownvalue.StringExact("totp"),
					),
				},
			},
			{
				// Update triggers PATCH with two roles.
				Config: testAccOrgUserUpdatedConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_user.this",
						tfjsonpath.New("organization_roles"),
						knownvalue.ListSizeExact(2),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_user.this",
						tfjsonpath.New("organization_roles").AtSliceIndex(0),
						knownvalue.StringExact("organization-admin"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_user.this",
						tfjsonpath.New("organization_roles").AtSliceIndex(1),
						knownvalue.StringExact("organization-member"),
					),
				},
			},
		},
	})

	// Destroy is a no-op because deregister_on_delete=false; user still exists.
	if !testMockServer.OrganizationUserExists(orgUserOrgId, orgUserUserId) {
		t.Error("expected org user to still exist after destroy with deregister_on_delete=false")
	}
}

// TestAcc_organization_user_deregister_on_delete verifies that DELETE is called when
// deregister_on_delete=true, removing the user from the organization on destroy.
func TestAcc_organization_user_deregister_on_delete(t *testing.T) {
	testMockServer.Reset()
	seedTestOrgUser(orgUserOrgId, orgUserUserId, []string{"organization-member"})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: func(_ *terraform.State) error {
			if testMockServer.OrganizationUserExists(orgUserOrgId, orgUserUserId) {
				return fmt.Errorf("org user %s still exists in org %s after destroy", orgUserUserId, orgUserOrgId)
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: testAccOrgUserDeregisterConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_user.this",
						tfjsonpath.New("deregister_on_delete"),
						knownvalue.Bool(true),
					),
				},
			},
		},
	})
}

// TestAcc_organization_user_import imports an existing org user by composite ID
// and verifies that the imported state matches the seeded data.
func TestAcc_organization_user_import(t *testing.T) {
	const (
		importOrgId  = "test-org-id-001"
		importUserId = "user-import-test"
	)
	importID := fmt.Sprintf("%s,%s", importOrgId, importUserId)

	testMockServer.Reset()
	lastActivity := "2024-06-01T12:00:00Z"
	testMockServer.SeedOrganizationUser(importOrgId, client.OrganizationUserData{
		UserId:                     importUserId,
		Email:                      "import@example.com",
		ExemptFromAutomaticRemoval: false,
		LastActivityAt:             &lastActivity,
		MfaEnrollmentStatus:        "not_enrolled",
		MfaEnrolledMethods:         []client.MfaMethodData{},
		OrganizationRoles:          []string{"organization-admin"},
	})

	importConfig := fmt.Sprintf(`%s
resource "neo4jaura_organization_user" "this" {
  id                   = "%s"
  organization_id      = "%s"
  organization_roles   = ["organization-admin"]
  deregister_on_delete = false
}
`, defaultProviderConfig, importUserId, importOrgId)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             importConfig,
				ResourceName:       "neo4jaura_organization_user.this",
				ImportState:        true,
				ImportStateId:      importID,
				ImportStatePersist: true,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_user.this",
						tfjsonpath.New("id"),
						knownvalue.StringExact(importUserId),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_user.this",
						tfjsonpath.New("organization_id"),
						knownvalue.StringExact(importOrgId),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_user.this",
						tfjsonpath.New("organization_roles").AtSliceIndex(0),
						knownvalue.StringExact("organization-admin"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_user.this",
						tfjsonpath.New("email"),
						knownvalue.StringExact("import@example.com"),
					),
				},
			},
		},
	})
}

// TestAcc_organization_user_owner_role_protected verifies that a user with the
// organization-owner role is NOT deleted via the API even when deregister_on_delete=true.
func TestAcc_organization_user_owner_role_protected(t *testing.T) {
	const ownerUserId = "user-owner-test"

	testMockServer.Reset()
	seedOrgUser := client.OrganizationUserData{
		UserId:              ownerUserId,
		Email:               "owner@example.com",
		MfaEnrollmentStatus: "enrolled",
		MfaEnrolledMethods:  []client.MfaMethodData{{Id: "totp", EnrolledAt: "2024-01-01T00:00:00Z"}},
		OrganizationRoles:   []string{"organization-owner"},
	}
	testMockServer.SeedOrganizationUser(orgUserOrgId, seedOrgUser)

	ownerConfig := fmt.Sprintf(`%s
resource "neo4jaura_organization_user" "this" {
  id                   = "%s"
  organization_id      = "%s"
  organization_roles   = ["organization-owner"]
  deregister_on_delete = true
}
`, defaultProviderConfig, ownerUserId, orgUserOrgId)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		// User should still exist after destroy because owner role blocks deletion.
		CheckDestroy: func(_ *terraform.State) error {
			if !testMockServer.OrganizationUserExists(orgUserOrgId, ownerUserId) {
				return fmt.Errorf("org user %s should still exist (owner role protected) but was deleted", ownerUserId)
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: ownerConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_user.this",
						tfjsonpath.New("organization_roles").AtSliceIndex(0),
						knownvalue.StringExact("organization-owner"),
					),
				},
			},
		},
	})
}
