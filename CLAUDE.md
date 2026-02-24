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

## Security TODOs

- ~~**Log Fields masking**~~: Done. `SanitizeFields()` in `pkg/logger/logger.go` masks keys matching `token`, `key`, `secret`, `password`, `authorization`, `credential`. Applied in `RecentLogs()` and `wsLogs()` stream.

## Known Gaps

- **Mini App log viewer has no frontend tests**: `renderLogs()` in `pkg/miniapp/static/index.html` is inline vanilla JS with no unit/E2E test coverage. Backend (Go) tests cover `RecentLogs`, `SanitizeFields`, and JSON serialization, but nothing verifies the JS rendering. This allowed the Fields display bug (fields sent but not rendered) to ship undetected.
- **No human intervention for heartbeat worktrees**: Heartbeat sessions create git worktrees (`.picoclaw/worktrees/heartbeat-YYYYMMDD/`) but there is no CLI or Mini App command to list, inspect, or manually dispose them. Need a `/plan worktrees` command (or similar) that shows active worktrees with branch/commit info and allows manual merge/dispose. `PruneOrphaned` on startup only removes directories without auto-committing first, so uncommitted changes in orphaned worktrees are silently lost.

## Memory Optimization Candidates

> Reviewed 2026-02-24 on branch `memory-optimization-review`. False positives included intentionally.
> Legend: 🔴 High / 🟡 Medium / 🟢 Low

### A. ホットパスでの文字列結合 (strings.Builder 未使用)

| 重要度 | ファイル | 行 | 内容 |
|--------|----------|----|------|
| 🔴 | `pkg/tools/web.go` | 73-85 | `BraveSearchProvider.Search()` — slice append + Join を Builder に |
| 🔴 | `pkg/tools/web.go` | 155-167 | `TavilySearchProvider.Search()` — 同上パターン |
| 🔴 | `pkg/tools/web.go` | 211-254 | `DuckDuckGoSearchProvider.extractResults()` — ループ内 append+Join |
| 🔴 | `pkg/tools/web.go` | 592-617 | `WebFetchTool.extractText()` — cleanLines を Builder で |
| 🔴 | `pkg/skills/loader.go` | 234-250 | `BuildSkillsSummary()` — `[]string` + Join で XML 組み立て (要素数×アロケーション) → Builder へ |
| 🔴 | `pkg/channels/telegram.go` | 789-806 | `extractCodeBlocks()` — codes スライス無容量 + ReplaceAllStringFunc の fmt.Sprintf |
| 🔴 | `pkg/channels/telegram.go` | 813-830 | `extractInlineCodes()` — 同上パターン |
| 🟡 | `pkg/agent/context.go` | 247 | `BuildSystemPrompt()` — `systemPrompt +=` で連結 → Builder へ |
| 🟡 | `pkg/logger/logger.go` | 241-246 | `formatFields()` — parts slice + Join → Builder へ |
| 🟡 | `pkg/skills/loader.go` | 217-225 | `LoadSkillsForContext()` — parts + Join → Builder へ |
| 🟡 | `pkg/channels/discord.go` | 162-168 | `appendContent()` — `+` 演算子で結合 → Builder へ |
| 🟡 | `pkg/channels/slack.go` | 234-272 | `handleMessageEvent()` — ループ内文字列連結 → Builder へ |
| 🟢 | `pkg/git/worktree.go` | 61-63 | `SanitizeBranchName()` — `strings.ReplaceAll` ループ |

### B. スライスの事前容量確保漏れ

| 重要度 | ファイル | 行 | 内容 |
|--------|----------|----|------|
| 🟡 | `pkg/tools/toolloop.go` | 87-96 | `RunToolLoop()` — normalizedToolCalls / toolNames を make([]T, 0, 推定値) に |
| 🟡 | `pkg/config/config.go` | 628 | `findMatches()` — `var matches []ModelConfig` → 容量ヒントを付与 |
| 🟡 | `pkg/config/migration.go` | 48 | `ConvertProvidersToModelList()` — result に make([]ModelConfig, 0, 20) |
| 🟡 | `pkg/skills/registry.go` | 183 | `SearchAll()` — merged に make([]SearchResult, 0, len(regs)*limit) |
| 🟡 | `pkg/skills/loader.go` | 73 | `ListSkills()` — skills に make([]SkillInfo, 0, 20) 程度 |
| 🟡 | `pkg/channels/telegram.go` | 832-861 | `extractMarkdownTables()` — out / tables スライスに容量ヒント |
| 🟢 | `pkg/skills/search_cache.go` | 42-43 | `NewSearchCache()` — entries map / order slice に maxEntries をヒント |
| 🟢 | `pkg/agent/session_tracker.go` | 121 | `ListActive()` — result スライスに容量ヒント |

### C. 不要な []byte ↔ string 変換 / 重複変換

| 重要度 | ファイル | 行 | 内容 |
|--------|----------|----|------|
| 🔴 | `pkg/channels/telegram.go` | 1071-1111 | `wrapByDisplayWidth()` — ループ内で `string(r)` (rune→string) を毎イテレーション実行 |
| 🟡 | `pkg/tools/web.go` | 545-562 | `WebFetchTool.Execute()` — `string(body)` を最大5回呼び出し → 1回に集約 |
| 🟡 | `pkg/tools/web.go` | 289 | `PerplexitySearchProvider.Search()` — `string(payloadBytes)` + `strings.NewReader` → `bytes.NewReader` を直接使用 |
| 🟢 | `pkg/utils/string.go` | 50 | `wrapLine()` — ASCII 主体なのに `[]rune(line)` |
| 🟢 | `pkg/utils/string.go` | 100 | `Truncate()` — 長さ確認前に `[]rune(s)` |
| 🟢 | `pkg/git/worktree.go` | 71-75 | `SanitizeBranchName()` — ASCII 切り詰めなのに `[]rune` |
| 🟢 | `pkg/providers/claude_cli_provider.go` | 133 | `string(paramsJSON)` 後に Builder へ書き込み → bytes.Write |

### D. JSON Marshal/Unmarshal の重複・ホットパス

| 重要度 | ファイル | 行 | 内容 |
|--------|----------|----|------|
| 🔴 | `pkg/providers/openai_compat/provider.go` | 274, 362, 621 | ストリーミングループ内でツール引数を複数回 Unmarshal |
| 🟡 | `pkg/providers/anthropic/provider.go` | 213 | `json.Unmarshal(tu.Input, &args)` — map にサイズヒントなし |
| 🟡 | `pkg/providers/codex_cli_provider.go` | 154-155 | ツール定義ループ内で `json.Marshal(parameters)` |

### E. 大きな struct の値渡し / ループ内コピー

| 重要度 | ファイル | 行 | 内容 |
|--------|----------|----|------|
| 🔴 | `pkg/agent/session_tracker.go` | 125 | `ListActive()` — `*entry` を値コピーして append → ポインタ slice に |
| 🟡 | `pkg/session/manager.go` | 98-100 | `GetHistory()` — messages 全コピー (スレッド安全のため意図的。COW 検討) |
| 🟡 | `pkg/session/manager.go` | 187-188 | `Save()` — messages 全コピー (同上) |
| 🟡 | `pkg/skills/registry.go` | 132-133 | `SearchAll()` — `[]SkillRegistry` を全コピーしてからロック解除 |
| 🟢 | `pkg/logger/logger.go` | 88-92 | `recent()` — LogEntry を値コピーして返却 → ポインタ slice 検討 |

### F. sync.Pool / バッファ再利用の検討

| 重要度 | ファイル | 行 | 内容 |
|--------|----------|----|------|
| 🟡 | `pkg/tools/web.go` | 592-617 | `extractText()` — HTML 解析用 Builder を sync.Pool で再利用 |
| 🟡 | `pkg/channels/telegram.go` | 757-861 | Markdown 変換系関数群 — メッセージ毎に多数のバッファを生成 → Pool 化 |
| 🟢 | `pkg/utils/download.go` | 43 | `DownloadToFile()` — エラー読み取り用 `make([]byte, 512)` → 共有バッファ |

### G. LRU / アルゴリズムレベルの最適化

| 重要度 | ファイル | 行 | 内容 |
|--------|----------|----|------|
| 🟡 | `pkg/skills/search_cache.go` | 161 | `moveToEndLocked()` — slice slicing で O(n) LRU 更新 → doubly-linked list で O(1) に |

### H. パッケージレベル変数化 (関数呼び出しのたびに再生成)

| 重要度 | ファイル | 行 | 内容 |
|--------|----------|----|------|
| 🟢 | `pkg/utils/media.go` | 18-19 | `IsAudioFile()` — `audioExtensions` / `audioTypes` スライスを毎回生成 → var に |
| 🟢 | `pkg/skills/clawhub_registry.go` | 114 | `fmt.Sprintf("%d", limit)` → `strconv.Itoa(limit)` |

### H. 重複 strings.Split / Join (memory.go)

| 重要度 | ファイル | 行 | 内容 |
|--------|----------|----|------|
| 🟡 | `pkg/agent/memory.go` | 233, 285, 352, 381 | `extractPhaseContent` / `GetPlanPhases` / `MarkStep` / `AddStep` — 同一 MEMORY.md を関数毎に Split → 統合 or キャッシュ |

---

### 設計レベルの根本原因 — 「見落とし」ではなく「構造的に不可避」な問題

個別の最適化候補の多くは、書いた人の不注意ではなく、**設計上の選択が特定のアロケーションパターンを必然的に引き起こしている**ことが読み取れる。以下はその根本原因を設計レベルで整理したもの。

#### D-1. MemoryStore が「ファイル = 正」の設計で、パース済み表現をキャッシュできない

`MemoryStore` の各メソッドはほぼ全員が `ReadLongTerm()` → `strings.Split()` → scan → `strings.Join()` を独立して実行する。`GetMemoryContext()` を1回呼ぶだけで、内部で `ReadLongTerm()` が3回以上呼ばれる連鎖が起きる。

```
GetMemoryContext()
  └─ HasActivePlan()    → ReadLongTerm() → ファイルI/O
  └─ GetPlanStatus()   → ReadLongTerm() → ファイルI/O
  └─ GetPlanContext()  → ReadLongTerm() → ファイルI/O
       └─ GetCurrentPhase() → ReadLongTerm() → ファイルI/O
       └─ GetTotalPhases()  → ReadLongTerm() → ファイルI/O
```

**なぜこうなったか**: MEMORY.md をユーザーが直接編集できる外部ファイルとして設計したため、「ファイルが常に最新の正」という前提が成立している。インメモリキャッシュを持つと外部編集が反映されなくなる恐れがあり、キャッシュを自然に導入できない。

**設計上の選択肢**: (a) `content` を引数として受け取る内部 pure function 群 + 高レベルメソッドだけが1回 ReadLongTerm() を呼ぶ、(b) ウォッチ付きキャッシュ (`fsnotify`)、(c) エージェントループ内で1ターンに1回だけ読む「ターンスコープキャッシュ」。

---

#### D-2. `FunctionCall.Arguments` が JSON 文字列のまま型として定義されている

```go
// protocoltypes/types.go
type FunctionCall struct {
    Name      string `json:"name"`
    Arguments string `json:"arguments"`  // ← ワイヤフォーマット (JSON文字列) をそのままドメイン型に
}
```

ツール引数はワイヤ上 `"arguments": "{\"key\":\"value\"}"` の形で届くが、この型定義はその文字列をそのまま保持する。使う側は毎回 `json.Unmarshal([]byte(tc.Function.Arguments), &args)` しなければならず、これがストリーミングループ内の重複 Unmarshal の根本原因になっている。

**対比**: `ToolCall.Arguments map[string]any json:"-"` というパース済みフィールドは存在するが、openai_compat の streaming path ではこの `map[string]any` フィールドではなく `Function.Arguments string` から直接読んでいる。両方のフィールドが中途半端に共存している。

---

#### D-3. `ToolFunctionDefinition.Parameters` が `map[string]any` で、シリアライズ済み形式を保持できない

```go
type ToolFunctionDefinition struct {
    Name        string         `json:"name"`
    Description string         `json:"description"`
    Parameters  map[string]any `json:"parameters"`  // ← プロバイダーへ送るたびに Marshal が必要
}
```

ツール定義はエージェント起動時に一度決まり、実行中は変化しない。しかし `map[string]any` として保持しているため、各プロバイダーへの送信のたびに `json.Marshal` → `string` 変換が発生する。`json.RawMessage` にしておけば「一度 marshal したバイト列をそのまま複数プロバイダーへ流す」設計が可能になる。

---

#### D-4. 検索プロバイダー群に共通フォーマット抽象がなく、同じ欠陥が3箇所に複製されている

`BraveSearchProvider`, `TavilySearchProvider`, `DuckDuckGoSearchProvider` は全て独立して「結果 → 文字列」の変換ロジックを実装している。共通の `ResultFormatter` インターフェースや `formatSearchResult(title, url, snippet string)` ヘルパーがないため、同じ `[]string + strings.Join` パターンが3箇所に独立してコピーされた。最適化漏れも3箇所に同時に発生する。

**設計の示唆**: プロバイダーの `Search()` 戻り値を `string` にせず構造体 (`[]SearchResult`) にして、フォーマットを呼び出し側に移譲する設計なら、フォーマットロジックは1箇所で済む。

---

#### D-5. `Session.Messages` が可変スライスで、読み取りに構造的な全コピーが必要

```go
type Session struct {
    Messages []providers.Message  // ← 可変。append で追記される
}

func (sm *SessionManager) GetHistory(key string) []providers.Message {
    history := make([]providers.Message, len(session.Messages))
    copy(history, session.Messages)  // ← 安全のために必須
    return history
}
```

`session.Messages` は `append` で追記される可変スライスで、外部から参照を渡すと内部状態が壊れるリスクがある。そのため `GetHistory()`, `Save()`, `SetHistory()` の全てでコピーが必要になる。コメントにも「to strictly isolate internal state from the caller's slice」と明記されており、これは意図的な設計だがコピーコストを構造的に固定している。

**代替設計**: メッセージログを append-only な不変構造 (`[]*Message` のリンクリストや、インデックスで管理するリングバッファ) にすれば、参照の共有が安全になりコピーを排除できる。

---

#### D-6. `MemoryStore` のメソッド境界が「ファイル操作単位」で切られており、呼び出し側が合成できない

```go
// 呼び出し側は content を持てないため、内部で毎回 ReadLongTerm() を呼ぶ
phases := ms.GetPlanPhases()       // ReadLongTerm() 内包
current := ms.GetCurrentPhase()    // ReadLongTerm() 内包
status := ms.GetPlanStatus()       // ReadLongTerm() 内包
```

各 public メソッドが「ファイルを読んでパースして1つの値を返す」単位で設計されているため、呼び出し側は複数の値が必要なときでもメソッドを複数回呼ぶしか選択肢がない。`content` を受け取る private 関数群 (`extractPhaseContent(content, phase)` など) は存在するが、public API からは使えない。

---

### コードのにおい — 見落としやすいパターン集

上記の個別発見を横断して見ると、このコードベースに繰り返し現れる**7つの構造的なにおい**がある。新しいコードを書くとき・レビューするときのチェックリストとして使う。

#### 1. 「先に集めてから結合」パターン (`[]string` + `strings.Join`)

```go
// においのある書き方
var parts []string
for _, x := range items {
    parts = append(parts, fmt.Sprintf("...%s...", x))
}
return strings.Join(parts, "\n")
```

`var parts []string` → ループ内 `append` → 最後に `strings.Join` という3ステップの流れ。見た目が整理されているため気づきにくいが、中間スライスと最終結合の2回アロケーションが発生する。`strings.Builder` に一本化すれば1回で済む。**web.go の検索プロバイダー4箇所、logger.go、skills/loader.go など計10箇所以上で観察された。**

#### 2. 「変換してから渡す」パターン ([]byte ↔ string の橋渡し)

```go
// においのある書き方
payload, _ := json.Marshal(body)
req, _ := http.NewRequest("POST", url, strings.NewReader(string(payload)))
//                                    ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
//                                    []byte → string → io.Reader と2段変換
```

`json.Marshal` は `[]byte` を返すのに、直後に `string()` へキャストして `strings.NewReader` に渡す。`bytes.NewReader(payload)` で変換ゼロで済む。**web.go の Perplexity プロバイダー、各 CLI プロバイダーで観察された。**

#### 3. 「ループ内で静的なものを毎回生成」パターン

```go
// においのある書き方
for _, tool := range tools {
    paramsJSON, _ := json.Marshal(tool.Parameters) // ← ループ内 Marshal
    prompt += fmt.Sprintf("...", string(paramsJSON))
}
```

ループ内で毎イテレーション行われる処理のうち、**入力が変わらないものが含まれていないか**を疑う。典型例：
- ループ内での `json.Marshal` (引数が定数的なとき)
- ループ内での `string(rune)` 変換 (1文字ずつ変換)
- ループ内でのスライス/マップリテラル生成

**telegram.go の `wrapByDisplayWidth`、openai_compat の streaming ループ、codex の tool 定義ループで観察された。**

#### 4. 「防衛的コピーが広すぎる」パターン (スレッド安全の過剰適用)

```go
// においのある書き方
func (m *Manager) GetHistory() []Message {
    m.mu.RLock()
    defer m.mu.RUnlock()
    result := make([]Message, len(m.messages))
    copy(result, m.messages)   // ← 全件コピーしてからロック解除
    return result
}
```

並行安全のため slice 全体を防衛的にコピーするのは正しいが、**コピー範囲が呼び出し側の実際の用途より広い**ことがある。読み取り専用なら `sync.RWMutex` + ポインタ返却 + immutable 制約、または Copy-on-Write で代替できる場合がある。**session/manager.go の GetHistory・Save で観察された。**

#### 5. 「ファイルを読むたびにパース」パターン (ステートレスな繰り返しパース)

```go
// においのある書き方
func GetPlanPhases(content string) []string {
    lines := strings.Split(content, "\n")   // ← 呼び出し毎にフルスキャン
    ...
}
func MarkStep(content, step string) string {
    lines := strings.Split(content, "\n")   // ← 同じ content を再度スキャン
    ...
}
```

同一のファイル内容を受け取る複数の関数がそれぞれ独立して `strings.Split` → スキャン → `strings.Join` している。呼び出し側でパース済み表現（行スライスなど）を保持して渡すか、パース結果をキャッシュする設計にすると複数回のアロケーションを削減できる。**memory.go の4関数で観察された。**

#### 6. 「`var x []T` から始まる容量なし append」パターン

```go
// においのある書き方
var result []ModelConfig          // cap=0 から開始
for _, p := range providers {
    result = append(result, ...)  // 倍々に再アロケーション
}
```

`var x []T` や `make([]T, 0)` で始まり、ループ内で `append` を重ねる。**ソースの長さが事前にわかっている場合**（別スライスの len、定数上限など）は `make([]T, 0, n)` で初期容量を与えれば再アロケーションをゼロにできる。見落とされやすい理由は「append は自動で伸びるから大丈夫」という習慣。**config/migration.go、skills/registry.go、skills/loader.go ほか6箇所で観察された。**

#### 7. 「Unicode 安全のための過剰な []rune 変換」パターン

```go
// においのある書き方
func Truncate(s string, max int) string {
    runes := []rune(s)       // ← 全文字を変換してから長さ確認
    if len(runes) <= max {
        return s
    }
    return string(runes[:max])
}
```

文字数を正しく数えるために `[]rune` へ変換するのは正しい。しかし **①変換前に `len(s)` で byte 長をチェックして早期 return できる**（ASCII なら byte 長 == rune 長）、**②実際の入力が ASCII 主体であれば `utf8.RuneCountInString` + `utf8.RuneError` チェックでアロケーションなしに処理できる**。`[]rune(s)` は文字列全体をヒープにコピーするため、長い文字列では無視できないコストになる。**utils/string.go の2関数、git/worktree.go で観察された。**

---

## ストレージ保護設計 — 書き込みの遅延・バッチ化

> 追記 2026-02-24。microSD上で動作する前提でのFS書き込み最適化。

### 「誰がこのデータを必要とするか」マップ

現状の永続化データを**消費者**と**書き込み頻度**で整理すると、書き込みを遅延できる余地が大きく異なる。

| データ | プロセス内読者 | プロセス外読者 | 書き込み頻度(現状) | 損失許容度 |
|--------|--------------|--------------|-----------------|----------|
| `sessions/*.json` | AgentLoop (ターン毎 `GetHistory`) | **なし**（起動時ロードのみ） | **メッセージ毎** | 中（会話消失は困るが致命ではない） |
| `state/stats.json` | StatusAPI, Mini App（in-process） | CLI `cmd_status` | **LLM呼び出し毎 + ユーザーメッセージ毎** | 低（数件のロスは許容） |
| `memory/MEMORY.md` | AgentLoop (ターン毎) | CLI, Mini App, **外部エディタ** | ステップ完了毎・LLM edit_file | 高（プラン状態が失われると復帰不能） |
| `memory/YYYYMM/DD.md` | AgentLoop（プランなし時） | 外部エディタ | 日次ノート追記時（低頻度） | 低 |

### 重要な観察: セッションファイルはプロセス内専用データ

`sessions/*.json` は**稼働中に外部プロセスが読まない**。唯一の利用タイミングは起動時の `loadSessions()`。つまり書き込みの目的は「クラッシュリカバリ」だけであり、**メッセージ毎の即時書き込みは過剰**。

同様に `state/stats.json` も、Mini App や CLI はプロセス内の `Tracker.GetStats()` 経由でメモリから読む。ファイルはプロセス再起動時の引き継ぎ専用。

### 推奨書き込み戦略

#### sessions/*.json — Write-behind (ダーティフラグ + 定期フラッシュ)

```
AddFullMessage() → in-memory のみ更新、dirty フラグ立て
                                ↓
                  定期タイマー (5分) or メッセージ数閾値 (20件)
                  またはシャットダウンフック → Save()
```

- リカバリウィンドウ: 最大5分 or 20メッセージ分
- 書き込み回数削減率: 会話速度次第だが **10〜50倍**
- 実装: `SessionManager` に `dirtyKeys map[string]bool` + バックグラウンドフラッシャーgoroutine

#### state/stats.json — 定期フラッシュのみ

```
RecordUsage() / RecordPrompt() → in-memory のみ更新
                                       ↓
                         定期タイマー (5分) → save()
                         + シャットダウンフック
```

- 損失リスク: 最大5分分の統計カウント（許容範囲）
- 書き込み回数削減率: **LLM呼び出し頻度 × 5分** = 数十〜数百倍

#### memory/MEMORY.md — ターンスコープキャッシュ (書き込みは即時維持)

書き込みは現状通り即時。読み取りの問題だけ解決する。

```
エージェントターン開始 → content := ReadLongTerm() を1回だけ
                         ↓ content を引数として全ヘルパーに渡す
                         (HasActivePlan(content), GetPlanStatus(content), ...)
エージェントターン終了 → content キャッシュ破棄
```

- 外部エディタとの整合: ターン境界でリフレッシュされるので1ターン以内の外部編集のみ見逃す（許容範囲）
- LLM の edit_file 経由の書き込み: ファイルシステムに即座に書かれるため次ターンで自動反映
- 読み取り回数削減: 1ターンあたり `5回以上 → 1回`

### microSD 寿命への影響試算

一般的な会話セッション（1時間、60メッセージ、10 LLM呼び出し/分）の場合:

| データ | 現状の書き込み回数/時 | 改善後 | 削減率 |
|--------|-------------------|--------|-------|
| sessions/*.json | ~60回 (メッセージ毎) | ~12回 (5分毎) | **80%減** |
| stats.json | ~660回 (LLM呼+prompt毎) | ~12回 (5分毎) | **98%減** |
| MEMORY.md | ステップ数分（変わらず） | 同左 | — |
| **合計** | **720+ 回/時** | **~24回/時** | **97%減** |

### 実装上の注意点

- **シャットダウンフック必須**: `SIGTERM` / `SIGINT` で dirty なデータを強制フラッシュ。フラッシュ失敗時はログに記録。
- **クラッシュ後のリカバリ**: dirty データが失われた場合、セッション履歴は最後のチェックポイント以降が消える。ユーザーへの通知が必要か検討。
- **フラッシュ中の競合**: フラッシュgoroutineと `Save()` の同時呼び出しを防ぐため、既存の mutex を流用。
- **MEMORY.md のターンキャッシュ**: `edit_file` ツールが MEMORY.md を書き込んだ場合、**同ターン内のキャッシュを無効化**する仕組みが必要（`MemoryStore.InvalidateCache()` を edit_file のコールバックから呼ぶなど）。そうしないと同ターン内の後続の `GetPlanStatus()` などが古いキャッシュを読む。

### RAM が潤沢な場合の設計変更

対象デバイスは RAM 7GB / available 5GB 超（例: `free -m` で available ~5260MB）。
この前提が上記の各戦略に与える影響を整理する。

#### 読み取りレイテンシの実態

`buff/cache` が 4.6GB 程度を占めるということは、OS のページキャッシュが空き RAM をほぼ全て使い切っている状態。`ReadLongTerm()` の複数回呼び出しは**実際にはディスクアクセスしていない**（2回目以降はページキャッシュヒット、マイクロ秒オーダー）。

読み取りの実コストは「ディスクI/O」ではなく「**syscall + 文字列 Split/Join のアロケーション**」。ターンスコープキャッシュの主な効果はレイテンシ削減より**GC 圧力の軽減**に変わる。

#### 書き込み寿命はRAMに影響されない

書き込みは `O_SYNC` ではなくても `os.Rename` でアトミックに書かれるが、カーネルはライトバックキャッシュを経由して最終的に SD に書く。ページキャッシュが書き込みを吸収しても**最終的な NAND への書き込み回数は変わらない**。書き込み削減の優先度は変わらず高い。

#### インメモリ表現の常駐が現実的になる

RAM が逼迫していない場合、`MemoryStore` に `*ParsedPlan` をフィールドとして持たせる設計（D-1, D-6 の解決策）のメモリコストは無視できる。MEMORY.md が数KB〜数十KB であっても、パース済み構造体として常駐させて差し支えない。

```go
// 設計案: MemoryStore がパース済み状態を保持
type MemoryStore struct {
    workspace  string
    memoryFile string
    mu         sync.RWMutex
    cached     *ParsedPlan  // nil = 未ロード
    cachedAt   time.Time
}
// edit_file ツールが書き込んだ後に InvalidateCache() を呼ぶことで
// 同ターン内の再読み込みをトリガーできる
```

これにより `GetMemoryContext()` 内の `ReadLongTerm()` 多重呼び出し問題（D-1）と、
public メソッドが `content` を隠す問題（D-6）が同時に解消される。

#### write-behind 窓をさらに広げられる

RAM が十分にあるため、セッションデータを長時間インメモリに保持するリスクがない。
write-behind の戦略を「5分 or 20件」から**「グレースフルシャットダウン時のみ + 30分タイマー」**に緩和しても、
クラッシュ時の損失（最大30分の会話）と実装の単純さのトレードオフとして許容できる可能性がある。
プロジェクトの可用性要件に応じて判断する。

#### 優先実装順の修正

RAM 制約がない前提での推奨順:

1. **`stats.json` の write-behind** — 実装が最も単純（タイマー1本追加）、書き込み削減率が最大（98%）
2. **`sessions/*.json` の write-behind** — セッション単位の dirty フラグ + シャットダウンフック
3. **`MemoryStore` への `*ParsedPlan` 常駐** — D-1/D-6 を根本解決、読み取りアロケーションをゼロに
4. **ターンスコープキャッシュ** — 3 が実装されれば自然に解決するため不要になる可能性あり

---

## 改修計画 — メモリ最適化の実装フェーズ

> 作成 2026-02-24。レビュー結果 (A〜H + D-1〜D-6 + ストレージ保護) を実装可能な単位に分割。
> 各フェーズは `go build ./... && go test ./... && go vet ./...` が通る状態で完結する。

### フェーズ 0: 機械的な置き換え (低リスク・高カバレッジ)

**目的**: コード構造を変えず、同じ関数内でパターンを置き換えるだけの修正。レビューが容易で回帰リスクが最小。

#### 0-1. strings.Builder 置き換え (カテゴリ A 残り)

| ファイル | 関数 | 優先度 |
|----------|------|--------|
| `pkg/tools/web.go` | `BraveSearchProvider.Search()` L73-85 | 🔴 |
| `pkg/tools/web.go` | `TavilySearchProvider.Search()` L155-167 | 🔴 |
| `pkg/tools/web.go` | `DuckDuckGoSearchProvider.extractResults()` L211-254 | 🔴 |
| `pkg/tools/web.go` | `WebFetchTool.extractText()` L592-617 | 🔴 |
| `pkg/skills/loader.go` | `BuildSkillsSummary()` L234-250 | 🔴 |
| `pkg/agent/context.go` | `BuildSystemPrompt()` L247 — `+=` を Builder に | 🟡 |
| `pkg/logger/logger.go` | `formatFields()` L241-246 | 🟡 |
| `pkg/skills/loader.go` | `LoadSkillsForContext()` L217-225 | 🟡 |
| `pkg/channels/discord.go` | `appendContent()` L162-168 | 🟡 |
| `pkg/channels/slack.go` | `handleMessageEvent()` L234-272 | 🟡 |

#### 0-2. スライス事前容量 (カテゴリ B 残り)

| ファイル | 変更 |
|----------|------|
| `pkg/skills/loader.go:73` | `make([]SkillInfo, 0)` → `make([]SkillInfo, 0, 20)` |
| `pkg/config/config.go:628` | `var matches` → `make([]ModelConfig, 0, 4)` |
| `pkg/config/migration.go:48` | `var result` → `make([]ModelConfig, 0, len(providers)*4)` |
| `pkg/skills/registry.go:183` | `var merged` → `make([]SearchResult, 0, len(regs)*limit)` |
| `pkg/agent/session_tracker.go:125` | `var result` → 容量ヒント付き |
| `pkg/skills/search_cache.go:42-43` | map/slice に `maxEntries` ヒント |

#### 0-3. byte/string 変換の削減 (カテゴリ C)

| ファイル | 変更 |
|----------|------|
| `pkg/tools/web.go:289` | `strings.NewReader(string(payloadBytes))` → `bytes.NewReader(payloadBytes)` |
| `pkg/tools/web.go:545-562` | 複数の `string(body)` → 1回だけ変換して変数に保持 |
| `pkg/providers/claude_cli_provider.go:133` | `string(paramsJSON)` → `sb.Write(paramsJSON)` |
| `pkg/utils/string.go:100` | `Truncate()` — `len(s) <= max` で早期 return (ASCII fast path) |
| `pkg/utils/string.go:50` | `wrapLine()` — 同上 ASCII fast path |
| `pkg/git/worktree.go:71-75` | `[]rune` → byte 長チェックで早期 return |

#### 0-4. パッケージ変数化 (カテゴリ H)

| ファイル | 変更 |
|----------|------|
| `pkg/utils/media.go:18-19` | `audioExtensions`/`audioTypes` を関数外の `var` に |
| `pkg/skills/clawhub_registry.go:114` | `fmt.Sprintf("%d", limit)` → `strconv.Itoa(limit)` |

**コミット単位**: 0-1, 0-2, 0-3, 0-4 をそれぞれ個別コミット。

---

### フェーズ 1: 値渡し・コピーの最適化 (カテゴリ E)

**目的**: struct の不要なコピーを削減。型シグネチャが変わるため呼び出し側の修正が必要。

| ファイル | 変更 | 注意 |
|----------|------|------|
| `pkg/agent/session_tracker.go:125` | `ListActive()` の戻り値を `[]*SessionEntry` に | 呼び出し側 (`cmd_gateway.go`, `miniapp.go`) の型合わせ |
| `pkg/logger/logger.go:88-92` | `recent()` の戻り値を `[]*LogEntry` 検討 | JSON シリアライズへの影響確認 |
| `pkg/skills/registry.go:132-133` | `SearchAll()` 内の registries コピーをポインタスライスに | ロック範囲の再確認 |

**コミット**: 1つにまとめる。

---

### フェーズ 2: JSON ホットパスの最適化 (カテゴリ D)

**目的**: ストリーミングループ内の重複 Marshal/Unmarshal を排除。

#### 2-1. openai_compat streaming の Arguments 重複 Unmarshal

`pkg/providers/openai_compat/provider.go` L274, 362, 621
— ストリーム完了時に1回だけ Unmarshal するよう制御フローを整理。

#### 2-2. codex/claude CLI の Parameters 重複 Marshal

`pkg/providers/codex_cli_provider.go:154-155`, `pkg/providers/claude_cli_provider.go:133`
— ツール定義はループ外で1回 Marshal してキャッシュ、またはループ内で `json.RawMessage` 直接書き込み。

**コミット**: 2-1, 2-2 を個別。

---

### フェーズ 3: MemoryStore の読み取り最適化 (設計 D-1, D-6)

**目的**: `GetMemoryContext()` 1回で `ReadLongTerm()` が 5回以上呼ばれる問題を解消。

#### 方式: content パススルー (最小侵襲)

既存の private 関数群 (`extractPhaseContent`, `getPlanPhasesFromContent` 等) は既に `content string` を受け取る設計。
public メソッド側に「content を引数に取るバリアント」を追加し、`GetMemoryContext()` で1回だけ Read する。

```go
// 新設: 1回の Read で全情報を取得
func (ms *MemoryStore) GetMemoryContextCached() string {
    content := ms.ReadLongTerm()
    // content を全ヘルパーに渡す
    hasPlan := hasActivePlanFrom(content)
    status  := getPlanStatusFrom(content)
    ctx     := getPlanContextFrom(content)
    ...
}
```

既存の public メソッド (`HasActivePlan()`, `GetPlanStatus()` 等) は互換性のため残す（単体テスト・CLI から呼ばれる可能性）。

#### memory.go 内の重複 Split/Join (カテゴリ H 後半)

`extractPhaseContent`, `GetPlanPhases`, `MarkStep`, `AddStep` が個別に `strings.Split` する問題は、
content パススルー方式で自然に解消される（Split は `GetMemoryContext()` 内で1回のみ）。

**コミット**: 1つ。

---

### フェーズ 4: ストレージ保護 — write-behind (設計セクション)

**目的**: microSD 書き込み回数を 97% 削減。

#### 4-1. stats.json の write-behind

- `SessionTracker` (または stats 管理構造体) に dirty フラグ + タイマー (5分)
- `RecordUsage()` / `RecordPrompt()` はインメモリのみ更新
- シャットダウンフックで強制フラッシュ
- 実装量: 最小。タイマー goroutine 1本 + `Close()` メソッド

#### 4-2. sessions/*.json の write-behind

- `SessionManager` に `dirtyKeys map[string]bool` + バックグラウンドフラッシャー
- `AddFullMessage()` → dirty 記録のみ、`Save()` はフラッシャーから呼ぶ
- フラッシュ間隔: 5分 or 20メッセージ
- シャットダウンフックで全 dirty セッションをフラッシュ

**コミット**: 4-1, 4-2 を個別。

---

### フェーズ 5: 発展的最適化 (任意)

実装コストが高い or 効果が限定的なもの。必要に応じて着手。

| 項目 | 内容 | 見送り理由 |
|------|------|-----------|
| F: sync.Pool | web.go extractText, telegram.go Markdown 変換 | 呼び出し頻度が低く Pool の効果が薄い可能性 |
| G: LRU O(1) 化 | search_cache.go を doubly-linked list に | maxEntries=100 で O(n) でも十分高速 |
| D-2: FunctionCall.Arguments 型変更 | `string` → `json.RawMessage` | 全プロバイダーに波及、破壊的変更 |
| D-3: Parameters を RawMessage に | 同上 | 同上 |
| D-5: Session.Messages を immutable に | COW or linked list | セッション管理の根本再設計が必要 |
| telegram.go wrapByDisplayWidth | ループ内 `string(r)` | runewidth ライブラリ依存、別途検討 |

---

### 実装順サマリー

```
Phase 0 ──→ Phase 1 ──→ Phase 2 ──→ Phase 3 ──→ Phase 4
 機械的      値渡し       JSON       Memory      Storage
 置き換え    最適化     ホットパス    読み取り    write-behind
 (4 commits) (1 commit) (2 commits) (1 commit)  (2 commits)
```

- Phase 0〜2: **アロケーション削減** (GC 圧力軽減)
- Phase 3: **syscall + Split/Join 削減** (CPU + アロケーション)
- Phase 4: **ディスク書き込み削減** (microSD 寿命保護)
- Phase 5: 必要に応じて個別判断
