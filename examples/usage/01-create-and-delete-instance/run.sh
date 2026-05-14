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

read -p "Press enter when you want to destroy"

terraform destroy
