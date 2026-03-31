> [README](../../../README.ja.md) に戻る

# Line

PicoClaw は LINE Messaging API と Webhook コールバックを通じて LINE をサポートします。

## 設定

```json
{
  "channels": {
    "line": {
      "enabled": true,
      "channel_secret": "YOUR_CHANNEL_SECRET",
      "channel_access_token": "YOUR_CHANNEL_ACCESS_TOKEN",
      "webhook_path": "/webhook/line",
      "allow_from": []
    }
  }
}
```

| フィールド           | 型     | 必須   | 説明                                                               |
| -------------------- | ------ | ------ | ------------------------------------------------------------------ |
| enabled              | bool   | はい   | LINE チャンネルを有効にするかどうか                                |
| channel_secret       | string | はい   | LINE Messaging API の Channel Secret                               |
| channel_access_token | string | はい   | LINE Messaging API の Channel Access Token                         |
| webhook_path         | string | いいえ | Webhook のパス（デフォルト: /webhook/line）                        |
| allow_from           | array  | いいえ | ユーザーIDのホワイトリスト。空の場合は全ユーザーを許可             |

## セットアップ手順

1. [LINE Developers Console](https://developers.line.biz/console/) にアクセスし、サービスプロバイダーと Messaging API チャンネルを作成する
2. Channel Secret と Channel Access Token を取得する
3. Webhook を設定する:
   - LINE は Webhook に HTTPS が必要なため、HTTPS 対応サーバーをデプロイするか、ngrok などのリバースプロキシツールを使用してローカルサーバーをインターネットに公開する必要があります
   - PicoClaw は共有の Gateway HTTP サーバーを使用してすべてのチャンネルの Webhook コールバックを受信します。デフォルトのリッスンアドレスは 127.0.0.1:18790 です
   - Webhook URL を `https://your-domain.com/webhook/line` に設定し、外部ドメインをローカルの Gateway（デフォルトポート 18790）にリバースプロキシする
   - Webhook を有効にして URL を検証する
4. Channel Secret と Channel Access Token を設定ファイルに入力する
