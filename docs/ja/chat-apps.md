# 💬 チャットアプリ設定

> [README](../../README.ja.md) に戻る

## 💬 チャットアプリ連携

PicoClaw は複数のチャットプラットフォームをサポートしており、Agent をどこにでも接続できます。

> **注意**: すべての Webhook ベースのチャネル（LINE、WeCom など）は、共有 Gateway HTTP サーバー（`gateway.host`:`gateway.port`、デフォルト `127.0.0.1:18790`）上で提供されます。チャネルごとにポートを設定する必要はありません。注意：飛書（Feishu）は WebSocket/SDK モードを使用し、共有 HTTP Webhook サーバーは使用しません。

### チャネル一覧

| チャネル             | セットアップ難易度 | 特徴                                      | ドキュメント                                                                                                    |
| -------------------- | ------------------ | ----------------------------------------- | --------------------------------------------------------------------------------------------------------------- |
| **Telegram**         | ⭐ 簡単            | 推奨、音声テキスト変換対応、ロングポーリング（公開 IP 不要） | [ドキュメント](../channels/telegram/README.zh.md)                                                             |
| **Discord**          | ⭐ 簡単            | Socket Mode、グループ/DM 対応、Bot エコシステム充実 | [ドキュメント](../channels/discord/README.zh.md)                                                              |
| **WhatsApp**         | ⭐ 簡単            | ネイティブ (QR スキャン) または Bridge URL | [ドキュメント](../channels/whatsapp/README.zh.md)                                                             |
| **Slack**            | ⭐ 簡単            | **Socket Mode** (公開 IP 不要)、エンタープライズ対応 | [ドキュメント](../channels/slack/README.zh.md)                                                                |
| **Matrix**           | ⭐⭐ 中程度        | フェデレーションプロトコル、セルフホスト対応 | [ドキュメント](../channels/matrix/README.zh.md)                                                              |
| **QQ**               | ⭐⭐ 中程度        | 公式ボット API、中国コミュニティ向け       | [ドキュメント](../channels/qq/README.zh.md)                                                                   |
| **DingTalk**         | ⭐⭐ 中程度        | Stream モード（公開 IP 不要）、企業向け    | [ドキュメント](../channels/dingtalk/README.zh.md)                                                             |
| **LINE**             | ⭐⭐⭐ やや難      | HTTPS Webhook が必要                       | [ドキュメント](../channels/line/README.zh.md)                                                                 |
| **WeCom (企業微信)** | ⭐⭐⭐ やや難      | グループ Bot (Webhook)、カスタムアプリ (API)、AI Bot 対応 | [Bot](../channels/wecom/wecom_bot/README.zh.md) / [App](../channels/wecom/wecom_app/README.zh.md) / [AI Bot](../channels/wecom/wecom_aibot/README.zh.md) |
| **Feishu (飛書)**    | ⭐⭐⭐ やや難      | エンタープライズコラボレーション、機能豊富 | [ドキュメント](../channels/feishu/README.zh.md)                                                               |
| **IRC**              | ⭐⭐ 中程度        | サーバー + TLS 設定                        | -                                                                                                               |
| **OneBot**           | ⭐⭐ 中程度        | NapCat/Go-CQHTTP 互換、コミュニティエコシステム充実 | [ドキュメント](../channels/onebot/README.zh.md)                                                               |
| **MaixCam**          | ⭐ 簡単            | Sipeed AI カメラハードウェア統合チャネル   | [ドキュメント](../channels/maixcam/README.zh.md)                                                              |
| **Pico**             | ⭐ 簡単            | PicoClaw ネイティブプロトコルチャネル     |                                                                                                               |

---

<details>
<summary><b>Telegram</b>（推奨）</summary>

**1. Bot を作成**

* Telegram を開き、`@BotFather` を検索
* `/newbot` を送信し、プロンプトに従う
* Token をコピー

**2. 設定**

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "YOUR_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"]
    }
  }
}
```

> Telegram の `@userinfobot` から User ID を取得できます。

**3. 実行**

```bash
picoclaw gateway
```

**4. Telegram コマンドメニュー（起動時に自動登録）**

PicoClaw は統一されたコマンド定義を使用します。起動時に Telegram がサポートするコマンド（例: `/start`、`/help`、`/show`、`/list`）を Bot コマンドメニューに自動登録し、メニュー表示と実際の動作を一致させます。
Telegram 側はコマンドメニュー登録機能を保持し、汎用コマンドの実行は Agent Loop 内の commands executor で統一的に処理されます。

ネットワークや API の一時的なエラーで登録に失敗しても、チャネルの起動はブロックされません。システムがバックグラウンドで自動リトライします。

</details>

<details>
<summary><b>Discord</b></summary>

**1. Bot を作成**

* <https://discord.com/developers/applications> にアクセス
* アプリケーションを作成 → Bot → Bot を追加
* Bot Token をコピー

**2. Intents を有効化**

* Bot 設定で **MESSAGE CONTENT INTENT** を有効化
* （オプション）メンバーデータに基づくホワイトリストが必要な場合は **SERVER MEMBERS INTENT** を有効化

**3. User ID を取得**

* Discord 設定 → 詳細設定 → **開発者モード** を有効化
* アバターを右クリック → **ユーザー ID をコピー**

**4. 設定**

```json
{
  "channels": {
    "discord": {
      "enabled": true,
      "token": "YOUR_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"]
    }
  }
}
```

**5. Bot を招待**

* OAuth2 → URL Generator
* Scopes: `bot`
* Bot Permissions: `Send Messages`, `Read Message History`
* 生成された招待リンクを開き、Bot をサーバーに追加

**オプション：グループトリガーモード**

デフォルトでは Bot はサーバーチャネル内のすべてのメッセージに応答します。@メンション時のみ応答するには：

```json
{
  "channels": {
    "discord": {
      "group_trigger": { "mention_only": true }
    }
  }
}
```

キーワードプレフィックスでトリガーすることもできます（例: `!bot`）：

```json
{
  "channels": {
    "discord": {
      "group_trigger": { "prefixes": ["!bot"] }
    }
  }
}
```

**6. 実行**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>WhatsApp</b>（ネイティブ whatsmeow）</summary>

PicoClaw は 2 つの WhatsApp 接続方式をサポートしています：

- **ネイティブ（推奨）：** プロセス内で [whatsmeow](https://github.com/tulir/whatsmeow) を使用。独立した Bridge は不要です。`"use_native": true` に設定し、`bridge_url` を空にします。初回実行時に WhatsApp で QR コードをスキャン（リンクデバイス）。セッションはワークスペース配下（例: `workspace/whatsapp/`）に保存されます。ネイティブチャネルは**オプション**ビルドで、`-tags whatsapp_native` でコンパイルします（例: `make build-whatsapp-native` または `go build -tags whatsapp_native ./cmd/...`）。
- **Bridge：** 外部 WebSocket Bridge に接続。`bridge_url`（例: `ws://localhost:3001`）を設定し、`use_native` を false のままにします。

**設定（ネイティブ）**

```json
{
  "channels": {
    "whatsapp": {
      "enabled": true,
      "use_native": true,
      "session_store_path": "",
      "allow_from": []
    }
  }
}
```

`session_store_path` が空の場合、セッションは `<workspace>/whatsapp/` に保存されます。`picoclaw gateway` を実行し、初回実行時にターミナルに表示される QR コードをスキャンしてください（WhatsApp → リンクデバイス）。

</details>

<details>
<summary><b>Matrix</b></summary>

**1. Bot アカウントを準備**

* お好みの homeserver（例: `https://matrix.org` またはセルフホスト）を使用
* Bot ユーザーを作成し、access token を取得

**2. 設定**

```json
{
  "channels": {
    "matrix": {
      "enabled": true,
      "homeserver": "https://matrix.org",
      "user_id": "@your-bot:matrix.org",
      "access_token": "YOUR_MATRIX_ACCESS_TOKEN",
      "allow_from": []
    }
  }
}
```

**3. 実行**

```bash
picoclaw gateway
```

すべてのオプション（`device_id`、`join_on_invite`、`group_trigger`、`placeholder`、`reasoning_channel_id`）については [Matrix チャネル設定ガイド](../channels/matrix/README.md) を参照してください。

</details>

<details>
<summary><b>QQ</b></summary>

**1. Bot を作成**

- [QQ 開放プラットフォーム](https://q.qq.com/#) にアクセス
- アプリケーションを作成 → **AppID** と **AppSecret** を取得

**2. 設定**

```json
{
  "channels": {
    "qq": {
      "enabled": true,
      "app_id": "YOUR_APP_ID",
      "app_secret": "YOUR_APP_SECRET",
      "allow_from": []
    }
  }
}
```

> `allow_from` を空にするとすべてのユーザーを許可します。QQ 番号を指定してアクセスを制限することもできます。

**3. 実行**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>Slack</b></summary>

**1. Slack App を作成**

* [Slack API](https://api.slack.com/apps) でアプリを作成
* **Socket Mode** を有効化
* **Bot Token** と **App-Level Token** を取得

**2. 設定**

```json
{
  "channels": {
    "slack": {
      "enabled": true,
      "bot_token": "xoxb-YOUR_BOT_TOKEN",
      "app_token": "xapp-YOUR_APP_TOKEN",
      "allow_from": []
    }
  }
}
```

**3. 実行**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>IRC</b></summary>

**1. 設定**

```json
{
  "channels": {
    "irc": {
      "enabled": true,
      "server": "irc.libera.chat:6697",
      "nick": "picoclaw-bot",
      "use_tls": true,
      "channels_to_join": ["#your-channel"],
      "allow_from": []
    }
  }
}
```

**2. 実行**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>DingTalk</b></summary>

**1. Bot を作成**

* [開放プラットフォーム](https://open.dingtalk.com/) にアクセス
* 内部アプリを作成
* Client ID と Client Secret をコピー

**2. 設定**

```json
{
  "channels": {
    "dingtalk": {
      "enabled": true,
      "client_id": "YOUR_CLIENT_ID",
      "client_secret": "YOUR_CLIENT_SECRET",
      "allow_from": []
    }
  }
}
```

> `allow_from` を空にするとすべてのユーザーを許可します。DingTalk ユーザー ID を指定してアクセスを制限することもできます。

**3. 実行**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>LINE</b></summary>

**1. LINE 公式アカウントを作成**

- [LINE Developers Console](https://developers.line.biz/) にアクセス
- Provider を作成 → Messaging API チャネルを作成
- **Channel Secret** と **Channel Access Token** をコピー

**2. 設定**

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

> LINE Webhook は共有 Gateway サーバー（`gateway.host`:`gateway.port`、デフォルト `127.0.0.1:18790`）上で提供されます。

**3. Webhook URL を設定**

LINE は HTTPS Webhook が必要です。リバースプロキシまたはトンネルを使用してください：

```bash
# 例：ngrok を使用（Gateway デフォルトポートは 18790）
ngrok http 18790
```

LINE Developers Console で Webhook URL を `https://your-domain/webhook/line` に設定し、**Use webhook** を有効にしてください。

**4. 実行**

```bash
picoclaw gateway
```

> グループチャットでは、Bot は @メンション時のみ応答します。返信は元のメッセージを引用します。

</details>

<details>
<summary><b>Feishu (飛書)</b></summary>

**1. アプリを作成**

* [飛書開放プラットフォーム](https://open.feishu.cn/) にアクセス
* 企業カスタムアプリを作成
* **App ID** と **App Secret** を取得

**2. 設定**

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

**3. 実行**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>WeCom (企業微信)</b></summary>

PicoClaw は 3 種類の WeCom 統合をサポートしています：

**方式 1: グループ Bot (Bot)** — セットアップ簡単、グループチャット対応
**方式 2: カスタムアプリ (App)** — より多機能、プロアクティブメッセージング、プライベートチャットのみ
**方式 3: AI Bot** — 公式 AI Bot、ストリーミング返信、グループ・プライベートチャット対応

詳細なセットアップ手順は [WeCom AI Bot 設定ガイド](../channels/wecom/wecom_aibot/README.zh.md) を参照してください。

**クイックセットアップ — グループ Bot：**

**1. Bot を作成**

* WeCom 管理コンソール → グループチャット → グループ Bot を追加
* Webhook URL をコピー（形式：`https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx`）

**2. 設定**

```json
{
  "channels": {
    "wecom": {
      "enabled": true,
      "token": "YOUR_TOKEN",
      "encoding_aes_key": "YOUR_ENCODING_AES_KEY",
      "webhook_url": "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY",
      "webhook_path": "/webhook/wecom",
      "allow_from": []
    }
  }
}
```

> WeCom Webhook は共有 Gateway サーバー（`gateway.host`:`gateway.port`、デフォルト `127.0.0.1:18790`）上で提供されます。

**クイックセットアップ — カスタムアプリ：**

**1. アプリを作成**

* WeCom 管理コンソール → アプリ管理 → アプリを作成
* **AgentId** と **Secret** をコピー
* 「マイ企業」ページで **CorpID** をコピー

**2. メッセージ受信を設定**

* アプリ詳細で「メッセージ受信」→「API を設定」をクリック
* URL を `http://your-server:18790/webhook/wecom-app` に設定
* **Token** と **EncodingAESKey** を生成

**3. 設定**

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
      "allow_from": []
    }
  }
}
```

**4. 実行**

```bash
picoclaw gateway
```

> **注意**: WeCom Webhook コールバックは Gateway ポート（デフォルト 18790）で提供されます。HTTPS にはリバースプロキシを使用してください。

**クイックセットアップ — AI Bot：**

**1. AI Bot を作成**

* WeCom 管理コンソール → アプリ管理 → AI Bot
* AI Bot 設定でコールバック URL を設定：`http://your-server:18791/webhook/wecom-aibot`
* **Token** をコピーし、「ランダム生成」をクリックして **EncodingAESKey** を取得

**2. 設定**

```json
{
  "channels": {
    "wecom_aibot": {
      "enabled": true,
      "token": "YOUR_TOKEN",
      "encoding_aes_key": "YOUR_43_CHAR_ENCODING_AES_KEY",
      "webhook_path": "/webhook/wecom-aibot",
      "allow_from": [],
      "welcome_message": "こんにちは！何かお手伝いできますか？"
    }
  }
}
```

**3. 実行**

```bash
picoclaw gateway
```

> **注意**: WeCom AI Bot はストリーミングプルプロトコルを使用しており、返信タイムアウトの心配はありません。長時間タスク（30 秒超）は自動的に `response_url` プッシュ配信に切り替わります。

</details>

<details>
<summary><b>OneBot</b></summary>

**1. 設定**

NapCat / Go-CQHTTP などの OneBot 実装と互換性があります。

```json
{
  "channels": {
    "onebot": {
      "enabled": true,
      "allow_from": []
    }
  }
}
```

**2. 実行**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>MaixCam</b></summary>

Sipeed AI カメラハードウェア向けの統合チャネルです。

```json
{
  "channels": {
    "maixcam": {
      "enabled": true
    }
  }
}
```

```bash
picoclaw gateway
```

</details>
