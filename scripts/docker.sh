#!/usr/bin/env bash
set -euo pipefail

IMAGE_NAME="picoclaw"
TAG="${1:-latest}"

# ── version from git ─────────────────────────────────────────
if [[ "$TAG" == "latest" ]]; then
    GIT_TAG="$(git describe --tags --always --dirty 2>/dev/null || echo "dev")"
else
    GIT_TAG="$TAG"
fi

echo "==> Building Docker image: ${IMAGE_NAME}:${TAG}"
echo "    Version: ${GIT_TAG}"

docker build \
    -t "${IMAGE_NAME}:${TAG}" \
    -t "${IMAGE_NAME}:${GIT_TAG}" \
    .

echo ""
echo "Build complete:"
echo "  ${IMAGE_NAME}:${TAG}"
echo "  ${IMAGE_NAME}:${GIT_TAG}"
echo ""
echo "Run with:"
echo "  docker run --rm ${IMAGE_NAME}:${TAG} version"
echo "  docker run -v config.json:/home/picoclaw/.picoclaw/config.json:ro ${IMAGE_NAME}:${TAG} gateway"
