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

## Subagent Orchestration (実装済み)

- **Startup flag**: `--orchestration` で on/off。`SubagentsConfig.Enabled` で gate。
- **Conductor identity**: orchestration 有効時に conductor identity + spawn/subagent guidance を system prompt へ注入。
- **Sandbox/Spawn**: `pkg/tools/sandbox.go`, `pkg/tools/spawn.go` 実装済み。
- **AgentReporter**: `orch.AgentReporter` / `orch.Noop` / `orch.Broadcaster` で統一。main/heartbeat/subagent 全セッションが同一 Broadcaster に発火。Mini App は `agentLoop.GetOrchBroadcaster()` → `handler.SetOrchBroadcaster()` で受信。
- **Container Model (Q&A escalation)**:
  - `ContainerMessage` + `inCh`/`outCh` channels on `SubagentTask` — deliberate preset (coder/worker/coordinator) のみ
  - `ask_conductor` tool — subagent → conductor question (blocking)
  - `answer_subagent` tool — conductor → subagent answer
  - `submit_plan` tool — subagent → conductor plan review (blocking)
  - `review_subagent_plan` tool — conductor → subagent approve/reject
  - `PendingQuestions()` で conductor LLM loop に question/plan_review を注入
- **Deliberate Plan Mode**: `SubagentPlanState` (Clarifying → Review → Executing → Completed)
  - `runDeliberateTask()`: clarifying phase (ask_conductor + submit_plan のみ) → executing phase (全ツール)
  - `runExploratoryTask()`: exploratory preset の single-phase loop
- **Environment injection**: `extractPlanContext()` で MEMORY.md から Context/Commands/Orchestration セクションを抽出 → subagent system prompt に注入
- **SessionRecorder 拡張**: `RecordQuestion()` / `RecordPlanSubmit()` + `TurnQuestion` / `TurnPlanSubmit` TurnKind

## Session DAG (Phase 0–3 実装済み)

- **SQLite SessionStore**: `pkg/session/sqlite.go` — `modernc.org/sqlite` (CGO不要)、WAL モード、`sessions` + `turns` テーブル
- **SessionStore interface**: `pkg/session/store.go` — Create/Get/List/Append/Turns/Compact/Fork/Prune 等15メソッド
- **LegacyAdapter**: `pkg/session/legacy_adapter.go` — SessionStore をラップし SessionManager と同一 API を提供。`Store()` / `AdvanceStored()` で直接 DAG 操作も可能
- **CompactOldTurns**: `LegacyAdapter.CompactOldTurns(key, keepLast, summary)` — flush → turn 単位で cut point 算出 → SQLite Compact → キャッシュ更新。`summarizeSession()` から呼び出し (fallback 付き)
- **SessionGraph**: `pkg/session/graph.go` — `SessionGraph` + `BeginTurn()` / `TurnWriter`。`LegacyAdapter.Graph()` で取得。将来の段階移行準備
- **JSON → SQLite migration**: `pkg/session/migrate.go` — 起動時に `sessions/*.json` を検出 → SQLite import → `.json.migrated` にリネーム
- **配線**: `pkg/agent/instance.go` の `Sessions` 型が `*LegacyAdapter` に変更。`sessions.db` を workspace 直下に生成
- **SessionRecorder**: `pkg/tools/session_recorder.go` (interface) + `pkg/agent/session_recorder.go` (impl) — SubagentManager から Fork/Turn/Completion/Report を記録
- **Prune**: 起動時 `store.Prune(7d)` + `flushLoop` 内 6h 定期 prune。`AgentLoop.gcLoop` で 30分毎に idle `sessionLocks` を GC
- **CLI `/session` コマンド**: `list` / `graph` / `fork [label]` / `reset` サブコマンド。default は DAG summary + token stats
- **Mini App Session Graph**: `/miniapp/api/sessions/graph` endpoint。SSE `session` event に `graph` 含む。フロントエンドで tree rendering

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

以下の `todo/` ファイルに分割。基本は別ブランチで並列実装可能（※ TASKS-2 は TASKS-1 の型変更前提あり）。

| ファイル | 概要 |
|---|---|
| [`todo/TASKS-1.md`](todo/TASKS-1.md) | ~~**Memory & Performance Optimization**~~ ✅ 実装済み（MemoryStore キャッシュ＋パース済み state、FunctionCall.Arguments map統一、ToolDefinition.Parameters RawMessage化、検索結果フォーマット共通化、stats 定期フラッシュ） |
| [`todo/TASKS-2.md`](todo/TASKS-2.md) | ~~**Subagent Orchestration (Container Model)**~~ ✅ 実装済み（Container Q&A escalation、Deliberate Plan Mode、Environment injection、SessionRecorder 拡張） |
| [`todo/TASKS-3.md`](todo/TASKS-3.md) | ~~**Session DAG (SQLite Store)**~~ ✅ 実装済み（Phase 0–3: SQLite SessionStore、LegacyAdapter、Fork/Report、CompactOldTurns、`/session` CLI コマンド、Mini App グラフ UI） |
| [`todo/TASKS-4.md`](todo/TASKS-4.md) | ~~**Mini App & Static Serving**~~ ✅ 実装済み（`http.FileServer` 統合、テンプレート注入、Bun ビルド導線、frontend unit test + CI `pnpm test`） |
| [`todo/TASKS-5.md`](todo/TASKS-5.md) | ~~**Heartbeat Worktree Management**~~ ✅ 実装済み（`/plan worktrees` の `list/inspect/merge/dispose`、安全化した `PruneOrphaned`、Mini App `/miniapp/api/worktrees` + Git タブ UI） |
| [`todo/TASKS-6.md`](todo/TASKS-6.md) | **SOUL.md — AI Persona Evolution** — 睡眠フェーズで体験を統合・忘却し人格を再構成。TASKS-2 完了後に着手 |
| [`todo/TASKS-7.md`](todo/TASKS-7.md) | **Provider Wire Compatibility Hardening** — openai_compat の provider 別 wire 分岐（OpenAI strict / Gemini）、thought_signature round-trip 保全、互換テスト追加 |


