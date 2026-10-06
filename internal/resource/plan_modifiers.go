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

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// UseStateForUnknownOrNull behaves like stringplanmodifier.UseStateForUnknown, except
// that it also copies a *null* state value into the plan.
//
// The stock modifier treats a null state value as "the resource is being created" and
// leaves the planned value unknown. That is wrong for attributes the Aura API only ever
// returns once (username and password, which are returned by POST /instances and by no
// other endpoint): after `terraform import` those attributes are null in state forever,
// so every subsequent plan renders them as "(known after apply)" and any apply that
// reaches Update writes an unknown value back to state, which Terraform rejects.
//
// Resource creation is still detected via the whole prior state being null, so a genuinely
// new resource keeps an unknown planned value for Create to fill in. Terraform re-plans
// with a null prior state when a resource is being replaced, so replacement behaves the
// same way.
func UseStateForUnknownOrNull() planmodifier.String {
	return useStateForUnknownOrNullModifier{}
}

type useStateForUnknownOrNullModifier struct{}

func (m useStateForUnknownOrNullModifier) Description(_ context.Context) string {
	return "Once set, the value of this attribute in state will not change, even when the value in state is null."
}

func (m useStateForUnknownOrNullModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m useStateForUnknownOrNullModifier) PlanModifyString(_ context.Context, request planmodifier.StringRequest, response *planmodifier.StringResponse) {
	// The resource is being created (or replaced, which Terraform re-plans with a null
	// prior state): leave the value unknown so Create can populate it.
	if request.State.Raw.IsNull() {
		return
	}

	// The resource is being destroyed.
	if request.Plan.Raw.IsNull() {
		return
	}

	// A known planned value must not be overwritten.
	if !request.PlanValue.IsUnknown() {
		return
	}

	// The configuration is unknown (e.g. it references another resource's unknown
	// value), so the planned value cannot be resolved yet.
	if request.ConfigValue.IsUnknown() {
		return
	}

	response.PlanValue = request.StateValue
}
