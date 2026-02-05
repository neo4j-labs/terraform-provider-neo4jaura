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

package resource

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/domain"
)

func TestValidateCdcTier(t *testing.T) {
	t.Parallel()

	validator := &cdcTierValidator{}

	cases := map[string]struct {
		model         InstanceResourceModel
		expectedError string
	}{
		"valid_enterprise_db_with_cdc": {
			model: InstanceResourceModel{
				Type:              types.StringValue(domain.InstanceTypeEnterpriseDb),
				CdcEnrichmentMode: types.StringValue(domain.CdcEnrichmentModeDiff),
			},
		},
		"valid_enterprise_ds_with_cdc": {
			model: InstanceResourceModel{
				Type:              types.StringValue(domain.InstanceTypeEnterpriseDs),
				CdcEnrichmentMode: types.StringValue(domain.CdcEnrichmentModeFull),
			},
		},
		"valid_business_critical_with_cdc": {
			model: InstanceResourceModel{
				Type:              types.StringValue(domain.InstanceTypeBusinessCritical),
				CdcEnrichmentMode: types.StringValue(domain.CdcEnrichmentModeDiff),
			},
		},
		"invalid_free_db_with_cdc": {
			model: InstanceResourceModel{
				Type:              types.StringValue(domain.InstanceTypeFreeDb),
				CdcEnrichmentMode: types.StringValue(domain.CdcEnrichmentModeDiff),
			},
			expectedError: "CDC enrichment mode is only supported on business-critical and enterprise instance types. Instance type 'free-db' does not support CDC.",
		},
		"invalid_professional_db_with_cdc": {
			model: InstanceResourceModel{
				Type:              types.StringValue(domain.InstanceTypeProfessionalDb),
				CdcEnrichmentMode: types.StringValue(domain.CdcEnrichmentModeDiff),
			},
			expectedError: "CDC enrichment mode is only supported on business-critical and enterprise instance types. Instance type 'professional-db' does not support CDC.",
		},
		"cdc_not_set": {
			model: InstanceResourceModel{
				Type: types.StringValue(domain.InstanceTypeFreeDb),
			},
		},
		"cdc_null": {
			model: InstanceResourceModel{
				Type:              types.StringValue(domain.InstanceTypeFreeDb),
				CdcEnrichmentMode: types.StringNull(),
			},
		},
		"type_not_set": {
			model: InstanceResourceModel{
				CdcEnrichmentMode: types.StringValue(domain.CdcEnrichmentModeDiff),
			},
		},
		"type_null": {
			model: InstanceResourceModel{
				Type:              types.StringNull(),
				CdcEnrichmentMode: types.StringValue(domain.CdcEnrichmentModeDiff),
			},
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			resp := &resource.ValidateConfigResponse{}
			validator.validateCdcTier(tc.model, resp)

			if tc.expectedError == "" {
				if resp.Diagnostics.HasError() {
					t.Fatalf("expected no error, got: %v", resp.Diagnostics)
				}
			} else {
				if !resp.Diagnostics.HasError() {
					t.Fatalf("expected error, got none")
				}
				found := false
				for _, d := range resp.Diagnostics {
					if d.Summary() == "Invalid Configuration" && d.Detail() == tc.expectedError {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("expected error detail %q, got: %v", tc.expectedError, resp.Diagnostics)
				}
			}
		})
	}
}

func TestValidateVectorOptimized(t *testing.T) {
	t.Parallel()

	validator := &vectorOptimizedValidator{}

	cases := map[string]struct {
		model         InstanceResourceModel
		expectedError string
	}{
		"valid_4gb_optimized": {
			model: InstanceResourceModel{
				Memory:          types.StringValue(domain.InstanceMemory4GB),
				VectorOptimized: types.BoolValue(true),
			},
		},
		"valid_1gb_not_optimized": {
			model: InstanceResourceModel{
				Memory:          types.StringValue(domain.InstanceMemory1GB),
				VectorOptimized: types.BoolValue(false),
			},
		},
		"invalid_1gb_optimized": {
			model: InstanceResourceModel{
				Memory:          types.StringValue(domain.InstanceMemory1GB),
				VectorOptimized: types.BoolValue(true),
			},
			expectedError: "Vector optimization is not supported for instances with 1GB memory.",
		},
		"invalid_2gb_optimized": {
			model: InstanceResourceModel{
				Memory:          types.StringValue(domain.InstanceMemory2GB),
				VectorOptimized: types.BoolValue(true),
			},
			expectedError: "Vector optimization is not supported for instances with 2GB memory.",
		},
		"memory_not_set": {
			model: InstanceResourceModel{
				VectorOptimized: types.BoolValue(true),
			},
		},
		"vector_optimized_not_set": {
			model: InstanceResourceModel{
				Memory: types.StringValue(domain.InstanceMemory1GB),
			},
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			resp := &resource.ValidateConfigResponse{}
			validator.validateVectorOptimized(tc.model, resp)

			if tc.expectedError == "" {
				if resp.Diagnostics.HasError() {
					t.Fatalf("expected no error, got: %v", resp.Diagnostics)
				}
			} else {
				if !resp.Diagnostics.HasError() {
					t.Fatalf("expected error, got none")
				}
				found := false
				for _, d := range resp.Diagnostics {
					if d.Summary() == "Invalid Configuration" && d.Detail() == tc.expectedError {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("expected error detail %q, got: %v", tc.expectedError, resp.Diagnostics)
				}
			}
		})
	}
}
