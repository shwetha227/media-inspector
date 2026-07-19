# Media Inspector

A gRPC service that inspects media files (video/audio) and returns container, codec, and stream metadata. Built with Go, CGO, and GStreamer's Discoverer API, served over gRPC with Protocol Buffers, and packaged as a multi-stage Docker image.

## Tech Stack

- Go (server, gRPC)
- C + CGO (GStreamer Discoverer bindings)
- GStreamer Discoverer
- Protocol Buffers / gRPC
- Docker (multi-stage build)
- Make

## Prerequisites

- Docker Desktop with WSL2 integration enabled
- `make`
- `grpcurl` (for manual testing) — install via `go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest`

## Quick Start

Build the runtime image:

    make build

Run the server (listens on `:50051`, mounts `testdata/` for local testing):

    make run

In a separate terminal, run a test request against a sample file:

    make test

## Project Structure

    media-inspector/
    ├── c/           # C wrapper around GStreamer Discoverer
    ├── internal/    # Internal Go packages (CGO bindings, business logic)
    ├── proto/       # Protobuf service/message definitions + generated Go code
    ├── server/      # gRPC server entrypoint
    ├── testdata/    # Sample media files for manual/local testing
    ├── Dockerfile   # Multi-stage build (build + runtime stages)
    └── Makefile     # build, run, test targets

## API

**Service:** `inspector.MediaInspector`

**RPC:** `Inspect(InspectRequest) returns (InspectResponse)`

Request:

    message InspectRequest {
      string file_path = 1;
    }

Response:

    message InspectResponse {
      string container = 1;
      double duration_seconds = 2;
      repeated Stream streams = 3;
    }

## Testing

With the server running (`make run`), test against a valid media file:

    make test

This calls `Inspect` on `testdata/sample.mp4` and returns container, codec, and stream metadata.

Error handling can be verified against a non-media file:

    grpcurl -plaintext \
      -proto proto/inspector.proto \
      -import-path proto \
      -d '{"file_path": "/testdata/not_media.txt"}' \
      localhost:50051 inspector.MediaInspector/Inspect

This returns a gRPC `InvalidArgument` error rather than crashing the server.
