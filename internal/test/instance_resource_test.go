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
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/stretchr/testify/assert"
)

var freeTierInstanceConfig = fmt.Sprintf(`
%[1]s
data "neo4jaura_projects" "this" {}

resource "neo4jaura_instance" "this" {
  name           = "MyTestFreeInstance"
  cloud_provider = "gcp"
  region         = "europe-west1"
  memory         = "1GB"
  type           = "free-db"
  project_id     = data.neo4jaura_projects.this.projects.0.id
}
`, defaultProviderConfig)

func TestAcc_can_create_instance_resource(t *testing.T) {
	connectionUrlCapturer := &Capturer[string]{}
	usernameCapturer := &Capturer[string]{}
	passwordCapturer := &Capturer[string]{}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create instance
				Config: freeTierInstanceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("instance_id"),
						knownvalue.StringFunc(nonEmptyString),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("name"),
						knownvalue.StringExact("MyTestFreeInstance"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("connection_url"),
						knownvalue.StringFunc(connectionUrlCapturer.Capture(nonEmptyString)),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("username"),
						knownvalue.StringFunc(usernameCapturer.Capture(nonEmptyString)),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("password"),
						knownvalue.StringFunc(passwordCapturer.Capture(nonEmptyString)),
					),
				},
			},
			{
				// Insert some data
				PreConfig: func() {
					err := executeCypher(context.Background(), connectionUrlCapturer.Value, usernameCapturer.Value, passwordCapturer.Value,
						"CREATE (a: Actor {name: 'Keanu Reeves'})-[:PLAYS]->(b: Movie {title: 'The Matrix'})")
					assert.NoError(t, err)
					time.Sleep(time.Minute)
				},
				RefreshState: true,
			},
			{
				// Verify counters are updated
				Config: freeTierInstanceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("graph_nodes"),
						knownvalue.Int64Exact(2),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("graph_relationships"),
						knownvalue.Int64Exact(1),
					),
				},
			},
		},
	})
}
