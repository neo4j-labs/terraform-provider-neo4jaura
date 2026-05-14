data "neo4jaura_projects" "this" {}

output "projects" {
  value = data.neo4jaura_projects.this.projects
}
