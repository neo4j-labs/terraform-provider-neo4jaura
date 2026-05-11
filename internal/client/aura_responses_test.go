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
	"testing"
	"time"
)

// helpers to make pointer literals inline
func strPtr(s string) *string { return &s }

// ---------------------------------------------------------------------------
// CanBePaused
// ---------------------------------------------------------------------------

func TestCanBePaused(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{"running (lower) can be paused", "running", true},
		{"running (upper) can be paused", "RUNNING", true},
		{"running (mixed) can be paused", "Running", true},
		{"paused cannot be paused", "paused", false},
		{"creating cannot be paused", "creating", false},
		{"destroying cannot be paused", "destroying", false},
		{"pausing cannot be paused", "pausing", false},
		{"resuming cannot be paused", "resuming", false},
		{"updating cannot be paused", "updating", false},
		{"empty status cannot be paused", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := GetInstanceData{Status: tt.status}
			got := d.CanBePaused()
			if got != tt.want {
				t.Errorf("CanBePaused() = %v, want %v (status=%q)", got, tt.want, tt.status)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CanBeResumed
// ---------------------------------------------------------------------------

func TestCanBeResumed(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{"paused (lower) can be resumed", "paused", true},
		{"paused (upper) can be resumed", "PAUSED", true},
		{"paused (mixed) can be resumed", "Paused", true},
		{"running cannot be resumed", "running", false},
		{"creating cannot be resumed", "creating", false},
		{"destroying cannot be resumed", "destroying", false},
		{"pausing cannot be resumed", "pausing", false},
		{"resuming cannot be resumed", "resuming", false},
		{"updating cannot be resumed", "updating", false},
		{"empty status cannot be resumed", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := GetInstanceData{Status: tt.status}
			got := d.CanBeResumed()
			if got != tt.want {
				t.Errorf("CanBeResumed() = %v, want %v (status=%q)", got, tt.want, tt.status)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CreatedAtAsTime
// ---------------------------------------------------------------------------

func TestCreatedAtAsTime(t *testing.T) {
	validTimestamp := "2024-01-15T10:30:00Z"
	invalidTimestamp := "not-a-timestamp"
	emptyTimestamp := ""

	tests := []struct {
		name      string
		createdAt *string
		wantTime  time.Time
		wantErr   bool
	}{
		{
			name:      "nil CreatedAt returns zero time without error",
			createdAt: nil,
			wantTime:  time.Time{},
			wantErr:   false,
		},
		{
			name:      "valid RFC3339 timestamp parses correctly",
			createdAt: strPtr(validTimestamp),
			wantTime:  mustParseTime(t, validTimestamp),
			wantErr:   false,
		},
		{
			name:      "invalid timestamp returns error",
			createdAt: strPtr(invalidTimestamp),
			wantErr:   true,
		},
		{
			name:      "empty string returns error",
			createdAt: strPtr(emptyTimestamp),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := GetInstanceData{CreatedAt: tt.createdAt}
			got, err := d.CreatedAtAsTime()
			if (err != nil) != tt.wantErr {
				t.Errorf("CreatedAtAsTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.Equal(tt.wantTime) {
				t.Errorf("CreatedAtAsTime() = %v, want %v", got, tt.wantTime)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TimestampAsTime
// ---------------------------------------------------------------------------

func TestTimestampAsTime(t *testing.T) {
	validTimestamp := "2024-03-20T08:00:00Z"
	invalidTimestamp := "2024/03/20 08:00:00"
	emptyTimestamp := ""

	tests := []struct {
		name      string
		timestamp string
		wantTime  time.Time
		wantErr   bool
	}{
		{
			name:      "valid RFC3339 timestamp parses correctly",
			timestamp: validTimestamp,
			wantTime:  mustParseTime(t, validTimestamp),
			wantErr:   false,
		},
		{
			name:      "invalid format returns error",
			timestamp: invalidTimestamp,
			wantErr:   true,
		},
		{
			name:      "empty string returns error",
			timestamp: emptyTimestamp,
			wantErr:   true,
		},
		{
			name:      "RFC3339 with timezone offset parses correctly",
			timestamp: "2024-06-01T12:00:00+02:00",
			wantTime:  mustParseTime(t, "2024-06-01T12:00:00+02:00"),
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := GetSnapshotData{Timestamp: tt.timestamp}
			got, err := d.TimestampAsTime()
			if (err != nil) != tt.wantErr {
				t.Errorf("TimestampAsTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.Equal(tt.wantTime) {
				t.Errorf("TimestampAsTime() = %v, want %v", got, tt.wantTime)
			}
		})
	}
}

// mustParseTime is a test helper that parses an RFC3339 timestamp or fails the test.
func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("mustParseTime: failed to parse %q: %v", s, err)
	}
	return parsed
}
