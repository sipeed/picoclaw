> [README](../../project/README.ja.md) に戻る

# DingTalk

DingTalkはアリババの企業向けコミュニケーションプラットフォームで、中国のビジネス環境で広く利用されています。ストリーミング SDK を使用して持続的な接続を維持します。

## 設定

```json
{
  "channel_list": {
    "dingtalk": {
      "enabled": true,
      "type": "dingtalk",
      "client_id": "YOUR_CLIENT_ID",
      "client_secret": "YOUR_CLIENT_SECRET",
      "allow_from": []
    }
  }
}
```

| フィールド    | 型     | 必須 | 説明                                         |
| ------------- | ------ | ---- | -------------------------------------------- |
| enabled       | bool   | はい | DingTalk チャンネルを有効にするかどうか      |
| client_id     | string | はい | DingTalk アプリケーションの Client ID        |
| client_secret | string | はい | DingTalk アプリケーションの Client Secret    |
| allow_from    | array  | いいえ | ユーザーIDのホワイトリスト。空の場合は全ユーザーを許可 |

## セットアップ手順

1. [DingTalk オープンプラットフォーム](https://open.dingtalk.com/) にアクセスする
2. 企業内部アプリケーションを作成する
3. アプリケーション設定から Client ID と Client Secret を取得する
4. OAuth とイベントサブスクリプションを設定する（必要な場合）
5. Client ID と Client Secret を設定ファイルに入力する
