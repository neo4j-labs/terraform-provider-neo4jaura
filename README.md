# Neo4j Aura Terraform Provider

Available as a Neo4j Labs Project ( See Disclaimer further down this README )  Neo4j Aura Terraform Provider enables a declarative, infrastructure-as-code (IaC) approach to infrastructure.  This codifies the interaction with Aura's management API for the provisioning and management of AuraDB infrastructure. Specifically Neo4j Aura Terraform provider allows for:- 

* Obtaining information about a project ( tenant )
* Create, modify, pause, resume and delete operations for AuraDB instances
* Take and restore AuraDB snapshots.  
* Creating an Aura instance from a snapshot

__Neo4j Aura Terraform Provider is a Neo4j Labs Project.  Please read the Disclaimer at the bottom of this page before use.__

## Using from the Terraform Provider Registry

To use directly from the [Terraform Provider Registry](https://registry.terraform.io/providers/neo4j-labs/neo4jaura/latest), copy and paste this code into your Terraform configuration, adjusting the configuration options to meet your requirements.  


```Text
terraform {
  required_providers {
    neo4jaura = {
      source = "neo4j-labs/neo4jaura"
      version = "0.0.2-beta"
    }
  }
}
provider "neo4jaura" {
  # Configuration options
}
```

Then run ```terraform init```


See [Examples](https://github.com/neo4j-labs/terraform-provider-neo4jaura/tree/main/examples) for the various possible configuration options


## Using from GitHUb repository

This is route to take if you wish to experiment with your own development of the provider or just try it out. 


### Requirements

* Go 1.25+
* Terraform 1.13.4+
* A Client Id and Client Secret for access to the Aura API.  To obtain these, follow the guidance in the [Neo4j AuraDB documentation](https://neo4j.com/docs/aura/api/authentication/)


### Installation

Clone the repositry

```Text
git clone https://github.com/neo-technology/neo4j-aura-terraform-provider-poc.git
```

Build the provider

```Text
cd neo4j-aura-terraform-provider-poc/
./build.sh
```

Add .terraformrc file to your $HOME folder


```Text
provider_installation {
  filesystem_mirror {
    path    = "$YOUR_HOME_PATH/.terraform.d/plugins"
  }
  direct {
    exclude = ["terraform.local/*/*"]
  }
}
```

## Example configurations 

Several example configurations are provided in the [Examples](https://github.com/neo4j-labs/terraform-provider-neo4jaura/tree/main/examples) folder of this repository. You will need to set  TF_VAR_client_id and TF_VAR_client_secret environment variables before running any of the examples. 

```Text
export TF_VAR_client_id="$AURA_CLIENT_ID"
export TF_VAR_client_secret="$AURA_CLIENT_SECRET"
```

Move into the examples folder and then, to run an example

```Text
/execute_example.sh <example folder name>
```

You may be prompted to enter values or text during execution.   



___The terraform files used in the examples may require editing to match your Neo4j AuraDB environment.  In particular , those that create or modify AuraDB Instances are likely to need changes.___


## Contributing 

We welcome contributions to improve and extend the capabilities of the Neo4j Aura Terraform Provider.  If you wish to contribute, then follow these steps:-

* Sign the [contributors agreement](https://neo4j.com/developer/contributing-code/#sign-cla)
* Fork the [repository](https://github.com/neo-technology/neo4j-aura-terraform-provider-poc)
* Create a branch for your contribution on your _forked repo_
* Submit a PR from your fork back to the Neo4j Aura Terraform Provider repository


___A good pull request is focused on one feature or issue and includes a clear title that summarizes the change. In the description, you should explain what you changed and why, and reference any related issues using syntax like "Fixes #123".___


If you get stuck, start by checking existing GitHub issues to see if others have encountered similar problems. You can also ask questions directly in pull request discussions, where maintainers and other contributors can provide guidance. For complex architectural questions or decisions that might affect the project's design, reach out to maintainers directly to get their input before investing too much time in a particular approach.

Thank you for contributing to make this better!


## Feedback, Support and Issues

All feedback is welcome and can be posted either in the Issues area of the [GitHub Reposity](https://github.com/neo4j-labs/terraform-provider-neo4jaura/issues) or by posting in [Neo4j Communities Integrations](https://community.neo4j.com/c/integrations).  Communities is also a great place for asking questions.

Neo4j Aura Terraform Provider is a Neo4j Labs project which means it is not officially supported by Neo4j.  Please report any issue what you may have in the [GitHub Reposity](https://github.com/neo4j-labs/terraform-provider-neo4jaura/issues)


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
| Terraform provider registry | [https://registry.terraform.io/providers/neo4j-labs/neo4jaura/latest](https://registry.terraform.io/providers/neo4j-labs/neo4jaura/latest ) |
| Terraform plugin framework | [https://developer.hashicorp.com/terraform/plugin/framework](https://developer.hashicorp.com/terraform/plugin/framework) |
| Terraform provider scaffolding framework | [https://github.com/hashicorp/terraform-provider-scaffolding-framework](https://github.com/hashicorp/terraform-provider-scaffolding-framework) |
| Aura API specifcation | [https://neo4j.com/docs/aura/platform/api/specification/](https://neo4j.com/docs/aura/platform/api/specification/) |


