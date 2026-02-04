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
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

var professionalTierInstanceConfig = fmt.Sprintf(`
%[1]s
data "neo4jaura_projects" "this" {}

resource "neo4jaura_instance" "this" {
  name           = "MyTestProfessionalInstance"
  cloud_provider = "gcp"
  region         = "europe-west1"
  memory         = "1GB"
  type           = "professional-db"
  project_id     = data.neo4jaura_projects.this.projects.0.id
}
`, defaultProviderConfig)

var businessCriticalTierInstanceConfig = fmt.Sprintf(`
%[1]s
data "neo4jaura_projects" "this" {}

resource "neo4jaura_instance" "this" {
  name                = "TestBusinessCritInstance"
  cloud_provider      = "gcp"
  region              = "us-central1"
  memory              = "8GB"
  type                = "business-critical"
  project_id          = data.neo4jaura_projects.this.projects.0.id
  cdc_enrichment_mode = "FULL"
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

// Test for issue #6: CDC enrichment mode should not cause inconsistent state
// https://github.com/neo4j-labs/terraform-provider-neo4jaura/issues/6
func TestAcc_cdc_enrichment_mode_default_value(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create instance with CDC enrichment mode OFF
				Config: businessCriticalTierInstanceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("instance_id"),
						knownvalue.StringFunc(nonEmptyString),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("name"),
						knownvalue.StringExact("TestBusinessCritInstance"),
					),
					// Verify CDC enrichment mode is correctly set to FULL
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("cdc_enrichment_mode"),
						knownvalue.StringExact("FULL"),
					),
				},
			},
			{
				RefreshState: true,
			},
			{
				// Refresh state to verify no drift (issue #6 bug check)
				// Before the fix, this would cause "inconsistent result" error
				Config: businessCriticalTierInstanceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					// Verify CDC enrichment mode remains FULL after refresh
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("cdc_enrichment_mode"),
						knownvalue.StringExact("FULL"),
					),
				},
			},
		},
	})
}

func TestAcc_can_import_instance_resource(t *testing.T) {
	api := newTestAuraApi()
	examples := []struct {
		name               string
		createResourceFunc func(*testing.T) string
		config             string
		extraStateChecks   []statecheck.StateCheck
		parallel           bool
	}{
		{
			name: "free tier",
			createResourceFunc: func(tt *testing.T) string {
				ctx := context.Background()
				instance, err := api.PostInstance(ctx, client.PostInstanceRequest{
					Version:       "5",
					Name:          "TestFreeTier",
					CloudProvider: "gcp",
					Region:        "europe-west1",
					Memory:        "1GB",
					Type:          "free-db",
					TenantId:      os.Getenv("AURA_PROJECT_ID"),
				})
				require.NoError(tt, err)

				_, err = api.WaitUntilInstanceIsInState(ctx, instance.Data.Id, func(r client.GetInstanceResponse) bool {
					return r.Data.Status == "running"
				})
				require.NoError(tt, err)

				err = executeCypher(ctx, instance.Data.ConnectionUrl, instance.Data.Username, instance.Data.Password,
					"CREATE (a: Actor {name: 'Keanu Reeves'})-[:PLAYS]->(b: Movie {title: 'The Matrix'})")
				require.NoError(tt, err)
				// wait for instance to update nodes sand relationships counters
				time.Sleep(time.Minute)

				return instance.Data.Id
			},
			config: freeTierInstanceConfig,
			extraStateChecks: []statecheck.StateCheck{
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
			parallel: false,
		},

		{
			name: "professional tier",
			createResourceFunc: func(tt *testing.T) string {
				ctx := context.Background()
				instance, err := api.PostInstance(ctx, client.PostInstanceRequest{
					Version:       "5",
					Name:          "TestProfessionalTier",
					CloudProvider: "gcp",
					Region:        "europe-west1",
					Memory:        "1GB",
					Type:          "professional-db",
					TenantId:      os.Getenv("AURA_PROJECT_ID"),
				})
				require.NoError(tt, err)

				_, err = api.WaitUntilInstanceIsInState(ctx, instance.Data.Id, func(r client.GetInstanceResponse) bool {
					return r.Data.Status == "running"
				})
				require.NoError(tt, err)

				return instance.Data.Id
			},
			config:           professionalTierInstanceConfig,
			extraStateChecks: []statecheck.StateCheck{},
			parallel:         true,
		},

		{
			name: "business critical tier",
			createResourceFunc: func(tt *testing.T) string {
				ctx := context.Background()
				instance, err := api.PostInstance(ctx, client.PostInstanceRequest{
					Version:       "5",
					Name:          "TestBusinessCritInstance",
					CloudProvider: "gcp",
					Region:        "us-central1",
					Memory:        "8GB",
					Type:          "business-critical",
					TenantId:      os.Getenv("AURA_PROJECT_ID"),
				})
				require.NoError(tt, err)

				_, err = api.WaitUntilInstanceIsInState(ctx, instance.Data.Id, func(r client.GetInstanceResponse) bool {
					return r.Data.Status == "running"
				})
				require.NoError(tt, err)

				cdcEnrichmentMode := "FULL"
				_, err = api.PatchInstanceById(ctx, instance.Data.Id, client.PatchInstanceRequest{
					CdcEnrichmentMode: &cdcEnrichmentMode,
				})
				require.NoError(tt, err)
				// wait for instance to update CDC enrichment mode
				time.Sleep(time.Minute)

				return instance.Data.Id
			},
			config: businessCriticalTierInstanceConfig,
			extraStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue(
					"neo4jaura_instance.cdc_test",
					tfjsonpath.New("cdc_enrichment_mode"),
					knownvalue.StringExact("FULL"),
				),
			},
			parallel: true,
		},
	}

	for _, example := range examples {
		t.Run(example.name, func(tt *testing.T) {
			if example.parallel {
				tt.Parallel()
			}
			instanceId := example.createResourceFunc(tt)
			defer api.DeleteInstanceById(context.Background(), instanceId)

			stateChecks := []statecheck.StateCheck{
				statecheck.ExpectKnownValue(
					"neo4jaura_instance.this",
					tfjsonpath.New("instance_id"),
					knownvalue.StringExact(instanceId),
				),
			}
			stateChecks = append(stateChecks, example.extraStateChecks...)
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:            example.config,
						ResourceName:      "neo4jaura_instance.this",
						ImportState:       true,
						ImportStateId:     instanceId,
						ConfigStateChecks: stateChecks,
					},
				},
			})
		})
	}
}
