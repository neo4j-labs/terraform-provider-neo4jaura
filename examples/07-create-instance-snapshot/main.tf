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
  name           = "MySourceInstance"
  cloud_provider = "gcp"
  region         = "europe-west2"
  memory         = "1GB"
  type           = "professional-db"
  tenant_id      = data.aura_tenants.this.tenants.0.id
}

resource "aura_snapshot" "this" {
  instance_id = aura_instance.this.instance_id
}

data "aura_tenants" "this" {}

variable "client_id" {}
variable "client_secret" {}

variable "create_another" {
  type    = bool
  default = false
}

output "snapshot_profile" {
  value = aura_snapshot.this.profile
}

output "snapshot_timestamp" {
  value = aura_snapshot.this.timestamp
}
