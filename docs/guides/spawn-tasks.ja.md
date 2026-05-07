# 🔄 非同期タスクと Spawn

> [README](../project/README.ja.md) に戻る

### Spawn を使用した非同期タスク

長時間実行タスク（Web 検索、API 呼び出し）には、`spawn` ツールを使用して**サブ Agent (subagent)** を作成します：

```markdown
# Periodic Tasks

## Quick Tasks (respond directly)

- Report current time

## Long Tasks (use spawn for async)

- Search the web for AI news and summarize
- Check email and report important messages
```

**主な動作：**

| 特性             | 説明                                             |
| ---------------- | ------------------------------------------------ |
| **spawn**        | 非同期サブ Agent を作成、メインハートビートをブロックしない |
| **独立コンテキスト** | サブ Agent は独自のコンテキストを持ち、セッション履歴なし |
| **message tool** | サブ Agent は message ツールでユーザーと直接通信   |
| **ノンブロッキング** | spawn 後、ハートビートは次のタスクに進む         |

#### サブ Agent の通信の仕組み

```
ハートビートトリガー (Heartbeat triggers)
    ↓
Agent が HEARTBEAT.md を読み取り
    ↓
長時間タスクの場合: サブ Agent を spawn
    ↓                           ↓
次のタスクに進む             サブ Agent が独立して作業
    ↓                           ↓
すべてのタスク完了           サブ Agent が "message" ツールを使用
    ↓                           ↓
HEARTBEAT_OK を応答          ユーザーが直接結果を受信
```

サブ Agent はツール（message、web_search など）にアクセスでき、メイン Agent を経由せずにユーザーと独立して通信できます。

**設定：**

```json
{
  "heartbeat": {
    "enabled": true,
    "interval": 30
  }
}
```

| オプション | デフォルト値 | 説明                           |
| ---------- | ------------ | ------------------------------ |
| `enabled`  | `true`       | ハートビートの有効/無効        |
| `interval` | `30`         | チェック間隔（分単位、最小: 5）|

**環境変数:**

- `PICOCLAW_HEARTBEAT_ENABLED=false` で無効化
- `PICOCLAW_HEARTBEAT_INTERVAL=60` で間隔を変更
