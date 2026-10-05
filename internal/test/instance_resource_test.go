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
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
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

var professionalTierMutableFieldsInitialConfig = fmt.Sprintf(`
%[1]s
data "neo4jaura_projects" "this" {}

resource "neo4jaura_instance" "this" {
  name                   = "MyTestProfessionalInstance"
  cloud_provider         = "gcp"
  region                 = "europe-west1"
  memory                 = "4GB"
  storage                = "8GB"
  type                   = "professional-db"
  project_id             = data.neo4jaura_projects.this.projects.0.id
  vector_optimized       = false
  graph_analytics_plugin = false
}
`, defaultProviderConfig)

var professionalTierMutableFieldsUpdatedConfig = fmt.Sprintf(`
%[1]s
data "neo4jaura_projects" "this" {}

resource "neo4jaura_instance" "this" {
  name                   = "MyTestProfessionalInstance"
  cloud_provider         = "gcp"
  region                 = "europe-west1"
  memory                 = "4GB"
  storage                = "16GB"
  type                   = "professional-db"
  project_id             = data.neo4jaura_projects.this.projects.0.id
  vector_optimized       = true
  graph_analytics_plugin = true
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

func TestAcc_can_update_instance_mutable_patch_fields(t *testing.T) {
	testMockServer.Reset()

	updatedStateChecks := []statecheck.StateCheck{
		statecheck.ExpectKnownValue(
			"neo4jaura_instance.this",
			tfjsonpath.New("storage"),
			knownvalue.StringExact(domain.InstanceStorage16GB),
		),
		statecheck.ExpectKnownValue(
			"neo4jaura_instance.this",
			tfjsonpath.New("vector_optimized"),
			knownvalue.Bool(true),
		),
		statecheck.ExpectKnownValue(
			"neo4jaura_instance.this",
			tfjsonpath.New("graph_analytics_plugin"),
			knownvalue.Bool(true),
		),
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: professionalTierMutableFieldsInitialConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("storage"),
						knownvalue.StringExact(domain.InstanceStorage8GB),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("vector_optimized"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("graph_analytics_plugin"),
						knownvalue.Bool(false),
					),
				},
			},
			{
				Config:            professionalTierMutableFieldsUpdatedConfig,
				ConfigStateChecks: updatedStateChecks,
			},
			{
				RefreshState: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("neo4jaura_instance.this", "storage", domain.InstanceStorage16GB),
					resource.TestCheckResourceAttr("neo4jaura_instance.this", "vector_optimized", "true"),
					resource.TestCheckResourceAttr("neo4jaura_instance.this", "graph_analytics_plugin", "true"),
				),
			},
		},
	})
}

func TestAcc_can_import_instance_resource(t *testing.T) {
	cdcEnrichmentModeFull := domain.CdcEnrichmentModeFull
	secondariesCount := 1

	examples := []struct {
		name     string
		instance client.GetInstanceData
		config   string
		// extraAttrs holds tier specific attributes expected in the imported state,
		// in addition to the ones every instance has.
		extraAttrs map[string]string
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
			config: professionalTierInstanceConfig,
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
			extraAttrs: map[string]string{
				"cdc_enrichment_mode": domain.CdcEnrichmentModeFull,
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
			extraAttrs: map[string]string{
				"secondaries_count": "1",
			},
		},
	}

	for _, example := range examples {
		t.Run(example.name, func(tt *testing.T) {
			testMockServer.Reset()
			testMockServer.SeedInstance(example.instance)
			instanceId := example.instance.Id

			expectedAttrs := map[string]string{
				"instance_id": instanceId,
				"name":        example.instance.Name,
				// Issue #43: project_id and version must be populated on import,
				// otherwise the next plan proposes replacing a live instance.
				"project_id": example.instance.TenantId,
				"version":    domain.InstanceVersion5,
			}
			for name, value := range example.extraAttrs {
				expectedAttrs[name] = value
			}
			resource.Test(tt, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(tt) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				CheckDestroy:             testAccCheckInstanceDestroyed(testMockServer),
				Steps: []resource.TestStep{
					{
						Config:        example.config,
						ResourceName:  "neo4jaura_instance.this",
						ImportState:   true,
						ImportStateId: instanceId,
						// The Aura API only returns the credentials on creation.
						ImportStateCheck: checkImportedAttributes(expectedAttrs, "username", "password"),
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

// TestAcc_instance_pause_resume verifies the three-step pause/resume lifecycle:
//  1. Create a free-tier instance (reaches "running" state).
//  2. Update status to "paused" → triggers PauseInstanceById + WaitUntilInstanceIsInState.
//  3. Update status back to "running" → triggers ResumeInstanceById + WaitUntilInstanceIsInState.
//
// This exercises handlePauseInstance and handleResumeInstance in the mock server,
// as well as the pauseInstance and resumeInstance helpers in InstanceResource.
// Do NOT call t.Parallel() — this test calls testMockServer.Reset().
func TestAcc_instance_pause_resume(t *testing.T) {
	testMockServer.Reset()

	configRunning := fmt.Sprintf(`
%[1]s
data "neo4jaura_projects" "this" {}

resource "neo4jaura_instance" "this" {
  name           = "PauseResumeInstance"
  cloud_provider = "gcp"
  region         = "europe-west1"
  memory         = "1GB"
  type           = "free-db"
  project_id     = data.neo4jaura_projects.this.projects.0.id
  status         = "running"
}
`, defaultProviderConfig)

	configPaused := fmt.Sprintf(`
%[1]s
data "neo4jaura_projects" "this" {}

resource "neo4jaura_instance" "this" {
  name           = "PauseResumeInstance"
  cloud_provider = "gcp"
  region         = "europe-west1"
  memory         = "1GB"
  type           = "free-db"
  project_id     = data.neo4jaura_projects.this.projects.0.id
  status         = "paused"
}
`, defaultProviderConfig)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckInstanceDestroyed(testMockServer),
		Steps: []resource.TestStep{
			{
				// Step 1: create the instance; it should reach "running" state.
				Config: configRunning,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("instance_id"),
						knownvalue.StringFunc(nonEmptyString),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("status"),
						knownvalue.StringExact(domain.InstanceStatusRunning),
					),
				},
			},
			{
				// Step 2: update status to "paused" — exercises pauseInstance + PauseInstanceById.
				Config: configPaused,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("status"),
						knownvalue.StringExact(domain.InstanceStatusPaused),
					),
				},
			},
			{
				// Step 3: resume back to "running" — exercises resumeInstance + ResumeInstanceById.
				Config: configRunning,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("status"),
						knownvalue.StringExact(domain.InstanceStatusRunning),
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

// Test for issue #43: after importing an instance the next plan must be empty.
// Read has to populate project_id and version (both RequiresReplace) from the API,
// and username/password must stay null instead of becoming "known after apply".
// https://github.com/neo4j-labs/terraform-provider-neo4jaura/issues/43
func TestAcc_imported_instance_produces_clean_plan(t *testing.T) {
	const importedInstanceId = "import-clean-plan-id"
	const importedProjectId = "test-project-id-001"

	const importedInstanceConfig = defaultProviderConfig + `
resource "neo4jaura_instance" "this" {
  name           = "ImportedInstance"
  cloud_provider = "gcp"
  region         = "europe-west1"
  memory         = "2GB"
  type           = "professional-db"
  project_id     = "` + importedProjectId + `"
}
`

	const renamedInstanceConfig = defaultProviderConfig + `
resource "neo4jaura_instance" "this" {
  name           = "RenamedImportedInstance"
  cloud_provider = "gcp"
  region         = "europe-west1"
  memory         = "2GB"
  type           = "professional-db"
  project_id     = "` + importedProjectId + `"
}
`

	storage := domain.InstanceStorage4GB
	createdAt := "2024-06-01T12:00:00Z"
	vectorOptimized := false
	graphAnalyticsPlugin := false

	testMockServer.Reset()
	testMockServer.SeedInstance(client.GetInstanceData{
		Id:                   importedInstanceId,
		Name:                 "ImportedInstance",
		Status:               domain.InstanceStatusRunning,
		CloudProvider:        domain.CloudProviderGcp,
		Region:               "europe-west1",
		Memory:               domain.InstanceMemory2GB,
		Type:                 domain.InstanceTypeProfessionalDb,
		TenantId:             importedProjectId,
		ConnectionUrl:        "neo4j+s://import-clean-plan-id.databases.neo4j.io",
		Storage:              &storage,
		CreatedAt:            &createdAt,
		VectorOptimized:      &vectorOptimized,
		GraphAnalyticsPlugin: &graphAnalyticsPlugin,
	})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckInstanceDestroyed(testMockServer),
		Steps: []resource.TestStep{
			{
				Config:             importedInstanceConfig,
				ResourceName:       "neo4jaura_instance.this",
				ImportState:        true,
				ImportStateId:      importedInstanceId,
				ImportStatePersist: true,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("project_id"),
						knownvalue.StringExact(importedProjectId),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("version"),
						knownvalue.StringExact(domain.InstanceVersion5),
					),
					// The Aura API only returns the credentials on creation.
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("username"),
						knownvalue.Null(),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("password"),
						knownvalue.Null(),
					),
				},
			},
			{
				// Fails with "the plan was not empty" if the imported state drifts
				// from the configuration (before the fix: version forced a replacement).
				Config:   importedInstanceConfig,
				PlanOnly: true,
			},
			{
				// An in-place update of an imported instance must not error with an
				// unknown value for the credentials the API never returns, and must
				// not replace the instance.
				Config: renamedInstanceConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("neo4jaura_instance.this", plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("instance_id"),
						knownvalue.StringExact(importedInstanceId),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("name"),
						knownvalue.StringExact("RenamedImportedInstance"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("username"),
						knownvalue.Null(),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("password"),
						knownvalue.Null(),
					),
				},
			},
		},
	})
}
