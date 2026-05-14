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

package domain

const (
	InstanceStatusCreating      string = "creating"
	InstanceStatusDestroying    string = "destroying"
	InstanceStatusRunning       string = "running"
	InstanceStatusPausing       string = "pausing"
	InstanceStatusPaused        string = "paused"
	InstanceStatusSuspending    string = "suspending"
	InstanceStatusSuspended     string = "suspended"
	InstanceStatusResuming      string = "resuming"
	InstanceStatusLoading       string = "loading"
	InstanceStatusLoadingFailed string = "loading failed"
	InstanceStatusRestoring     string = "restoring"
	InstanceStatusUpdating      string = "updating"
	InstanceStatusOverwriting   string = "overwriting"
)

const (
	InstanceMemory1GB   string = "1GB"
	InstanceMemory2GB   string = "2GB"
	InstanceMemory4GB   string = "4GB"
	InstanceMemory8GB   string = "8GB"
	InstanceMemory16GB  string = "16GB"
	InstanceMemory24GB  string = "24GB"
	InstanceMemory32GB  string = "32GB"
	InstanceMemory48GB  string = "48GB"
	InstanceMemory64GB  string = "64GB"
	InstanceMemory128GB string = "128GB"
	InstanceMemory192GB string = "192GB"
	InstanceMemory256GB string = "256GB"
	InstanceMemory384GB string = "384GB"
	InstanceMemory512GB string = "512GB"
)

const (
	InstanceTypeEnterpriseDb     string = "enterprise-db"
	InstanceTypeEnterpriseDs     string = "enterprise-ds"
	InstanceTypeProfessionalDb   string = "professional-db"
	InstanceTypeProfessionalDs   string = "professional-ds"
	InstanceTypeFreeDb           string = "free-db"
	InstanceTypeBusinessCritical string = "business-critical"
)

const (
	CloudProviderGcp   string = "gcp"
	CloudProviderAws   string = "aws"
	CloudProviderAzure string = "azure"
)

const (
	InstanceVersion5 string = "5"
)

const (
	InstanceStorage2GB    string = "2GB"
	InstanceStorage4GB    string = "4GB"
	InstanceStorage8GB    string = "8GB"
	InstanceStorage16GB   string = "16GB"
	InstanceStorage32GB   string = "32GB"
	InstanceStorage48GB   string = "48GB"
	InstanceStorage64GB   string = "64GB"
	InstanceStorage96GB   string = "96GB"
	InstanceStorage128GB  string = "128GB"
	InstanceStorage192GB  string = "192GB"
	InstanceStorage256GB  string = "256GB"
	InstanceStorage384GB  string = "384GB"
	InstanceStorage512GB  string = "512GB"
	InstanceStorage768GB  string = "768GB"
	InstanceStorage1024GB string = "1024GB"
	InstanceStorage1536GB string = "1536GB"
	InstanceStorage2048GB string = "2048GB"
)

const (
	CdcEnrichmentModeOff  string = "OFF"
	CdcEnrichmentModeDiff string = "DIFF"
	CdcEnrichmentModeFull string = "FULL"
)
