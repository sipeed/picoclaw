# Project Map
_Generated: 2026-05-07 | Git: $(git rev-parse HEAD 2>/dev/null || echo "local")

## Directory Structure
cmd/ — CLI entry points (picoclaw main, membench, internal subcommands)
pkg/ — Core library packages (agent, channels, providers, tools, etc.)
web/frontend/ — Frontend UI (React/TypeScript with TanStack)
web/backend/ — Backend API server (Go, dashboard auth, middleware)
workspace/ — Runtime workspace (skills, memory)
docs/ — Documentation (architecture, channels, guides, migration, reference)
docs/reference/tools-api.md — Complete tools API documentation: available tools, data structures, backend API endpoints, MCP integration
docs/reference/exec-tool.md — Exec tool deep dive: how it works, security measures, how to disable/sandbox/remove completely
docs/superpowers-optimized/ — Superpowers workflow specs and plans
config/ — Configuration templates and examples
build/ — Build scripts and artifacts
docker/ — Docker containerization files
scripts/ — Automation and utility scripts
examples/ — Example projects (pico-echo-server)
assets/ — Static assets (logo, images)

## Key Files
cmd/picoclaw/main.go — Main CLI entry point using Cobra; registers subcommands (agent, auth, gateway, mcp, migrate, model, skills, etc.)
cmd/picoclaw/internal/ — Internal CLI command implementations (agent, auth, gateway, mcp, migrate, model, skills, status, version, onboard, cron, cliui)

### Core Packages
pkg/agent/ — Core agent logic: context management, pipelines, turn coordination, event handling, hooks, steering, thinking
pkg/agent/manager/ — Agent lifecycle management (manager.go, types.go)
pkg/agent/context_manager.go — Manages LLM context lifecycle, caching, and budget enforcement
pkg/agent/pipeline.go — Orchestrates agent execution phases (setup → LLM → tools → finalize)
pkg/channels/ — Multi-platform chat integrations: Discord, Telegram, Slack, WeChat, WeCom, Feishu, DingTalk, IRC, LINE, Matrix, VK, WhatsApp, OneBot, MaixCam, Pico
pkg/providers/ — AI model provider integrations: Anthropic, OpenAI-compatible, Azure, AWS Bedrock, CLI, HTTP API
pkg/config/ — Configuration loading, validation, and environment variable handling
pkg/skills/ — Skills system for extending agent capabilities
pkg/tools/ — Built-in tools: filesystem (fs), hardware interaction, shared utilities, integration tools
pkg/mcp/ — Model Context Protocol (MCP) server implementation for tool/resource exposure
pkg/memory/ — Agent memory management (short-term/long-term, persistence)
pkg/gateway/ — Gateway for routing messages between channels and agents (gateway.go, agent_api.go)
pkg/auth/ — Authentication and credential management (OAuth, API keys, encryption)
pkg/health/ — Health check endpoints and diagnostics server.go
pkg/bus/ — Internal event bus for decoupled communication

### Backend API (web/backend/api/)
- pico.go, router.go — Main API routing
- agents.go — Agent CRUD endpoints (/api/agents, /api/agent/*)
- skills.go — Skills management endpoints (/api/skills, /api/skills/*)

### Frontend (web/frontend/src/)
api/ — REST client wrappers
- agents.ts — Agent API client (listAgents, getAgent, createAgent, updateAgent, deleteAgent)
- skills.ts — Skills API client (listSkills, getSkill, searchSkills, installSkill, deleteSkill)

components/agent/
├── agents/ — Agent management UI
│   ├── agents-page.tsx — Main agents list with search, create, edit, delete
│   ├── agent-card.tsx — Agent display card
│   └── agent-form-modal.tsx — Create/edit agent form
├── cockpit/ — Main cockpit dashboard
│   ├── cockpit-page.tsx — Tools, Skills, Agents tabs with memory graph
│   └── use-agent-cockpit.ts — Cockpit state hook (tools, skills, agents, subagents)
├── research/ — Agent research/analysis features
│   ├── research-page.tsx — Research dashboard
│   ├── research-config.tsx — Configuration panel
│   ├── research-agents.tsx — Agent selection
│   ├── research-reports.tsx — Reports view
│   └── research-graph.tsx — Graph visualization
├── skills/ — Skills management UI
│   ├── skills-page.tsx — Skills list with search, delete
│   ├── skill-card.tsx — Skill display card
│   └── index.ts — Barrel exports
└── hub/ — Skill marketplace

hooks/ — React hooks
- use-agent-cockpit.ts — Main cockpit state
- use-cockpit-skills.ts — Skills state with TanStack Query
- use-agents.ts — Agent state management

routes/agent/ — Route definitions
- skills.tsx → /agent/skills
- research.tsx → /agent/research

## Cockpit UI Features
The cockpit provides tabs:
- **Tools**: Enable/disable tools with status badges
- **Agents**: CRUD for agent definitions (stored as .md in ~/.picoclaw/workspace/agents)
- **Skills**: View/delete installed skills
- **Sidebar**: Memory network graph, subagent manifest

go.mod — Go 1.25.9 module definition
Makefile — Build targets (build, test, lint, release)
.goreleaser.yaml — GoReleaser config for cross-platform releases

## Critical Constraints
- Target: Ultra-lightweight for $10 hardware with <10MB RAM
- Go 1.25.9 required
- Security: Prompt injection defense, tool abuse prevention, SSRF protection
- Workspace: ~/.picoclaw/workspace/ (agents, skills, memory)
- Frontend: TypeScript/React with TanStack router/query

## Hot Files
pkg/agent/manager/, pkg/gateway/, web/backend/api/agents.go, web/frontend/src/api/agents.ts, web/frontend/src/components/agent/cockpit/, web/frontend/src/components/agent/agents/, web/frontend/src/components/agent/research/