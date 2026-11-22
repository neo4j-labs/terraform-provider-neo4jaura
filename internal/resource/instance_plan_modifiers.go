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

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

const requiresReplaceIfStateKnownDescription = "If the value is already known in state, changing it destroys and recreates the resource. " +
	"Setting a value that state holds as null updates the resource in place."

// requiresReplaceIfStateKnown requires replacement only when state holds a known, non-null value that
// differs from the plan. Filling in a value that state holds as null (for example organization_id after
// a failed lookup) is an in-place update, so it cannot destroy the instance.
func requiresReplaceIfStateKnown() planmodifier.String {
	return stringplanmodifier.RequiresReplaceIf(
		func(_ context.Context, request planmodifier.StringRequest, response *stringplanmodifier.RequiresReplaceIfFuncResponse) {
			response.RequiresReplace = isKnownValue(request.StateValue)
		},
		requiresReplaceIfStateKnownDescription,
		requiresReplaceIfStateKnownDescription,
	)
}

// requiresReplaceIfStateKnownBool is the Bool variant of requiresReplaceIfStateKnown.
func requiresReplaceIfStateKnownBool() planmodifier.Bool {
	return boolplanmodifier.RequiresReplaceIf(
		func(_ context.Context, request planmodifier.BoolRequest, response *boolplanmodifier.RequiresReplaceIfFuncResponse) {
			response.RequiresReplace = isKnownValue(request.StateValue)
		},
		requiresReplaceIfStateKnownDescription,
		requiresReplaceIfStateKnownDescription,
	)
}

// isKnownValue reports whether value is neither null nor unknown.
func isKnownValue(value attr.Value) bool {
	return !value.IsNull() && !value.IsUnknown()
}
