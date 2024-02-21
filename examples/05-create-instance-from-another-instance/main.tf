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

resource "aura_instance" "source" {
  name           = "MySourceInstance"
  cloud_provider = "gcp"
  region         = "europe-west2"
  memory         = "1GB"
  type           = "professional-db"
  tenant_id      = data.aura_tenants.this.tenants.0.id
}

resource "aura_instance" "target" {
  count          = var.create_another ? 1 : 0
  name           = "MyTargetInstance"
  cloud_provider = "gcp"
  region         = "europe-west2"
  memory         = "1GB"
  type           = "professional-db"
  tenant_id      = data.aura_tenants.this.tenants.0.id

  source = {
    instance_id = aura_instance.source.instance_id
  }

  lifecycle {
    ignore_changes = [source]
  }
}

data "aura_tenants" "this" {}

variable "client_id" {}
variable "client_secret" {}

variable "create_another" {
  type    = bool
  default = false
}
