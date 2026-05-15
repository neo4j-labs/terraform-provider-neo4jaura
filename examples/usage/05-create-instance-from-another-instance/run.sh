#!/bin/bash

set -e

function cleanup() {
    rm -rf .terraform || echo ""
    rm .terraform* || echo ""
    rm terraform.tfstate* || echo ""
}

cleanup
trap cleanup EXIT

terraform init
terraform apply

read -p "Press enter to create another instance"

terraform apply -var="create_another=true"

read -p "Press enter to delete"

terraform destroy
