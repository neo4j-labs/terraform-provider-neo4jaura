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
	"testing"
)

func TestNoDiagnosticsError(t *testing.T) {
	de := NoDiagnosticsError()
	if de.Message != "" {
		t.Errorf("NoDiagnosticsError().Message = %q, want empty string", de.Message)
	}
	if de.Details != "" {
		t.Errorf("NoDiagnosticsError().Details = %q, want empty string", de.Details)
	}
}

func TestNewDiagnosticsError(t *testing.T) {
	tests := []struct {
		name    string
		message string
		detail  string
	}{
		{
			name:    "both fields set",
			message: "something went wrong",
			detail:  "instance_id=abc123: connection refused",
		},
		{
			name:    "empty message",
			message: "",
			detail:  "some detail",
		},
		{
			name:    "empty detail",
			message: "error occurred",
			detail:  "",
		},
		{
			name:    "both empty",
			message: "",
			detail:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			de := NewDiagnosticsError(tt.message, tt.detail)
			if de.Message != tt.message {
				t.Errorf("Message = %q, want %q", de.Message, tt.message)
			}
			if de.Details != tt.detail {
				t.Errorf("Details = %q, want %q", de.Details, tt.detail)
			}
		})
	}
}

func TestDiagnosticsError_IsNotEmpty(t *testing.T) {
	tests := []struct {
		name string
		de   DiagnosticsError
		want bool
	}{
		{
			name: "empty error is not non-empty",
			de:   NoDiagnosticsError(),
			want: false,
		},
		{
			name: "error with message is non-empty",
			de:   NewDiagnosticsError("error", ""),
			want: true,
		},
		{
			name: "error with detail only is non-empty",
			de:   NewDiagnosticsError("", "some detail"),
			want: true,
		},
		{
			name: "error with both fields is non-empty",
			de:   NewDiagnosticsError("error", "detail"),
			want: true,
		},
		{
			name: "zero value is not non-empty",
			de:   DiagnosticsError{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.de.IsNotEmpty()
			if got != tt.want {
				t.Errorf("IsNotEmpty() = %v, want %v (de=%+v)", got, tt.want, tt.de)
			}
		})
	}
}

func TestDiagnosticsError_NoDiagnosticsError_IsNotNonEmpty(t *testing.T) {
	// Verify that NoDiagnosticsError() and DiagnosticsError{} are equal (same zero value).
	zero := DiagnosticsError{}
	noDiag := NoDiagnosticsError()
	if zero != noDiag {
		t.Errorf("DiagnosticsError{} != NoDiagnosticsError(): got %+v vs %+v", zero, noDiag)
	}
}
