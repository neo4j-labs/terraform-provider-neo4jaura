data "neo4jaura_projects" "protected" {}

resource "neo4jaura_instance" "protected" {
  name           = "example-protected-db"
  cloud_provider = "gcp"
  region         = "europe-west2"
  memory         = "2GB"
  storage        = "4GB"
  type           = "professional-db"
  version        = "5"
  project_id     = data.neo4jaura_projects.protected.projects.0.id

  lifecycle {
    prevent_destroy = true
  }
}
