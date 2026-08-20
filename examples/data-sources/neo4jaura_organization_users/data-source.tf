data "neo4jaura_organization_users" "this" {
  organization_id = var.organization_id
}

# List only the members of a specific project.
data "neo4jaura_organization_users" "project_members" {
  organization_id = var.organization_id
  project_id      = var.project_id
}

variable "organization_id" {
  type = string
}

variable "project_id" {
  type = string
}

output "user_ids_by_email" {
  value = { for u in data.neo4jaura_organization_users.this.users : u.email => u.id }
}
