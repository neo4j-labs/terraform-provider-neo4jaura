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
