data "neo4jaura_organization_invites" "this" {
  organization_id = var.organization_id
}

variable "organization_id" {
  type = string
}

output "pending_invites" {
  value = [
    for invite in data.neo4jaura_organization_invites.this.invites : invite
    if invite.status == "active"
  ]
}

# user_id is only populated for invites with status "accepted".
output "accepted_user_ids" {
  value = {
    for invite in data.neo4jaura_organization_invites.this.invites :
    invite.email => invite.user_id
    if invite.status == "accepted"
  }
}
