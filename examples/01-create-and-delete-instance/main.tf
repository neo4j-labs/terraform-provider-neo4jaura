terraform {
  required_version = ">= 1.13.4"
  required_providers {
    neo4jaura = {
      source  = "terraform.local/local/neo4jaura"
      version = "0.0.1"
    }
  }
}

provider "neo4jaura" {
  client_id     = var.client_id
  client_secret = var.client_secret
}

resource "neo4jaura_instance" "this" {
  name           = "MyFirstAuraInstance"
  cloud_provider = "gcp"
  region         = "europe-west2"
  memory         = "1GB"
  type           = "professional-db"
  project_id     = data.neo4jaura_projects.this.projects.0.id
}

data "neo4jaura_projects" "this" {}

variable "client_id" {}
variable "client_secret" {}

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
