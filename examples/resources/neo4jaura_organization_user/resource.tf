resource "neo4jaura_organization_user" "this" {
  organization_id      = var.organization_id
  id                   = var.user_id
  organization_roles   = ["organization-member"]
  deregister_on_delete = false
}

variable "organization_id" {
  type = string
}

variable "user_id" {
  type = string
}

output "email" {
  value = neo4jaura_organization_user.this.email
}
