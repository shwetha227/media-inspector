# ---- build stage -----------------------------------------------------
FROM golang:1.25-bookworm AS build
RUN apt-get update && apt-get update && apt-get install -y --no-install-recommends \
        pkg-config \
        build-essential \
        protobuf-compiler \
        libgstreamer1.0-dev \
        libgstreamer-plugins-base1.0-dev \
        gstreamer1.0-plugins-base \
        gstreamer1.0-plugins-good \
        gstreamer1.0-plugins-ugly \
        gstreamer1.0-libav \
        gstreamer1.0-plugins-bad \
    && rm -rf /var/lib/apt/lists/*
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest \
    && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
ENV PATH="$PATH:/root/go/bin"
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o /out/media-inspector-server ./server

# ---- runtime stage -----------------------------------------------------
FROM debian:bookworm-slim AS runtime
RUN apt-get update && apt-get update && apt-get install -y --no-install-recommends \
        libgstreamer1.0-0 \
        libgstreamer-plugins-base1.0-0 \
        gstreamer1.0-plugins-base \
        gstreamer1.0-plugins-good \
        gstreamer1.0-plugins-ugly \
        gstreamer1.0-libav \
        gstreamer1.0-plugins-bad \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /out/media-inspector-server /usr/local/bin/media-inspector-server
CMD ["/usr/local/bin/media-inspector-server"]