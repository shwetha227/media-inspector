.PHONY: build build-local run test test-grpc client

build:
	docker build -t media-inspector:latest .

run:
	docker run --rm -p 50051:50051 \
		-v $(shell pwd)/testdata:/testdata \
		media-inspector:latest

test:
	go test -race ./...

test-grpc:
	grpcurl -plaintext \
		-proto proto/inspector.proto \
		-import-path proto \
		-d '{"file_path": "/testdata/sample.mp4"}' \
		localhost:50051 inspector.MediaInspector/Inspect

build-local:
	CGO_ENABLED=1 go build -o bin/media-inspector-server ./server

client:
	go run client/main.go $(FILE)
