#!/usr/bin/env bash
# Build/push local da imagem multi-arch do Traefix para o Docker Hub.
#
#   scripts/traefix/docker-build.sh            # tag = git describe, push
#   TAG=v3.8.0-tx1 scripts/traefix/docker-build.sh
#   PUSH=0 PLATFORMS=linux/arm64 scripts/traefix/docker-build.sh   # só build local (--load)
#
# Requer `docker login` na conta tmsoftbrasil.
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

IMAGE="${IMAGE:-tmsoftbrasil/traefix}"
PLATFORMS="${PLATFORMS:-linux/amd64,linux/arm64}"
TAG="${TAG:-$(git describe --tags --always --dirty)}"
PUSH="${PUSH:-1}"
BUILDER="${BUILDER:-traefix-builder}"

docker buildx inspect "$BUILDER" >/dev/null 2>&1 || docker buildx create --name "$BUILDER" --driver docker-container >/dev/null

if [[ "$PUSH" == "1" ]]; then
  OUTPUT=(--push)
else
  OUTPUT=(--load)
fi

TAGS=(-t "$IMAGE:$TAG")
[[ "${LATEST:-0}" == "1" ]] && TAGS+=(-t "$IMAGE:latest")

set -x
docker buildx build \
  --builder "$BUILDER" \
  -f Dockerfile.traefix \
  --platform "$PLATFORMS" \
  --build-arg VERSION="$TAG" \
  --build-arg REVISION="$(git rev-parse HEAD)" \
  --build-arg CREATED="$(date -u '+%Y-%m-%dT%H:%M:%SZ')" \
  "${TAGS[@]}" \
  "${OUTPUT[@]}" \
  .
