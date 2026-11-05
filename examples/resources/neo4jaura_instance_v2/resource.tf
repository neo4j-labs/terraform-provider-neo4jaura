resource "neo4jaura_instance_v2" "this" {
  organization_id = var.organization_id
  project_id      = var.project_id
  name            = "example-multidb-instance"
  cloud_provider  = "gcp"
  region          = "europe-west1"
  memory          = "16GB"
  type            = "business-critical"
  multi_database  = true
}

variable "organization_id" {
  type = string
}

variable "project_id" {
  type = string
}

output "instance_id" {
  value = neo4jaura_instance_v2.this.instance_id
}

output "connection_url" {
  value = neo4jaura_instance_v2.this.connection_url
}
