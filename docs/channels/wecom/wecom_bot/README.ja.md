> [README](../../../../README.ja.md) に戻る

# 企業WeChat ボット

企業WeChat ボットは、企業WeChatが提供するWebhook URLを通じてメッセージを受信できる迅速な連携方式です。

## 設定

```json
{
  "channels": {
    "wecom": {
      "enabled": true,
      "token": "YOUR_TOKEN",
      "encoding_aes_key": "YOUR_ENCODING_AES_KEY",
      "webhook_url": "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY",
      "webhook_path": "/webhook/wecom",
      "allow_from": [],
      "reply_timeout": 5
    }
  }
}
```

| フィールド | 型 | 必須 | 説明 |
| ---------------- | ------ | ---- | -------------------------------------------- |
| token | string | はい | 署名検証トークン |
| encoding_aes_key | string | はい | 復号化に使用する43文字のAESキー |
| webhook_url | string | はい | 返信送信に使用する企業WeChatグループボットのWebhook URL |
| webhook_path | string | いいえ | Webhookエンドポイントパス（デフォルト：/webhook/wecom） |
| allow_from | array | いいえ | ユーザーIDの許可リスト（空 = 全ユーザーを許可） |
| reply_timeout | int | いいえ | 返信タイムアウト（秒、デフォルト：5） |

## セットアップ手順

1. 企業WeChatグループにボットを追加
2. Webhook URLを取得
3. （メッセージを受信する場合）ボット設定ページでメッセージ受信APIアドレス（コールバックURL）、Token、EncodingAESKeyを設定
4. 関連情報を設定ファイルに入力

   注意：PicoClawは現在、すべてのチャンネルのwebhookコールバックを受信するために共有のGateway HTTPサーバーを使用しています。デフォルトのリスニングアドレスは127.0.0.1:18790です。公共インターネットからコールバックを受信するには、外部ドメインをGateway（デフォルトポート18790）にリバースプロキシしてください。
