resource "neo4jaura_organization_invite" "this" {
  organization_id    = var.organization_id
  email              = "new.teammate@example.com"
  organization_roles = ["organization-member"]

  project_invites = [
    {
      project_id    = var.project_id
      project_roles = ["namespace-member"]
    }
  ]
}

variable "organization_id" {
  type = string
}

variable "project_id" {
  type = string
}

output "invite_status" {
  value = neo4jaura_organization_invite.this.status
}

# Populated once the invite is accepted — see the "Managing users in Aura" guide.
output "accepted_user_id" {
  value = neo4jaura_organization_invite.this.user_id
}
