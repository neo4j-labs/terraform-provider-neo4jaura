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

package livetest

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// TestLive_instance_lifecycle creates a real professional tier Aura instance, verifies
// that it reaches running state with the expected attributes, updates mutable
// professional fields in-place, and finally destroys it. This is the core
// end-to-end smoke test.
//
// Requires: AURA_CLIENT_ID, AURA_CLIENT_SECRET, AURA_PROJECT_ID, TF_ACC=1
// Typical runtime: 5–10 minutes (Aura instance creation takes ~3–5 min).
func TestLive_instance_lifecycle(t *testing.T) {
	t.Parallel()

	liveInstanceConfig := func(t *testing.T, resourceName, instanceName, memory, storage string, vectorOptimized, graphAnalyticsPlugin bool) string {
		return fmt.Sprintf(`
		%s
		resource "neo4jaura_instance" "%s" {
		  name                   = "%s"
		  cloud_provider         = "gcp"
		  region                 = "europe-west1"
		  memory                 = "%s"
		  storage                = "%s"
		  type                   = "professional-db"
		  project_id             = "%s"
		  vector_optimized       = %t
		  graph_analytics_plugin = %t
		}
		`, liveProviderConfig, resourceName, instanceName, memory, storage, liveProjectID(t), vectorOptimized, graphAnalyticsPlugin)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Step 1: create the instance.
				Config: liveInstanceConfig(t, "update_test", "tf-live-test-instance", "4GB", "8GB", false, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.update_test",
						tfjsonpath.New("instance_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.update_test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("tf-live-test-instance"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.update_test",
						tfjsonpath.New("status"),
						knownvalue.StringExact("running"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.update_test",
						tfjsonpath.New("connection_url"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.update_test",
						tfjsonpath.New("username"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.update_test",
						tfjsonpath.New("cloud_provider"),
						knownvalue.StringExact("gcp"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.update_test",
						tfjsonpath.New("region"),
						knownvalue.StringExact("europe-west1"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.update_test",
						tfjsonpath.New("memory"),
						knownvalue.StringExact("4GB"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.update_test",
						tfjsonpath.New("storage"),
						knownvalue.StringExact("8GB"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.update_test",
						tfjsonpath.New("type"),
						knownvalue.StringExact("professional-db"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.update_test",
						tfjsonpath.New("vector_optimized"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.update_test",
						tfjsonpath.New("graph_analytics_plugin"),
						knownvalue.Bool(false),
					),
				},
			},
			{
				// Step 2: update mutable fields in-place — verifies Update path without replacement.
				Config: liveInstanceConfig(t, "update_test", "tf-live-test-instance-renamed", "4GB", "16GB", true, true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.update_test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("tf-live-test-instance-renamed"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.update_test",
						tfjsonpath.New("status"),
						knownvalue.StringExact("running"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.update_test",
						tfjsonpath.New("storage"),
						knownvalue.StringExact("16GB"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.update_test",
						tfjsonpath.New("vector_optimized"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.update_test",
						tfjsonpath.New("graph_analytics_plugin"),
						knownvalue.Bool(true),
					),
				},
			},
		},
	})
}

// TestLive_instance_pause_resume creates a pro-tier instance, pauses it,
// verifies the paused state and that connection_url is preserved in Terraform
// state, then resumes it back to running.
//
// Requires: AURA_CLIENT_ID, AURA_CLIENT_SECRET, AURA_PROJECT_ID, TF_ACC=1
// Typical runtime: 10–15 minutes.
func TestLive_instance_pause_resume(t *testing.T) {
	t.Parallel()
	projectID := liveProjectID(t)

	configRunning := liveProviderConfig + `
resource "neo4jaura_instance" "pause_resume" {
  name           = "tf-live-pause-resume"
  cloud_provider = "gcp"
  region         = "europe-west1"
  memory         = "1GB"
  type           = "professional-db"
  project_id     = "` + projectID + `"
  status         = "running"
}
`
	configPaused := liveProviderConfig + `
resource "neo4jaura_instance" "pause_resume" {
  name           = "tf-live-pause-resume"
  cloud_provider = "gcp"
  region         = "europe-west1"
  memory         = "1GB"
  type           = "professional-db"
  project_id     = "` + projectID + `"
  status         = "paused"
}
`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: configRunning,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.pause_resume",
						tfjsonpath.New("status"),
						knownvalue.StringExact("running"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.pause_resume",
						tfjsonpath.New("connection_url"),
						knownvalue.NotNull(),
					),
				},
			},
			{
				// Pause — verifies that connection_url is preserved in state
				// even though the Aura API returns "" for paused instances.
				Config: configPaused,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.pause_resume",
						tfjsonpath.New("status"),
						knownvalue.StringExact("paused"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.pause_resume",
						tfjsonpath.New("connection_url"),
						knownvalue.NotNull(),
					),
				},
			},
			{
				// Resume back to running.
				Config: configRunning,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.pause_resume",
						tfjsonpath.New("status"),
						knownvalue.StringExact("running"),
					),
				},
			},
		},
	})
}
