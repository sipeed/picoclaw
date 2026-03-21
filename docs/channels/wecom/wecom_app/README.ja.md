> [README](../../../../README.ja.md) に戻る

# 企業WeChat 自社開発アプリ

企業WeChat 自社開発アプリとは、企業が企業WeChat内で作成するアプリケーションで、主に社内利用を目的としています。企業WeChat 自社開発アプリを通じて、企業は従業員との効率的なコミュニケーションと協業を実現し、業務効率を向上させることができます。

## 設定

```json
{
  "channels": {
    "wecom_app": {
      "enabled": true,
      "corp_id": "wwxxxxxxxxxxxxxxxx",
      "corp_secret": "YOUR_CORP_SECRET",
      "agent_id": 1000002,
      "token": "YOUR_TOKEN",
      "encoding_aes_key": "YOUR_ENCODING_AES_KEY",
      "webhook_path": "/webhook/wecom-app",
      "allow_from": [],
      "reply_timeout": 5
    }
  }
}
```

| フィールド | 型 | 必須 | 説明 |
| ---------------- | ------ | ---- | ---------------------------------------- |
| corp_id | string | はい | 企業ID |
| corp_secret | string | はい | アプリケーションシークレット |
| agent_id | int | はい | アプリケーションエージェントID |
| token | string | はい | コールバック検証トークン |
| encoding_aes_key | string | はい | 43文字のAESキー |
| webhook_path | string | いいえ | Webhookパス（デフォルト：/webhook/wecom-app） |
| allow_from | array | いいえ | ユーザーIDの許可リスト |
| reply_timeout | int | いいえ | 返信タイムアウト（秒） |

## セットアップ手順

1. [企業WeChat管理コンソール](https://work.weixin.qq.com/) にログイン
2. 「アプリ管理」→「アプリを作成」に進む
3. 企業ID（CorpID）とアプリのSecretを取得
4. アプリ設定で「メッセージ受信」を設定し、TokenとEncodingAESKeyを取得
5. コールバックURLを `http://<your-server-ip>:<port>/webhook/wecom-app` に設定
6. CorpID、Secret、AgentIDなどの情報を設定ファイルに入力

   注意：PicoClawは現在、すべてのチャンネルのwebhookコールバックを受信するために共有のGateway HTTPサーバーを使用しています。デフォルトのリスニングアドレスは127.0.0.1:18790です。公共インターネットからコールバックを受信するには、外部ドメインをGateway（デフォルトポート18790）にリバースプロキシしてください。
