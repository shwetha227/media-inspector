build:
	docker build --target runtime -t media-inspector-runtime-test .

run:
	docker run --rm -p 50051:50051 \
		-v $(shell pwd)/testdata:/testdata \
		media-inspector-runtime-test

test:
	go test -race ./...

test-grpc:
	grpcurl -plaintext \
		-proto proto/inspector.proto \
		-import-path proto \
		-d '{"file_path": "/testdata/sample.mp4"}' \
		localhost:50051 inspector.MediaInspector/Inspect
