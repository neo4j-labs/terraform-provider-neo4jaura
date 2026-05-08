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
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/client"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/domain"
)

const (
	snapTestInstanceID = "snap-test-instance-001"
	snapTestSnapshotID = "snap-test-snapshot-001"
)

const testAccSnapshotDataSourceConfig = defaultProviderConfig + `
data "neo4jaura_snapshot" "this" {
  instance_id = "` + snapTestInstanceID + `"
  most_recent = true
}
`

func TestAcc_can_read_snapshot_datasource(t *testing.T) {
	testMockServer.Reset()

	// Pre-seed a running instance so the provider can resolve the instance.
	testMockServer.SeedInstance(client.GetInstanceData{
		Id:     snapTestInstanceID,
		Name:   "Snapshot Test Instance",
		Status: domain.InstanceStatusRunning,
	})

	// Pre-seed a snapshot associated with the known instance ID.
	testMockServer.SeedSnapshot(client.GetSnapshotData{
		InstanceId: snapTestInstanceID,
		SnapshotId: snapTestSnapshotID,
		Profile:    domain.SnapshotProfileScheduled,
		Status:     domain.SnapshotStatusCompleted,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSnapshotDataSourceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.neo4jaura_snapshot.this",
						tfjsonpath.New("instance_id"),
						knownvalue.StringExact(snapTestInstanceID),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_snapshot.this",
						tfjsonpath.New("snapshot_id"),
						knownvalue.StringExact(snapTestSnapshotID),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_snapshot.this",
						tfjsonpath.New("profile"),
						knownvalue.StringExact("Scheduled"),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_snapshot.this",
						tfjsonpath.New("status"),
						knownvalue.StringFunc(oneOf("Completed", "InProgress", "Pending")),
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
