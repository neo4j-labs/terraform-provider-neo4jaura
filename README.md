# Neo4j Aura Terraform Provider Proof of Concept

This is a proof of concept of Terraform Provider for Aura. It
uses [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework)
on [protocol version 6](https://developer.hashicorp.com/terraform/plugin/terraform-plugin-protocol#protocol-version-6).

## Requirements

* Go 1.22+
* Terraform 1.7.3+

## Build

The provider is not registered and at the moment could be installed locally:

* Run `/.build.sh` in the root to compile.
* Add the `.terraformrc` file to your $HOME folder:

```
provider_installation {
  filesystem_mirror {
    path    = "$YOUR_HOME_PATH/.terraform.d/plugins"
  }
  direct {
    exclude = ["terraform.local/*/*"]
  }
}
```

## Examples

There are several examples of the provider usage available on [`examples`](examples) folder. In order to run them you
need to
export the Aura credentials to your environment:

```
export TF_VAR_client_id="$AURA_CLIENT_ID"
export TF_VAR_client_secret="$AURA_CLIENT_SECRET"
```

and execute the script

```
./execute_example.sh $EXAMPLE_FOLDER_NAME
```

## Resources

* [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework) - documentation of the latest
  version of the plugin framework
* [Terraform Provider Scaffolding Framework](https://github.com/hashicorp/terraform-provider-scaffolding-framework) -
  template repository, which is good
* [Aura API Specification](https://neo4j.com/docs/aura/platform/api/specification/) - OpenAPI specification for Aura API 