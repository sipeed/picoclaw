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

## Known Gaps

- **Mini App log viewer has no frontend tests**: `renderLogs()` in `pkg/miniapp/static/index.html` is inline vanilla JS with no unit/E2E test coverage. Backend (Go) tests cover `RecentLogs`, `SanitizeFields`, and JSON serialization, but nothing verifies the JS rendering. This allowed the Fields display bug (fields sent but not rendered) to ship undetected.
- **No human intervention for heartbeat worktrees**: Heartbeat sessions create git worktrees (`.worktrees/heartbeat-YYYYMMDD/`) but there is no CLI or Mini App command to list, inspect, or manually dispose them. Need a `/plan worktrees` command (or similar) that shows active worktrees with branch/commit info and allows manual merge/dispose. `PruneOrphaned` on startup only removes directories without auto-committing first, so uncommitted changes in orphaned worktrees are silently lost.

---

## Subagent Orchestration Design

> Designed 2026-02-25 on branch `sub-agent-technical-breakdown`.

### なぜオーケストレーションか

単純な指示から可能性の木を広げることが目的。conductor は一人でやり遂げるのではなく、探索・深化・fork をサブエージェントに委ねながら大局観を保つ。

```
without orchestration:
  human → conductor → (全部自分でやる) → result
  常にボトルネック、逐次処理

with orchestration:
  human → conductor ─┬─ scout A ─┐
                     ├─ scout B ─┼─ synthesize → deeper insight
                     └─ scout C ─┘
  conductor は次を考えながら並走
```

**3つの核心原則:**

1. **Fork** — 同じ問いに複数の切り口で同時探索。sequential queue ではなく tree の展開。
2. **管理職の原則** — conductor は subagent の完了を待たない。spawn したら即座に次を設計する。人員を遊ばせないことがポイント。
3. **会話の fork** — main thread (conductor ↔ human) は高レベル・戦略的に保つ。subagent への細かい指示出しは branch thread で行い、main thread を汚染しない。subagent の進捗はサマリーだけ main thread に上げる。

**spawn がデフォルト、subagent は例外:**
```
spawn   = conductor が次を考え続けられる (正しい姿)
subagent = conductor が止まる (結果が絶対必要な時だけ)
```

### Architecture Overview

```
Conductor goroutine
    │  ContainerRequest (task, preset, environment)
    ▼
Container goroutine: provision → run → finalize
    │  ContainerMessage (question / result / status)
    ▼
Conductor goroutine
    │  answer (question への回答)
    ▼  (わからなければ human に escalate)
Container goroutine (再開)
```

**escalation chain:**
```
subagent (clarifying) → question → conductor
  → conductor が答えられる: inCh に回答
  → conductor もわからない: message tool で human に投げ、回答を転送
```

### SubagentContainer

goroutine と channel でライフサイクルを表現。goroutine がブロックしている場所が現在の状態。

```go
type ContainerMessage struct {
    Type    string // "question" | "result" | "status"
    Content string
}

type SubagentContainer struct {
    inCh    chan string           // conductor → subagent (回答)
    outCh   chan ContainerMessage // subagent → conductor (質問・結果・進捗)
    cancel  context.CancelFunc
}
```

spawn (async) は outCh を返して即リターン。subagent (sync) はその場で result を待つ。

**tasks map 問題の解消:** goroutine 終了時に `defer orchestrator.active.Delete(id)` + `defer close(outCh)` で自動 GC。

### SubagentEnvironment (Context Injection)

conductor は subagent に必要なコンテキストを明示的に渡す。MEMORY.md からの自動注入で冗長な手動記述を排除。

```go
type SubagentEnvironment struct {
    // 自動注入 (harness が埋める)
    Workspace    string   // workspace パス
    WorktreeDir  string   // write先、workDir として透過的に機能

    // MEMORY.md から自動抽出 (inject_plan_context: true の場合)
    PlanTask     string   // > Task: の内容
    PlanContext  string   // ## Context セクション
    Commands     string   // ## Commands セクション (build/test/lint)
    CurrentPhase string   // 対象 Phase の内容

    // conductor が明示的に追加
    Background   string   // 追加の背景・意図
    Constraints  string   // 制約
    ContextFiles []string // 参照すべきファイルリスト
}
```

spawn パラメータ例:
```json
{
  "task": "Phase 2 Step 1: implement the rate limiter",
  "preset": "coder",
  "context_files": ["pkg/ratelimit/ratelimit.go"],
  "inject_plan_context": true
}
```

### SandboxConfig

ToolRegistry.Execute() の入口で一括 enforcement。subagent は普通に tool call するつもりで透過的に sandboxed になる。

```go
type SandboxConfig struct {
    Preset           string
    WriteRoot        string          // write 系ツールのパス制限
    AllowedTools     map[string]bool
    ExecPolicy       *ExecPolicy     // nil = exec 不可
    SpawnablePresets []string        // nil = spawn 不可
}

type ExecPolicy struct {
    AllowPattern string // 先頭一致 regex; マッチしたコマンドだけ実行可
}
```

**透過的隔離:** workDir = worktreeDir として設定することで、AI は自分が隔離されていることに気づかずに振る舞う。picoclaw 側で CoW 的にファイルを引き渡せる。

### Presets (5種)

| preset | 性格 | write | exec | search | spawn |
|---|---|---|---|---|---|
| `scout` | Exploratory | ✗ | ✗ | ✓ | ✗ |
| `analyst` | Exploratory | ✗ | go test/vet, git log/diff, grep | ✓ | ✗ |
| `coder` | Deliberate | ✓ sandbox | test/lint/fmt 系 | ✓ | ✗ |
| `worker` | Deliberate | ✓ sandbox | build/package manager 系 | ✓ | ✗ |
| `coordinator` | Deliberate | ✓ sandbox | go/pnpm/bun/curl 系 | ✓ | scout〜worker のみ |

**性格の分類:**
- **Exploratory** (scout/analyst): open-ended、見てきて報告。clarifying フェーズなし。
- **Deliberate** (coder/worker/coordinator): 成果物を作る。目標があいまいだと失敗する。clarifying フェーズあり。

**境界:**
- `coder` = 書いて自分で検証できる (package 追加・deploy 不可)
- `worker` = インフラも含めてやりきる (package install, CI pipeline 等)
- `coordinator` = coordinator を spawn できない (深さ自然制限)

**npm は全 preset で禁止:** git worktree に node_modules が作られると大量ファイルが生じるため。pnpm は symbolic link で済む、bun も同様。

**websearch/webfetch は全 preset で許可:** read-only・非破壊のため制限不要。

#### exec allowlist regex

```go
var presetExecPatterns = map[string]string{
    "scout":   ``,
    "analyst": `^(go\s+(test|vet)|git\s+(log|diff|status)|curl|wget|grep|find)\b`,
    "coder": `^(` +
        `go\s+(test|vet|fmt)|gofmt|goimports|golangci-lint|` +
        `prettier|eslint|` +
        `black|ruff|` +
        `cargo\s+(test|fmt|clippy)|` +
        `pnpm\s+(test|run\s+(test|lint|format))|` +
        `bun\s+(test|run\s+(test|lint|format))|` +
        `uv\s+run\s+` +
        `)\b`,
    "worker": `^(` +
        `go\s+|` +
        `pnpm\s+(install|add|run|test|build)|` +
        `bun\s+(install|add|run|test|build)|` +
        `uv\s+(run|sync|add|pip\s+install)|` +
        `pip\s+install|` +
        `cargo\s+` +
        `)\b`,
    "coordinator": `^(go\s+|pnpm\s+|bun\s+|curl|wget)\b`,
}
```

### Subagent Plan Mode

Deliberate な preset (coder/worker/coordinator) は in-memory のミニ plan mode を持つ。MEMORY.md には一切触れない (ファイル参照・編集を避けるため)。

```go
type SubagentPlanState int

const (
    PlanStateNone       SubagentPlanState = iota // Exploratory preset
    PlanStateClarifying                          // 目的・制約を確認中
    PlanStateReview                              // conductor の承認待ち
    PlanStateExecuting                           // 実行中
)

type SubagentPlan struct {
    State    SubagentPlanState
    Goal     string   // clarifying で合意した目的
    Approach []string // proposed なステップ
    QA       []QAItem // 質問・回答の履歴
    mu       sync.Mutex
}
```

SubagentContainer がフィールドとして保持。goroutine 終了とともに消える。

**Deliberate preset の system prompt:**
```
You are in clarifying mode. Before executing, confirm with the conductor:
1. What is the exact goal?
2. What are the constraints and acceptance criteria?
3. Are there relevant files I should know about?
Use the `message` tool to ask questions.
When you have clear answers, propose your approach (steps) for review.
Do NOT start executing until the conductor approves.
```

**Exploratory preset の system prompt:**
```
Explore and return findings. Use your best judgment when encountering ambiguity.
```

**fractal 構造:**
```
human
  ↕ plan mode (MEMORY.md, file-based, 永続)
conductor
  ↕ subagent plan mode (in-memory, 揮発)
subagent (deliberate)
```

### MEMORY.md Orchestration Section

executing 中に conductor が自由に書き込める専用エリア。システムはパースしない。

```markdown
## Orchestration

### Delegated
<!-- subagent に委譲したタスクの記録 (spawn 直後に書く) -->
- coder-1 (coder): rate limiter 実装 → Phase 2 Step 1
- scout-1 (scout): pkg/auth の構造調査

### Findings
<!-- subagent の結果から蓄積した知見 (結果受信後に書く) -->
- pkg/auth は middleware パターン、入口は middleware.go (scout-1)
- セッションストアは存在しない、JWT が有効 (scout-2)

### Decisions
<!-- fork の意思決定ログ (方向選択時に書く) -->
- auth: OAuth2 より JWT を選択 (外部依存なし、scout-2 推奨)
```

conductor の guidance に追記:
```
After spawning a subagent, record the assignment in ## Orchestration > Delegated.
When a subagent reports back, move key findings to ## Orchestration > Findings.
When you choose one direction over another, log the rationale in ## Orchestration > Decisions.
```

### Conductor Identity (System Prompt)

`pkg/agent/context.go` の `getIdentity()` に追加予定:

```
You are picoclaw, a conductor AI agent. Your role is to orchestrate:
break work into tasks, delegate them to subagents, and synthesize results —
rather than doing everything inline yourself.

## Orchestration

You are the conductor, not the performer. Prefer delegation over doing everything inline.

Use `spawn` (non-blocking) when:
- Tasks can run in parallel or in the background
- Multiple independent tasks can run simultaneously (spawn each one)
- You don't need the result to decide the next step
- The operation is long-running (builds, fetches, analysis, file processing)

Use `subagent` (blocking) when:
- You need the result before you can continue
- Correctness of next steps depends on the outcome

Do inline only when:
- It's a single fast tool call (read a file, quick search)
- Delegation overhead clearly outweighs the benefit

Default bias: if a task involves more than 2-3 tool calls or can run
independently, delegate it. When you spawn, immediately plan what comes next —
blocking means you've stopped thinking.

Fork aggressively: explore multiple directions simultaneously.
After spawning a subagent, record the assignment in ## Orchestration > Delegated.
When results come back, synthesize and decide the next fork.
```

### Startup Flag

テスト用途で起動時にオーケストレーション機能を on/off できるようにする。

**変更箇所:**
1. `pkg/config/config.go` — `SubagentsConfig` に `Enabled bool` を追加
2. `cmd/picoclaw/cmd_agent.go` — `--orchestration` フラグを追加 (default: false for now)
3. `pkg/agent/loop.go` — `registerSharedTools()` で spawn tool 登録を `Enabled` で gate

```go
// pkg/config/config.go
type SubagentsConfig struct {
    Enabled     bool              `json:"enabled"`
    AllowAgents []string          `json:"allow_agents,omitempty"`
    Model       *AgentModelConfig `json:"model,omitempty"`
}

// cmd/picoclaw/cmd_agent.go
case "--orchestration":
    cfg.Agents.Defaults.Subagents.Enabled = true  // or toggle
```

### Implementation Files (予定)

```
pkg/tools/
  container.go      — SubagentContainer, ContainerRequest, ContainerMessage,
                      SubagentEnvironment, SubagentPlan
  sandbox.go        — SandboxConfig, ExecPolicy, preset 定義
  orchestrator.go   — Orchestrator (SubagentManager を置き換え)
  spawn.go          — preset / inject_plan_context パラメータ追加
pkg/agent/
  context.go        — conductor identity + orchestration guidance 追加
```

### AgentReporter 抽象化 (実装済み 2026-02-25)

> branch `sub-agent-technical-breakdown`

`Broadcaster` を `SubagentManager` 内部で生成する密結合を解消し、
`orch.AgentReporter` インターフェースを中心に置くリファクタリングを実施。

#### オーナーシップ

```
AgentLoop
  ├─ owns: *orch.Broadcaster  (orchBroadcaster — nil when disabled)
  └─ holds: orch.AgentReporter (orchReporter = Broadcaster or Noop)
       ├─ passes to → SubagentManager.reporter
       │               └─ passes to → ToolLoopConfig.Reporter
       └─ calls directly for main/heartbeat sessions
            ├─ runAgentLoop: ReportSpawn / ReportGC
            └─ runLLMIteration: ReportStateChange

cmd_gateway.go
  └─ agentLoop.GetOrchBroadcaster() → handler.SetOrchBroadcaster()

miniapp.Handler
  └─ borrows *orch.Broadcaster for Subscribe/Snapshot (WS 配信)
```

#### インターフェース (`pkg/orch/reporter.go`)

```go
type AgentReporter interface {
    ReportSpawn(id, label, task string)
    ReportStateChange(id, state, tool string)
    ReportConversation(from, to, text string)
    ReportGC(id, reason string)
}
var Noop AgentReporter = &noopReporter{} // nil-free; 全メソッドが no-op
```

`Broadcaster` は `AgentReporter` を満たす (`ReportSpawn` 等が `Publish` のラッパー)。

#### Noop パターン

```
--orchestration なし: orchReporter = orch.Noop → 全 Report* が空振り (panic なし)
--orchestration あり: orchReporter = *Broadcaster → WS 配信
```

呼び出し側は `if reporter != nil` チェック不要。

#### イベント発火の責任分担

| 発火元 | イベント | 経由 |
|--------|---------|------|
| `runAgentLoop` | `ReportSpawn` / `ReportGC` | `al.reporter()` |
| `runLLMIteration` | `ReportStateChange("waiting"/"toolcall")` | `al.reporter()` |
| `SubagentManager.Spawn` | `ReportSpawn` | `sm.reporter` |
| `SubagentManager.runTask` | `ReportConversation` / `ReportGC` | `sm.reporter` |
| `RunToolLoop` | `ReportStateChange` | `config.Reporter` |

main / heartbeat / subagent の全セッションが同一 Broadcaster に発火するため、
canvas には全エージェントが統一して表示される。

#### 変更ファイル

- `pkg/orch/reporter.go` — **新規** インターフェース + Noop
- `pkg/orch/broadcaster.go` — `ReportSpawn/StateChange/Conversation/GC` 追加
- `pkg/tools/toolloop.go` — `OnStateChange func` → `Reporter AgentReporter + AgentID`
- `pkg/tools/subagent.go` — constructor に `reporter` 受け取り、内部 broadcaster 廃止、`GetBroadcaster()` 削除
- `pkg/agent/loop.go` — `orchBroadcaster`/`orchReporter` フィールド追加、`SetOrchReporter`/`GetOrchBroadcaster` 追加、`registerSharedTools` シグネチャに `al *AgentLoop` 追加
- `cmd/picoclaw/cmd_gateway.go` — `GetOrchBroadcaster()` → `handler.SetOrchBroadcaster()`

---

## Memory Optimization Notes

> Reviewed 2026-02-24 on branch `memory-optimization-review`.

### 設計レベルの根本原因

#### D-1. MemoryStore が「ファイル = 正」でパース済み表現をキャッシュできない

`GetMemoryContext()` 1回で `ReadLongTerm()` が5回以上呼ばれる連鎖。MEMORY.md を外部エディタが直接編集できる設計上、インメモリキャッシュを自然に導入できない。

対策: (a) content パススルー方式 — 高レベルメソッドだけが1回 ReadLongTerm() を呼び、content を private ヘルパーに渡す。(b) `*ParsedPlan` 常駐 — RAM が潤沢なので MemoryStore にパース済み構造体を持たせる。edit_file 後に `InvalidateCache()` を呼ぶ。

#### D-2. `FunctionCall.Arguments` が JSON 文字列のままドメイン型に

ストリーミングループ内の重複 Unmarshal の根本原因。`ToolCall.Arguments map[string]any` のパース済みフィールドも存在するが中途半端に共存している。

#### D-3. `ToolFunctionDefinition.Parameters` が `map[string]any`

プロバイダーへ送るたびに Marshal が必要。`json.RawMessage` にすれば一度の marshal で済む。

#### D-4. 検索プロバイダーに共通フォーマット抽象がない

`[]string + strings.Join` パターンが3箇所に複製。`Search()` 戻り値を `string` でなく構造体にすれば1箇所で済む。

#### D-5. `Session.Messages` が可変スライスで全コピーが必要

`GetHistory()` / `Save()` での防衛的コピーは意図的設計。COW または append-only immutable 構造で解消できる。

#### D-6. `MemoryStore` のメソッド境界が「ファイル操作単位」

呼び出し側は複数の値が必要でも複数回呼ぶしかない。D-1 の解決策 (ParsedPlan 常駐) と合わせて解消。

### コードの匂い — チェックリスト

新しいコードを書くとき・レビューするときの確認事項:

1. **`[]string` + `strings.Join`** → `strings.Builder` に一本化
2. **`[]byte → string → io.Reader`** → `bytes.NewReader(b)` を直接使用
3. **ループ内で静的なものを毎回生成** → ループ外で1回生成してキャッシュ
4. **全件コピーが呼び出し側の用途より広い** → COW または RWMutex + ポインタ返却を検討
5. **同じ content を複数関数が独立して Split** → 呼び出し側で1回 Split して渡す
6. **`var x []T` から始まる容量なし append** → ソース長が既知なら `make([]T, 0, n)`
7. **`[]rune(s)` 変換前に長さチェックなし** → `len(s) <= max` で ASCII fast path を先に

### ストレージ保護設計 (microSD 寿命)

| データ | 現状 | 推奨戦略 | 削減率 |
|---|---|---|---|
| `sessions/*.json` | メッセージ毎書き込み | write-behind (dirty flag + 5分タイマー) | 80% |
| `state/stats.json` | LLM呼び出し毎 | 定期フラッシュのみ (5分) | 98% |
| `memory/MEMORY.md` | 即時 (変えない) | ターンスコープキャッシュ (読み取りのみ最適化) | — |

**実装ポイント:**
- `SessionManager` に `dirtyKeys map[string]bool` + バックグラウンドフラッシャー goroutine
- `stats.Tracker` に `Close()` メソッド追加 (タイマー停止 + 最終 save)
- SIGTERM/SIGINT でシャットダウンフック必須
- MEMORY.md の edit_file 書き込み後にターンキャッシュを無効化 (`InvalidateCache()`)

---

## Session Management (Future)

現状の設計は「正確性」は成熟しているが「ライフサイクル」が欠落している。

**近期:**
- `SessionManager.Delete(key)` + TTL エビクション
- `sessionLocks sync.Map` (loop.go) の GC
- 起動時の `loadSessions()` を遅延ロード化

**中期:**
- チェックポイント / ロールバック (`Session.Messages` を append-only immutable に)
- 名前付きセッション (`/new-session`, `/switch-session`, `/list-sessions`)

**長期:**
- `Session.ParentKey` でサブエージェントセッションをグラフ化
- クロスセッション検索 (MEMORY.md の補完として)

設計の哲学: `Session.Messages` (履歴中心) と `MEMORY.md` (知識中心) が現状共存している。どちらを主軸にするかで発展方向が変わる。
