# TASKS-6: SOUL.md — AI Persona Evolution (Sleep-Driven)

AI が human との関わりの中でペルソナを育てる仕組み。
SOUL.md の追記ではなく「再構成」— 睡眠フェーズで体験を統合・忘却し、人格を更新する。

**前提:** TASKS-3 (Session DAG) ✅、TASKS-2 (Orchestration) 完了後に着手。

## 設計原則

1. **追記ではなく再構成** — エピソードを足すのではなく、SOUL.md 全体を「今の自分」として書き直す
2. **忘却に根拠が要る** — 「なぜ覚えているか」の根拠がない記憶は次の睡眠で消える圧力を受ける
3. **圧力は物理的** — token 上限・行数制限が脳の容量として機能する
4. **移ろいがある** — 一時的な気分・印象は覚醒中だけ存在し、睡眠で統合 or 消失する

## アーキテクチャ概要

```
覚醒 (human との会話中)
  → エピソードが turns として SQLite に蓄積
  → SOUL.md は読むだけ、書かない
  → RAM 上の transient state (気分・印象) が変動

睡眠 (idle 検出 / heartbeat トリガー)
  → LLM が「今日の体験」を session turns から読み出し
  → SOUL.md を再構成:
      - 新しい印象・学びを統合
      - 根拠の薄い記憶に忘却圧力 (容量制限)
      - 「何を手放すか」を自分で決める
  → 次の覚醒時に更新された SOUL.md を読み込み
```

## 未定項目 (TASKS-2 完了後に具体化)

- SOUL.md のフォーマット (セクション構成、容量制限)
- 睡眠トリガー条件 (idle 時間閾値、heartbeat 連携)
- 忘却圧力の実装 (LLM prompt 設計、根拠スコアリング)
- transient state の表現 (session metadata? 専用フィールド?)
- 覚醒中の印象収集 (会話中にどう「気分」を蓄積するか)
