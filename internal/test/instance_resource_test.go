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
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/client"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/domain"
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

var businessCriticalWithSecondariesConfig = fmt.Sprintf(`
%[1]s
data "neo4jaura_projects" "this" {}

resource "neo4jaura_instance" "this" {
  name                = "TestBusinessCritSecondaries"
  cloud_provider      = "gcp"
  region              = "us-central1"
  memory              = "8GB"
  type                = "business-critical"
  project_id          = data.neo4jaura_projects.this.projects.0.id
  secondaries_count   = 1
}
`, defaultProviderConfig)

func TestAcc_can_create_instance_resource(t *testing.T) {
	testMockServer.Reset()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckInstanceDestroyed(testMockServer),
		Steps: []resource.TestStep{
			{
				// Create instance and verify it reaches running state with mock preset values
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
						tfjsonpath.New("graph_nodes"),
						knownvalue.Int64Exact(10),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("graph_relationships"),
						knownvalue.Int64Exact(5),
					),
				},
			},
		},
	})
}

// Test for issue #6: CDC enrichment mode should not cause inconsistent state
// https://github.com/neo4j-labs/terraform-provider-neo4jaura/issues/6
func TestAcc_cdc_enrichment_mode_default_value(t *testing.T) {
	testMockServer.Reset()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckInstanceDestroyed(testMockServer),
		Steps: []resource.TestStep{
			{
				// Create instance with CDC enrichment mode FULL
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
						knownvalue.StringExact(domain.CdcEnrichmentModeFull),
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
						knownvalue.StringExact(domain.CdcEnrichmentModeFull),
					),
				},
			},
		},
	})
}

// Test that secondaries_count is applied via PATCH after create and does not drift when API omits it on read
func TestAcc_secondaries_count_no_drift(t *testing.T) {
	testMockServer.Reset()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckInstanceDestroyed(testMockServer),
		Steps: []resource.TestStep{
			{
				// Create instance with secondaries_count
				Config: businessCriticalWithSecondariesConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("instance_id"),
						knownvalue.StringFunc(nonEmptyString),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("name"),
						knownvalue.StringExact("TestBusinessCritSecondaries"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("secondaries_count"),
						knownvalue.Int32Exact(1),
					),
				},
			},
			{
				RefreshState: true,
			},
			{
				// Verify secondaries_count remains after refresh (no drift when API omits field)
				Config: businessCriticalWithSecondariesConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("secondaries_count"),
						knownvalue.Int32Exact(1),
					),
				},
			},
		},
	})
}

func TestAcc_can_import_instance_resource(t *testing.T) {
	cdcEnrichmentModeFull := domain.CdcEnrichmentModeFull
	secondariesCount := 1

	examples := []struct {
		name             string
		instance         client.GetInstanceData
		config           string
		extraStateChecks []statecheck.StateCheck
	}{
		{
			name: "free tier",
			instance: client.GetInstanceData{
				Id:            "import-free-tier-id",
				Name:          "MyTestFreeInstance",
				Status:        domain.InstanceStatusRunning,
				CloudProvider: domain.CloudProviderGcp,
				Region:        "europe-west1",
				Memory:        domain.InstanceMemory1GB,
				Type:          domain.InstanceTypeFreeDb,
				TenantId:      "test-project-id-001",
			},
			config: freeTierInstanceConfig,
			extraStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue(
					"neo4jaura_instance.this",
					tfjsonpath.New("graph_nodes"),
					knownvalue.Int64Exact(10),
				),
				statecheck.ExpectKnownValue(
					"neo4jaura_instance.this",
					tfjsonpath.New("graph_relationships"),
					knownvalue.Int64Exact(5),
				),
			},
		},

		{
			name: "professional tier",
			instance: client.GetInstanceData{
				Id:            "import-professional-tier-id",
				Name:          "MyTestProfessionalInstance",
				Status:        domain.InstanceStatusRunning,
				CloudProvider: domain.CloudProviderGcp,
				Region:        "europe-west1",
				Memory:        domain.InstanceMemory1GB,
				Type:          domain.InstanceTypeProfessionalDb,
				TenantId:      "test-project-id-001",
			},
			config:           professionalTierInstanceConfig,
			extraStateChecks: []statecheck.StateCheck{},
		},

		{
			name: "business critical tier",
			instance: client.GetInstanceData{
				Id:                "import-business-critical-tier-id",
				Name:              "TestBusinessCritInstance",
				Status:            domain.InstanceStatusRunning,
				CloudProvider:     domain.CloudProviderGcp,
				Region:            "us-central1",
				Memory:            domain.InstanceMemory8GB,
				Type:              domain.InstanceTypeBusinessCritical,
				TenantId:          "test-project-id-001",
				CdcEnrichmentMode: &cdcEnrichmentModeFull,
			},
			config: businessCriticalTierInstanceConfig,
			extraStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue(
					"neo4jaura_instance.this",
					tfjsonpath.New("cdc_enrichment_mode"),
					knownvalue.StringExact(domain.CdcEnrichmentModeFull),
				),
			},
		},

		{
			name: "business critical tier with secondaries_count",
			instance: client.GetInstanceData{
				Id:               "import-business-critical-secondaries-id",
				Name:             "TestBusinessCritSecondaries",
				Status:           domain.InstanceStatusRunning,
				CloudProvider:    domain.CloudProviderGcp,
				Region:           "us-central1",
				Memory:           domain.InstanceMemory8GB,
				Type:             domain.InstanceTypeBusinessCritical,
				TenantId:         "test-project-id-001",
				SecondariesCount: &secondariesCount,
			},
			config: businessCriticalWithSecondariesConfig,
			extraStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue(
					"neo4jaura_instance.this",
					tfjsonpath.New("secondaries_count"),
					knownvalue.Int32Exact(1),
				),
			},
		},
	}

	for _, example := range examples {
		t.Run(example.name, func(tt *testing.T) {
			testMockServer.Reset()
			testMockServer.SeedInstance(example.instance)
			instanceId := example.instance.Id

			stateChecks := []statecheck.StateCheck{
				statecheck.ExpectKnownValue(
					"neo4jaura_instance.this",
					tfjsonpath.New("instance_id"),
					knownvalue.StringExact(instanceId),
				),
			}
			stateChecks = append(stateChecks, example.extraStateChecks...)
			resource.Test(tt, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(tt) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				CheckDestroy:             testAccCheckInstanceDestroyed(testMockServer),
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

// TestAcc_instance_update verifies that an in-place name change is applied
// without replacing the resource (the instance_id must remain the same).
func TestAcc_instance_update(t *testing.T) {
	testMockServer.Reset()

	configStep1 := fmt.Sprintf(`
%[1]s
data "neo4jaura_projects" "this" {}

resource "neo4jaura_instance" "this" {
  name           = "OriginalName"
  cloud_provider = "gcp"
  region         = "europe-west1"
  memory         = "1GB"
  type           = "free-db"
  project_id     = data.neo4jaura_projects.this.projects.0.id
}
`, defaultProviderConfig)

	configStep2 := fmt.Sprintf(`
%[1]s
data "neo4jaura_projects" "this" {}

resource "neo4jaura_instance" "this" {
  name           = "UpdatedName"
  cloud_provider = "gcp"
  region         = "europe-west1"
  memory         = "1GB"
  type           = "free-db"
  project_id     = data.neo4jaura_projects.this.projects.0.id
}
`, defaultProviderConfig)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckInstanceDestroyed(testMockServer),
		Steps: []resource.TestStep{
			{
				// Step 1: create the instance with the original name.
				Config: configStep1,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("name"),
						knownvalue.StringExact("OriginalName"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("instance_id"),
						knownvalue.StringFunc(nonEmptyString),
					),
				},
			},
			{
				// Step 2: update the name in-place. The instance_id must not change.
				Config: configStep2,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("name"),
						knownvalue.StringExact("UpdatedName"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("instance_id"),
						knownvalue.StringFunc(nonEmptyString),
					),
				},
			},
		},
	})
}

// TestAcc_instance_disappears verifies that when an instance is deleted
// out-of-band (without going through Terraform), the post-apply refresh
// produces a non-empty plan (recreation proposed) rather than erroring.
// This relies on InstanceResource.Read calling resp.State.RemoveResource
// on 404 (task-001). The framework automatically runs a refresh after the
// step's Check; ExpectNonEmptyPlan: true asserts that refresh shows a diff.
func TestAcc_instance_disappears(t *testing.T) {
	testMockServer.Reset()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckInstanceDestroyed(testMockServer),
		Steps: []resource.TestStep{
			{
				// Create the instance, then delete it out-of-band inside Check.
				// The framework's post-step refresh then calls Read, which returns
				// 404 and removes the resource from state, producing a non-empty plan.
				Config:             freeTierInstanceConfig,
				Check:              deleteInstanceOutOfBand(testMockServer, "neo4jaura_instance.this"),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
