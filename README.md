# Neo4j Aura Terraform Provider

Available as a Neo4j Labs Project ( See Disclaimer further down this README )  Neo4j Aura Terraform Provider enables a declarative, infrastructure-as-code (IaC) approach to infrastructure.  This codifies the interaction with Aura's management API for the provisioning and management of AuraDB infrastructure. Specifically Neo4j Aura Terraform provider allows for:- 

* Obtaining information about a project ( tenant )
* Create, modify, pause, resume, delete, and import operations for AuraDB instances
* Take and restore AuraDB snapshots
* Creating an Aura instance from a snapshot
* Import existing snapshots into Terraform state
* Configuration validation for CDC enrichment mode, vector optimisation, and graph analytics plugin

__Neo4j Aura Terraform Provider is a Neo4j Labs Project.  Please read the Disclaimer at the bottom of this page before use.__

## Using from the Terraform Provider Registry

To use directly from the [Terraform Provider Registry](https://registry.terraform.io/providers/neo4j-labs/neo4jaura/latest), copy and paste this code into your Terraform configuration, adjusting the configuration options to meet your requirements.  


```hcl
terraform {
  required_providers {
    neo4jaura = {
      source  = "neo4j-labs/neo4jaura"
      version = "0.0.3-beta"
    }
  }
}

provider "neo4jaura" {
  # client_id and client_secret can be set here or via the
  # AURA_CLIENT_ID and AURA_CLIENT_SECRET environment variables.
}
```

Then run `terraform init`


See [Examples](https://github.com/neo4j-labs/terraform-provider-neo4jaura/tree/main/examples) for the various possible configuration options


## Provider configuration

The provider supports two ways to supply credentials:

**HCL provider block** (takes precedence):

```hcl
provider "neo4jaura" {
  client_id     = "your-client-id"
  client_secret = "your-client-secret"
}
```

**Environment variables** (fallback when the provider block values are absent):

```bash
export AURA_CLIENT_ID="your-client-id"
export AURA_CLIENT_SECRET="your-client-secret"
```

To obtain a client ID and secret, follow the guidance in the [Neo4j AuraDB documentation](https://neo4j.com/docs/aura/api/authentication/).


## Using from GitHub repository

This is the route to take if you wish to experiment with your own development of the provider or just try it out. 


### Requirements

* Go 1.25+
* Terraform 1.14.8+
* A Client ID and Client Secret for access to the Aura API.  To obtain these, follow the guidance in the [Neo4j AuraDB documentation](https://neo4j.com/docs/aura/api/authentication/)


### Installation

Clone the repository:

```bash
git clone https://github.com/neo4j-labs/terraform-provider-neo4jaura.git
cd terraform-provider-neo4jaura
```

Build and install the provider binary into your local Go bin path:

```bash
make install
```

Add a `~/.terraformrc` file (or extend an existing one) to tell Terraform to use the local binary instead of the registry:

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/neo4j-labs/neo4jaura" = "/path/to/your/go/bin"
  }
  direct {}
}
```

Replace `/path/to/your/go/bin` with the output of `go env GOPATH`/bin (e.g. `~/go/bin`).

With `dev_overrides` in place you do not need to run `terraform init` for the provider — Terraform uses the local binary directly.


## Example configurations 

Several example configurations are provided in the [Examples](https://github.com/neo4j-labs/terraform-provider-neo4jaura/tree/main/examples) folder of this repository. Set your credentials before running any of the examples:

```bash
export AURA_CLIENT_ID="your-client-id"
export AURA_CLIENT_SECRET="your-client-secret"
```

Or, if using the TF_VAR pattern:

```bash
export TF_VAR_client_id="$AURA_CLIENT_ID"
export TF_VAR_client_secret="$AURA_CLIENT_SECRET"
```

Move into the examples folder and then, to run an example:

```bash
./examples/execute_example.sh <example folder name>
```

You may be prompted to enter values or text during execution.   


___The terraform files used in the examples may require editing to match your Neo4j AuraDB environment.  In particular, those that create or modify AuraDB Instances are likely to need changes.___


## Contributing

We welcome contributions to improve and extend the capabilities of the Neo4j Aura Terraform Provider.  If you wish to contribute, then follow these steps:

* Sign the [contributors agreement](https://neo4j.com/developer/contributing-code/#sign-cla)
* Fork the [repository](https://github.com/neo4j-labs/terraform-provider-neo4jaura)
* Create a branch for your contribution on your _forked repo_
* Add a changelog entry (see below)
* Submit a PR from your fork back to the Neo4j Aura Terraform Provider repository

___A good pull request is focused on one feature or issue and includes a clear title that summarizes the change. In the description, you should explain what you changed and why, and reference any related issues using syntax like "Fixes #123".___

If you get stuck, start by checking existing GitHub issues to see if others have encountered similar problems. You can also ask questions directly in pull request discussions, where maintainers and other contributors can provide guidance. For complex architectural questions or decisions that might affect the project's design, reach out to maintainers directly to get their input before investing too much time in a particular approach.

Thank you for contributing to make this better!

### Running tests locally

```bash
# Unit tests (no live infrastructure required)
make test

# Acceptance tests against the in-process mock server
make mock-acceptance

# Acceptance tests against the live Aura API (requires credentials)
make live-acceptance   # needs AURA_CLIENT_ID / AURA_CLIENT_SECRET set
```

### Changelog entries

Every pull request should include a changelog entry so that changes are captured in the release notes. Entries live as small YAML files under `.changes/unreleased/` and are merged into `CHANGELOG.md` automatically at release time.

Create a file named `.changes/unreleased/<short-description>.yaml` with the following content:

```yaml
kind: <kind>
body: '<description of the change>'
time: <RFC3339 timestamp, e.g. 2026-01-01T00:00:00Z>
```

The `kind` field determines how the version number is bumped:

| Kind | When to use |
|------|-------------|
| `Added` | New feature or capability |
| `Changed` | Breaking change to existing behaviour |
| `Deprecated` | Feature marked for future removal |
| `Removed` | Feature removed |
| `Fixed` | Bug fix |
| `Security` | Security fix |

If you have [changie](https://changie.dev) installed you can run `changie new` to create the entry interactively.

### Releasing (maintainers only)

Releases are created using the **Prepare Release** GitHub Actions workflow. It batches the unreleased changelog entries, regenerates `CHANGELOG.md`, opens a PR, and pushes the version tag. The **Release** workflow fires immediately on the new tag and publishes binaries to GitHub Releases and the Terraform Registry.

**Steps to cut a release:**

1. Ensure all changes intended for the release are merged to `main` and each has a changelog entry under `.changes/unreleased/`.

2. Go to **Actions → Prepare Release → Run workflow** on GitHub.

3. Enter the version to release (e.g. `0.0.4-beta`). The version must match the changie-supported format — do **not** include a `v` prefix.

4. The workflow will:
   - Run `changie batch <version>` to gather unreleased entries
   - Run `changie merge` to regenerate `CHANGELOG.md`
   - Push a `chore/release-v<version>` branch and open a PR
   - Push a `v<version>` tag — this triggers the **Release** workflow immediately

5. The **Release** workflow builds the provider binaries, creates a GitHub Release, and publishes to the Terraform Registry.

6. Merge the PR opened in step 4 to bring the updated `CHANGELOG.md` into `main`.

> **Note:** Because the tag is pushed by `github-actions[bot]`, GitHub's security model prevents it from automatically triggering the release workflow. If the Release workflow does not start within a minute, delete and re-push the tag from your local machine:
> ```bash
> git fetch --tags
> git push origin :refs/tags/v<version>   # delete remote tag
> git push origin v<version>              # re-push to trigger workflow
> ```


## Feedback, Support and Issues

All feedback is welcome and can be posted either in the Issues area of the [GitHub Repository](https://github.com/neo4j-labs/terraform-provider-neo4jaura/issues) or by posting in [Neo4j Communities Integrations](https://community.neo4j.com/c/integrations).  Communities is also a great place for asking questions.

Neo4j Aura Terraform Provider is a Neo4j Labs project which means it is not officially supported by Neo4j.  Please report any issue you may have in the [GitHub Repository](https://github.com/neo4j-labs/terraform-provider-neo4jaura/issues).


## Disclaimer

Neo4j Aura Terraform Provider is a Neo4j Labs project.  Neo4j Labs projects are useful ecosystem tools that are meant to benefit all Neo4j users. 
They are not officially supported by Neo4j. Use them at your own risk.

Neo4j Labs projects, while trying to apply sound engineering principles, are provided as is - with no guarantees, liabilities or warranty for function, API stability or continued maintenance. Support for Neo4j Labs projects happens by the community and maintainers as a best-effort through GitHub issues and community forums. These projects are examples that use public Neo4j APIs to show how to implement a certain capability.


## Relevant Links

| Topic   | Link |
| -------- | ------- |
| Releases | [https://github.com/neo4j-labs/terraform-provider-neo4jaura/releases](https://github.com/neo4j-labs/terraform-provider-neo4jaura/releases) |
| Source | [https://github.com/neo4j-labs/terraform-provider-neo4jaura](https://github.com/neo4j-labs/terraform-provider-neo4jaura) |
| Issues | [https://github.com/neo4j-labs/terraform-provider-neo4jaura/issues](https://github.com/neo4j-labs/terraform-provider-neo4jaura/issues) |
| Terraform provider registry | [https://registry.terraform.io/providers/neo4j-labs/neo4jaura/latest](https://registry.terraform.io/providers/neo4j-labs/neo4jaura/latest) |
| Terraform plugin framework | [https://developer.hashicorp.com/terraform/plugin/framework](https://developer.hashicorp.com/terraform/plugin/framework) |
| Terraform provider scaffolding framework | [https://github.com/hashicorp/terraform-provider-scaffolding-framework](https://github.com/hashicorp/terraform-provider-scaffolding-framework) |
| Aura API specification | [https://neo4j.com/docs/aura/platform/api/specification/](https://neo4j.com/docs/aura/platform/api/specification/) |
