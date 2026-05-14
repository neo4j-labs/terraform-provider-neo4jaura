data "neo4jaura_projects" "this" {}

resource "neo4jaura_instance" "this" {
  name           = "example-professional-db"
  cloud_provider = "gcp"
  region         = "europe-west2"
  memory         = "2GB"
  storage        = "4GB"
  type           = "professional-db"
  project_id     = data.neo4jaura_projects.this.projects.0.id
}

output "connection_url" {
  value = neo4jaura_instance.this.connection_url
}

output "username" {
  value = neo4jaura_instance.this.username
}

output "password" {
  value     = neo4jaura_instance.this.password
  sensitive = true
}
