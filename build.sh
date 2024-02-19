#!/bin/bash

PROVIDER_NAME=aura
VERSION=0.0.1
PLUGIN_FOLDER="${HOME}/.terraform.d/plugins/terraform.local/local/${PROVIDER_NAME}/${VERSION}/darwin_arm64/terraform-provider-${PROVIDER_NAME}_v${VERSION}"
env GOOS=darwin GOARCH=arm64 go build -o "$PLUGIN_FOLDER"