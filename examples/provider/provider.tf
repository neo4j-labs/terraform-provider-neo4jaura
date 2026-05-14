provider "neo4jaura" {
  client_id     = var.client_id
  client_secret = var.client_secret
}

variable "client_id" {
  type      = string
  sensitive = true
}

variable "client_secret" {
  type      = string
  sensitive = true
}
