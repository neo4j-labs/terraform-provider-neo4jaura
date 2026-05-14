variable "source_instance_id" {
  type = string
}

data "neo4jaura_projects" "restored" {}

resource "neo4jaura_instance" "restored" {
  name           = "example-restored-db"
  cloud_provider = "gcp"
  region         = "europe-west2"
  memory         = "2GB"
  storage        = "4GB"
  type           = "professional-db"
  project_id     = data.neo4jaura_projects.restored.projects.0.id

  source = {
    instance_id = var.source_instance_id
  }

  lifecycle {
    ignore_changes = [source]
  }
}
