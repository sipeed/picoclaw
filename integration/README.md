# Integration Test Suites

This directory contains the integration test suites that CI runs before merging PRs and before building `main`.

These tests exist to catch regressions that are easy to miss with unit tests alone, especially when different PRs touch adjacent code paths and the breakage only appears once the pieces are wired together. Typical examples are:

- protocol compatibility across package boundaries
- request and response behavior over real transports
- subprocess, CLI, or container wiring
- configuration passed through environment variables
- startup, discovery, and teardown flows involving more than one component

## Two Layers of Integration Testing

PicoClaw currently uses two related mechanisms:

1. Go integration tests, usually in `*_integration_test.go` files and guarded by `//go:build integration`
2. Docker-backed suites in `integration/suites/` that start real dependencies and run one or more of those Go tests in CI

That distinction matters:

- a tagged Go integration test is the test implementation
- a Docker-backed suite is how we make that test reproducible and CI-safe

If a test should actively protect merges between PRs, it should be reachable from [`scripts/run-integration-tests.sh`](../scripts/run-integration-tests.sh), either by extending an existing suite or by adding a new one.

Some integration-tagged tests are intentionally opt-in and not part of the Docker suites. For example, real CLI smoke tests under `pkg/providers/cli/` depend on external binaries being installed locally. Those are useful for manual verification, but they do not gate PR merges.

## What CI Runs

The integration jobs in [`.github/workflows/pr.yml`](../.github/workflows/pr.yml) and [`.github/workflows/build.yml`](../.github/workflows/build.yml) both execute:

```bash
bash ./scripts/run-integration-tests.sh
```

The runner auto-discovers every suite under `integration/suites/`, so adding a suite does not require editing the GitHub Actions workflow.

## How the Runner Works

- The shared runner is defined in [`integration/docker-compose.runner.yml`](docker-compose.runner.yml).
- Each suite lives in `integration/suites/<suite-name>/`.
- The runner script loads `suite.env`, merges the shared compose file with the suite-specific compose files, starts dependency services, runs the suite command, and then tears everything down.
- The shared runner container sets `GOFLAGS=-tags=goolm,stdjson,integration`, so tests run with the same build tags used by CI.

In practice, each suite gives us:

- a deterministic dependency graph
- a stable execution environment
- automatic cleanup after the run
- a clean place to encode the exact regression we want to prevent

## Current Reference Suite

[`integration/suites/mcp-streamable/`](suites/mcp-streamable/) is the reference example today.

It does three things:

- builds and starts a fixture MCP server from [`integration/fixtures/mcp-streamable-server/`](fixtures/mcp-streamable-server/)
- injects connection details into the runner container through environment variables
- runs [`TestIntegration_RealConfiguredServer`](../pkg/mcp/manager_real_server_integration_test.go) to verify that PicoClaw can connect to a real server, discover tools, invoke one, and validate the response payload

That suite complements [`TestIntegration_StreamableHTTPCompatibility`](../pkg/mcp/manager_integration_test.go), which exercises the same area in-process. Together they cover both protocol behavior and real service wiring.

## Suite Layout

Each suite directory must contain:

- `suite.env`
- at least one `docker-compose.yml` or `docker-compose.*.yml`

Example:

```text
integration/suites/my-suite/
├── docker-compose.yml
└── suite.env
```

## Required Manifest Fields

`suite.env` is sourced by the runner script and must define:

- `TEST_COMMAND`: shell command executed inside the integration runner container

Optional fields:

- `RUNNER_SERVICE`: override the default runner service name (`integration-runner`)

Example:

```bash
TEST_COMMAND='go test ./pkg/mcp -run TestIntegration_RealConfiguredServer -v'
```

## Running Integration Tests Locally

### Prerequisites

- Docker with the `docker compose` plugin for Docker-backed suites
- Go 1.25+ only if you want to run tagged integration tests directly on your host instead of through Docker

### Run Everything That CI Runs

```bash
make integration-test
```

Equivalent direct command:

```bash
bash ./scripts/run-integration-tests.sh
```

### Run a Single Suite

```bash
bash ./scripts/run-integration-tests.sh mcp-streamable
```

This is the fastest way to reproduce exactly what the CI integration job does for one suite.

### Run a Tagged Integration Test Directly

For faster iteration while writing the test, you can run the Go test itself without Docker:

```bash
go test -tags=goolm,stdjson,integration ./pkg/mcp -run TestIntegration_StreamableHTTPCompatibility -v
```

You can also run the real-server smoke test directly if you provide the same environment variables that the Docker suite would inject:

```bash
PICOCLAW_MCP_REAL_SERVER_JSON='{"enabled":true,"type":"http","url":"http://127.0.0.1:8080/mcp"}' \
PICOCLAW_MCP_REAL_TOOL_NAME=echo \
PICOCLAW_MCP_REAL_TOOL_ARGS_JSON='{"message":"hello"}' \
PICOCLAW_MCP_REAL_EXPECT_SUBSTRING=hello \
go test -tags=goolm,stdjson,integration ./pkg/mcp -run TestIntegration_RealConfiguredServer -v
```

Notes:

- avoid `-short`: the current integration tests skip in short mode
- use direct `go test` for tight feedback loops, then validate the Docker suite before committing

## When to Add an Integration Test

Reach for an integration test when the risk lives in the interaction, not just in the function body. Good candidates include:

- code that crosses process or container boundaries
- transport-specific behavior such as HTTP, SSE, stdio, or streamable MCP flows
- CLI parsing and wiring that depends on real subprocess execution
- configuration propagation through files, env vars, headers, or service discovery
- regressions that usually appear only after merging separately reasonable PRs

Prefer a unit test when the behavior is pure, local, and fully controllable in-process.

## How to Add a New Integration Test

### 1. Start from the regression you want to prevent

Describe the real workflow that could break after a merge. The sharper the scenario, the better the test will age.

Examples:

- "PicoClaw can still connect to a streamable MCP server after transport normalization changes."
- "A provider wrapper still parses the real CLI output format after refactors in response handling."

### 2. Implement the Go test

Add or extend a `*_integration_test.go` file in the package that owns the behavior and gate it with:

```go
//go:build integration
```

Guidelines:

- keep the assertions focused on observable behavior
- use bounded timeouts
- prefer deterministic fixtures over internet access or shared external state
- skip only when the dependency is intentionally optional, such as a locally installed third-party CLI

### 3. Decide whether it must gate CI merges

Use this rule of thumb:

- if the test is only a manual smoke check, a tagged Go test may be enough
- if the test should prevent regressions from landing through PR merges, wire it into a Docker-backed suite

### 4. Reuse or add a suite

If an existing suite already exercises the same subsystem, extend it. Otherwise create:

```text
integration/suites/<name>/
├── docker-compose.yml
└── suite.env
```

Use `integration/fixtures/` for reusable helper services or fake servers.

### 5. Define the suite command

In `suite.env`, point `TEST_COMMAND` at the Go test you want CI to run.

Examples:

```bash
TEST_COMMAND='go test ./pkg/mcp -run TestIntegration_RealConfiguredServer -v'
```

```bash
TEST_COMMAND='go test ./pkg/somepkg -run TestIntegration_MyScenario -v'
```

You can also run multiple tests if they share the same environment, but keep suites cohesive and easy to diagnose when they fail.

### 6. Model the dependencies in Docker Compose

Suite compose files can:

- define dependency services needed by the tests
- extend or override the shared `integration-runner` service
- inject environment variables into the runner for the tests to consume

Practical advice:

- prefer Docker service names over hard-coded host ports
- add health checks when the runner depends on service readiness
- keep the suite self-contained and deterministic

### 7. Validate locally before committing

At minimum:

```bash
go test -tags=goolm,stdjson,integration ./path/to/package -run TestIntegration_Name -v
bash ./scripts/run-integration-tests.sh <suite-name>
```

The first command helps while authoring. The second proves that the CI path works end to end.

## Review Checklist for New Suites

Before opening the PR, check that the new suite:

- reproduces a realistic cross-component failure mode
- is deterministic and isolated
- does not require manual setup in CI
- has clear failure output
- finishes in a reasonable amount of time
- cleans up after itself through the normal runner teardown

Once committed, the suite will be auto-discovered by the CI integration job.
