# TASKS-2: Subagent Orchestration (Container Model)

## TASKS-1 反映メモ (2026-03-05)

TASKS-2 実装時は以下の型変更を前提にすること。

- `FunctionCall.Arguments` は JSON 文字列ではなく `map[string]any` 扱い。
  - 旧来の `json.Unmarshal([]byte(tc.Function.Arguments), ...)` 前提コードは不要。
- `ToolFunctionDefinition.Parameters` は `json.RawMessage`。
  - 生成時は `providers.MustMarshalParameters(...)` か `SetParametersMap(...)` を利用。
  - `map[string]any` を直接代入しない。
- `MemoryStore` はキャッシュ化済み。
  - `GetPlanTaskName` / `GetPlanWorkDir` / `GetMemoryContext` を優先して利用し、`ReadLongTerm()` 直叩きは最小化する。

SubagentManager を Container ベースの Orchestrator に進化させる。
セッション管理の内部実装 (TASKS-3) には依存しない — 現行の SessionManager 上で動作させ、後から SessionStore に差し替える。

## 設計コンテキスト

### アーキテクチャ

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

### escalation chain

```
subagent (clarifying) → question → conductor
  → conductor が答えられる: inCh に回答
  → conductor もわからない: message tool で human に投げ、回答を転送
```

### Presets (5種)

| preset | 性格 | write | exec | search | spawn |
|---|---|---|---|---|---|
| `scout` | Exploratory | x | x | o | x |
| `analyst` | Exploratory | x | go test/vet, git log/diff, grep | o | x |
| `coder` | Deliberate | o sandbox | test/lint/fmt 系 | o | x |
| `worker` | Deliberate | o sandbox | build/package manager 系 | o | x |
| `coordinator` | Deliberate | o sandbox | go/pnpm/bun/curl 系 | o | scout-worker のみ |

**性格の分類:**
- **Exploratory** (scout/analyst): open-ended、見てきて報告。clarifying フェーズなし。
- **Deliberate** (coder/worker/coordinator): 成果物を作る。clarifying フェーズあり。

**npm は全 preset で禁止** (git worktree に node_modules が生じるため)。pnpm/bun は可。
**websearch/webfetch は全 preset で許可** (read-only のため)。

### exec allowlist regex

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

---

## タスク一覧

### 1. SubagentContainer 実装

goroutine + channel でサブエージェントのライフサイクルを表現。

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

- spawn (async): outCh を返して即リターン
- subagent (sync): その場で result を待つ
- goroutine 終了時に `defer orchestrator.active.Delete(id)` + `defer close(outCh)` で自動 GC

**対象ファイル:** `pkg/tools/container.go` (新規)

---

### 2. Orchestrator 実装

SubagentManager を置き換える上位構造。

- Container の生成・管理
- escalation chain の実装 (subagent → conductor → human)
- spawn/subagent の使い分けロジック

**対象ファイル:** `pkg/tools/orchestrator.go` (新規)

---

### 3. SubagentEnvironment (Context Injection)

conductor が subagent に渡すコンテキストの構造化。

```go
type SubagentEnvironment struct {
    Workspace    string
    WorktreeDir  string
    PlanTask     string   // MEMORY.md > Task: から自動抽出
    PlanContext  string   // MEMORY.md ## Context から自動抽出
    Commands     string   // MEMORY.md ## Commands から自動抽出
    CurrentPhase string
    Background   string   // conductor が明示的に追加
    Constraints  string
    ContextFiles []string
}
```

`inject_plan_context: true` で MEMORY.md からの自動注入を有効化。

**対象ファイル:** `pkg/tools/container.go` または `pkg/tools/environment.go` (新規)

---

### 4. SandboxConfig enforcement

`ToolRegistry.Execute()` の入口で一括 enforcement。subagent は透過的に sandboxed になる。

```go
type SandboxConfig struct {
    Preset           string
    WriteRoot        string
    AllowedTools     map[string]bool
    ExecPolicy       *ExecPolicy
    SpawnablePresets []string
}
```

workDir = worktreeDir として設定し、AI が隔離に気づかず振る舞う透過的隔離を実現。

**対象ファイル:** `pkg/tools/sandbox.go` (既存拡張), `pkg/tools/registry.go`

---

### 5. Preset enforcement

preset 定義と exec allowlist regex を SandboxConfig 経由で enforce。
`pkg/tools/sandbox.go` に既存の定義があるが、ToolRegistry.Execute() での一括 enforcement がまだ。

---

### 6. Subagent Plan Mode (in-memory)

Deliberate preset 用のミニ plan mode。MEMORY.md には触れない。

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
    Goal     string
    Approach []string
    QA       []QAItem
    mu       sync.Mutex
}
```

SubagentContainer がフィールドとして保持。goroutine 終了とともに消える。

**System prompt 使い分け:**
- Deliberate: clarifying → review → executing の3段階
- Exploratory: 即座に探索開始、判断は自律的に

---

### 7. MEMORY.md Orchestration Section

conductor の guidance に Orchestration セクションの使い方を追記。

```markdown
## Orchestration

### Delegated
- coder-1 (coder): rate limiter 実装 → Phase 2 Step 1

### Findings
- pkg/auth は middleware パターン (scout-1)

### Decisions
- auth: JWT を選択 (外部依存なし、scout-2 推奨)
```

conductor は spawn 後に Delegated に記録、結果受信後に Findings に追記、方向選択時に Decisions に記録。

**対象ファイル:** `pkg/agent/context.go` (guidance 追記)

---

## 完了基準

- `go test ./...` 全パス
- scout/analyst/coder preset で spawn → 結果受信 → conductor 応答の E2E フロー動作
- Deliberate preset の clarifying → review → executing フロー動作
- SandboxConfig による exec 制限が全 preset で正しく enforcement
- escalation chain (subagent → conductor → human) が動作



