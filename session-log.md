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

## 2026-05-05 00:03 [saved]
Goal: Implement exec tool permission system via subagent-driven-development
Decisions:
- Executed plan docs/superpowers-optimized/plans/2026-05-05-exec-permission.md
- Task 1: Created PermissionCache (commit 843552ad)
- Task 2: Added PermissionCache tests (commit cf494c7b)
- Task 3: Created RequestPermissionTool (commit 7804b120)
- Task 4: Modified ExecTool to check permissions (commit 9ac94f32)
- Task 5: Added ask_permission to config (commit 8feb6892)
- Task 6: Wired up PermissionCache in agent (commit 9455c2d6)
- Task 7: Build passes, permission tests PASS
- Task 8: Updated documentation (commit 4c74b13c)
- Fixed test file to use exported field names (PermissionCache, AskPermission) (commit f46cf4c7)
Rejected: None
Open: None

Open: None
