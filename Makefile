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