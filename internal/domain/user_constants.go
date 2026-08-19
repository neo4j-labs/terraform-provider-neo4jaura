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

import "strings"

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

// The organization-invite endpoints spell project roles with a "namespace-"
// prefix instead of "project-", even though they represent the same
// permission levels as ProjectRoles. The provider only exposes the
// "project-" spelling to users (for consistency with ProjectRoles elsewhere)
// and converts to/from "namespace-" at the API boundary.
const (
	projectRolePrefix = "project-"
	inviteRolePrefix  = "namespace-"
)

// ProjectRoleToInviteRole converts a "project-"-prefixed role to the
// "namespace-"-prefixed spelling the invite endpoints expect on the wire.
func ProjectRoleToInviteRole(role string) string {
	if suffix, ok := strings.CutPrefix(role, projectRolePrefix); ok {
		return inviteRolePrefix + suffix
	}
	return role
}

// InviteRoleToProjectRole converts a "namespace-"-prefixed role, as returned
// by the invite endpoints, back to the "project-" spelling used everywhere
// else in the provider.
func InviteRoleToProjectRole(role string) string {
	if suffix, ok := strings.CutPrefix(role, inviteRolePrefix); ok {
		return projectRolePrefix + suffix
	}
	return role
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
