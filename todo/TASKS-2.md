# TASKS-2: Subagent Orchestration (Container Model)

SubagentManager を Container ベースの Orchestrator に進化させる。
escalation chain（質問→回答）と Deliberate preset の plan mode を追加。

## 既存実装の現状

TASKS-2 の下地はかなり実装済み。以下を前提として差分のみ実装する。

### 実装済み ✅

| 要素 | ファイル | 状態 |
|---|---|---|
| **SubagentManager** | `pkg/tools/subagent.go` | Spawn/SubagentTool、タスク管理、goroutine lifecycle、WaitAll |
| **SpawnTool / SubagentTool** | `pkg/tools/spawn.go`, `subagent.go` | async/blocking 両パターン |
| **Preset 定義** | `pkg/tools/sandbox.go` | 5 preset、`SandboxConfig`、`ExecPolicy`、`AllowedToolsForPreset()` |
| **Registry 隔離** | `subagent.go` `buildPresetRegistry()` | preset 毎に専用 ToolRegistry + ExecTool インスタンス |
| **orch.Broadcaster** | `pkg/orch/broadcaster.go` | Spawn/StateChange/Conversation/GC イベント配信 |
| **SessionRecorder** | `pkg/tools/session_recorder.go` + `pkg/agent/session_recorder.go` | Fork/Turn/Completion/Report の DAG 記録 |
| **SubagentEnvironment 型** | `pkg/tools/sandbox.go` | Workspace/WorktreeDir/Background/Constraints/ContextFiles |
| **Async Callback** | `pkg/agent/loop.go` `processRequest` | spawn 完了 → MessageBus → conductor に結果注入 |
| **Orchestration Nudge** | `pkg/agent/loop.go` `buildOrchReminder()` | plan 実行中に spawn/subagent 使用を促すリマインダ |

### 未実装 ❌ → TASKS-2 スコープ

| # | 要素 | 概要 |
|---|---|---|
| 1 | **ContainerMessage channel** | subagent→conductor の question/status/result 双方向通信 |
| 2 | **Escalation chain** | subagent question → conductor 回答 or → human escalate |
| 3 | **Deliberate Plan Mode** | coder/worker/coordinator の clarifying→review→executing 状態遷移 |
| 4 | **SubagentEnvironment injection** | MEMORY.md からの自動コンテキスト注入 |
| 5 | **MEMORY.md Orchestration section** | conductor guidance に delegated/findings/decisions 追記 |

---

## 設計コンテキスト

### 現行アーキテクチャ（spawn 非同期フロー）

```
Conductor (main loop)
  │ SpawnTool.Execute({"task":..., "preset":"scout"})
  │
  ├─ manager.Spawn() → goroutine 起動 → AsyncResult 即返却
  │
  │  [goroutine]
  │  ├─ buildPresetRegistry(preset)  ← 隔離された ToolRegistry
  │  ├─ RunToolLoop(messages, tools)  ← LLM + tool 実行ループ
  │  ├─ recorder.RecordCompletion()
  │  ├─ reporter.ReportGC()
  │  └─ callback() → msgBus.PublishInbound() → conductor に結果到着
  │
  ▼ conductor 次イテレーションで結果を受信
```

### Container 追加後のアーキテクチャ

```
Conductor (main loop)
  │ SpawnTool.Execute({"task":..., "preset":"coder"})
  │
  ├─ manager.Spawn() → goroutine 起動 → AsyncResult 即返却
  │
  │  [goroutine: Container]
  │  ├─ buildPresetRegistry(preset)
  │  ├─ RunToolLoop() — subagent が ask_conductor tool を呼ぶ
  │  │   └─ AskConductorTool.Execute()
  │  │       ├─ outCh ← ContainerMessage{Type:"question", Content:...}
  │  │       ├─ ← inCh (ブロック: conductor の回答を待つ)
  │  │       └─ return ToolResult{ForLLM: answer}
  │  │
  │  ├─ recorder.RecordCompletion()
  │  └─ callback() → conductor に結果到着
  │
  ▼ conductor は outCh からの question を受信し回答 or escalate
```

### escalation chain

```
subagent (clarifying) → outCh: question → conductor 受信
  → conductor が答えられる → inCh: answer → subagent 再開
  → conductor もわからない → message tool で human に投げ → 回答を転送 → inCh
```

### Presets (5種、定義済み)

| preset | 性格 | write | exec | search | spawn |
|---|---|---|---|---|---|
| `scout` | Exploratory | x | x | o | x |
| `analyst` | Exploratory | x | go test/vet, git log/diff, grep | o | x |
| `coder` | Deliberate | o sandbox | test/lint/fmt 系 | o | x |
| `worker` | Deliberate | o sandbox | build/package manager 系 | o | x |
| `coordinator` | Deliberate | o sandbox | go/pnpm/bun/curl 系 | o | scout-worker のみ |

---

## Phase 1: Container + Escalation

### Task 1: ContainerMessage channel 追加

**目的:** subagent goroutine と conductor 間の双方向通信を既存の SubagentManager に追加。
新規ファイルは作らず SubagentTask を拡張する。

**対象ファイル:** `pkg/tools/subagent.go`

**変更内容:**

1. SubagentTask にチャネルフィールド追加:

```go
// 既存の SubagentTask に追加
type SubagentTask struct {
    // ... existing fields ...

    // Container channels — nil for Exploratory presets
    inCh  chan string            // conductor → subagent (回答)
    outCh chan ContainerMessage  // subagent → conductor (質問・結果)
}

type ContainerMessage struct {
    Type    string // "question" | "status"
    Content string
    TaskID  string // 送信元タスクID
}
```

2. `Spawn()` で Deliberate preset の場合のみチャネル生成:

```go
func (sm *SubagentManager) Spawn(...) (string, error) {
    // ... existing code ...
    if isDeliberatePreset(preset) {
        task.inCh = make(chan string, 1)
        task.outCh = make(chan ContainerMessage, 4)
    }
    // ...
}
```

3. `PendingQuestions()` メソッド追加 — conductor が polling で question を回収:

```go
// PendingQuestions returns all pending questions from active subagents.
// Non-blocking: returns nil if no questions.
func (sm *SubagentManager) PendingQuestions() []ContainerMessage {
    sm.mu.RLock()
    defer sm.mu.RUnlock()
    var questions []ContainerMessage
    for _, t := range sm.tasks {
        if t.outCh == nil { continue }
        select {
        case msg := <-t.outCh:
            questions = append(questions, msg)
        default:
        }
    }
    return questions
}
```

4. `AnswerQuestion()` メソッド追加:

```go
func (sm *SubagentManager) AnswerQuestion(taskID, answer string) error {
    sm.mu.RLock()
    t, ok := sm.tasks[taskID]
    sm.mu.RUnlock()
    if !ok { return fmt.Errorf("task %s not found", taskID) }
    if t.inCh == nil { return fmt.Errorf("task %s has no question channel", taskID) }
    select {
    case t.inCh <- answer:
        return nil
    default:
        return fmt.Errorf("task %s answer channel full", taskID)
    }
}
```

**テスト:** `pkg/tools/subagent_container_test.go`
- Deliberate preset → outCh/inCh 生成確認
- Exploratory preset → outCh/inCh = nil 確認
- PendingQuestions の non-blocking 動作
- AnswerQuestion の正常系/異常系

---

### Task 2: AskConductorTool

**目的:** subagent goroutine 内で conductor に質問を送り、回答をブロック待ちする tool。

**対象ファイル:** `pkg/tools/ask_conductor.go` (新規)

**実装:**

```go
type AskConductorTool struct {
    taskID string
    outCh  chan<- ContainerMessage
    inCh   <-chan string
}

func NewAskConductorTool(taskID string, outCh chan<- ContainerMessage, inCh <-chan string) *AskConductorTool

func (t *AskConductorTool) Name() string { return "ask_conductor" }

func (t *AskConductorTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
    question := args["question"].(string)

    // 1. Send question to conductor
    select {
    case t.outCh <- ContainerMessage{Type: "question", Content: question, TaskID: t.taskID}:
    case <-ctx.Done():
        return ErrorResult("canceled while sending question")
    }

    // 2. Block waiting for answer (with context cancellation)
    select {
    case answer := <-t.inCh:
        return SuccessResult(answer)
    case <-ctx.Done():
        return ErrorResult("canceled while waiting for answer")
    }
}
```

**Parameters:**

```go
{
    "type": "object",
    "properties": {
        "question": {
            "type": "string",
            "description": "A clarifying question for the conductor about the task requirements, constraints, or approach."
        }
    },
    "required": ["question"]
}
```

**Registry 登録:** `buildPresetRegistry()` で Deliberate preset のみ登録:

```go
// In buildPresetRegistry():
if task.inCh != nil && task.outCh != nil {
    reg.Register(NewAskConductorTool(task.ID, task.outCh, task.inCh))
}
```

**テスト:** `pkg/tools/ask_conductor_test.go`
- question 送信 → answer 受信の正常フロー
- context キャンセルでの中断
- channel 閉鎖時の graceful error

---

### Task 3: Conductor 側の question 処理

**目的:** conductor の main loop が subagent question を検出し、LLM に判断させる。

**対象ファイル:** `pkg/agent/loop.go`

**方式:** processRequest のイテレーションごとに pending questions をチェック。
questions があれば system message として LLM に注入。

**変更内容:**

1. `runAgentLoop()` のイテレーション先頭に question チェック追加:

```go
// In runAgentLoop(), after getting LLM response:
if agent.SubagentMgr != nil {
    questions := agent.SubagentMgr.PendingQuestions()
    for _, q := range questions {
        // LLM の次の応答で answer_subagent tool call を期待
        messages = append(messages, providers.Message{
            Role: "user",
            Content: fmt.Sprintf("[Subagent %s asks]: %s\n\nRespond using the answer_subagent tool.",
                q.TaskID, q.Content),
        })
    }
}
```

2. `AnswerSubagentTool` 追加 (`pkg/tools/answer_subagent.go`、新規):

```go
type AnswerSubagentTool struct {
    manager *SubagentManager
}

func (t *AnswerSubagentTool) Name() string { return "answer_subagent" }

func (t *AnswerSubagentTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
    taskID := args["task_id"].(string)
    answer := args["answer"].(string)
    if err := t.manager.AnswerQuestion(taskID, answer); err != nil {
        return ErrorResult(err.Error())
    }
    return SuccessResult("Answer delivered to " + taskID)
}
```

**Parameters:**

```go
{
    "type": "object",
    "properties": {
        "task_id": {"type": "string", "description": "The subagent task ID that asked the question"},
        "answer":  {"type": "string", "description": "Your answer to the subagent's question"}
    },
    "required": ["task_id", "answer"]
}
```

3. conductor の ToolRegistry に `answer_subagent` 登録:

```go
// In registerSharedTools(), after creating SubagentManager:
agent.Tools.Register(tools.NewAnswerSubagentTool(subagentManager))
```

**escalation (conductor → human):**
conductor LLM が answer_subagent で答えられないと判断した場合:
- message tool で human に質問を転送
- human 回答後に conductor が answer_subagent を呼ぶ
- **追加実装不要** — 既存の message tool + LLM 判断で自然に発生

**テスト:** `pkg/agent/loop_escalation_test.go`
- question 注入 → LLM が answer_subagent を呼ぶフロー（mock LLM）
- 複数 subagent から同時に question が来るケース

---

## Phase 2: Deliberate Plan Mode

### Task 4: SubagentPlanState

**目的:** Deliberate preset (coder/worker/coordinator) の subagent に clarifying→review→executing の状態遷移を追加。

**対象ファイル:** `pkg/tools/subagent.go` (SubagentTask に追加)

**変更内容:**

1. 状態定義:

```go
type SubagentPlanState int

const (
    PlanStateNone       SubagentPlanState = iota // Exploratory — 即実行
    PlanStateClarifying                          // 目的・制約を確認中
    PlanStateReview                              // conductor の承認待ち
    PlanStateExecuting                           // 実行中
    PlanStateCompleted                           // 完了
)

// SubagentTask に追加
type SubagentTask struct {
    // ... existing ...
    PlanState SubagentPlanState
    PlanGoal  string   // clarifying で確定した目標
    PlanSteps []string // review で提出したステップ
}
```

2. `runTask()` の Deliberate preset フロー変更:

```go
func (sm *SubagentManager) runTask(ctx context.Context, task *SubagentTask, preset string, callback AsyncCallback) {
    p := Preset(preset)

    if isDeliberatePreset(p) {
        // Phase A: Clarifying — ask_conductor で不明点を解消
        task.PlanState = PlanStateClarifying
        clarifyResult := sm.runClarifyingPhase(ctx, task, p)
        if ctx.Err() != nil { return }

        // Phase B: Review — conductor に計画を提出、承認待ち
        task.PlanState = PlanStateReview
        approved := sm.submitPlanForReview(ctx, task, clarifyResult)
        if !approved || ctx.Err() != nil { return }

        // Phase C: Executing — 承認済み計画を実行
        task.PlanState = PlanStateExecuting
        sm.runExecutingPhase(ctx, task, p, callback)
    } else {
        // Exploratory — 既存フロー（即実行）
        sm.runExploratoryTask(ctx, task, p, callback)
    }
}
```

3. 各フェーズの system prompt テンプレート:

```go
func clarifyingSystemPrompt(task, preset string) string {
    return fmt.Sprintf(`You are a %s subagent. Your task:
%s

PHASE: CLARIFYING
You must understand the task fully before executing.
- If anything is unclear, use ask_conductor to ask questions.
- When you are confident you understand the task, call submit_plan with your goal and approach.
- Do NOT write code or make changes yet.`, preset, task)
}

func executingSystemPrompt(task, preset, plan string) string {
    return fmt.Sprintf(`You are a %s subagent. Your task:
%s

APPROVED PLAN:
%s

PHASE: EXECUTING
Execute the approved plan step by step. You may use your tools to implement.
Do not deviate from the plan without asking via ask_conductor.`, preset, task, plan)
}
```

**テスト:** `pkg/tools/subagent_plan_test.go`
- Deliberate preset → clarifying→review→executing 遷移
- Exploratory preset → PlanStateNone のまま即実行
- clarifying 中に context cancel → graceful exit

---

### Task 5: SubmitPlanTool + review フロー

**目的:** subagent が clarifying 完了後に計画を conductor に提出する tool。

**対象ファイル:** `pkg/tools/submit_plan.go` (新規)

**実装:**

```go
type SubmitPlanTool struct {
    taskID string
    outCh  chan<- ContainerMessage
    inCh   <-chan string
}

func (t *SubmitPlanTool) Name() string { return "submit_plan" }

func (t *SubmitPlanTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
    goal := args["goal"].(string)
    steps := args["steps"]  // []any → []string

    planText := fmt.Sprintf("Goal: %s\nSteps:\n", goal)
    for i, s := range steps.([]any) {
        planText += fmt.Sprintf("  %d. %s\n", i+1, s)
    }

    // Send plan to conductor for review
    select {
    case t.outCh <- ContainerMessage{Type: "plan_review", Content: planText, TaskID: t.taskID}:
    case <-ctx.Done():
        return ErrorResult("canceled")
    }

    // Wait for approval/rejection
    select {
    case response := <-t.inCh:
        if response == "approved" || strings.HasPrefix(response, "approved") {
            return SuccessResult("Plan approved. Proceed with execution.")
        }
        return SuccessResult("Plan rejected: " + response + "\nRevise your plan and resubmit.")
    case <-ctx.Done():
        return ErrorResult("canceled while waiting for review")
    }
}
```

**Parameters:**

```go
{
    "type": "object",
    "properties": {
        "goal":  {"type": "string", "description": "One-sentence goal of the task"},
        "steps": {"type": "array", "items": {"type": "string"}, "description": "Ordered implementation steps"}
    },
    "required": ["goal", "steps"]
}
```

**Conductor 側:** `review_subagent_plan` tool 追加 (`pkg/tools/answer_subagent.go` に同居):

```go
type ReviewSubagentPlanTool struct {
    manager *SubagentManager
}

func (t *ReviewSubagentPlanTool) Name() string { return "review_subagent_plan" }

func (t *ReviewSubagentPlanTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
    taskID := args["task_id"].(string)
    decision := args["decision"].(string)  // "approved" or "rejected: reason"
    if err := t.manager.AnswerQuestion(taskID, decision); err != nil {
        return ErrorResult(err.Error())
    }
    return SuccessResult("Review delivered to " + taskID)
}
```

**Question injection:** conductor の LLM には plan_review も question と同じ経路で注入:

```go
// PendingQuestions already returns plan_review messages
// Conductor LLM sees:
// "[Subagent coder-1 submits plan for review]:
//   Goal: ...
//   Steps: ...
//  Use review_subagent_plan to approve or reject."
```

**テスト:** `pkg/tools/submit_plan_test.go`
- submit → approved フロー
- submit → rejected → 再 submit → approved フロー
- context cancel 中断

---

## Phase 3: Context Injection + Guidance

### Task 6: SubagentEnvironment injection

**目的:** `SubagentEnvironment` (既存型) に MEMORY.md からの自動コンテキスト注入を実装。

**対象ファイル:** `pkg/tools/subagent.go` (runTask 内)

**変更内容:**

1. `extractPlanContext()` ヘルパー追加:

```go
// extractPlanContext reads MEMORY.md and extracts Task/Context/Commands sections.
func extractPlanContext(workspace string) (task, context, commands string) {
    data, err := os.ReadFile(filepath.Join(workspace, "MEMORY.md"))
    if err != nil { return }
    // Parse markdown sections:
    // - "Task:" or "# Task" line → task
    // - "## Context" section → context
    // - "## Commands" section → commands
    // Simple line-based parser, no external deps
    return
}
```

2. `runTask()` で Environment を system prompt に注入:

```go
func (sm *SubagentManager) runTask(...) {
    // ... existing ...
    env := SubagentEnvironment{
        Workspace:   sm.workspace,
        WorktreeDir: sm.workspace, // TODO: worktree 有効時は worktree path
    }

    // Auto-inject from MEMORY.md if available
    env.PlanTask, env.PlanContext, env.Commands = extractPlanContext(sm.workspace)

    // Build system prompt with environment
    systemPrompt := buildSubagentSystemPrompt(preset, task.Task, env)
    // ...
}
```

3. `buildSubagentSystemPrompt()` に environment セクション追加:

```go
func buildSubagentSystemPrompt(preset, task string, env SubagentEnvironment) string {
    var sb strings.Builder
    // ... existing preset-based prompt ...

    if env.PlanTask != "" || env.PlanContext != "" {
        sb.WriteString("\n## Project Context\n")
        if env.PlanTask != "" {
            sb.WriteString("Current task: " + env.PlanTask + "\n")
        }
        if env.PlanContext != "" {
            sb.WriteString(env.PlanContext + "\n")
        }
    }
    if env.Commands != "" {
        sb.WriteString("\n## Build Commands\n" + env.Commands + "\n")
    }
    if env.Constraints != "" {
        sb.WriteString("\n## Constraints\n" + env.Constraints + "\n")
    }
    return sb.String()
}
```

**テスト:** `pkg/tools/subagent_env_test.go`
- MEMORY.md ありの場合にセクション抽出確認
- MEMORY.md なし → 空文字列、エラーなし

---

### Task 7: MEMORY.md Orchestration section guidance

**目的:** conductor が spawn/結果受信/方向決定時に MEMORY.md の Orchestration セクションを更新するよう guidance を追加。

**対象ファイル:** `pkg/agent/context.go`

**変更内容:**

既存の `buildOrchGuidance()` に Orchestration セクションの使い方を追記:

```go
func buildOrchGuidance() string {
    return `## Orchestration Memory

When orchestrating subagents, maintain an "## Orchestration" section in MEMORY.md:

### Delegated
Track active delegations:
- <label> (<preset>): <task summary> → <plan phase/step>

### Findings
Record discoveries from subagent reports:
- <finding> (<source subagent>)

### Decisions
Record architectural/design decisions:
- <decision> (<rationale>, <recommending subagent>)

Update this section after each spawn and after receiving each subagent report.`
}
```

**テスト:** 既存の context_test.go で guidance 文字列に "Orchestration" が含まれることを確認。

---

## ファイル一覧

| ファイル | Action | Phase |
|---|---|---|
| `pkg/tools/subagent.go` | 拡張: ContainerMessage, inCh/outCh, PendingQuestions, AnswerQuestion, PlanState | 1, 2 |
| `pkg/tools/ask_conductor.go` | 新規: AskConductorTool | 1 |
| `pkg/tools/answer_subagent.go` | 新規: AnswerSubagentTool, ReviewSubagentPlanTool | 1, 2 |
| `pkg/tools/submit_plan.go` | 新規: SubmitPlanTool | 2 |
| `pkg/agent/loop.go` | 拡張: question polling + 注入, answer/review tool 登録 | 1, 2 |
| `pkg/agent/context.go` | 拡張: Orchestration guidance | 3 |
| `pkg/tools/subagent_container_test.go` | 新規 | 1 |
| `pkg/tools/ask_conductor_test.go` | 新規 | 1 |
| `pkg/tools/subagent_plan_test.go` | 新規 | 2 |
| `pkg/tools/submit_plan_test.go` | 新規 | 2 |
| `pkg/tools/subagent_env_test.go` | 新規 | 3 |

## Phase 間の依存

```
Phase 1: Container + Escalation (Task 1-3)
    ↓ (inCh/outCh が前提)
Phase 2: Deliberate Plan Mode (Task 4-5)
    ↓ (subagent の system prompt 拡張が前提)
Phase 3: Context Injection + Guidance (Task 6-7)  ← 独立して先行も可
```

## 完了基準

- `go test ./...` 全パス
- Exploratory preset (scout/analyst): 既存フロー変更なし、即実行
- Deliberate preset (coder/worker): clarifying → review → executing の 3 段階動作
- `ask_conductor` → conductor LLM 回答 → subagent 再開 のラウンドトリップ
- conductor が回答不可 → message tool で human escalate → 回答転送
- MEMORY.md Orchestration セクションが conductor guidance に含まれる
- 既存の spawn/subagent E2E フロー（Exploratory）に regression なし
