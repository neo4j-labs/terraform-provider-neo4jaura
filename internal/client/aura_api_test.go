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
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestTranslatedToV2beta1(t *testing.T) {
	tests := []struct {
		instanceType       string
		wantType           string
		wantGraphAnalytics string
	}{
		{instanceType: "enterprise-db", wantType: "virtual-dedicated-cloud", wantGraphAnalytics: "serverless"},
		{instanceType: "enterprise-ds", wantType: "virtual-dedicated-cloud", wantGraphAnalytics: "plugin"},
		{instanceType: "professional-db", wantType: "professional", wantGraphAnalytics: "serverless"},
		{instanceType: "professional-ds", wantType: "professional", wantGraphAnalytics: "plugin"},
		{instanceType: "business-critical", wantType: "business-critical"},
	}
	for _, tt := range tests {
		t.Run(tt.instanceType, func(t *testing.T) {
			got := translatedToV2beta1(PostInstanceV2Request{Type: tt.instanceType})
			if got.Type != tt.wantType {
				t.Errorf("type = %q, want %q", got.Type, tt.wantType)
			}
			gotGraphAnalytics := ""
			if got.GraphAnalytics != nil {
				gotGraphAnalytics = *got.GraphAnalytics
			}
			if gotGraphAnalytics != tt.wantGraphAnalytics {
				t.Errorf("graph_analytics = %q, want %q", gotGraphAnalytics, tt.wantGraphAnalytics)
			}
		})
	}
}

// newOrganizationLookupServer serves two organizations. Listing projects of "forbidden-org" returns 403;
// "owner-org" owns ownedProjectId.
func newOrganizationLookupServer(t *testing.T, ownedProjectId string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, _ *http.Request) {
		writeTestJSON(t, w, TokenResponse{AccessToken: "token", ExpiresIn: 3600})
	})
	mux.HandleFunc("/v2beta1/organizations", func(w http.ResponseWriter, _ *http.Request) {
		organizations := GetOrganizationsResponse{Data: []OrganizationData{{Id: "forbidden-org"}, {Id: "owner-org"}}}
		writeTestJSON(t, w, organizations)
	})
	mux.HandleFunc("/v2beta1/organizations/forbidden-org/projects", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	})
	mux.HandleFunc("/v2beta1/organizations/owner-org/projects", func(w http.ResponseWriter, _ *http.Request) {
		writeTestJSON(t, w, GetProjectsResponse{Data: []ProjectResponseData{{Id: ownedProjectId}}})
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func writeTestJSON(t *testing.T, w http.ResponseWriter, body any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(body); err != nil {
		t.Errorf("encode response: %v", err)
	}
}

func newTestAuraApi(baseURL string) *AuraApi {
	v1Client := NewAuraClient("id", "secret", "test", baseURL, "v1")
	v2beta1Client := NewAuraClient("id", "secret", "test", baseURL, "v2beta1")
	return NewAuraApi(v1Client, v2beta1Client, nil, nil)
}

func TestFindOrganizationIdByProjectId_skipsOrganizationsThatFail(t *testing.T) {
	server := newOrganizationLookupServer(t, "project-1")
	api := newTestAuraApi(server.URL)

	organizationId, err := api.FindOrganizationIdByProjectId(context.Background(), "project-1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if organizationId != "owner-org" {
		t.Errorf("organization id = %q, want %q", organizationId, "owner-org")
	}
}

func TestFindOrganizationIdByProjectId_reportsSkippedOrganizationsWhenNotFound(t *testing.T) {
	server := newOrganizationLookupServer(t, "project-1")
	api := newTestAuraApi(server.URL)

	_, err := api.FindOrganizationIdByProjectId(context.Background(), "unknown-project")

	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
	if !strings.Contains(err.Error(), "forbidden-org") {
		t.Errorf("error %q does not mention the skipped organization", err.Error())
	}
}
