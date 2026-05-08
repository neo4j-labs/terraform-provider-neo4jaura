terraform {
  required_version = ">= 1.13.4"
  required_providers {
    neo4jaura = {
      source  = "neo4j-labs/neo4jaura"
      version = "0.0.2-beta"
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
