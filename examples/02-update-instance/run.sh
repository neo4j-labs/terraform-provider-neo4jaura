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

read -p "Press enter when you want instance to be updated"

terraform apply -var="name=UpdatedInstance" -var="memory=2GB"

read -p "Press enter when you want to destroy"

terraform destroy
