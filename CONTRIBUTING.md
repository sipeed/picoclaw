# Contributing to ForgeClaw

ForgeClaw is a personal fork of [PicoClaw](https://github.com/sipeed/picoclaw).
It keeps the upstream Go runtime and binary names, but carries deployment-driven
changes for personal automation, MCP-heavy workflows, task delivery, media
handling, context management, and multi-agent workflows.

This repository is not the upstream PicoClaw project. Contributions here should
optimize for the running ForgeClaw deployment while keeping upstream merges as
low-conflict as practical.

## Repository Layout

- `origin`: `git@github.com:bogdanovich/forgeclaw.git`
- `upstream`: `https://github.com/sipeed/picoclaw.git`
- `main`: active ForgeClaw branch.
- `upstream/main`: read-only tracking branch for upstream PicoClaw.

The binary, Go module path, config directory, and most command examples still
use `picoclaw` intentionally. Renaming those would create unnecessary upstream
merge conflicts and migration churn.

## Development Setup

Prerequisites:

- Go 1.25 or later
- `make`
- Node.js 22+ and pnpm 10.33.0+ for launcher/frontend changes

Build and test:

```bash
make deps
make build
make test
make lint-docs
```

For frontend/launcher work:

```bash
(cd web/frontend && pnpm install --frozen-lockfile)
make build-launcher
```

## Branching

For ForgeClaw changes:

```bash
git checkout main
git pull origin main
git checkout -b feat/short-description
```

Target ForgeClaw PRs at `main`.

For upstreamable changes:

1. Start from the latest `upstream/main`.
2. Create a clean topic branch.
3. Cherry-pick or manually port only the intended upstream patch.
4. Avoid bringing ForgeClaw-only deployment behavior into upstream PRs.

Do not open upstream PRs directly from ForgeClaw `main`.

## Keeping Up With Upstream

Periodically merge upstream into ForgeClaw:

```bash
git fetch upstream
git checkout main
git merge upstream/main
```

When resolving conflicts:

- keep ForgeClaw branding in the root README;
- keep deployment-specific fork notes unless the feature was truly merged
  upstream;
- preserve upstream bug fixes and dependency bumps unless they conflict with a
  fork-specific behavior that is still required;
- prefer small compatibility shims over broad rewrites.

## Code Style

- Keep changes narrowly scoped.
- Prefer existing package boundaries and local helper APIs.
- Avoid unnecessary abstractions.
- Add or update tests for behavioral changes.
- Run focused package tests before pushing.
- Run full lint when touching formatting-sensitive Go code:

```bash
make lint
```

The pre-push hook runs the same linter rules expected by GitHub checks.

## Documentation

The root `README.md` is the authoritative ForgeClaw entry document.

Use `docs/README.md` for documentation layout and naming conventions. Run:

```bash
make lint-docs
```

Documentation guidelines:

- Keep ForgeClaw-specific docs in English unless translations are intentionally
  maintained.
- Do not reintroduce upstream PicoClaw marketing, hardware sales, crypto scam
  warnings, star-count news, or `picoclaw.io` download instructions into the
  ForgeClaw root README.
- Keep command names such as `picoclaw`, paths such as `~/.picoclaw`, and Go
  module references when they describe the actual current binary/runtime.

## PR Expectations

PRs should include:

- a concise description of the change;
- the reason for the change;
- test commands run;
- screenshots/logs when user-facing behavior changes;
- AI assistance disclosure when relevant.

Reviewers should prioritize:

1. correctness and regressions;
2. security and tool-safety boundaries;
3. concurrency and async delivery behavior;
4. context/session isolation;
5. maintainability and simplicity;
6. test coverage.

## AI-Assisted Development

AI assistance is acceptable, but the author remains responsible for the change.

Before opening or merging AI-assisted code:

- read the diff carefully;
- verify behavior with tests or a concrete manual run;
- inspect security-sensitive paths yourself;
- remove speculative or over-engineered output.

## Commit Messages

Use concise imperative messages, preferably with a functional scope:

```text
fix(agent): preserve media delivery status
feat(tasks): add task board readiness view
docs: clarify fork maintenance workflow
```

Avoid `[codex]` prefixes in commit or PR titles.

## Communication

Use GitHub issues and PR comments for durable project discussion. For local
deployment notes that should not become upstream-facing docs, prefer workspace
documentation outside this source repository.
