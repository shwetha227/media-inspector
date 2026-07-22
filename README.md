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

Build the client binary (rebuild this any time `client/main.go` changes):

    make build-client

Run the server (listens on `:50051`, mounts `testdata/` for local testing):

    make run

In a separate terminal, run a test request against a sample file:

    make test

## Inspecting Files

### Via `inspect.sh` (recommended — handles starting/stopping the server container for you)

Accepts one or more file paths, including Windows-style paths under WSL:

    ./inspect.sh testdata/sample.mp4
    ./inspect.sh testdata/sample.mp4 testdata/video.avi
    ./inspect.sh "C:\Users\you\Downloads\video.mp4" "C:\Users\you\Downloads\clip.mp3"

Each file is inspected independently and concurrently. If one file is missing or fails to inspect, its error is printed and the remaining files are still processed normally.

### Via `make client` (server must already be running separately, e.g. via `make run`)

    make client FILES="testdata/sample.mp4 testdata/video.avi"

All paths go inside one quoted `FILES` value, space-separated.

## Project Structure

    media-inspector/
    ├── c/           # C wrapper around GStreamer Discoverer
    ├── client/      # gRPC client CLI (accepts one or more file paths)
    ├── internal/    # Internal Go packages (CGO bindings, business logic)
    ├── proto/       # Protobuf service/message definitions + generated Go code
    ├── server/      # gRPC server entrypoint
    ├── testdata/    # Sample media files for manual/local testing
    ├── Dockerfile   # Multi-stage build (build + runtime stages)
    ├── inspect.sh   # Runs the server in Docker and inspects one or more files
    └── Makefile     # build, run, test, client targets

## API

**Service:** `inspector.MediaInspector`

**RPC:** `Inspect(InspectRequest) returns (InspectResponse)`

The client sends one file path per RPC call. Multiple files are handled by making multiple concurrent `Inspect` calls — the RPC itself remains a simple unary request/response; there is no batch or streaming variant.

Request:

    message InspectRequest {
      string file_path = 1;
    }

Response:

    message InspectResponse {
      string container = 1;
      double duration_seconds = 2;
      repeated Stream streams = 3;
      string error = 4;
    }

If `error` is non-empty, inspection did not succeed for this file — `container`, `duration_seconds`, and `streams` will be unset. This covers problems with the file itself (missing, unreadable, or not valid media) and is returned as a normal, successful RPC response, not a gRPC error status.

A `Stream` describes one video or audio track. Which kind it is comes from which field of the `details` oneof is set — video streams populate `video`, audio streams populate `audio`, and there is no separate type field:

    message Stream {
      string codec = 1;
      uint32 bitrate = 2;
      oneof details {
        VideoDetails video = 3;
        AudioDetails audio = 4;
      }
    }

Malformed requests (e.g. an empty `file_path`) are reported as a gRPC error status (`InvalidArgument`), not through the `error` field.

## Testing

With the server running (`make run`), test against a valid media file:

    make test

This calls `Inspect` on `testdata/sample.mp4` and returns container, codec, and stream metadata.

Error handling can be verified against a non-media file — this returns a normal RPC response with the `error` field populated, not a gRPC error status:

    grpcurl -plaintext \
      -proto proto/inspector.proto \
      -import-path proto \
      -d '{"file_path": "/testdata/not_media.txt"}' \
      localhost:50051 inspector.MediaInspector/Inspect

The same can be verified end-to-end through the client:

    ./inspect.sh testdata/not_media.txt