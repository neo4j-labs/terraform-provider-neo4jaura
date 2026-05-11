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
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/client"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/domain"
)

func TestAcc_can_import_snapshot(t *testing.T) {
	instanceId := "import-snap-instance-001"
	snapshotId := "import-snap-snapshot-001"

	testConfig := fmt.Sprintf(`
%s
resource "neo4jaura_snapshot" "this" {}
`, defaultProviderConfig)

	testMockServer.Reset()
	testMockServer.SeedInstance(client.GetInstanceData{
		Id:     instanceId,
		Name:   "import-snap-instance",
		Status: domain.InstanceStatusRunning,
	})
	testMockServer.SeedSnapshot(client.GetSnapshotData{
		InstanceId: instanceId,
		SnapshotId: snapshotId,
		Profile:    domain.SnapshotProfileScheduled,
		Status:     domain.SnapshotStatusCompleted,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:        testConfig,
				ResourceName:  "neo4jaura_snapshot.this",
				ImportState:   true,
				ImportStateId: fmt.Sprintf("%s,%s", instanceId, snapshotId),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_snapshot.this",
						tfjsonpath.New("instance_id"),
						knownvalue.StringExact(instanceId),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_snapshot.this",
						tfjsonpath.New("snapshot_id"),
						knownvalue.StringExact(snapshotId),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_snapshot.this",
						tfjsonpath.New("profile"),
						knownvalue.StringExact(domain.SnapshotProfileScheduled),
					),
					statecheck.ExpectKnownValue(
						"neo4jaura_snapshot.this",
						tfjsonpath.New("status"),
						knownvalue.StringExact(domain.SnapshotStatusCompleted),
					),
				},
			},
		},
	})
}
