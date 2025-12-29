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
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

var testAccProjectsDataSourceConfig = fmt.Sprintf(`
%[1]s
data "neo4jaura_projects" "this" {}
`, defaultProviderConfig)

func TestAcc_can_read_projects_datasource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccProjectsDataSourceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.neo4jaura_projects.this",
						tfjsonpath.New("projects").AtSliceIndex(0).AtMapKey("id"),
						knownvalue.StringRegexp(uuidRegex),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_projects.this",
						tfjsonpath.New("projects").AtSliceIndex(0).AtMapKey("name"),
						knownvalue.StringFunc(nonEmptyString),
					),
				},
			},
		},
	})
}
