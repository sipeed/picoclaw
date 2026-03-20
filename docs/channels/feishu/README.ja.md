> [README](../../../README.ja.md) に戻る

# 飛書（Feishu）

飛書（国際名：Lark）は ByteDance が提供するエンタープライズコラボレーションプラットフォームです。イベント駆動型の WebSocket 接続を通じて、中国および世界市場の両方をサポートします。

## 設定

```json
{
  "channels": {
    "feishu": {
      "enabled": true,
      "app_id": "cli_xxx",
      "app_secret": "xxx",
      "encrypt_key": "",
      "verification_token": "",
      "allow_from": []
    }
  }
}
```

| フィールド            | 型     | 必須   | 説明                                                              |
| --------------------- | ------ | ------ | ----------------------------------------------------------------- |
| enabled               | bool   | はい   | 飛書チャンネルを有効にするかどうか                                |
| app_id                | string | はい   | 飛書アプリケーションの App ID（`cli_` で始まる）                  |
| app_secret            | string | はい   | 飛書アプリケーションの App Secret                                 |
| encrypt_key           | string | いいえ | イベントコールバックの暗号化キー                                  |
| verification_token    | string | いいえ | Webhook イベント検証に使用するトークン                            |
| allow_from            | array  | いいえ | 許可するユーザーIDのリスト。空の場合はすべてのユーザーを許可     |
| random_reaction_emoji | array  | いいえ | ランダムに追加する絵文字のリスト。空の場合はデフォルトの "Pin" を使用 |

## セットアップ手順

1. [飛書オープンプラットフォーム](https://open.feishu.cn/) にアクセスしてアプリケーションを作成する
2. アプリケーション設定で**ボット**機能を有効にする
3. バージョンを作成してアプリケーションを公開する（公開後に設定が有効になる）
4. **App ID**（`cli_` で始まる）と **App Secret** を取得する
5. PicoClaw 設定ファイルに App ID と App Secret を入力する
6. `picoclaw gateway` を実行してサービスを起動する
7. 飛書でボット名を検索して会話を始める

> PicoClaw は WebSocket/SDK モードで飛書に接続するため、公開コールバックアドレスや Webhook URL の設定は不要です。
>
> `encrypt_key` と `verification_token` はオプションですが、本番環境ではイベント暗号化を有効にすることを推奨します。
>
> カスタム絵文字の参考：[飛書絵文字リスト](https://open.larkoffice.com/document/server-docs/im-v1/message-reaction/emojis-introduce)
