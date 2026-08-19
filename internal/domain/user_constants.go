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
	OrganizationRoleAdmin  string = "organization-admin"
	OrganizationRoleMember string = "organization-member"
	OrganizationRoleOwner  string = "organization-owner"
)

var OrganizationRoles = []string{
	OrganizationRoleAdmin,
	OrganizationRoleMember,
	OrganizationRoleOwner,
}

const (
	ProjectRoleAdmin                    string = "project-admin"
	ProjectRoleMember                   string = "project-member"
	ProjectRoleMetricsIntegrationReader string = "project-metrics-integration-reader"
	ProjectRoleViewer                   string = "project-viewer"
)

var ProjectRoles = []string{
	ProjectRoleAdmin,
	ProjectRoleMember,
	ProjectRoleMetricsIntegrationReader,
	ProjectRoleViewer,
}

// InviteProjectRoles are the project-role enum values accepted by the
// organization-invite endpoints. These represent the same permission levels
// as ProjectRoles but are spelled with a "namespace-" prefix instead of
// "project-" — the two enums are not interchangeable.
const (
	InviteProjectRoleAdmin                    string = "namespace-admin"
	InviteProjectRoleMember                   string = "namespace-member"
	InviteProjectRoleMetricsIntegrationReader string = "namespace-metrics-integration-reader"
	InviteProjectRoleViewer                   string = "namespace-viewer"
)

var InviteProjectRoles = []string{
	InviteProjectRoleAdmin,
	InviteProjectRoleMember,
	InviteProjectRoleMetricsIntegrationReader,
	InviteProjectRoleViewer,
}

const (
	InviteStatusActive   string = "active"
	InviteStatusAccepted string = "accepted"
	InviteStatusRevoked  string = "revoked"
	InviteStatusExpired  string = "expired"
	InviteStatusDeclined string = "declined"
)

var InviteStatuses = []string{
	InviteStatusActive,
	InviteStatusAccepted,
	InviteStatusRevoked,
	InviteStatusExpired,
	InviteStatusDeclined,
}
