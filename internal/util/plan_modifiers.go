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

package util

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SortedStringList returns a plan modifier that normalizes a List(String) attribute's planned
// value into alphabetical order. This keeps role-list attributes stable regardless of the order
// they're written in configuration or the order the Aura API happens to return them in.
func SortedStringList() planmodifier.List {
	return sortedStringListModifier{}
}

type sortedStringListModifier struct{}

func (m sortedStringListModifier) Description(ctx context.Context) string {
	return m.MarkdownDescription(ctx)
}

func (sortedStringListModifier) MarkdownDescription(context.Context) string {
	return "Stores the list in alphabetical order, independent of configuration or API order."
}

func (sortedStringListModifier) PlanModifyList(_ context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if req.PlanValue.IsUnknown() || req.PlanValue.IsNull() {
		return
	}

	elements := req.PlanValue.Elements()
	values := make([]string, len(elements))
	for i, e := range elements {
		s, ok := e.(types.String)
		if !ok {
			return
		}
		values[i] = s.ValueString()
	}

	sortedValues := SortedStrings(values)
	sortedElements := make([]attr.Value, len(sortedValues))
	for i, v := range sortedValues {
		sortedElements[i] = types.StringValue(v)
	}

	sortedList, diags := types.ListValue(types.StringType, sortedElements)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.PlanValue = sortedList
}
