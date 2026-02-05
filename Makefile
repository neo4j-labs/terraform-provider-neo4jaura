default: fmt test install generate

build:
	go build -v ./...

install:
	go install -v ./...

generate:
	cd tools; go generate ./...

fmt:
	gofmt -s -w -e .

test:
	TF_ACC= go run gotest.tools/gotestsum@latest --format testname -- -cover -timeout=120s -parallel=10 ./...

acceptance:
	TF_ACC=1 go run gotest.tools/gotestsum@latest --format testname -- -cover -timeout=1h -parallel=10 ./...