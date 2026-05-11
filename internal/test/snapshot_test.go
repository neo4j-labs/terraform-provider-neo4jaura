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
	"regexp"
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

// testAccSnapshotDataSourceByIdConfig selects a snapshot by explicit snapshot_id,
// exercising the readSnapshotById code path.
const testAccSnapshotDataSourceByIdConfig = defaultProviderConfig + `
data "neo4jaura_snapshot" "this" {
  instance_id = "` + snapTestInstanceID + `"
  snapshot_id = "` + snapTestSnapshotID + `"
}
`

// snapTestInstanceID2 / snapTestSnapshotID2 / snapTestSnapshotID3 are used for the
// multi-snapshot "most recent" test so they don't collide with the main snap constants.
const (
	snapTestInstanceID2  = "snap-test-instance-002"
	snapTestSnapshotID2a = "snap-test-snapshot-002a"
	snapTestSnapshotID2b = "snap-test-snapshot-002b"
)

const testAccSnapshotDataSourceMostRecentConfig = defaultProviderConfig + `
data "neo4jaura_snapshot" "this" {
  instance_id = "` + snapTestInstanceID2 + `"
  most_recent = true
}
`

// snapTestInstanceID3 is for the recently-created instance wait test.
const snapTestInstanceID3 = "snap-test-instance-003"

const testAccSnapshotDataSourceRecentlyCreatedConfig = defaultProviderConfig + `
data "neo4jaura_snapshot" "this" {
  instance_id = "` + snapTestInstanceID3 + `"
  most_recent = true
}
`

// snapTestInstanceID4 is for the "no snapshots, not recently created" error test.
const snapTestInstanceID4 = "snap-test-instance-004"

const testAccSnapshotDataSourceNoSnapshotsConfig = defaultProviderConfig + `
data "neo4jaura_snapshot" "this" {
  instance_id = "` + snapTestInstanceID4 + `"
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
		PreCheck:                 func() { testAccPreCheck(t) },
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

// TestAcc_snapshot_datasource_by_id exercises the readSnapshotById code path by
// specifying snapshot_id explicitly in the datasource config (no most_recent flag).
func TestAcc_snapshot_datasource_by_id(t *testing.T) {
	testMockServer.Reset()

	testMockServer.SeedInstance(client.GetInstanceData{
		Id:     snapTestInstanceID,
		Name:   "Snapshot Test Instance",
		Status: domain.InstanceStatusRunning,
	})
	testMockServer.SeedSnapshot(client.GetSnapshotData{
		InstanceId: snapTestInstanceID,
		SnapshotId: snapTestSnapshotID,
		Profile:    domain.SnapshotProfileScheduled,
		Status:     domain.SnapshotStatusCompleted,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSnapshotDataSourceByIdConfig,
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
						knownvalue.StringExact("Completed"),
					),
				},
			},
		},
	})
}

// TestAcc_snapshot_datasource_most_recent_multiple seeds two snapshots with distinct
// timestamps and verifies that readMostRecentSnapshot returns the one with the later
// timestamp when most_recent=true is used.
func TestAcc_snapshot_datasource_most_recent_multiple(t *testing.T) {
	testMockServer.Reset()

	olderTimestamp := time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)
	newerTimestamp := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)

	testMockServer.SeedInstance(client.GetInstanceData{
		Id:     snapTestInstanceID2,
		Name:   "Multi-Snapshot Test Instance",
		Status: domain.InstanceStatusRunning,
	})
	// Older snapshot seeded first.
	testMockServer.SeedSnapshot(client.GetSnapshotData{
		InstanceId: snapTestInstanceID2,
		SnapshotId: snapTestSnapshotID2a,
		Profile:    domain.SnapshotProfileScheduled,
		Status:     domain.SnapshotStatusCompleted,
		Timestamp:  olderTimestamp,
	})
	// Newer snapshot seeded second.
	testMockServer.SeedSnapshot(client.GetSnapshotData{
		InstanceId: snapTestInstanceID2,
		SnapshotId: snapTestSnapshotID2b,
		Profile:    domain.SnapshotProfileScheduled,
		Status:     domain.SnapshotStatusCompleted,
		Timestamp:  newerTimestamp,
	})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSnapshotDataSourceMostRecentConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.neo4jaura_snapshot.this",
						tfjsonpath.New("instance_id"),
						knownvalue.StringExact(snapTestInstanceID2),
					),
					// The most-recent (newer) snapshot should be returned.
					statecheck.ExpectKnownValue(
						"data.neo4jaura_snapshot.this",
						tfjsonpath.New("snapshot_id"),
						knownvalue.StringExact(snapTestSnapshotID2b),
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

// TestAcc_snapshot_datasource_recently_created_waits exercises the
// isInstanceRecentlyCreated guard in readMostRecentSnapshot. The mock is set up so
// that the first GET /snapshots call returns an empty list (simulating a brand-new
// instance) and subsequent calls return the seeded snapshot. The datasource should
// detect the recently-created instance, invoke WaitUntilSnapshotsMatchCondition, and
// ultimately return the snapshot once the mock starts serving it.
func TestAcc_snapshot_datasource_recently_created_waits(t *testing.T) {
	testMockServer.Reset()

	recentCreatedAt := time.Now().UTC().Add(-1 * time.Minute).Format(time.RFC3339)
	testMockServer.SeedInstance(client.GetInstanceData{
		Id:        snapTestInstanceID3,
		Name:      "Recently Created Instance",
		Status:    domain.InstanceStatusRunning,
		CreatedAt: &recentCreatedAt,
	})
	// Seed the snapshot that will appear after the first empty poll.
	snapshotTimestamp := time.Now().UTC().Format(time.RFC3339)
	testMockServer.SeedSnapshot(client.GetSnapshotData{
		InstanceId: snapTestInstanceID3,
		SnapshotId: "snap-test-snapshot-003",
		Profile:    domain.SnapshotProfileScheduled,
		Status:     domain.SnapshotStatusCompleted,
		Timestamp:  snapshotTimestamp,
	})
	// Make the first call to GET /snapshots return empty, so the datasource
	// invokes isInstanceRecentlyCreated and then WaitUntilSnapshotsMatchCondition.
	testMockServer.HoldSnapshotList(snapTestInstanceID3, 1)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSnapshotDataSourceRecentlyCreatedConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.neo4jaura_snapshot.this",
						tfjsonpath.New("instance_id"),
						knownvalue.StringExact(snapTestInstanceID3),
					),
					statecheck.ExpectKnownValue(
						"data.neo4jaura_snapshot.this",
						tfjsonpath.New("snapshot_id"),
						knownvalue.StringExact("snap-test-snapshot-003"),
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

// TestAcc_snapshot_datasource_no_snapshots_error verifies that readMostRecentSnapshot
// returns an error when the instance has no snapshots and was not recently created.
// The isInstanceRecentlyCreated check returns false (CreatedAt is nil → zero time),
// so the datasource surfaces "Cannot find snapshot" rather than waiting.
func TestAcc_snapshot_datasource_no_snapshots_error(t *testing.T) {
	testMockServer.Reset()

	// Seed an instance with no CreatedAt — CreatedAtAsTime returns time.Time{} (year
	// 0001) which is NOT after now-5m, so isInstanceRecentlyCreated returns false.
	testMockServer.SeedInstance(client.GetInstanceData{
		Id:     snapTestInstanceID4,
		Name:   "Old Instance No Snapshots",
		Status: domain.InstanceStatusRunning,
	})
	// No snapshots seeded for this instance.

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccSnapshotDataSourceNoSnapshotsConfig,
				ExpectError: regexp.MustCompile(`Cannot find snapshot`),
			},
		},
	})
}
