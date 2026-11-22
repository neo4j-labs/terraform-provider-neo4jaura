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

package resource

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
)

// existingResourceState and existingResourcePlan mark the request as an update of an existing
// resource; RequiresReplaceIf skips create (null state) and destroy (null plan).
var existingResourceState = tfsdk.State{Raw: tftypes.NewValue(tftypes.Object{}, map[string]tftypes.Value{})}
var existingResourcePlan = tfsdk.Plan{Raw: tftypes.NewValue(tftypes.Object{}, map[string]tftypes.Value{})}

func TestRequiresReplaceIfStateKnown(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		state           types.String
		plan            types.String
		requiresReplace bool
	}{
		"null_state_filled_in":   {state: types.StringNull(), plan: types.StringValue("org-1"), requiresReplace: false},
		"known_state_changed":    {state: types.StringValue("org-1"), plan: types.StringValue("org-2"), requiresReplace: true},
		"known_state_unchanged":  {state: types.StringValue("org-1"), plan: types.StringValue("org-1"), requiresReplace: false},
		"known_state_to_unknown": {state: types.StringValue("org-1"), plan: types.StringUnknown(), requiresReplace: true},
	}

	for name, example := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			request := planmodifier.StringRequest{
				State:       existingResourceState,
				Plan:        existingResourcePlan,
				StateValue:  example.state,
				PlanValue:   example.plan,
				ConfigValue: example.plan,
			}
			response := &planmodifier.StringResponse{PlanValue: example.plan}

			requiresReplaceIfStateKnown().PlanModifyString(context.Background(), request, response)

			assert.Equal(t, example.requiresReplace, response.RequiresReplace)
		})
	}
}

func TestRequiresReplaceIfStateKnownBool(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		state           types.Bool
		plan            types.Bool
		requiresReplace bool
	}{
		"null_state_filled_in":  {state: types.BoolNull(), plan: types.BoolValue(true), requiresReplace: false},
		"known_state_changed":   {state: types.BoolValue(false), plan: types.BoolValue(true), requiresReplace: true},
		"known_state_unchanged": {state: types.BoolValue(true), plan: types.BoolValue(true), requiresReplace: false},
	}

	for name, example := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			request := planmodifier.BoolRequest{
				State:       existingResourceState,
				Plan:        existingResourcePlan,
				StateValue:  example.state,
				PlanValue:   example.plan,
				ConfigValue: example.plan,
			}
			response := &planmodifier.BoolResponse{PlanValue: example.plan}

			requiresReplaceIfStateKnownBool().PlanModifyBool(context.Background(), request, response)

			assert.Equal(t, example.requiresReplace, response.RequiresReplace)
		})
	}
}
