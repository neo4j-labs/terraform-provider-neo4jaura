package main

import (
	"context"
	"flag"
	"log"

	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/provider"
)

import (
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

var version = "0.0.3-dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "enable debugging")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/neo4j-labs/neo4jaura",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)

	if err != nil {
		log.Fatal(err.Error())
	}

}
