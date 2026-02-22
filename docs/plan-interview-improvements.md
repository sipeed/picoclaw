# /plan interview 改善検討

## 現象

`/plan <task>` でinterviewモードに入った後、AIが:

1. MEMORY.mdに何も書かずに会話だけ続ける
2. interviewを無視して実装を始めようとする（exec, ファイル書き込み）
3. 結局フェーズ/ステップ/コマンドが書かれないまま executing に遷移する

## 原因分析

### Layer 1: 技術的障壁（修正済み）

| 問題 | 原因 | 修正 |
|---|---|---|
| XML tool callがパースされない | MiniMaxの開閉タグ不一致（`<minimax:toolcall>` vs `</minimax:tool_call>`） | regex + normalizeAlpha + 編集距離で fuzzy matching |
| tool名が不一致で実行失敗 | `readfile` vs `read_file` | `ToolRegistry.Get()` に normalizeAlpha フォールバック |
| interview許可リストも完全一致 | `isToolAllowedDuringInterview` が exact match | normalizeAlpha で比較 |

### Layer 2: AI行動の問題（未対応）

技術的障壁を除去しても、モデル（特に小規模モデル）の命令追従に起因する問題が残る:

- **「会話しながらファイル編集」が複合タスクとして難しい** — interview中にMEMORY.mdを更新する行為は、会話とファイル操作の並行処理。小さいモデルにはハードルが高い
- **AIがワークフローを無視して実装に走る** — interview指示よりも「ユーザーの要求を直接解決しよう」というバイアスが強い
- **tool callイテレーション中に目的を忘れる** — read_fileでファイルを読み始めると、そのまま実装に入ろうとする

## 検討した案と判断

### 案: ユーザー発言の自動追記（却下）

MEMORY.mdにユーザー発言を自動追記 → ちゃんとしたplanにはならない。生データの蓄積であって構造化された計画ではない。

### 案: 専用tool `save_context`（却下）

tool仕様を変えても、AIがtoolを適切に呼ばない根本問題は解決しない。

### 案: 別パスでplan生成LLMコール（却下）

AIが「情報が揃った」と判断するトリガーの設計が難しい。キーワード検出は不安定。

### 案: `/plan draft` コマンド（却下）

ユーザーがトリガーできても、その前にAIがワークフローを無視して実装を始める問題は残る。また状態とコマンドが増えてユーザーが混乱する。

### 案: 毎ターン固定メッセージ注入（却下）

探索的なマルチターン会話で誤爆する。AIがファイルを読んだりリサーチしている途中のターンで「edit_fileしろ」は邪魔。

## 採用方針: tool callイテレーション内リマインド

### 既知の知見

通常の開発モード（executing）で、tool callが反復される中でユーザー指示が忘れられる問題に対し、リマインド注入で自律開発がスムーズに進むようになった実績がある。同じパターンをinterview中にも適用する。

### 設計

**1. interviewフェーズのtool制限（実装済み）**

```
許可: read_file, list_dir, web_search, web_fetch
許可: edit_file / write_file / append_file（MEMORY.mdのみ）
ブロック: exec, その他write系
```

AIが実装に走ろうとしても物理的にできない。

**2. tool callイテレーション内でリマインド注入（未実装）**

`runLLMIteration` 内で、tool結果をLLMに返す直前（= 次のLLMコールの直前）にリマインドを差し込む。

```go
// tool結果メッセージの後、次のLLMコール前
if isPlanPreExecution(agent.ContextBuilder.GetPlanStatus()) {
    messages = append(messages, providers.Message{
        Role:    "user",
        Content: "[System] You are interviewing. Ask questions and save findings " +
                 "to ## Context in memory/MEMORY.md. " +
                 "When ready, write ## Phase and ## Commands sections.",
    })
}
```

- **注入タイミング**: tool callループ内のみ。ユーザーとの会話ターンには入れない
- **注入条件**: interviewing または review 状態の時
- **内容**: 固定。短く、具体的に何をすべきか指示

**3. 状態遷移は既存のまま**

```
/plan <task> → interviewing（AIが質問、read系+MEMORY.md書き込み許可）
             → AIがStatus:executingに変更しようとする
             → システムがreviewに横取り（phases > 0 の場合）
             → ユーザーにplan表示
/plan start  → executing（全toolアンロック）
```

新しい状態・新しいコマンドなし。

## 実装タスク

- [ ] `runLLMIteration` 内のtool callループにリマインド注入を追加
- [ ] リマインド内容をステータスごとに分岐（interviewing / review / executing）
- [ ] 既存のstaleness nudge（2ターン無更新で警告）との統合・整理
- [ ] テスト追加

## 未解決の懸念

- **リマインドの効果がモデル依存**: 大きいモデルには効くが、小さいモデルでは無視される可能性
- **リマインドの頻度**: 毎イテレーション注入でトークン消費が増える（ただし1行程度なので軽微）
- **interview→plan書き込みのタイミング**: AIが「もう十分」と判断する基準はモデル任せ。staleness nudgeが補助するが確実ではない
