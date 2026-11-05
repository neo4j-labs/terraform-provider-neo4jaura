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
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/client"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/domain"
)

const (
	instanceV2OrgId      = "test-org-id-001"
	instanceV2ProjectId  = "instance-v2-test-project"
	instanceV2InstanceId = "instance-001"
)

const testAccInstanceV2Config = defaultProviderConfig + `
resource "neo4jaura_instance_v2" "this" {
  organization_id = "` + instanceV2OrgId + `"
  project_id      = "` + instanceV2ProjectId + `"
  name            = "v2-test-instance"
  region          = "europe-west1"
  memory          = "4GB"
  storage         = "8GB"
  multi_database  = true
}
`

const testAccInstanceV2ResizedConfig = defaultProviderConfig + `
resource "neo4jaura_instance_v2" "this" {
  organization_id = "` + instanceV2OrgId + `"
  project_id      = "` + instanceV2ProjectId + `"
  name            = "v2-test-instance"
  region          = "europe-west1"
  memory          = "8GB"
  storage         = "16GB"
  multi_database  = true
}
`

// TestAcc_instance_v2_lifecycle covers create via the v2beta1 payload (with
// multi_database enabled), in-place memory/storage resize (PATCH), and destroy.
func TestAcc_instance_v2_lifecycle(t *testing.T) {
	testMockServer.Reset()

	// The mock generates the instance ID at creation time, so it is captured
	// from state and reused by CheckDestroy.
	var createdInstanceId string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: func(_ *terraform.State) error {
			if createdInstanceId != "" && testMockServer.InstanceExists(createdInstanceId) {
				return fmt.Errorf("instance %s still exists after destroy", createdInstanceId)
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceV2Config,
				Check: func(s *terraform.State) error {
					rs, ok := s.RootModule().Resources["neo4jaura_instance_v2.this"]
					if !ok {
						return fmt.Errorf("neo4jaura_instance_v2.this not found in state")
					}
					createdInstanceId = rs.Primary.Attributes["instance_id"]
					return nil
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_instance_v2.this",
						tfjsonpath.New("organization_id"),
						knownvalue.StringExact(instanceV2OrgId),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance_v2.this",
						tfjsonpath.New("project_id"),
						knownvalue.StringExact(instanceV2ProjectId),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance_v2.this",
						tfjsonpath.New("name"),
						knownvalue.StringExact("v2-test-instance"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance_v2.this",
						tfjsonpath.New("instance_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance_v2.this",
						tfjsonpath.New("status"),
						knownvalue.StringExact("running"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance_v2.this",
						tfjsonpath.New("multi_database"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance_v2.this",
						tfjsonpath.New("connection_url"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance_v2.this",
						tfjsonpath.New("username"),
						knownvalue.StringExact("neo4j"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance_v2.this",
						tfjsonpath.New("password"),
						knownvalue.StringExact("mock-password"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance_v2.this",
						tfjsonpath.New("default_database_id"),
						knownvalue.StringExact("neo4j"),
					),
				},
			},
			{
				// In-place memory/storage resize via the v2beta1 PATCH.
				Config: testAccInstanceV2ResizedConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_instance_v2.this",
						tfjsonpath.New("memory"),
						knownvalue.StringExact("8GB"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance_v2.this",
						tfjsonpath.New("storage"),
						knownvalue.StringExact("16GB"),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance_v2.this",
						tfjsonpath.New("status"),
						knownvalue.StringExact("running"),
					),
				},
			},
		},
	})
}

// TestAcc_instance_v2_import imports an existing instance by composite ID
// ({organization_id}/{project_id}/{instance_id}) and verifies the imported
// state round-trips without a diff.
func TestAcc_instance_v2_import(t *testing.T) {
	testMockServer.Reset()

	lastActivity := "2024-06-01T12:00:00Z"
	testMockServer.SeedInstance(client.GetInstanceData{
		Id:            instanceV2InstanceId,
		Name:          "imported-v2-instance",
		Status:        domain.InstanceStatusRunning,
		TenantId:      instanceV2ProjectId,
		CloudProvider: "gcp",
		Region:        "europe-west1",
		Type:          domain.InstanceTypeBusinessCritical,
		Memory:        "4GB",
		CreatedAt:     &lastActivity,
	})

	importID := instanceV2OrgId + "/" + instanceV2ProjectId + "/" + instanceV2InstanceId

	importConfig := defaultProviderConfig + `
resource "neo4jaura_instance_v2" "this" {
  organization_id = "` + instanceV2OrgId + `"
  project_id      = "` + instanceV2ProjectId + `"
  name            = "imported-v2-instance"
  region          = "europe-west1"
  memory          = "4GB"
}
`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             importConfig,
				ResourceName:       "neo4jaura_instance_v2.this",
				ImportState:        true,
				ImportStateId:      importID,
				ImportStatePersist: true,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_instance_v2.this",
						tfjsonpath.New("organization_id"),
						knownvalue.StringExact(instanceV2OrgId),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance_v2.this",
						tfjsonpath.New("project_id"),
						knownvalue.StringExact(instanceV2ProjectId),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance_v2.this",
						tfjsonpath.New("instance_id"),
						knownvalue.StringExact(instanceV2InstanceId),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_instance_v2.this",
						tfjsonpath.New("name"),
						knownvalue.StringExact("imported-v2-instance"),
					),
				},
			},
			{
				// Fails with "the plan was not empty" if the imported state does not
				// round-trip after Read refreshes it from the API.
				Config:   importConfig,
				PlanOnly: true,
			},
		},
	})
}

// TestAcc_instance_v2_multi_database_invalid_tier verifies that enabling
// multi_database on an instance type that does not support it is rejected by
// the plan-time validator.
func TestAcc_instance_v2_multi_database_invalid_tier(t *testing.T) {
	testMockServer.Reset()

	invalidTierConfig := defaultProviderConfig + `
resource "neo4jaura_instance_v2" "this" {
  organization_id = "` + instanceV2OrgId + `"
  project_id      = "` + instanceV2ProjectId + `"
  name            = "v2-professional-instance"
  region          = "europe-west1"
  memory          = "1GB"
  type            = "professional-db"
  multi_database  = true
}
`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      invalidTierConfig,
				ExpectError: regexp.MustCompile(`Unsupported instance type for multi_database`),
			},
		},
	})
}
