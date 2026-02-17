#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BINARY_PATH="${PICOCLAW_BINARY_PATH:-${ROOT_DIR}/build/picoclaw}"
BUDGET_KB="${PICOCLAW_MEMORY_BUDGET_KB:-20480}"

if [[ ! -x "${BINARY_PATH}" ]]; then
  echo "[memory-check] binary not found at ${BINARY_PATH}; building..."
  make -C "${ROOT_DIR}" build >/dev/null
fi

tmp_time_out="$(mktemp)"
trap 'rm -f "${tmp_time_out}"' EXIT

if [[ "$(uname -s)" == "Darwin" ]]; then
  /usr/bin/time -l "${BINARY_PATH}" version >/dev/null 2>"${tmp_time_out}"
  rss_bytes="$(awk '/maximum resident set size/{print $1}' "${tmp_time_out}" | tail -n1)"
  rss_kb="$((rss_bytes / 1024))"
else
  /usr/bin/time -v "${BINARY_PATH}" version >/dev/null 2>"${tmp_time_out}"
  rss_kb="$(awk -F: '/Maximum resident set size/{gsub(/^[ \t]+/, "", $2); print $2}' "${tmp_time_out}" | tail -n1)"
fi

if [[ -z "${rss_kb}" ]]; then
  echo "[memory-check] failed to parse peak RSS from /usr/bin/time output"
  cat "${tmp_time_out}"
  exit 1
fi

echo "[memory-check] peak RSS: ${rss_kb} KiB (budget: ${BUDGET_KB} KiB)"

if (( rss_kb > BUDGET_KB )); then
  echo "[memory-check] FAILED: peak RSS exceeds budget"
  exit 1
fi

echo "[memory-check] PASS"
