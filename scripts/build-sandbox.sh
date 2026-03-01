#!/bin/bash
# PicoClaw Sandbox Build Script

set -e

# Base directory
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DOCKERFILE="${REPO_ROOT}/Dockerfile.sandbox"
IMAGE_NAME="picoclaw-sandbox:bookworm-slim"

echo "Building PicoClaw sandbox image: ${IMAGE_NAME}..."

if [ ! -f "${DOCKERFILE}" ]; then
    echo "Error: Dockerfile.sandbox not found at ${DOCKERFILE}"
    exit 1
fi

docker build --no-cache -t "${IMAGE_NAME}" -f "${DOCKERFILE}" "${REPO_ROOT}"

echo "Successfully built ${IMAGE_NAME}"
