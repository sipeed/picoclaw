#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONFIG_FILE="${ROOT_DIR}/.goreleaser.yaml"

failed=0
while IFS= read -r main_path; do
  main_path="${main_path#./}"
  if [ ! -e "${ROOT_DIR}/${main_path}" ]; then
    echo "missing goreleaser build main: ${main_path}" >&2
    failed=1
  fi
done < <(sed -n 's/^[[:space:]]*main:[[:space:]]*//p' "${CONFIG_FILE}")

if [ "${failed}" -ne 0 ]; then
  exit 1
fi

echo "all goreleaser build mains exist"
