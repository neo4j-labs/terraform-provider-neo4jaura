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

read -p "Press enter to create another instance"

terraform apply -var="create_another=true"

read -p "Press enter to delete"

terraform destroy
