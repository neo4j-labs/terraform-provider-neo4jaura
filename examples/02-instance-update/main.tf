terraform {
  required_version = ">= 1.7.3"
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
  name           = var.name
  cloud_provider = "gcp"
  region         = "europe-west2"
  memory         = var.memory
  type           = "professional-db"
  tenant_id      = data.aura_tenants.this.tenants.0.id
}

data "aura_tenants" "this" {}

variable "client_id" {}
variable "client_secret" {}
variable "name" {
  default = "MySecondInstance"
}
variable "memory" {
  default = "1GB"
}
