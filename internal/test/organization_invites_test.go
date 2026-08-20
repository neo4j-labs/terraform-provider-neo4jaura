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
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/client"
)

const testAccInvitesConfig = defaultProviderConfig + `
data "neo4jaura_organization_invites" "this" {
  organization_id = "` + inviteOrgId + `"
}
`

// TestAcc_can_read_invites_datasource verifies that an accepted invite's
// user_id is resolved by matching email against the organization's user
// list, while a still-pending invite's user_id stays null.
func TestAcc_can_read_invites_datasource(t *testing.T) {
	testMockServer.Reset()

	// "alice@example.com" is one of the mock server's always-present static
	// organization users (see handleGetOrgUsers), so an accepted invite for
	// her email should resolve to her static user id.
	testMockServer.SeedOrganizationInvite(inviteOrgId, client.OrganizationInviteData{
		Id:                "invite-ds-accepted",
		Email:             "alice@example.com",
		InvitedBy:         "user-morpheus",
		ExpiresAt:         "2099-01-01T00:00:00Z",
		Status:            "accepted",
		OrganizationRoles: []string{"organization-member"},
	})
	testMockServer.SeedOrganizationInvite(inviteOrgId, client.OrganizationInviteData{
		Id:                "invite-ds-pending",
		Email:             "unknown@nebuchadnezzar.net",
		InvitedBy:         "user-morpheus",
		ExpiresAt:         "2099-01-01T00:00:00Z",
		Status:            "active",
		OrganizationRoles: []string{"organization-member"},
	})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInvitesConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.neo4jaura_organization_invites.this",
						tfjsonpath.New("invites"),
						knownvalue.ListSizeExact(2),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_organization_invites.this",
						tfjsonpath.New("invites").AtSliceIndex(0).AtMapKey("id"),
						knownvalue.StringExact("invite-ds-accepted"),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_organization_invites.this",
						tfjsonpath.New("invites").AtSliceIndex(0).AtMapKey("user_id"),
						knownvalue.StringExact("user-001"),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_organization_invites.this",
						tfjsonpath.New("invites").AtSliceIndex(1).AtMapKey("id"),
						knownvalue.StringExact("invite-ds-pending"),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_organization_invites.this",
						tfjsonpath.New("invites").AtSliceIndex(1).AtMapKey("user_id"),
						knownvalue.Null(),
					),
				},
			},
		},
	})

	testMockServer.Reset()
}
