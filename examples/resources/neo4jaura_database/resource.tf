resource "neo4jaura_database" "this" {
  organization_id = var.organization_id
  project_id      = var.project_id
  instance_id     = var.instance_id
  name            = "my-database"
}

variable "organization_id" {
  type = string
}

variable "project_id" {
  type = string
}

variable "instance_id" {
  type = string
}

output "database_id" {
  value = neo4jaura_database.this.database_id
}
