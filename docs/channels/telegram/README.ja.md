> [README](../../../README.ja.md) に戻る

# Telegram

Telegram チャンネルは、Telegram Bot API を使用したロングポーリングによるボットベースの通信を実装しています。テキストメッセージ、メディア添付ファイル（写真、音声、オーディオ、ドキュメント）、Groq Whisper による音声文字起こし、および組み込みコマンドハンドラーをサポートしています。

## 設定

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "123456789:ABCdefGHIjklMNOpqrsTUVwxyz",
      "allow_from": ["123456789"],
      "proxy": ""
    }
  }
}
```

| フィールド | 型     | 必須 | 説明                                                              |
| ---------- | ------ | ---- | ----------------------------------------------------------------- |
| enabled    | bool   | はい | Telegram チャンネルを有効にするかどうか                           |
| token      | string | はい | Telegram Bot API トークン                                         |
| allow_from | array  | いいえ | 許可するユーザーIDのリスト。空の場合はすべてのユーザーを許可     |
| proxy      | string | いいえ | Telegram API への接続に使用するプロキシ URL (例: http://127.0.0.1:7890) |

## セットアップ手順

1. Telegram で `@BotFather` を検索する
2. `/newbot` コマンドを送信し、指示に従って新しいボットを作成する
3. HTTP API トークンを取得する
4. 設定ファイルにトークンを入力する
5. (任意) `allow_from` を設定して、対話を許可するユーザー ID を制限する（ID は `@userinfobot` で取得可能）
