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

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/client"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestAcc_can_import_snapshot(t *testing.T) {
	SkipIfNotAcceptance(t)
	t.Parallel()

	testInstanceConfig := fmt.Sprintf(`
%[1]s
data "neo4jaura_projects" "this" {}

resource "neo4jaura_instance" "this" {
  name           = "TestProInstanceWithSnapshot"
  cloud_provider = "gcp"
  region         = "europe-west1"
  memory         = "1GB"
  type           = "professional-db"
  project_id     = data.neo4jaura_projects.this.projects.0.id
}
`, defaultProviderConfig)

	testInstanceWithSnapshot := fmt.Sprintf(`
%[1]s
resource "neo4jaura_snapshot" "this" {}
`, testInstanceConfig)

	api := newTestAuraApi()

	instanceIdCapturer := &Capturer[string]{}
	var snapshotId string
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		// Create an instance
		Steps: []resource.TestStep{
			{
				// Create an instance
				Config: testInstanceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_instance.this",
						tfjsonpath.New("instance_id"),
						knownvalue.StringFunc(instanceIdCapturer.Capture(nonEmptyString)),
					),
				},
			},
			{
				// Wait for snapshot to be created
				PreConfig: func() {
					resp, err := api.WaitUntilSnapshotsMatchCondition(context.Background(), instanceIdCapturer.Value, func(data client.GetSnapshotsResponse) bool {
						return len(data.Data) > 0
					})
					require.NoError(t, err)
					snapshotId = resp.Data[0].SnapshotId
				},
				RefreshState: true,
			},
			{
				Config:       testInstanceWithSnapshot,
				ResourceName: "neo4jaura_snapshot.this",
				ImportState:  true,
				ImportStateIdFunc: func(_ *terraform.State) (string, error) {
					return fmt.Sprintf("%s,%s", instanceIdCapturer.Value, snapshotId), nil
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"neo4jaura_snapshot.this",
						tfjsonpath.New("instance_id"),
						knownvalue.StringExact(instanceIdCapturer.Value),
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
						tfjsonpath.New("profile"),
						knownvalue.StringFunc(oneOf(
							domain.SnapshotStatusInProgress,
							domain.SnapshotStatusCompleted,
							domain.SnapshotStatusPending,
						)),
					),
				},
			},
		},
	})

}
