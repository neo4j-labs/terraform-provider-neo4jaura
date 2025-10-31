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

data "neo4jaura_projects" "this" {}

variable "client_id" {}
variable "client_secret" {}

output "test" {
  value = data.neo4jaura_projects.this
}
