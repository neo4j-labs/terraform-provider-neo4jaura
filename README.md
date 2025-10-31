# Neo4j Aura Terraform Provider

This is an experimental Terraform Provider for Neo4j Aura. It
uses [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework)
on [protocol version 6](https://developer.hashicorp.com/terraform/plugin/terraform-plugin-protocol#protocol-version-6).

## Requirements

* Go 1.25+
* Terraform 1.13.4+

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