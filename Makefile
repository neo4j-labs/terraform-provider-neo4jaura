default: fmt vet test install generate

build:
	go build -v ./...

install:
	go install -v ./...

generate:
	cd tools; go generate ./...

fmt:
	gofmt -s -w -e .

vet:
	go vet -v ./...

test:
	TF_ACC= go run gotest.tools/gotestsum@latest --format testname -- -cover -timeout=120s -parallel=10 ./...

acceptance:
	TF_ACC=1 go run gotest.tools/gotestsum@latest --format testname -- -cover -timeout=2m ./internal/test/...

live-acceptance:
	./live_acceptance.sh
# Build and install the provider into the local Terraform plugin mirror so it
# can be consumed via a terraform.local/local/neo4jaura provider source.
LOCAL_PROVIDER_NAME ?= neo4jaura
LOCAL_VERSION ?= 0.0.1-dev
LOCAL_OS_ARCH ?= $(shell go env GOOS)_$(shell go env GOARCH)
LOCAL_PLUGIN_DIR = $(HOME)/.terraform.d/plugins/terraform.local/local/$(LOCAL_PROVIDER_NAME)/$(LOCAL_VERSION)/$(LOCAL_OS_ARCH)

build-local:
	mkdir -p "$(LOCAL_PLUGIN_DIR)"
	go build -o "$(LOCAL_PLUGIN_DIR)/terraform-provider-$(LOCAL_PROVIDER_NAME)_v$(LOCAL_VERSION)"
	@echo "Installed terraform-provider-$(LOCAL_PROVIDER_NAME)_v$(LOCAL_VERSION) to $(LOCAL_PLUGIN_DIR)"

.PHONY: default build install generate fmt vet test acceptance live-acceptance build-local
