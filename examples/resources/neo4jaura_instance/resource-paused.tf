data "neo4jaura_projects" "paused" {}

resource "neo4jaura_instance" "paused" {
  name           = "example-paused-db"
  cloud_provider = "gcp"
  region         = "europe-west2"
  memory         = "2GB"
  storage        = "4GB"
  type           = "professional-db"
  project_id     = data.neo4jaura_projects.paused.projects.0.id
  status         = var.paused ? "paused" : "running"
}

variable "paused" {
  type    = bool
  default = true
}
