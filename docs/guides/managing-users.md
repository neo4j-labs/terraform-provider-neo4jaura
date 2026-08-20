---
page_title: "Managing users in Aura"
subcategory: ""
description: |-
  How to invite, discover, import, and manage users and their roles in an Aura organization and its projects.
---

# Managing users in Aura

This guide walks through onboarding a new teammate to an Aura organization with Terraform, and
managing access for people who already have Aura accounts. It ties together four resources and
data sources:

| Name | Scope | Purpose |
| --- | --- | --- |
| [`neo4jaura_organization_invite`](../resources/organization_invite.md) | Organization | Send an invite by email, granting initial project access at the same time. |
| [`neo4jaura_organization_invites`](../data-sources/organization_invites.md) | Organization | List pending and historical invites. |
| [`neo4jaura_organization_user`](../resources/organization_user.md) | Organization | Manage an existing member's organization-level roles. |
| [`neo4jaura_organization_users`](../data-sources/organization_users.md) | Organization / Project | List an organization's members; pass `project_id` to scope to one project's members. |
| [`neo4jaura_project_user`](../resources/project_user.md) | Project | Manage an existing member's roles within a specific project. |

There is no dedicated "project users" data source — use `neo4jaura_organization_users` with
`project_id` set for that.

## Typical flow: onboarding a new user

1. **Send the invite.** `neo4jaura_organization_invite` sends the invite email and grants an
   initial project role at the same time via `project_invites`:

```terraform
resource "neo4jaura_organization_invite" "this" {
  organization_id    = var.organization_id
  email              = "new.teammate@example.com"
  organization_roles = ["organization-member"]

  project_invites = [
    {
      project_id    = var.project_id
      project_roles = ["project-member"]
    }
  ]
}

variable "organization_id" {
  type = string
}

variable "project_id" {
  type = string
}

output "invite_status" {
  value = neo4jaura_organization_invite.this.status
}

# Populated once the invite is accepted — see the "Managing users in Aura" guide.
output "accepted_user_id" {
  value = neo4jaura_organization_invite.this.user_id
}
```

2. **The invitee accepts by email.** This step happens outside Terraform — the invitee clicks the
   link in the email Aura sends them.

3. **`user_id` becomes available.** The invite object never carries a `user_id` directly — Aura's
   API doesn't return one on an invite. Once `status` becomes `accepted`, this provider resolves
   the resulting `user_id` for you by matching the invite's `email` against the organization's
   user list, and exposes it as a computed attribute. Run `terraform apply` (or `terraform
   refresh`) again after the invite is accepted to pick it up:

```terraform
output "new_user_id" {
  value = neo4jaura_organization_invite.this.user_id
}
```

   To check status/`user_id` for many invites at once instead, use the data source:

```terraform
data "neo4jaura_organization_invites" "this" {
  organization_id = var.organization_id
}

variable "organization_id" {
  type = string
}

output "pending_invites" {
  value = [
    for invite in data.neo4jaura_organization_invites.this.invites : invite
    if invite.status == "active"
  ]
}

# user_id is only populated for invites with status "accepted".
output "accepted_user_ids" {
  value = {
    for invite in data.neo4jaura_organization_invites.this.invites :
    invite.email => invite.user_id
    if invite.status == "accepted"
  }
}
```

4. **Bring the user under management with `neo4jaura_organization_user`.** Import using the
   `user_id` from step 3, then manage their organization-level roles going forward:

```terraform
resource "neo4jaura_organization_user" "this" {
  organization_id      = var.organization_id
  id                   = var.user_id
  organization_roles   = ["organization-member"]
  deregister_on_delete = false
}

variable "organization_id" {
  type = string
}

variable "user_id" {
  type = string
}

output "email" {
  value = neo4jaura_organization_user.this.email
}
```

5. **Grant access to additional projects with `neo4jaura_project_user`.** The invite from step 1
   only grants the project(s) listed in `project_invites`. To add the user to a *different*
   project later, use `neo4jaura_project_user` — but the user must already exist at the
   organization level first (i.e. have an accepted invite), or `Create` will fail with "User not
   found in organization":

```terraform
resource "neo4jaura_project_user" "this" {
  organization_id      = var.organization_id
  project_id           = var.project_id
  user_id              = var.user_id
  project_roles        = ["project-member"]
  deregister_on_delete = false
}

variable "organization_id" {
  type = string
}

variable "project_id" {
  type = string
}

variable "user_id" {
  type = string
}
```

## Update and delete semantics

- **Invites are immutable.** Changing
  `email`, `organization_roles`, or `project_invites` on `neo4jaura_organization_invite` revokes
  the existing invite and creates a new one (`RequiresReplace` on all three). Deleting an invite
  that isn't `active` (already `accepted`, `revoked`, `expired`, or `declined`) is a local no-op —
  the provider does not call the API, since the Aura API rejects deleting a non-active
  invite.
- **`deregister_on_delete` defaults to `false`** on both `neo4jaura_organization_user` and
  `neo4jaura_project_user`. By default, destroying either resource only removes it from Terraform
  state — the user keeps their access in Aura. Set it to `true` if you want `terraform destroy` to
  actually revoke membership.
- **The last organization owner can't be deleted.** `neo4jaura_organization_user` refuses to
  remove a user holding the `organization-owner` role if they're the organization's last
  remaining owner, even with `deregister_on_delete = true`. Promote another user to
  `organization-owner` first.

## Importing existing invites and users

All three resources support `terraform import` with a composite, comma-separated ID:

| Resource | Import ID format |
| --- | --- |
| `neo4jaura_organization_invite` | `{organization_id},{invite_id}` |
| `neo4jaura_organization_user` | `{organization_id},{user_id}` |
| `neo4jaura_project_user` | `{organization_id},{project_id},{user_id}` |

```shell
terraform import neo4jaura_organization_invite.this <organization_id>,<invite_id>
```

```shell
terraform import neo4jaura_organization_user.this <organization_id>,<user_id>
```

```shell
terraform import neo4jaura_project_user.this <organization_id>,<project_id>,<user_id>
```

The values you supply for other **required** attributes in your configuration (e.g. `email`,
`organization_roles`) don't need to be accurate at import time — the provider immediately follows
the import with a `Read` that overwrites them from the API. What matters is that your
configuration's values match the real state *after* that read, so `terraform plan` comes back
clean.
