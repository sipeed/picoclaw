# picoclaw — Claude Code Rules

## What This Is
Tiny, fast, deployable-anywhere automation CLI — part of the claw family of AI assistants.

## Stack & Conventions
- **Go** (module `github.com/sipeed/picoclaw`). CLI entry in `cmd/`, config in `config/`, docker in `docker/`.
- Tests: `go test ./...`. Build: `make build` (see `Makefile`).
- This repo is a nuestra-ai fork that tracks an upstream; be conservative about restructuring that would conflict with upstream merges.

## Control Plane
Directives, skills, agent definitions, and the broader context tree live in the **agent-platform umbrella repo** at `../agent-platform/`. For cross-repo work or any `/directive` pipeline usage, open Claude from there:

```bash
cd ../agent-platform && claude
```

## Git Operations
NEVER perform git operations without explicit user approval.
