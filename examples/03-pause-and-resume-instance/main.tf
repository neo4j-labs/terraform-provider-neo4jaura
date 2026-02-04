terraform {
  required_version = ">= 1.13.4"
  required_providers {
    neo4jaura = {
      source = "neo4j-labs/neo4jaura"
      version = "0.0.2-beta"
    }
  }
}

provider "neo4jaura" {
  client_id     = var.client_id
  client_secret = var.client_secret
}

resource "neo4jaura_instance" "this" {
  name           = "MyThirdInstance"
  cloud_provider = "gcp"
  region         = "europe-west2"
  memory         = "1GB"
  type           = "professional-db"
  project_id     = data.neo4jaura_projects.this.projects.0.id
  status         = var.paused ? "paused" : "running"
}

data "neo4jaura_projects" "this" {}

variable "client_id" {}
variable "client_secret" {}

variable "paused" {
  type    = bool
  default = false
}
