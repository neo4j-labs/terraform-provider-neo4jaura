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

data "neo4jaura_projects" "this" {}

variable "provider_version" {}
variable "client_id" {}
variable "client_secret" {}

output "test" {
  value = data.neo4jaura_projects.this
}
