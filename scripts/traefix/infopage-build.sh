#!/usr/bin/env bash
# Build/push multi-arch da imagem tmsoftbrasil/traefix-infopage (pasta infopage/).
#
#   scripts/traefix/infopage-build.sh                 # tag = infopage-<git describe>, push com :latest
#   TAG=v1 scripts/traefix/infopage-build.sh
#   PUSH=0 PLATFORMS=linux/arm64 scripts/traefix/infopage-build.sh   # só build local (--load)
#
# Requer `docker login` na conta tmsoftbrasil.
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

IMAGE="${IMAGE:-tmsoftbrasil/traefix-infopage}"
PLATFORMS="${PLATFORMS:-linux/amd64,linux/arm64}"
TAG="${TAG:-$(git describe --tags --always --match 'fork-point*' --dirty)}"
PUSH="${PUSH:-1}"
BUILDER="${BUILDER:-traefix-builder}"

docker buildx inspect "$BUILDER" >/dev/null 2>&1 || docker buildx create --name "$BUILDER" --driver docker-container >/dev/null

TAGS=(-t "$IMAGE:$TAG")
if [[ "$PUSH" == "1" ]]; then
  OUTPUT=(--push)
  TAGS+=(-t "$IMAGE:latest")
else
  OUTPUT=(--load)
fi

set -x
docker buildx build \
  --builder "$BUILDER" \
  --platform "$PLATFORMS" \
  --build-arg VERSION="$TAG" \
  --build-arg REVISION="$(git rev-parse HEAD)" \
  --build-arg CREATED="$(date -u '+%Y-%m-%dT%H:%M:%SZ')" \
  "${TAGS[@]}" \
  "${OUTPUT[@]}" \
  infopage
