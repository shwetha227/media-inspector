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
