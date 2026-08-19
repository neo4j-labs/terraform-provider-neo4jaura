resource "neo4jaura_project_user" "this" {
  organization_id      = var.organization_id
  project_id           = var.project_id
  user_id              = var.user_id
  project_roles        = ["project-member"]
  deregister_on_delete = false
}

variable "organization_id" {
  type = string
}

variable "project_id" {
  type = string
}

variable "user_id" {
  type = string
}
