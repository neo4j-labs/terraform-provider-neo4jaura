#!/bin/bash

PROVIDER_NAME=neo4jaura
VERSION=0.0.3-dev
PLUGIN_FOLDER="${HOME}/.terraform.d/plugins/terraform.local/local/${PROVIDER_NAME}/${VERSION}/darwin_arm64/terraform-provider-${PROVIDER_NAME}_v${VERSION}"

go fmt
go build -o "$PLUGIN_FOLDER"