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

type PostInstanceRequest struct {
	Version              string  `json:"version"`
	Region               string  `json:"region"`
	Memory               string  `json:"memory"`
	Name                 string  `json:"name"`
	Type                 string  `json:"type"`
	TenantId             string  `json:"tenant_id"`
	CloudProvider        string  `json:"cloud_provider"`
	Storage              *string `json:"storage,omitempty"`
	SecondariesCount     *int32  `json:"secondaries_count,omitempty"`
	CdcEnrichmentMode    *string `json:"cdc_enrichment_mode,omitempty"`
	VectorOptimized      *bool   `json:"vector_optimized,omitempty"`
	GraphAnalyticsPlugin *bool   `json:"graph_analytics_plugin,omitempty"`
	SourceInstanceId     *string `json:"source_instance_id,omitempty"`
	SourceSnapshotId     *string `json:"source_snapshot_id,omitempty"`
}

type PatchInstanceRequest struct {
	Name              *string `json:"name,omitempty"`
	Memory            *string `json:"memory,omitempty"`
	CdcEnrichmentMode *string `json:"cdc_enrichment_mode,omitempty"`
}
