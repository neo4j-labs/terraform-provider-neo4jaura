resource "neo4jaura_snapshot" "this" {
  instance_id = var.instance_id
}

variable "instance_id" {
  type = string
}

output "snapshot_id" {
  value = neo4jaura_snapshot.this.snapshot_id
}

output "snapshot_timestamp" {
  value = neo4jaura_snapshot.this.timestamp
}
