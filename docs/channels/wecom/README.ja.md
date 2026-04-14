> [README](../../../README.ja.md) に戻る

# WeCom

PicoClaw は WeCom を公式 WeCom AI Bot WebSocket API に基づく単一の `channels.wecom` チャンネルとして公開します。
従来の `wecom`、`wecom_app`、`wecom_aibot` の分割を統一された設定モデルに置き換えました。

> パブリックな Webhook コールバック URL は不要です。PicoClaw は WeCom へのアウトバウンド WebSocket 接続を確立します。

## サポートされる機能

- ダイレクトチャットとグループチャット
- WeCom AI Bot プロトコルによるチャンネル側ストリーミング返信
- テキスト、音声、画像、ファイル、動画、ミックスメッセージの受信
- テキストおよびメディア返信の送信（`image`、`file`、`voice`、`video`）
- Web UI または CLI による QR コードオンボーディング
- 共有許可リストと `reasoning_channel_id` ルーティング

---

## クイックスタート

### オプション 1：Web UI QR バインディング（推奨）

Web UI を開き、**Channels → WeCom** に移動して、QR バインディングボタンをクリックします。WeCom で QR コードをスキャンし、アプリ内で確認すると、認証情報が自動的に保存されます。

<p align="center">
<img src="../../../assets/wecom-qr-binding.jpg" alt="Web UI での WeCom QR バインディング" width="600">
</p>

### オプション 2：CLI QR ログイン

実行：

```bash
picoclaw auth wecom
```

コマンドの動作：
1. WeCom に QR コードをリクエストし、ターミナルに表示します
2. ターミナルの QR コードがスキャンしにくい場合に備え、ブラウザで開ける **QR コードリンク** も表示します
3. 確認をポーリングします — スキャン後、**WeCom アプリ内でログインを確認** する必要があります
4. 成功すると、`bot_id` と `secret` を `channels.wecom` に書き込み、設定を保存します

デフォルトのタイムアウトは **5 分** です。`--timeout` で延長できます：

```bash
picoclaw auth wecom --timeout 10m
```

> ⚠️ QR コードのスキャンだけでは不十分です — WeCom アプリ内で **確認** をタップする必要があります。そうしないとコマンドがタイムアウトします。

### オプション 3：手動設定

WeCom AI Bot プラットフォームから `bot_id` と `secret` を既にお持ちの場合、直接設定できます：

```json
{
  "channels": {
    "wecom": {
      "enabled": true,
      "bot_id": "YOUR_BOT_ID",
      "secret": "YOUR_SECRET",
      "websocket_url": "wss://openws.work.weixin.qq.com",
      "send_thinking_message": true,
      "allow_from": [],
      "reasoning_channel_id": ""
    }
  }
}
```

---

## 設定

| フィールド | 型 | デフォルト | 説明 |
| ---------- | -- | ---------- | ---- |
| `enabled` | bool | `false` | WeCom チャンネルを有効にする。 |
| `bot_id` | string | — | WeCom AI Bot 識別子。有効時に必須。 |
| `secret` | string | — | WeCom AI Bot シークレット。`.security.yml` に暗号化して保存。有効時に必須。 |
| `websocket_url` | string | `wss://openws.work.weixin.qq.com` | WeCom WebSocket エンドポイント。 |
| `send_thinking_message` | bool | `true` | ストリーミング返信の開始前に `Processing...` メッセージを送信する。 |
| `allow_from` | array | `[]` | 送信者許可リスト。空の場合はすべての送信者を許可。 |
| `reasoning_channel_id` | string | `""` | 推論・思考出力を別の会話にルーティングするためのオプションのチャット ID。 |

### 環境変数

すべてのフィールドは `PICOCLAW_CHANNELS_WECOM_` プレフィックスの環境変数で上書きできます：

| 環境変数 | 対応フィールド |
| -------- | -------------- |
| `PICOCLAW_CHANNELS_WECOM_ENABLED` | `enabled` |
| `PICOCLAW_CHANNELS_WECOM_BOT_ID` | `bot_id` |
| `PICOCLAW_CHANNELS_WECOM_SECRET` | `secret` |
| `PICOCLAW_CHANNELS_WECOM_WEBSOCKET_URL` | `websocket_url` |
| `PICOCLAW_CHANNELS_WECOM_SEND_THINKING_MESSAGE` | `send_thinking_message` |
| `PICOCLAW_CHANNELS_WECOM_ALLOW_FROM` | `allow_from` |
| `PICOCLAW_CHANNELS_WECOM_REASONING_CHANNEL_ID` | `reasoning_channel_id` |

---

## ランタイム動作

- PicoClaw はアクティブな WeCom ターンを維持し、可能な限り同じストリームでストリーミング返信を継続します。
- ストリーミング返信の最大持続時間は **5.5 分**、最小送信間隔は **500ms** です。
- ストリーミングが利用できなくなった場合、返信はアクティブプッシュ配信にフォールバックします。
- チャットルートの関連付けは **30 分** の非アクティブ後に期限切れになります。
- 受信メディアはエージェントに渡される前にローカルメディアストアにダウンロードされます。
- 送信メディアは WeCom に一時ファイルとしてアップロードされ、メディアメッセージとして送信されます。
- 重複メッセージは検出され抑制されます（最新 1000 件のメッセージ ID のリングバッファ）。

---

## レガシー WeCom 設定からの移行

| 以前の設定 | 移行方法 |
| ---------- | -------- |
| `channels.wecom`（Webhook ボット） | `bot_id` + `secret` を使用する `channels.wecom` に置き換える。 |
| `channels.wecom_app` | 削除して `channels.wecom` を使用する。 |
| `channels.wecom_aibot` | `bot_id` と `secret` を `channels.wecom` に移動する。 |
| `token`、`encoding_aes_key`、`webhook_url`、`webhook_path` | 使用されなくなりました。設定から削除してください。 |
| `corp_id`、`corp_secret`、`agent_id` | 使用されなくなりました。設定から削除してください。 |
| `welcome_message`、`processing_message`、`max_steps` | WeCom チャンネル設定の一部ではなくなりました。 |

---

## トラブルシューティング

### QR バインディングがタイムアウトする

- QR コードをスキャンした後、**WeCom アプリ内でログインを確認** する必要があります。スキャンだけでは不十分です。
- より長い `--timeout` で再実行してください：`picoclaw auth wecom --timeout 10m`
- ターミナルの QR コードがスキャンしにくい場合は、その下に表示される **QR コードリンク** を使用してブラウザで開いてください。

### QR コードの有効期限切れ

- QR コードには有効期限があります。`picoclaw auth wecom` を再実行して新しいものを取得してください。

### WebSocket 接続の失敗

- `bot_id` と `secret` が正しいことを確認してください。
- ホストが `wss://openws.work.weixin.qq.com` に到達できることを確認してください（アウトバウンド WebSocket、インバウンドポートは不要）。

### 返信が届かない

- `allow_from` が送信者をブロックしていないか確認してください。
- `channels.wecom.bot_id` と `channels.wecom.secret` が設定されており、空でないことを確認してください。
