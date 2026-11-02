terraform {
  required_providers {
    neo4jaura = {
      source = "neo4j-labs/neo4jaura"
    }
  }
}

variable "organization_id" {
  type = string
}

variable "project_id" {
  type = string
}

resource "neo4jaura_instance" "multi_database" {
  name            = "example-multi-database"
  cloud_provider  = "aws"
  region          = "us-east-1"
  memory          = "4GB"
  storage         = "8GB"
  type            = "business-critical"
  organization_id = var.organization_id
  project_id      = var.project_id
  multi_database  = true

  lifecycle {
    prevent_destroy = true
  }
}
