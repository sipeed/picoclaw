#!/usr/bin/env bash
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

# Keep this command aligned with .github/workflows/pr.yml.
golangci-lint config verify
golangci-lint run --build-tags=goolm,stdjson
