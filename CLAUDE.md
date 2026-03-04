# CLAUDE.md

## Project: picoclaw

Go-based AI agent with multi-channel messaging (Telegram, Discord, Slack, etc.) and a Telegram Mini App UI.

## Build & Test

```bash
go build ./...
go test ./...
go vet ./...
```

Lint: `golangci-lint run`

## Plan Mode

- **Interview tool filtering**: `interviewAllowedTools` in `pkg/agent/loop.go` is the single source of truth for tools available during interview/review phases. Both `filterInterviewTools` (strips definitions before LLM call) and `isToolAllowedDuringInterview` (argument-level gating) reference this map.
- **History clear**: `/plan start clear` wipes session history and summary on transition to executing. The Mini App review UI offers two sliders: standard approve and approve-with-clear.

## Subagent Orchestration (実装済み部分)

- **Startup flag**: `--orchestration` で on/off。`SubagentsConfig.Enabled` で gate。
- **Conductor identity**: orchestration 有効時に conductor identity + spawn/subagent guidance を system prompt へ注入。
- **Sandbox/Spawn**: `pkg/tools/sandbox.go`, `pkg/tools/spawn.go` 実装済み。
- **AgentReporter**: `orch.AgentReporter` / `orch.Noop` / `orch.Broadcaster` で統一。main/heartbeat/subagent 全セッションが同一 Broadcaster に発火。Mini App は `agentLoop.GetOrchBroadcaster()` → `handler.SetOrchBroadcaster()` で受信。

## コードの匂い — チェックリスト

新しいコードを書くとき・レビューするときの確認事項:

1. **`[]string` + `strings.Join`** → `strings.Builder` に一本化
2. **`[]byte → string → io.Reader`** → `bytes.NewReader(b)` を直接使用
3. **ループ内で静的なものを毎回生成** → ループ外で1回生成してキャッシュ
4. **全件コピーが呼び出し側の用途より広い** → COW または RWMutex + ポインタ返却を検討
5. **同じ content を複数関数が独立して Split** → 呼び出し側で1回 Split して渡す
6. **`var x []T` から始まる容量なし append** → ソース長が既知なら `make([]T, 0, n)`
7. **`[]rune(s)` 変換前に長さチェックなし** → `len(s) <= max` で ASCII fast path を先に

---

## 未実装タスク

以下の `todo/` ファイルに分割。各ファイルは互いに依存関係がなく、別ブランチで並列実装可能。

| ファイル | 概要 |
|---|---|
| [`todo/TASKS-1.md`](todo/TASKS-1.md) | **Memory & Performance Optimization** — MemoryStore キャッシュ、FunctionCall/ToolDefinition 型整理、stats フラッシュ最適化 |
| [`todo/TASKS-2.md`](todo/TASKS-2.md) | **Subagent Orchestration (Container Model)** — SubagentContainer、Orchestrator、Presets enforcement、Subagent Plan Mode |
| [`todo/TASKS-3.md`](todo/TASKS-3.md) | **Session DAG (SQLite Store)** — セッション管理の SQLite 移行、Turn ベース線形+セッション間 DAG、Fork/Report フロー |
| [`todo/TASKS-4.md`](todo/TASKS-4.md) | **Mini App & Static Serving** — 静的配信の汎用化、バンドラ導入、フロントエンドテスト追加 |
| [`todo/TASKS-5.md`](todo/TASKS-5.md) | **Heartbeat Worktree Management** — worktree 一覧・点検・手動 merge/dispose の CLI/Mini App UI |
