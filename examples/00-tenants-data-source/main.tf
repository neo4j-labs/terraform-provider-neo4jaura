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

data "aura_tenants" "this" {}

variable "client_id" {}
variable "client_secret" {}

output "test" {
  value = data.aura_tenants.this
}
