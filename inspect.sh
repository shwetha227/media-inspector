#!/usr/bin/env bash
set -euo pipefail

if [ $# -lt 1 ]; then
  echo "usage: $0 <path-to-media-file> [more-paths...]" >&2
  exit 1
fi

CONTAINER_NAME=media-inspector-run
VOLUME_ARGS=()
CONTAINER_PATHS=()

i=0
for FILE in "$@"; do
  # Auto-convert Windows-style paths (C:\...) to WSL paths (/mnt/c/...)
  if [[ "$FILE" == *:\\* ]] && command -v wslpath >/dev/null 2>&1; then
    FILE="$(wslpath -u "$FILE")"
  fi

  if [ ! -f "$FILE" ]; then
    echo "=== $FILE ===" 
    echo "Error: file not found: $FILE"
    echo
    i=$((i + 1))
    continue
  fi

  ABS_PATH="$(cd "$(dirname "$FILE")" && pwd)/$(basename "$FILE")"
  BASENAME="$(basename "$ABS_PATH")"

  # Mount each file individually under its own index, so files with
  # the same basename in different directories don't collide.
  VOLUME_ARGS+=(-v "$ABS_PATH:/data/$i/$BASENAME:ro")
  CONTAINER_PATHS+=("/data/$i/$BASENAME")

  i=$((i + 1))
done

if [ "${#CONTAINER_PATHS[@]}" -eq 0 ]; then
  echo "no valid files to inspect" >&2
  exit 1
fi

cleanup() {
  docker rm -f "$CONTAINER_NAME" >/dev/null 2>&1 || true
}
trap cleanup EXIT

# free the port if a stale container is still holding it
docker ps --filter "publish=50051" -q | xargs -r docker stop >/dev/null 2>&1 || true

docker run -d --rm --name "$CONTAINER_NAME" \
  -p 50051:50051 \
  "${VOLUME_ARGS[@]}" \
  media-inspector:latest >/dev/null

for i in $(seq 1 30); do
  if (echo > /dev/tcp/127.0.0.1/50051) >/dev/null 2>&1; then
    break
  fi
  sleep 0.3
done

./bin/media-inspector-client "${CONTAINER_PATHS[@]}"