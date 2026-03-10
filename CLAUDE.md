# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

PicoClaw is an ultra-lightweight personal AI assistant written in Go, designed to run on minimal hardware (<10MB RAM). It supports multiple LLM providers and chat channels (Telegram, Discord, QQ, WhatsApp, Feishu, Slack, Matrix, etc.).

This repo is a **company internal fork**. The `main` branch mirrors upstream; all development happens on `internal_main`. Branch off `internal_main` for features/bugfixes, never develop directly on `main`.

## Build & Development Commands

```bash
# Build
make build                  # Build for current platform (runs go generate first)
make generate               # Run go generate only
make build-launcher         # Build web console (frontend + Go backend)

# Test
make test                   # All Go tests
go test ./pkg/agent/        # Test a specific package
go test -run TestName -v ./pkg/session/  # Run a single test

# Lint & Format
make lint                   # golangci-lint (strict config in .golangci.yaml)
make fmt                    # Format code (gci, gofmt, gofumpt, goimports, golines)
make vet                    # Static analysis
make check                  # Full pre-commit: deps + fmt + vet + test

# Frontend (web/frontend/)
cd web/frontend && pnpm install
pnpm dev                    # Vite dev server
pnpm build                  # Production build
pnpm build:backend          # Build into web/backend/dist for embedding
pnpm lint                   # ESLint
pnpm check                  # Prettier + ESLint fix

# Web console dev (web/)
cd web && make dev           # Start backend + frontend dev servers
```

## Go Build Configuration

- `CGO_ENABLED=0` — static binaries, no cgo
- Build tags: `stdjson` (standard library JSON)
- WhatsApp native support: build tag `whatsapp_native` (separate, larger binary)
- LDFLAGS inject version from git tags
- Module: `github.com/sipeed/picoclaw`, Go 1.25+

## Architecture

### Core Flow
```
Channel → MessageBus → Agent → LLM Provider
                         ↕
                     Tool System (exec, fs, web, MCP, cron, skills)
```

### Key Packages (pkg/)
- **agent/** — Agent loop, registry, memory, thinking. Multi-agent with per-agent workspace/model/tools
- **bus/** — Central MessageBus for decoupled channel↔agent communication
- **channels/** — Pluggable channel implementations (telegram, discord, qq, whatsapp_native, feishu, dingtalk, slack, matrix, irc, line, wechat_work, onebot, maixcam)
- **providers/** — LLM provider abstraction with fallback chains (Anthropic, OpenAI, OpenRouter, Gemini, Groq, Zhipu, etc.)
- **tools/** — Extensible tool registry with safety features (deny patterns, path restrictions)
- **mcp/** — Model Context Protocol integration (Go SDK, HTTP+stdio transport, BM25 tool search)
- **session/** — JSONL-based session persistence
- **skills/** — Installable skill packages with ClawHub registry
- **config/** — Configuration loading/validation
- **commands/** — Command registry and execution

### CLI Structure (cmd/picoclaw/)
Entry point is `cmd/picoclaw/main.go` using cobra. Subcommands in `cmd/picoclaw/internal/` (agent, gateway, auth, cron, migrate, onboard, skills, status, version).

### Frontend (web/frontend/)
React 19 + TypeScript + Vite + TailwindCSS 4 + TanStack Router + Radix UI + Jotai state + i18next. Built output embeds into the Go binary at `web/backend/dist/`.

## Code Style

- Formatting enforced by golangci-lint: gci (import ordering: std → third-party → local), gofumpt, golines (120 char max)
- Use `any` instead of `interface{}`
- Commit messages: conventional commits, imperative mood, English ("Add retry logic" not "Added retry logic")
- Reference issues in commits: `Fix session leak (#123)`

## Branching (Internal Fork)

```
upstream/main → main (mirror only) → internal_main (dev mainline) → feature/*, bugfix/*
```

PRs target `internal_main`. To contribute upstream: cherry-pick from `main` branch.
