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
	MfaEnrollmentStatusEnrolled             string = "enrolled"
	MfaEnrollmentStatusNotEnrolled          string = "not_enrolled"
	MfaEnrollmentStatusExternalAuthProvider string = "external_auth_provider"
	MfaEnrollmentStatusNonApplicable        string = "non_applicable"
)

var MfaEnrollmentStatuses = []string{
	MfaEnrollmentStatusEnrolled,
	MfaEnrollmentStatusNotEnrolled,
	MfaEnrollmentStatusExternalAuthProvider,
	MfaEnrollmentStatusNonApplicable,
}

const (
	OrganizationRoleOwner  string = "organization-owner"
	OrganizationRoleAdmin  string = "organization-admin"
	OrganizationRoleMember string = "organization-member"
)

var OrganizationRoles = []string{
	OrganizationRoleOwner,
	OrganizationRoleAdmin,
	OrganizationRoleMember,
}

const (
	ProjectRoleAdmin                    string = "project-admin"
	ProjectRoleMember                   string = "project-member"
	ProjectRoleViewer                   string = "project-viewer"
	ProjectRoleMetricsIntegrationReader string = "project-metrics-integration-reader"
)

var ProjectRoles = []string{
	ProjectRoleAdmin,
	ProjectRoleMember,
	ProjectRoleViewer,
	ProjectRoleMetricsIntegrationReader,
}
