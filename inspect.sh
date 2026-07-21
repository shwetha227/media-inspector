#!/usr/bin/env bash
set -euo pipefail

if [ $# -ne 1 ]; then
  echo "usage: $0 <path-to-media-file>" >&2
  exit 1
fi

FILE="$1"

# Auto-convert Windows-style paths (C:\...) to WSL paths (/mnt/c/...)
if [[ "$FILE" == *:\\* ]] && command -v wslpath >/dev/null 2>&1; then
  FILE="$(wslpath -u "$FILE")"
fi

if [ ! -f "$FILE" ]; then
  echo "file not found: $FILE" >&2
  exit 1
fi

ABS_PATH="$(cd "$(dirname "$FILE")" && pwd)/$(basename "$FILE")"
DIR="$(dirname "$ABS_PATH")"
BASENAME="$(basename "$ABS_PATH")"
CONTAINER_NAME=media-inspector-run

cleanup() {
  docker rm -f "$CONTAINER_NAME" >/dev/null 2>&1 || true
}
trap cleanup EXIT

# free the port if a stale contai ner is still holding it
docker ps --filter "publish=50051" -q | xargs -r docker stop >/dev/null 2>&1 || true

docker run -d --rm --name "$CONTAINER_NAME" \
  -p 50051:50051 \
  -v "$DIR:/data:ro" \
  media-inspector:latest >/dev/null 

for i in $(seq 1 30); do
  if (echo > /dev/tcp/127.0.0.1/50051) >/dev/null 2>&1; then
    break
  fi
  sleep 0.3
done

./bin/media-inspector-client "/data/$BASENAME"
