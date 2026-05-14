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

package client

import (
	"net/http"
	"testing"
)

func TestIsSuccessfulResponseStatus(t *testing.T) {
	tests := []struct {
		name   string
		method string
		status int
		want   bool
	}{
		{"patch 200", http.MethodPatch, http.StatusOK, true},
		{"patch 202", http.MethodPatch, http.StatusAccepted, true},
		{"put 200", http.MethodPut, http.StatusOK, true},
		{"put 202", http.MethodPut, http.StatusAccepted, true},
		{"get 200", http.MethodGet, http.StatusOK, true},
		{"post 200", http.MethodPost, http.StatusOK, true},
		{"delete 200", http.MethodDelete, http.StatusOK, true},
		{"patch 201", http.MethodPatch, http.StatusCreated, false},
		{"put 201", http.MethodPut, http.StatusCreated, false},
		{"get 202", http.MethodGet, http.StatusAccepted, false},
		{"post 202", http.MethodPost, http.StatusAccepted, true},
		{"post 200", http.MethodPost, http.StatusOK, true},
		{"delete 202", http.MethodDelete, http.StatusAccepted, true},
		{"delete 200", http.MethodDelete, http.StatusOK, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSuccessfulResponseStatus(tt.method, tt.status); got != tt.want {
				t.Errorf("isSuccessfulResponseStatus(%q, %d) = %v, want %v", tt.method, tt.status, got, tt.want)
			}
		})
	}
}
