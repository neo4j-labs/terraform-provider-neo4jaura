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

var testAccSnapshotDataSourceConfig = fmt.Sprintf(`
%[1]s
data "neo4jaura_projects" "this" {}

resource "neo4jaura_instance" "this" {
  name           = "MyTestInstance"
  cloud_provider = "gcp"
  region         = "europe-west2"
  memory         = "2GB"
  storage        = "4GB"
  type           = "professional-db"
  project_id     = data.neo4jaura_projects.this.projects.0.id
}

data "neo4jaura_snapshot" "this" {
  instance_id = neo4jaura_instance.this.instance_id
  most_recent = true
}
`, defaultProviderConfig)

func TestAcc_can_read_snapshot_datasource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				ResourceName: "data.neo4jaura_snapshot.this",
				Config:       testAccSnapshotDataSourceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.neo4jaura_snapshot.this",
						tfjsonpath.New("instance_id"),
						knownvalue.StringFunc(nonEmptyString),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_snapshot.this",
						tfjsonpath.New("snapshot_id"),
						knownvalue.StringFunc(nonEmptyString),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_snapshot.this",
						tfjsonpath.New("profile"),
						knownvalue.StringExact("Scheduled"),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_snapshot.this",
						tfjsonpath.New("status"),
						knownvalue.StringFunc(oneOf("Completed", "InProgress")),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_snapshot.this",
						tfjsonpath.New("most_recent"),
						knownvalue.Bool(true),
					),
				},
			},
		},
	})
}
