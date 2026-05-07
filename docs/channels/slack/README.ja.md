> [README](../../project/README.ja.md) に戻る

# Slack

Slack は世界をリードする企業向けインスタントメッセージングプラットフォームです。PicoClaw は Slack の Socket Mode を使用してリアルタイムの双方向通信を実現しており、公開 Webhook エンドポイントの設定は不要です。

## 設定

```json
{
  "channel_list": {
    "slack": {
      "enabled": true,
      "type": "slack",
      "bot_token": "xoxb-...",
      "app_token": "xapp-...",
      "allow_from": []
    }
  }
}
```

| フィールド | 型     | 必須   | 説明                                                                     |
| ---------- | ------ | ------ | ------------------------------------------------------------------------ |
| enabled    | bool   | はい   | Slack チャンネルを有効にするかどうか                                     |
| bot_token  | string | はい   | Slack ボットの Bot User OAuth Token（xoxb- で始まる）                    |
| app_token  | string | はい   | Slack アプリの Socket Mode App Level Token（xapp- で始まる）             |
| allow_from | array  | いいえ | ユーザーIDのホワイトリスト。空の場合は全ユーザーを許可                   |

## セットアップ手順

1. [Slack API](https://api.slack.com/) にアクセスして新しい Slack アプリを作成する
2. Socket Mode を有効にして App Level Token を取得する
3. Bot Token Scopes を追加する（例: `chat:write`、`im:history` など）
4. アプリをワークスペースにインストールして Bot User OAuth Token を取得する
5. Bot Token と App Token を設定ファイルに入力する
