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
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func nonEmptyString(s string) error {
	if len(strings.TrimSpace(s)) > 0 {
		return nil
	}
	return fmt.Errorf("expected non empty string")
}

func oneOf(options ...string) func(string) error {
	return func(s string) error {
		if slices.Contains(options, strings.TrimSpace(s)) {
			return nil
		}
		return fmt.Errorf("expected one of %v, got %s", options, s)
	}
}

type Capturer[T any] struct {
	Value T
}

func (c *Capturer[T]) Capture(f func(T) error) func(T) error {
	return func(t T) error {
		c.Value = t
		return f(t)
	}
}

func SkipIfNotAcceptance(t *testing.T) {
	if os.Getenv(resource.EnvTfAcc) == "" {
		t.Skipf("Acceptance tests skipped unless env '%s' set", resource.EnvTfAcc)
		return
	}
}

// testAccCheckInstanceDestroyed returns a CheckDestroy function that verifies
// every neo4jaura_instance present in the pre-destroy state no longer exists
// in the given mock server. The state passed by the framework is the state
// *before* destroy, so we read instance_id from that state to check the mock.
func testAccCheckInstanceDestroyed(ms *MockServer) func(*terraform.State) error {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "neo4jaura_instance" {
				continue
			}
			instanceID := rs.Primary.Attributes["instance_id"]
			if instanceID == "" {
				continue
			}
			if ms.InstanceExists(instanceID) {
				return fmt.Errorf("neo4jaura_instance %s still exists in the mock server after destroy", instanceID)
			}
		}
		return nil
	}
}

// testAccCheckSnapshotNotDeleted returns a CheckDestroy function for
// neo4jaura_snapshot resources. Because the Aura API does not expose a delete
// endpoint for snapshots, SnapshotResource.Delete is intentionally a no-op.
// The framework passes the pre-destroy Terraform state to CheckDestroy, so this
// function reads the snapshot IDs from that state and verifies each snapshot
// still exists in the mock server — confirming that no API-level delete was
// issued during destroy.
func testAccCheckSnapshotNotDeleted(ms *MockServer) func(*terraform.State) error {
	return func(s *terraform.State) error {
		for name, rs := range s.RootModule().Resources {
			if rs.Type != "neo4jaura_snapshot" {
				continue
			}
			instanceID := rs.Primary.Attributes["instance_id"]
			snapshotID := rs.Primary.Attributes["snapshot_id"]
			if instanceID == "" || snapshotID == "" {
				continue
			}
			if !ms.SnapshotExists(instanceID, snapshotID) {
				return fmt.Errorf(
					"neo4jaura_snapshot %s (instance_id=%s snapshot_id=%s) was unexpectedly deleted from the mock server during destroy; SnapshotResource.Delete should be a no-op",
					name, instanceID, snapshotID,
				)
			}
		}
		return nil
	}
}

// deleteInstanceOutOfBand returns a resource.TestCheckFunc that reads the
// instance_id attribute from Terraform state for resourceName, then removes
// that instance from the mock server without going through Terraform's delete
// path. Used by TestAcc_instance_disappears to simulate external deletion.
func deleteInstanceOutOfBand(ms *MockServer, resourceName string) func(*terraform.State) error {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}
		id := rs.Primary.Attributes["instance_id"]
		if id == "" {
			return fmt.Errorf("resource %s has no instance_id in state", resourceName)
		}
		ms.DeleteInstance(id)
		return nil
	}
}
