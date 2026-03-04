# TASKS-1: Memory & Performance Optimization

> ✅ 2026-03-04: D-1 / D-2 / D-3 / D-4 / D-6 と stats.Tracker 定期フラッシュを実装済み。

内部リファクタリング。外部APIの変更なし。他トラックへの依存なし。

## タスク一覧

### D-1. MemoryStore のパース済みキャッシュ導入

`GetMemoryContext()` 1回で `ReadLongTerm()` が5回以上呼ばれる連鎖を解消。

**方針 (いずれか):**
- (a) content パススルー — 高レベルメソッドだけが1回 `ReadLongTerm()` を呼び、content を private ヘルパーに渡す
- (b) `*ParsedPlan` 常駐 — MemoryStore にパース済み構造体を持たせ、`edit_file` 後に `InvalidateCache()` を呼ぶ

**対象ファイル:** `pkg/agent/memory.go`

---

### D-2. FunctionCall.Arguments の型整理

`FunctionCall.Arguments` が JSON 文字列のまま。ストリーミングループ内で重複 Unmarshal が発生。
`ToolCall.Arguments map[string]any` のパース済みフィールドと中途半端に共存している。

**やること:** パース済みフィールドに一本化し、JSON 文字列フィールドを内部に隠蔽。

**対象ファイル:** `pkg/providers/protocoltypes/types.go`, ストリーミング処理周辺

---

### D-3. ToolFunctionDefinition.Parameters → json.RawMessage

`Parameters` が `map[string]any` のため、プロバイダーへ送るたびに Marshal が必要。
`json.RawMessage` にすれば一度の marshal で済む。

**対象ファイル:** `pkg/providers/protocoltypes/types.go`, 各プロバイダー実装

---

### D-4. 検索プロバイダーの共通フォーマット抽象

`[]string + strings.Join` パターンが3箇所に複製。`Search()` 戻り値を構造体にすれば1箇所で済む。

**対象ファイル:** `pkg/search/` 配下

---

### D-6. MemoryStore メソッド境界の再設計

呼び出し側は複数の値が必要でも複数回 `ReadLongTerm()` を呼ぶしかない。
D-1 の解決策と合わせて、メソッド境界を「必要な情報の単位」に再編成。

**対象ファイル:** `pkg/agent/memory.go`

---

### stats.Tracker 定期フラッシュ

`state/stats.json` が LLM 呼び出し毎に書き込まれる。定期フラッシュ (5分) に変更して microSD 寿命を保護。

**やること:**
- `stats.Tracker` にインメモリバッファ + バックグラウンドフラッシャー goroutine を追加
- `Close()` メソッド追加 (タイマー停止 + 最終 save)
- SIGTERM/SIGINT でシャットダウンフック

**対象ファイル:** `pkg/state/state.go`

---

## 完了基準

- `go test ./...` 全パス
- MEMORY.md 読み取りが1ターンあたり1回以下に削減 (D-1)
- FunctionCall の Unmarshal が1回に集約 (D-2)
- ToolFunctionDefinition.Parameters の Marshal がプロバイダー初期化時のみ (D-3)
- stats.json の書き込み頻度が 98% 削減 (stats.Tracker)


## 作業報告 (2026-03-05)

- 実装完了: D-1 / D-2 / D-3 / D-4 / D-6、stats.Tracker 定期フラッシュ
- 主要変更:
  - MemoryStore に長期メモリキャッシュとパース済み plan state キャッシュを導入
  - FunctionCall.Arguments を map 中心に統一し、JSON 文字列は内部互換層で吸収
  - ToolFunctionDefinition.Parameters を json.RawMessage 化し、各 provider で必要時 decode
  - Web 検索（Brave/Tavily/DuckDuckGo）の結果整形を共通化
- 検証結果:
  - Linux (WSL): go generate ./... 成功
  - Linux (WSL): go test ./... 成功
  - 差分 lint: golangci-lint run -n 0 issues
