#!/bin/bash

set -e

function cleanup() {
    rm -rf .terraform || echo ""
    rm .terraform* || echo ""
    rm terraform.tfstate* || echo ""
}

cleanup
trap cleanup EXIT

if [[ -z "$TF_VAR_provider_version" ]]; then
    export TF_VAR_provider_version=$(cat ../../.version)
fi

terraform init
terraform apply

read -p "Press enter to unpause"

terraform apply -var="paused=false"

read -p "Press enter to delete"

terraform destroy
