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

import "strings"

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiredIn   int64  `json:"expires_in"`
}

type GetProjectsResponse struct {
	Data []ProjectResponseData `json:"data"`
}

type ProjectResponseData struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type PostInstanceResponse struct {
	Data PostInstanceData `json:"data"`
}

type PostInstanceData struct {
	Id            string `json:"id"`
	Name          string `json:"name"`
	TenantId      string `json:"tenant_id"`
	CloudProvider string `json:"cloud_provider"`
	ConnectionUrl string `json:"connection_url"`
	Region        string `json:"region"`
	Type          string `json:"type"`
	Username      string `json:"username"`
	Password      string `json:"password"`
}

type GetInstanceResponse struct {
	Data GetInstanceData `json:"data"`
}

type GetInstanceData struct {
	Id                    string  `json:"id"`
	Name                  string  `json:"name"`
	Status                string  `json:"status"`
	TenantId              string  `json:"tenant_id"`
	CloudProvider         string  `json:"cloud_provider"`
	ConnectionUrl         string  `json:"connection_url"`
	Region                string  `json:"region"`
	Type                  string  `json:"type"`
	Memory                string  `json:"memory"`
	Storage               *string `json:"storage"`
	CreatedAt             *string `json:"created_at"`
	MetricsIntegrationUrl *string `json:"metrics_integration_url"`
	GraphNodes            *string `json:"graph_nodes"`
	GraphRelationships    *string `json:"graph_relationships"`
	SecondariesCount      *int    `json:"secondaries_count"`
	CdcEnrichmentMode     *string `json:"cdc_enrichment_mode"`
	VectorOptimized       *bool   `json:"vector_optimized"`
	GraphAnalyticsPlugin  *bool   `json:"graph_analytics_plugin"`
}

func (d GetInstanceData) CanBePaused() bool {
	status := strings.ToLower(d.Status)
	return status == "running"
}

func (d GetInstanceData) CanBeResumed() bool {
	status := strings.ToLower(d.Status)
	return status == "paused"
}

type GetSnapshotsResponse struct {
	Data []GetSnapshotData `json:"data"`
}

type GetSnapshotData struct {
	InstanceId string `json:"instance_id"`
	SnapshotId string `json:"snapshot_id"`
	Profile    string `json:"profile"`
	Status     string `json:"status"`
	Timestamp  string `json:"timestamp"`
}

type GetSnapshotResponse struct {
	Data GetSnapshotData `json:"data"`
}

type PostSnapshotResponse struct {
	Data PostSnapshotData `json:"data"`
}

type PostSnapshotData struct {
	SnapshotId string `json:"snapshot_id"`
}
