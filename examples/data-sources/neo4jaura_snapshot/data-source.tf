data "neo4jaura_snapshot" "latest" {
  instance_id = var.instance_id
  most_recent = true
}

variable "instance_id" {
  type = string
}

output "snapshot_id" {
  value = data.neo4jaura_snapshot.latest.snapshot_id
}
