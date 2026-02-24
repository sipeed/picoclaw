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
