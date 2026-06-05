# ForgeClaw Roadmap

ForgeClaw is a deployment-focused fork of PicoClaw. The roadmap tracks work
that matters for the active ForgeClaw deployment while preserving the ability to
merge upstream PicoClaw changes.

## 1. Runtime Reliability

- Keep async task delivery deterministic across restarts.
- Continue consolidating tool delivery through the shared delivery coordinator.
- Reduce duplicate final replies after media/file/message tools.
- Keep task-registry records bounded, queryable, and useful for debugging.
- Preserve topic/session routing across Telegram, cron, spawn, and delegate
  paths.

## 2. Context Management

- Keep Seahorse compaction bounded and observable.
- Prefer asynchronous compaction where it is safe.
- Fail closed when context still exceeds provider limits after compaction.
- Keep prompt assembly predictable by reserving budget for tools, non-history
  prompt sections, and required routing context.
- Add focused regression tests for long-session and post-compaction behavior.

## 3. Tooling And Workflow State

- Expand `task_board` only where it improves truthful multi-step execution.
- Keep `task_status` as the primary user-facing progress/status command.
- Retire duplicated status surfaces when they no longer add value.
- Improve deterministic test coverage for tool loops, spawned work, and async
  completions.
- Extract a shared path-scope validation package only if non-`exec` tools need
  the same workspace/symlink/allowed-path rules currently enforced by
  `shellguard`.

## 4. Provider And MCP Behavior

- Keep OpenAI OAuth/Codex paths reliable and observable.
- Preserve streamed output and streamed tool-call behavior in provider adapters.
- Keep MCP transport failures explicit and fail-fast.
- Maintain deferred MCP/tool discovery behavior so large tool inventories do not
  pollute ordinary prompts.

## 5. Channels And Media

- Keep Telegram forum-topic routing stable.
- Preserve media-group handling and forwardable media captions.
- Keep generated images and files deliverable without duplicate completion
  messages.
- Keep channel feedback throttling controlled by real edit intervals.

## 6. Automation And Agent Workflows

- Keep core workflow primitives deployment-agnostic.
- Support durable queued work without assuming a specific domain or workspace.
- Make spawned/delegated work observable through shared task status surfaces.
- Keep webhook, cron, and manual trigger paths consistent.
- Let deployments layer domain-specific agents and policies outside the core
  runtime.

## 7. Upstream Compatibility

- Merge `upstream/main` regularly.
- Keep fork-specific behavior documented in the root README fork note.
- Avoid repo-wide renames of binary/config/module paths unless there is a strong
  reason.
- Prefer small compatibility patches over broad rewrites that increase merge
  conflict cost.
