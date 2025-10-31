terraform {
  required_version = ">= 1.13.4"
  required_providers {
    aura = {
      source  = "terraform.local/local/aura"
      version = "0.0.1"
    }
  }
}

provider "aura" {
  client_id     = var.client_id
  client_secret = var.client_secret
}

resource "aura_instance" "this" {
  name           = "MyThirdInstance"
  cloud_provider = "gcp"
  region         = "europe-west2"
  memory         = "1GB"
  type           = "professional-db"
  project_id      = data.aura_projects.this.projects.0.id
  status         = var.paused ? "paused" : "running"
}

data "aura_projects" "this" {}

variable "client_id" {}
variable "client_secret" {}

variable "paused" {
  type    = bool
  default = false
}
