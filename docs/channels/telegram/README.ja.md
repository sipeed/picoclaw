> [README](../../project/README.ja.md) に戻る

# Telegram

Telegram チャンネルは、Telegram Bot API を使用したロングポーリングによるボットベースの通信を実装しています。テキストメッセージ、メディア添付ファイル（写真、音声、オーディオ、ドキュメント）、Groq Whisper による音声文字起こし、および組み込みコマンドハンドラーをサポートしています。

## 設定

```json
{
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram",
      "allow_from": ["123456789"],
      "settings": {
        "proxy": "",
        "use_markdown_v2": false,
        "business_mode": false,
        "business_owner": "123456789",
        "business_commands_enable": false,
        "guest_mode": false
      }
    }
  }
}
```

| フィールド      | 型     | 必須 | 説明                                                              |
| --------------- | ------ | ---- | ----------------------------------------------------------------- |
| enabled         | bool   | はい | Telegram チャンネルを有効にするかどうか                           |
| allow_from      | array  | いいえ | 許可するユーザーIDのリスト。空の場合はすべてのユーザーを許可     |
| settings.proxy  | string | いいえ | Telegram API への接続に使用するプロキシ URL (例: http://127.0.0.1:7890) |
| settings.use_markdown_v2 | bool | いいえ | Telegram MarkdownV2 フォーマットを有効にする                      |
| settings.business_mode | bool | いいえ | Telegram Business メッセージ処理を有効にする                      |
| settings.business_owner | string | いいえ | 無視する Business アカウント所有者の Telegram ユーザー ID          |
| settings.business_commands_enable | bool | いいえ | Telegram Business チャットで Bot コマンドを処理する              |
| settings.guest_mode | bool | いいえ | Telegram Guest メッセージ処理と返信を有効にする                   |

## セットアップ手順

1. Telegram で `@BotFather` を検索する
2. `/newbot` コマンドを送信し、指示に従って新しいボットを作成する
3. HTTP API トークンを取得する
4. 設定ファイルにトークンを入力する
5. (任意) `allow_from` を設定して、対話を許可するユーザー ID を制限する（ID は `@userinfobot` で取得可能）

## Telegram Business モード

`settings.business_mode: true` を設定すると、接続された Business アカウントの Telegram Business メッセージを受信して返信できます。返信には受信した `business_connection_id` が使われ、Bot に `can_read_messages` 権限がある場合は受信 Business メッセージを既読にします。

`settings.business_owner` には Business アカウント所有者の Telegram ユーザー ID を設定します。このユーザーからの Business メッセージは無視されるため、接続済みアカウントから手動送信したメッセージへの自動返信を避けられます。

Business チャット内の Bot コマンドは既定で無視されます。`/new`、`/help`、`/show`、`/list`、`/use` を処理したい場合は `settings.business_commands_enable: true` を設定してください。

例：

```json
{
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram",
      "allow_from": ["123456789"],
      "settings": {
        "business_mode": true,
        "business_owner": "123456789",
        "business_commands_enable": true
      }
    }
  }
}
```

## Telegram Guest モード

`settings.guest_mode: true` を設定すると、`guest_message` 更新を受信し、Telegram の `answerGuestQuery` メソッドで返信できます。Guest メッセージは Bot が参加していないチャットから届くため、PicoClaw は受信した `guest_query_id` を使って別セッションとして扱います。

`settings.guest_mode` が `false` の場合、Guest 更新は要求されず、デコードされた Guest メッセージも無視されます。Telegram は単一の `answerGuestQuery` 応答のみを必要とするため、Guest 応答のプレースホルダーと入力インジケーターはスキップされます。

例：

```json
{
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram",
      "allow_from": ["123456789"],
      "settings": {
        "guest_mode": true
      }
    }
  }
}
```

## 高度なフォーマット

`use_markdown_v2: true` を設定することで、增强されたフォーマットオプションを有効にできます。これにより、ボットは Telegram MarkdownV2 の全機能（ネストされたスタイル、スポイラー、カスタム固定幅ブロックなど）を利用できます。

```json
{
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram",
      "allow_from": ["YOUR_USER_ID"],
      "settings": {
        "use_markdown_v2": true
      }
    }
  }
}
```
