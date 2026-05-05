## 2026-05-05 00:00 [saved]

Goal: Initial project setup - create memory files for PicoClaw project
Decisions:

- Initialized git repository for staleness tracking (enables precise project-map.md freshness checks)
- Created project-map.md with directory structure, key files, critical constraints, and hot files
- Created CLAUDE.md with build commands, environment setup, critical constraints, and architecture notes
- Project is PicoClaw: ultra-lightweight AI assistant in Go targeting low end hardware with <50MB RAM
Rejected: None (initial setup)
## 2026-05-05 00:01 [saved]
Goal: Document tools implementation and backend API interaction
Decisions:
- Created docs/tools-api.md with comprehensive tools documentation
- Documented all 20+ tools across 8 categories (filesystem, automation, web, communication, skills, agents, hardware, discovery)
- Documented backend API endpoints for tool management (GET/PUT /api/tools, web-search-config)
- Documented tool data structures (Tool interface, ToolResult, SubTurnConfig)
- Documented MCP integration and tool discovery flow
- Documented security considerations and configuration examples
Rejected: None
Open: None

## 2026-05-05 00:02 [saved]
Goal: Document exec tool internals and how to sandbox/remove it
Decisions:
- Created docs/reference/exec-tool.md with comprehensive exec tool documentation
- Documented exec flow: LLM invocation → action routing → sync/background execution → isolation wrapper
- Documented 40+ built-in deny patterns (rm -rf, dd, shutdown, sudo, docker, etc.)
- Documented 3 ways to disable: config (enabled=false), disable remote, strict timeout
- Documented complete removal steps (delete files, remove registration, config, defaults)
- Documented sandboxing options: bubblewrap (Linux), restricted tokens (Windows), workspace restriction, custom deny patterns
- Listed all key files: shell.go, session.go, spawn.go, instance.go, config.go, defaults.go, isolation/
Rejected: None
Open: None
