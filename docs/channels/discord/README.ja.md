> [README](../../../README.ja.md) に戻る

# Discord

Discord はコミュニティ向けに設計された無料の音声・ビデオ・テキストチャットアプリケーションです。PicoClaw は Discord Bot API を通じて Discord サーバーに接続し、メッセージの受信と送信をサポートします。

## 設定

```json
{
  "channels": {
    "discord": {
      "enabled": true,
      "token": "YOUR_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"],
      "group_trigger": {
        "mention_only": false
      }
    }
  }
}
```

| フィールド    | 型     | 必須   | 説明                                                              |
| ------------- | ------ | ------ | ----------------------------------------------------------------- |
| enabled       | bool   | はい   | Discord チャンネルを有効にするかどうか                            |
| token         | string | はい   | Discord ボットトークン                                            |
| allow_from    | array  | いいえ | 許可するユーザーIDのリスト。空の場合はすべてのユーザーを許可     |
| group_trigger | object | いいえ | グループトリガー設定（例: { "mention_only": false }）             |

## セットアップ手順

1. [Discord 開発者ポータル](https://discord.com/developers/applications) にアクセスして新しいアプリケーションを作成する
2. Intents を有効にする:
   - Message Content Intent
   - Server Members Intent
3. Bot トークンを取得する
4. 設定ファイルに Bot トークンを入力する
5. ボットをサーバーに招待し、必要な権限を付与する（例: メッセージの送信、メッセージ履歴の読み取りなど）
