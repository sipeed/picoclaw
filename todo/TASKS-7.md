# TASKS-7: Provider Wire Compatibility Hardening

## 背景 (発見済み懸念)

`TASKS-1` の型整理後、`openai_compat` の tool call 送信フォーマットが簡素化された。
この変更で、Gemini 系 endpoint で必要な thought signature 継続情報が履歴送信時に欠落する懸念がある。

### 懸念一覧 (今回レビューで確認した内容を全記録)

1. **送信時に `extra_content` と `function.thought_signature` が落ちる**
   - `stripSystemParts()` が `openaiToolCall/openaiFunctionCall` へ変換する際、wire struct に `extra_content` と `function.thought_signature` の受け皿がない。
   - 参照: `pkg/providers/openai_compat/provider.go` の `openaiToolCall/openaiFunctionCall` 定義と `toOpenAIWireToolCalls()`

2. **受信・保持はできているのに、再送で欠落して往復整合が崩れる**
   - `parseResponse()` では `extra_content.google.thought_signature` を取り込み `ToolCall.ExtraContent` と `Function.ThoughtSignature` に保持している。
   - `AgentLoop` でも assistant message に `ExtraContent`/`ThoughtSignature` を保存している。
   - しかし次回 LLM 呼び出し時、`stripSystemParts()` 側で落ちるため履歴再送で不整合になる。
   - 参照: `pkg/providers/openai_compat/provider.go` (`parseResponse` 周辺), `pkg/agent/loop.go` (tool call 履歴保存)

3. **provider 別の wire 方針が未分離**
   - 厳格な OpenAI 互換 endpoint は unknown field を嫌うため最小化が必要。
   - 一方で Gemini 互換 endpoint は thought signature 継続のため provider 拡張 field を維持すべき。
   - 現状は単一 wire で両者を同時に満たせていない。

4. **回帰を防ぐテストが不足**
   - `openai_compat` に thought signature の送受信ラウンドトリップを保証するテストが不足している。
   - 特に「OpenAI strict では落とす / Gemini では残す」の分岐テストが必要。

## 対応方針

大手 provider は専用対応を行う。
`openai_compat` 内で endpoint 特性に応じて tool call の wire フォーマットを切り替える。

## タスク一覧

### 1. Provider capability 判定の導入

- `apiBase` から endpoint 種別を判定する小さな関数を追加
- 最低限の区分:
  - OpenAI strict 互換 (unknown field 非許容想定)
  - Gemini 互換 (thought signature 継続を有効化)

### 2. Tool call 送信 wire の分岐実装

- OpenAI strict 互換:
  - 現行どおり最小フィールドのみ (`id/type/function.name/function.arguments`)
- Gemini 互換:
  - `extra_content.google.thought_signature` を再送
  - `function.thought_signature` も必要に応じて再送
- 既存互換を壊さないよう、デフォルトは strict-safe 側を維持

### 3. thought signature 解決ロジックの明示化

- `ToolCall` から署名を引く優先順を helper 化
  - `tc.ExtraContent.Google.ThoughtSignature`
  - `tc.Function.ThoughtSignature`
  - `tc.ThoughtSignature`
- 片側だけ更新されても送信値がぶれないように統一

### 4. 回帰テストの追加

- `parseResponse()` で thought signature 取り込みを検証
- `stripSystemParts()` / request body 生成で endpoint 別分岐を検証
  - OpenAI strict: 拡張 field が送信されない
  - Gemini 互換: 拡張 field が送信される
- 1ターン往復の簡易ラウンドトリップテストを追加

### 5. ドキュメント更新

- `CLAUDE.md` の未実装タスク一覧へ TASKS-7 を追加
- 実装完了後、該当行を簡略化して完了扱いへ更新

## 対象ファイル (予定)

- `pkg/providers/openai_compat/provider.go`
- `pkg/providers/openai_compat/provider_test.go`
- `CLAUDE.md`

## 完了基準

- OpenAI strict endpoint で既存リクエスト互換を維持
- Gemini 互換 endpoint で thought signature 継続情報が履歴再送される
- `go test ./pkg/providers/openai_compat/...` がパス
- `go test ./...` がパス
