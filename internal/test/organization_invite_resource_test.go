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

const inviteOrgId = "test-org-id-001"

const testAccOrgInviteConfig = defaultProviderConfig + `
resource "neo4jaura_organization_invite" "this" {
  organization_id    = "` + inviteOrgId + `"
  email               = "morpheus@nebuchadnezzar.net"
  organization_roles  = ["organization-member"]
}
`

// TestAcc_organization_invite_create covers create and destroy (revoke) of a
// plain org-level invite with no project_invites.
func TestAcc_organization_invite_create(t *testing.T) {
	testMockServer.Reset()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: func(_ *terraform.State) error {
			invites := testMockServer.OrganizationInvites(inviteOrgId)
			if len(invites) != 1 {
				return fmt.Errorf("expected exactly 1 invite to still exist (revoked) after destroy, got %d", len(invites))
			}
			if invites[0].Status != "revoked" {
				return fmt.Errorf("expected invite to be revoked after destroy, got status %q", invites[0].Status)
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: testAccOrgInviteConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_invite.this",
						tfjsonpath.New("id"),
						knownvalue.StringFunc(nonEmptyString),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_invite.this",
						tfjsonpath.New("organization_id"),
						knownvalue.StringExact(inviteOrgId),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_invite.this",
						tfjsonpath.New("email"),
						knownvalue.StringExact("morpheus@nebuchadnezzar.net"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_invite.this",
						tfjsonpath.New("organization_roles"),
						knownvalue.ListSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_invite.this",
						tfjsonpath.New("organization_roles").AtSliceIndex(0),
						knownvalue.StringExact("organization-member"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_invite.this",
						tfjsonpath.New("project_invites"),
						knownvalue.Null(),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_invite.this",
						tfjsonpath.New("status"),
						knownvalue.StringExact("active"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_invite.this",
						tfjsonpath.New("invited_by"),
						knownvalue.StringFunc(nonEmptyString),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_invite.this",
						tfjsonpath.New("expires_at"),
						knownvalue.StringFunc(nonEmptyString),
					),
				},
			},
		},
	})
}

// TestAcc_organization_invite_with_project_invites verifies project_invites
// round-trips through create.
func TestAcc_organization_invite_with_project_invites(t *testing.T) {
	testMockServer.Reset()

	config := defaultProviderConfig + `
resource "neo4jaura_organization_invite" "this" {
  organization_id   = "` + inviteOrgId + `"
  email             = "trinity@nebuchadnezzar.net"
  organization_roles = ["organization-member"]
  project_invites = [
    {
      project_id    = "89fcc404-a509-488e-a3a4-53d3e80b99b7"
      project_roles = ["namespace-member"]
    }
  ]
}
`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_invite.this",
						tfjsonpath.New("project_invites"),
						knownvalue.ListSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_invite.this",
						tfjsonpath.New("project_invites").AtSliceIndex(0).AtMapKey("project_id"),
						knownvalue.StringExact("89fcc404-a509-488e-a3a4-53d3e80b99b7"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_invite.this",
						tfjsonpath.New("project_invites").AtSliceIndex(0).AtMapKey("project_roles").AtSliceIndex(0),
						knownvalue.StringExact("namespace-member"),
					),
				},
			},
		},
	})
}

// TestAcc_organization_invite_import imports an existing invite by composite
// ID and verifies the imported state matches the seeded data.
func TestAcc_organization_invite_import(t *testing.T) {
	const importInviteId = "invite-import-test"
	importID := fmt.Sprintf("%s,%s", inviteOrgId, importInviteId)

	testMockServer.Reset()
	testMockServer.SeedOrganizationInvite(inviteOrgId, client.OrganizationInviteData{
		Id:                importInviteId,
		Email:             "neo@nebuchadnezzar.net",
		InvitedBy:         "user-morpheus",
		ExpiresAt:         "2099-01-01T00:00:00Z",
		Status:            "active",
		OrganizationRoles: []string{"organization-admin"},
	})

	importConfig := fmt.Sprintf(`%s
resource "neo4jaura_organization_invite" "this" {
  organization_id    = "%s"
  email              = "neo@nebuchadnezzar.net"
  organization_roles = ["organization-admin"]
}
`, defaultProviderConfig, inviteOrgId)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             importConfig,
				ResourceName:       "neo4jaura_organization_invite.this",
				ImportState:        true,
				ImportStateId:      importID,
				ImportStatePersist: true,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_invite.this",
						tfjsonpath.New("id"),
						knownvalue.StringExact(importInviteId),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_invite.this",
						tfjsonpath.New("organization_id"),
						knownvalue.StringExact(inviteOrgId),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_invite.this",
						tfjsonpath.New("email"),
						knownvalue.StringExact("neo@nebuchadnezzar.net"),
					),
				},
			},
		},
	})

	// Clean up so the invite doesn't leak into other tests sharing the mock server.
	testMockServer.Reset()
}

// TestAcc_organization_invite_delete_skips_terminal_status verifies that
// destroying an invite that is no longer "active" (e.g. already accepted)
// does not attempt to revoke it via the API.
func TestAcc_organization_invite_delete_skips_terminal_status(t *testing.T) {
	const acceptedInviteId = "invite-accepted-test"
	importID := fmt.Sprintf("%s,%s", inviteOrgId, acceptedInviteId)

	testMockServer.Reset()
	testMockServer.SeedOrganizationInvite(inviteOrgId, client.OrganizationInviteData{
		Id:                acceptedInviteId,
		Email:             "cypher@nebuchadnezzar.net",
		InvitedBy:         "user-morpheus",
		ExpiresAt:         "2099-01-01T00:00:00Z",
		Status:            "accepted",
		OrganizationRoles: []string{"organization-member"},
	})

	importConfig := fmt.Sprintf(`%s
resource "neo4jaura_organization_invite" "this" {
  organization_id    = "%s"
  email              = "cypher@nebuchadnezzar.net"
  organization_roles = ["organization-member"]
}
`, defaultProviderConfig, inviteOrgId)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: func(_ *terraform.State) error {
			status, ok := testMockServer.OrganizationInviteStatus(inviteOrgId, acceptedInviteId)
			if !ok {
				return fmt.Errorf("expected invite %s to still exist after destroy", acceptedInviteId)
			}
			if status != "accepted" {
				return fmt.Errorf("expected invite %s to remain %q, got %q — Delete should not call the API for terminal invites", acceptedInviteId, "accepted", status)
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config:             importConfig,
				ResourceName:       "neo4jaura_organization_invite.this",
				ImportState:        true,
				ImportStateId:      importID,
				ImportStatePersist: true,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_organization_invite.this",
						tfjsonpath.New("status"),
						knownvalue.StringExact("accepted"),
					),
				},
			},
		},
	})
}
