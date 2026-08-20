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

const testAccOrganizationsDataSourceConfig = defaultProviderConfig + `
data "neo4jaura_organizations" "this" {}
`

func TestAcc_can_read_organizations_datasource(t *testing.T) {
	testMockServer.Reset()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationsDataSourceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.neo4jaura_organizations.this",
						tfjsonpath.New("organizations").AtSliceIndex(0).AtMapKey("id"),
						knownvalue.StringExact("test-org-id-001"),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_organizations.this",
						tfjsonpath.New("organizations").AtSliceIndex(0).AtMapKey("name"),
						knownvalue.StringExact("Test Organization"),
					),
				},
			},
		},
	})
}
