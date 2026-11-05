data "neo4jaura_databases" "this" {
  organization_id = var.organization_id
  project_id      = var.project_id
  instance_id     = var.instance_id
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

output "databases" {
  value = data.neo4jaura_databases.this.databases
}
