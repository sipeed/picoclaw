> [README](../../../README.ja.md) に戻る

# Matrix チャンネル設定ガイド

## 1. 設定例

`config.json` に以下を追加してください：

```json
{
  "channels": {
    "matrix": {
      "enabled": true,
      "homeserver": "https://matrix.org",
      "user_id": "@your-bot:matrix.org",
      "access_token": "YOUR_MATRIX_ACCESS_TOKEN",
      "device_id": "",
      "join_on_invite": true,
      "allow_from": [],
      "group_trigger": {
        "mention_only": true
      },
      "placeholder": {
        "enabled": true,
        "text": "Thinking..."
      },
      "reasoning_channel_id": "",
      "message_format": "richtext"
    }
  }
}
```

## 2. フィールドリファレンス

| フィールド           | 型       | 必須 | 説明 |
|----------------------|----------|------|------|
| enabled              | bool     | はい | Matrix チャンネルの有効/無効 |
| homeserver           | string   | はい | Matrix ホームサーバー URL（例：`https://matrix.org`） |
| user_id              | string   | はい | ボットの Matrix ユーザー ID（例：`@bot:matrix.org`） |
| access_token         | string   | はい | ボットのアクセストークン |
| device_id            | string   | いいえ | オプションの Matrix デバイス ID |
| join_on_invite       | bool     | いいえ | 招待されたルームに自動参加 |
| allow_from           | []string | いいえ | ユーザーホワイトリスト（Matrix ユーザー ID） |
| group_trigger        | object   | いいえ | グループトリガー戦略（`mention_only` / `prefixes`） |
| placeholder          | object   | いいえ | プレースホルダーメッセージ設定 |
| reasoning_channel_id | string   | いいえ | 推論出力のターゲットチャンネル |
| message_format       | string   | いいえ | 出力形式：`"richtext"`（デフォルト）は markdown を HTML としてレンダリング；`"plain"` はプレーンテキストのみ送信 |

## 3. 現在サポートされている機能

- markdown レンダリング付きテキストメッセージ送受信（太字、斜体、見出し、コードブロックなど）
- 設定可能なメッセージ形式（`richtext` / `plain`）
- 受信画像/音声/動画/ファイルのダウンロード（MediaStore 優先、ローカルパスフォールバック）
- 受信音声の既存文字起こしフローへの正規化（`[audio: ...]`）
- 送信画像/音声/動画/ファイルのアップロードと送信
- グループトリガールール（メンションのみモードを含む）
- タイピング状態（`m.typing`）
- プレースホルダーメッセージ + 最終返信の置き換え
- 招待されたルームへの自動参加（無効化可能）

## 4. TODO

- リッチメディアメタデータの改善（例：画像/動画のサイズとサムネイル）
