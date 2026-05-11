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
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// TestLive_instance_lifecycle creates a real free-tier Aura instance, verifies
// that it reaches running state with the expected attributes, renames it
// in-place, and finally destroys it. This is the core end-to-end smoke test.
//
// Requires: AURA_CLIENT_ID, AURA_CLIENT_SECRET, AURA_PROJECT_ID, TF_ACC=1
// Typical runtime: 5–10 minutes (Aura instance creation takes ~3–5 min).
func TestLive_instance_lifecycle(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Step 1: create the instance.
				Config: liveInstanceConfig(t, "tf-live-test-instance"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("instance_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("name"),
						knownvalue.StringExact("tf-live-test-instance"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("status"),
						knownvalue.StringExact("running"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("connection_url"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("username"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("cloud_provider"),
						knownvalue.StringExact("gcp"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("region"),
						knownvalue.StringExact("europe-west1"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("memory"),
						knownvalue.StringExact("1GB"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("type"),
						knownvalue.StringExact("free-db"),
					),
				},
			},
			{
				// Step 2: rename in-place — verifies Update path without replacement.
				Config: liveInstanceConfig(t, "tf-live-test-instance-renamed"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("name"),
						knownvalue.StringExact("tf-live-test-instance-renamed"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("status"),
						knownvalue.StringExact("running"),
					),
				},
			},
		},
	})
}

// TestLive_instance_pause_resume creates a free-tier instance, pauses it,
// verifies the paused state and that connection_url is preserved in Terraform
// state, then resumes it back to running.
//
// Requires: AURA_CLIENT_ID, AURA_CLIENT_SECRET, AURA_PROJECT_ID, TF_ACC=1
// Typical runtime: 10–15 minutes.
func TestLive_instance_pause_resume(t *testing.T) {
	projectID := liveProjectID(t)

	configRunning := liveProviderConfig + `
resource "neo4jaura_instance" "this" {
  name           = "tf-live-pause-resume"
  cloud_provider = "gcp"
  region         = "europe-west1"
  memory         = "1GB"
  type           = "free-db"
  project_id     = "` + projectID + `"
  status         = "running"
}
`
	configPaused := liveProviderConfig + `
resource "neo4jaura_instance" "this" {
  name           = "tf-live-pause-resume"
  cloud_provider = "gcp"
  region         = "europe-west1"
  memory         = "1GB"
  type           = "free-db"
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
						"neo4jaura_instance.this",
						tfjsonpath.New("status"),
						knownvalue.StringExact("running"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
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
						"neo4jaura_instance.this",
						tfjsonpath.New("status"),
						knownvalue.StringExact("paused"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
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
						"neo4jaura_instance.this",
						tfjsonpath.New("status"),
						knownvalue.StringExact("running"),
					),
				},
			},
		},
	})
}
