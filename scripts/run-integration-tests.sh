#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BASE_COMPOSE="$ROOT_DIR/integration/docker-compose.runner.yml"
SUITES_DIR="$ROOT_DIR/integration/suites"

if [[ ! -f "$BASE_COMPOSE" ]]; then
  echo "missing base compose file: $BASE_COMPOSE" >&2
  exit 1
fi

if [[ ! -d "$SUITES_DIR" ]]; then
  echo "missing integration suites directory: $SUITES_DIR" >&2
  exit 1
fi

sanitize_project_name() {
  printf '%s' "$1" | tr '[:upper:]' '[:lower:]' | tr -cs 'a-z0-9' '-'
}

collect_suite_dirs() {
  if [[ "$#" -gt 0 ]]; then
    local suite
    for suite in "$@"; do
      printf '%s\n' "$SUITES_DIR/$suite"
    done
    return
  fi

  find "$SUITES_DIR" -mindepth 1 -maxdepth 1 -type d | sort
}

run_suite() {
  local suite_dir="$1"
  local suite_name
  suite_name="$(basename "$suite_dir")"
  local manifest="$suite_dir/suite.env"

  if [[ ! -f "$manifest" ]]; then
    echo "suite $suite_name is missing manifest: $manifest" >&2
    return 1
  fi

  local compose_args=()
  compose_args+=(--project-directory "$ROOT_DIR" -p "picoclaw-int-$(sanitize_project_name "$suite_name")")
  compose_args+=(-f "$BASE_COMPOSE")

  local compose_files=()
  while IFS= read -r compose_file; do
    compose_files+=("$compose_file")
    compose_args+=(-f "$compose_file")
  done < <(find "$suite_dir" -maxdepth 1 -type f \( -name 'docker-compose.yml' -o -name 'docker-compose.*.yml' \) | sort)

  if [[ "${#compose_files[@]}" -eq 0 ]]; then
    echo "suite $suite_name has no docker-compose file" >&2
    return 1
  fi

  (
    set -a
    # shellcheck disable=SC1090
    source "$manifest"
    set +a

    : "${TEST_COMMAND:?suite $suite_name must define TEST_COMMAND in $manifest}"
    runner_service="${RUNNER_SERVICE:-integration-runner}"

    cleanup() {
      docker compose "${compose_args[@]}" down -v --remove-orphans >/dev/null 2>&1 || true
    }
    trap cleanup EXIT

    echo "==> [$suite_name] resolving services"
    local services=()
    while IFS= read -r service; do
      services+=("$service")
    done < <(docker compose "${compose_args[@]}" config --services)

    local dependency_services=()
    for service in "${services[@]}"; do
      if [[ "$service" != "$runner_service" ]]; then
        dependency_services+=("$service")
      fi
    done

    if [[ "${#dependency_services[@]}" -gt 0 ]]; then
      echo "==> [$suite_name] starting docker services: ${dependency_services[*]}"
      docker compose "${compose_args[@]}" up -d --build --wait "${dependency_services[@]}"
    fi

    echo "==> [$suite_name] running: $TEST_COMMAND"
    # integration-runner already uses `bash -lc` as its entrypoint, so pass the
    # suite command as a single argument for Bash to execute directly.
    docker compose "${compose_args[@]}" run --rm "$runner_service" "$TEST_COMMAND"
  )
}

main() {
  local suite_dirs=()
  while IFS= read -r suite_dir; do
    suite_dirs+=("$suite_dir")
  done < <(collect_suite_dirs "$@")

  if [[ "${#suite_dirs[@]}" -eq 0 ]]; then
    echo "no integration suites found" >&2
    exit 1
  fi

  local suite_dir
  for suite_dir in "${suite_dirs[@]}"; do
    if [[ ! -d "$suite_dir" ]]; then
      echo "unknown integration suite: $suite_dir" >&2
      exit 1
    fi
    run_suite "$suite_dir"
  done
}

main "$@"
