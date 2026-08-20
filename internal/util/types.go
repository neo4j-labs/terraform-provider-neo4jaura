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
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ToStringSlice converts []types.String to []string.
func ToStringSlice(ts []types.String) []string {
	result := make([]string, len(ts))
	for i, t := range ts {
		result[i] = t.ValueString()
	}
	return result
}

// ToTypesStringSlice converts []string to []types.String.
func ToTypesStringSlice(ss []string) []types.String {
	result := make([]types.String, len(ss))
	for i, s := range ss {
		result[i] = types.StringValue(s)
	}
	return result
}

// SortedStrings returns a sorted copy of ss, leaving the input untouched.
func SortedStrings(ss []string) []string {
	sorted := append([]string(nil), ss...)
	sort.Strings(sorted)
	return sorted
}

// SlicesEqualIgnoringOrder checks whether two string slices contain the same elements (order-independent).
func SlicesEqualIgnoringOrder(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	counts := make(map[string]int, len(a))
	for _, r := range a {
		counts[r]++
	}
	for _, r := range b {
		counts[r]--
		if counts[r] < 0 {
			return false
		}
	}
	return true
}
