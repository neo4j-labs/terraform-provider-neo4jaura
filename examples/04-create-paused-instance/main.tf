terraform {
  required_version = ">= 1.13.4"
  required_providers {
    neo4jaura = {
      source  = "terraform.local/local/neo4jaura"
      version = var.provider_version
    }
  }
}

provider "neo4jaura" {
  client_id     = var.client_id
  client_secret = var.client_secret
}

resource "neo4jaura_instance" "this" {
  name           = "MyForthInstance"
  cloud_provider = "gcp"
  region         = "europe-west2"
  memory         = "2GB"
  storage        = "4GB"
  type           = "professional-db"
  project_id     = data.neo4jaura_projects.this.projects.0.id
  status         = var.paused ? "paused" : "running"
}

data "neo4jaura_projects" "this" {}

variable "provider_version" {}
variable "client_id" {}
variable "client_secret" {}

variable "paused" {
  type    = bool
  default = true
}
