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

resource "neo4jaura_instance" "source" {
  name           = "MySourceInstance"
  cloud_provider = "gcp"
  region         = "europe-west2"
  memory         = "2GB"
  storage        = "4GB"
  type           = "professional-db"
  project_id     = data.neo4jaura_projects.this.projects.0.id
}

resource "neo4jaura_instance" "target" {
  count          = var.create_another ? 1 : 0
  name           = "MyTargetInstance"
  cloud_provider = "gcp"
  region         = "europe-west2"
  memory         = "1GB"
  type           = "professional-db"
  project_id     = data.neo4jaura_projects.this.projects.0.id

  source = {
    instance_id = neo4jaura_instance.source.instance_id
    snapshot_id = data.neo4jaura_snapshot.source.snapshot_id
  }

  lifecycle {
    ignore_changes = [source]
  }
}

data "neo4jaura_snapshot" "source" {
  instance_id = neo4jaura_instance.source.instance_id
  most_recent = true
}

data "neo4jaura_projects" "this" {}

variable "client_id" {}
variable "client_secret" {}

variable "create_another" {
  type    = bool
  default = false
}

output "connection_url" {
  value = neo4jaura_instance.source.connection_url
}

output "password" {
  value     = neo4jaura_instance.source.password
  sensitive = true
}

output "target_password" {
  value     = var.create_another ? neo4jaura_instance.target.0.password : ""
  sensitive = true
}
