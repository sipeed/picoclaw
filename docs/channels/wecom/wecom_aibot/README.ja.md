> [README](../../../../README.ja.md) に戻る

# 企業WeChat AIボット

企業WeChat AIボット（AI Bot）は、企業WeChatが公式に提供するAI会話連携方式です。プライベートチャットとグループチャットの両方をサポートし、ストリーミングレスポンスプロトコルを内蔵しており、タイムアウト後に `response_url` を通じて最終返信をプッシュする機能もサポートしています。

## 他のWeCom チャンネルとの比較

| 機能 | WeCom Bot | WeCom App | **WeCom AI Bot** |
|------|-----------|-----------|-----------------|
| プライベートチャット | ✅ | ✅ | ✅ |
| グループチャット | ✅ | ❌ | ✅ |
| ストリーミング出力 | ❌ | ❌ | ✅ |
| タイムアウト時のプッシュ | ❌ | ✅ | ✅ |
| 設定の複雑さ | 低 | 高 | 中 |

## 設定

```json
{
  "channels": {
    "wecom_aibot": {
      "enabled": true,
      "token": "YOUR_TOKEN",
      "encoding_aes_key": "YOUR_43_CHAR_ENCODING_AES_KEY",
      "webhook_path": "/webhook/wecom-aibot",
      "allow_from": [],
      "welcome_message": "你好！有什么可以帮助你的吗？",
      "max_steps": 10
    }
  }
}
```

| フィールド | 型 | 必須 | 説明 |
| ---------------- | ------ | ---- | -------------------------------------------------- |
| token | string | はい | コールバック検証トークン。AIボット管理ページで設定 |
| encoding_aes_key | string | はい | 43文字のAESキー。AIボット管理ページでランダム生成 |
| webhook_path | string | いいえ | Webhookパス（デフォルト：/webhook/wecom-aibot） |
| allow_from | array | いいえ | ユーザーIDの許可リスト。空配列は全ユーザーを許可 |
| welcome_message | string | いいえ | ユーザーがチャットを開いたときに送信するウェルカムメッセージ。空白の場合は送信しない |
| reply_timeout | int | いいえ | 返信タイムアウト（秒、デフォルト：5） |
| max_steps | int | いいえ | エージェントの最大実行ステップ数（デフォルト：10） |

## セットアップ手順

1. [企業WeChat管理コンソール](https://work.weixin.qq.com/wework_admin) にログイン
2. 「アプリ管理」→「AIボット」に進み、AIボットを作成または選択
3. AIボット設定ページで「メッセージ受信」情報を入力：
   - **URL**：`http://<your-server-ip>:18790/webhook/wecom-aibot`
   - **Token**：ランダム生成またはカスタム
   - **EncodingAESKey**：「ランダム生成」をクリックして43文字のキーを取得
4. TokenとEncodingAESKeyをPicoClawの設定ファイルに入力し、サービスを起動してから管理コンソールに戻って保存（企業WeChatが検証リクエストを送信します）

> [!TIP]
> サーバーは企業WeChatのサーバーからアクセス可能である必要があります。イントラネットやローカル開発環境の場合は、[ngrok](https://ngrok.com) またはfrpを使用してトンネリングしてください。

## ストリーミングレスポンスプロトコル

WeCom AIボットは「ストリーミングプル」プロトコルを使用しており、通常のWebhookの一回限りの返信とは異なります：

```
ユーザーがメッセージを送信
  │
  ▼
PicoClawが即座に {finish: false} を返す（エージェントが処理開始）
  │
  ▼
企業WeChatが約1秒ごとに {msgtype: "stream", stream: {id: "..."}} でプル
  │
  ├─ エージェント未完了 → {finish: false} を返す（待機継続）
  │
  └─ エージェント完了 → {finish: true, content: "返信内容"} を返す
```

**タイムアウト処理**（タスクが30秒を超える場合）：

エージェントの処理時間が約30秒を超えた場合（企業WeChatの最大ポーリングウィンドウは6分）、PicoClawは：

1. 即座にストリームを閉じ、ユーザーに「⏳ 正在处理中，请稍候，结果将稍后发送。」と表示
2. エージェントはバックグラウンドで処理を継続
3. エージェント完了後、メッセージに含まれる `response_url` を通じて最終返信をユーザーにプッシュ

> `response_url` は企業WeChatが発行し、有効期限は1時間、使用は1回限りで、暗号化不要。マークダウンメッセージ本文をそのままPOSTするだけです。

## ウェルカムメッセージ

`welcome_message` を設定すると、ユーザーがAIボットとのチャットウィンドウを開いたとき（`enter_chat` イベント）に、PicoClawが自動的にそのメッセージを返信します。空白の場合は無視されます。

```json
"welcome_message": "你好！我是 PicoClaw AI 助手，有什么可以帮你？"
```

## よくある質問

### コールバックURL検証の失敗

- サーバーのファイアウォールで該当ポートが開放されているか確認（デフォルト18790）
- `token` と `encoding_aes_key` が正しく入力されているか確認
- PicoClawのログに企業WeChatからのGETリクエストが届いているか確認

### メッセージに返信がない

- `allow_from` が誤って送信者を制限していないか確認
- ログに `context canceled` またはエージェントエラーが出ていないか確認
- エージェント設定（`model_name` など）が正しいか確認

### 長時間タスクで最終プッシュが届かない

- メッセージコールバックに `response_url` が含まれているか確認（新バージョンの企業WeChat AIボットのみ対応）
- サーバーが外部ネットワークへのアウトバウンドリクエストを送信できるか確認（`response_url` へのPOSTが必要）
- ログのキーワード `response_url mode` と `Sending reply via response_url` を確認

## 参考ドキュメント

- [企業WeChat AIボット連携ドキュメント](https://developer.work.weixin.qq.com/document/path/100719)
- [ストリーミングレスポンスプロトコルの説明](https://developer.work.weixin.qq.com/document/path/100719)
- [response_url によるプロアクティブ返信](https://developer.work.weixin.qq.com/document/path/101138)
