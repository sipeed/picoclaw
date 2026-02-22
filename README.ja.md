<div align="center">
<img src="assets/logo.jpg" alt="PicoClaw" width="512">

<h1>PicoClaw: Go で書かれた超効率 AI アシスタント</h1>

<h3>$10 ハードウェア · 10MB RAM · 1秒起動 · 行くぜ、シャコ！</h3>
<h3></h3>

<p>
<img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
<img src="https://img.shields.io/badge/Arch-x86__64%2C%20ARM64%2C%20RISC--V-blue" alt="Hardware">
<img src="https://img.shields.io/badge/license-MIT-green" alt="License">
</p>

[中文](README.zh.md) | **日本語** | [Português](README.pt-br.md) | [Tiếng Việt](README.vi.md) | [Français](README.fr.md) | [English](README.md)

</div>


---

🦐 PicoClaw は [nanobot](https://github.com/HKUDS/nanobot) にインスパイアされた超軽量パーソナル AI アシスタントです。Go でゼロからリファクタリングされ、AI エージェント自身がアーキテクチャの移行とコード最適化を推進するセルフブートストラッピングプロセスで構築されました。

⚡️ $10 のハードウェアで 10MB 未満の RAM で動作：OpenClaw より 99% 少ないメモリ、Mac mini より 98% 安い！

<table align="center">
  <tr align="center">
    <td align="center" valign="top">
      <p align="center">
        <img src="assets/picoclaw_mem.gif" width="360" height="240">
      </p>
    </td>
    <td align="center" valign="top">
      <p align="center">
        <img src="assets/licheervnano.png" width="400" height="240">
      </p>
    </td>
  </tr>
</table>

## 📢 ニュース
2026-02-09 🎉 PicoClaw リリース！$10 ハードウェアで 10MB 未満の RAM で動く AI エージェントを 1 日で構築。🦐 行くぜ、シャコ！

## ✨ 特徴

🪶 **超軽量**: メモリフットプリント 10MB 未満 — Clawdbot のコア機能より 99% 小さい。

💰 **最小コスト**: $10 ハードウェアで動作 — Mac mini より 98% 安い。

⚡️ **超高速**: 起動時間 400 倍高速、0.6GHz シングルコアでも 1 秒で起動。

🌍 **真のポータビリティ**: RISC-V、ARM、x86 対応の単一バイナリ。ワンクリックで Go！

🤖 **AI ブートストラップ**: 自律的な Go ネイティブ実装 — コアの 95% が AI 生成、人間によるレビュー付き。

|  | OpenClaw  | NanoBot | **PicoClaw** |
| --- | --- | --- |--- |
| **言語** | TypeScript | Python | **Go** |
| **RAM** | >1GB |>100MB| **< 10MB** |
| **起動時間**</br>(0.8GHz コア) | >500秒 | >30秒 |  **<1秒** |
| **コスト** | Mac Mini 599$ | 大半の Linux SBC </br>~50$ |**あらゆる Linux ボード**</br>**最安 10$** |
<img src="assets/compare.jpg" alt="PicoClaw" width="512">


## 🦾 デモンストレーション
### 🛠️ スタンダードアシスタントワークフロー
<table align="center">
  <tr align="center">
    <th><p align="center">🧩 フルスタックエンジニア</p></th>
    <th><p align="center">🗂️ ログ＆計画管理</p></th>
    <th><p align="center">🔎 Web 検索＆学習</p></th>
  </tr>
  <tr>
    <td align="center"><p align="center"><img src="assets/picoclaw_code.gif" width="240" height="180"></p></td>
    <td align="center"><p align="center"><img src="assets/picoclaw_memory.gif" width="240" height="180"></p></td>
    <td align="center"><p align="center"><img src="assets/picoclaw_search.gif" width="240" height="180"></p></td>
  </tr>
  <tr>
    <td align="center">開発 · デプロイ · スケール</td>
    <td align="center">スケジュール · 自動化 · メモリ</td>
    <td align="center">発見 · インサイト · トレンド</td>
  </tr>
</table>

### 🐜 革新的な省フットプリントデプロイ
PicoClaw はほぼすべての Linux デバイスにデプロイできます！

- $9.9 [LicheeRV-Nano](https://www.aliexpress.com/item/1005006519668532.html) E(Ethernet) または W(WiFi6) バージョン、最小ホームアシスタントに
- $30~50 [NanoKVM](https://www.aliexpress.com/item/1005007369816019.html) または $100 [NanoKVM-Pro](https://www.aliexpress.com/item/1005010048471263.html) サーバー自動メンテナンスに
- $50 [MaixCAM](https://www.aliexpress.com/item/1005008053333693.html) または $100 [MaixCAM2](https://www.kickstarter.com/projects/zepan/maixcam2-build-your-next-gen-4k-ai-camera) スマート監視に

https://private-user-images.githubusercontent.com/83055338/547056448-e7b031ff-d6f5-4468-bcca-5726b6fecb5c.mp4

🌟 もっと多くのデプロイ事例が待っています！

## 📦 インストール

### コンパイル済みバイナリでインストール

[リリースページ](https://github.com/sipeed/picoclaw/releases) からお使いのプラットフォーム用のファームウェアをダウンロードしてください。

### ソースからインストール（最新機能、開発向け推奨）

```bash
git clone https://github.com/sipeed/picoclaw.git

cd picoclaw
make deps

# ビルド（インストール不要）
make build

# 複数プラットフォーム向けビルド
make build-all

# ビルドとインストール
make install
```

## 🐳 Docker Compose

Docker Compose を使えば、ローカルにインストールせずに PicoClaw を実行できます。

```bash
# 1. リポジトリをクローン
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw

# 2. API キーを設定
cp config/config.example.json config/config.json
vim config/config.json      # DISCORD_BOT_TOKEN, プロバイダーの API キーを設定

# 3. ビルドと起動
docker compose --profile gateway up -d

# 4. ログ確認
docker compose logs -f picoclaw-gateway

# 5. 停止
docker compose --profile gateway down
```

### Agent モード（ワンショット）

```bash
# 質問を投げる
docker compose run --rm picoclaw-agent -m "What is 2+2?"

# インタラクティブモード
docker compose run --rm picoclaw-agent
```

### リビルド

```bash
docker compose --profile gateway build --no-cache
docker compose --profile gateway up -d
```

### 🚀 クイックスタート（ネイティブ）

> [!TIP]
> `~/.picoclaw/config.json` に API キーを設定してください。
> API キーの取得先: [OpenRouter](https://openrouter.ai/keys) (LLM) · [Zhipu](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) (LLM)
> Web 検索は **任意** です - 無料の [Tavily API](https://tavily.com) (月 1000 クエリ無料) または [Brave Search API](https://brave.com/search/api) (月 2000 クエリ無料)

**1. 初期化**

```bash
picoclaw onboard
```

**2. 設定** (`~/.picoclaw/config.json`)

```json
{
  "model_list": [
    {
      "model_name": "gpt4",
      "model": "openai/gpt-5.2",
      "api_key": "sk-your-openai-key",
      "api_base": "https://api.openai.com/v1"
    }
  ],
  "agents": {
    "defaults": {
      "model": "gpt4"
    }
  },
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "YOUR_TELEGRAM_BOT_TOKEN",
      "allow_from": []
    }
  },
  "tools": {
    "web": {
      "search": {
        "api_key": "YOUR_BRAVE_API_KEY",
        "max_results": 5
      },
      "tavily": {
        "enabled": false,
        "api_key": "YOUR_TAVILY_API_KEY",
        "max_results": 5
      }
    },
    "cron": {
      "exec_timeout_minutes": 5
    }
  },
  "heartbeat": {
    "enabled": true,
    "interval": 30
  }
}
```

**3. API キーの取得**

- **LLM プロバイダー**: [OpenRouter](https://openrouter.ai/keys) · [Zhipu](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) · [Anthropic](https://console.anthropic.com) · [OpenAI](https://platform.openai.com) · [Gemini](https://aistudio.google.com/api-keys)
- **Web 検索**（任意）: [Tavily](https://tavily.com) - AI エージェント向けに最適化 (月 1000 リクエスト) · [Brave Search](https://brave.com/search/api) - 無料枠あり（月 2000 リクエスト）

> **注意**: 完全な設定テンプレートは `config.example.json` を参照してください。

**4. チャット**

```bash
picoclaw agent -m "What is 2+2?"
```

これだけです！2 分で AI アシスタントが動きます。

---

## 💬 チャットアプリ

Telegram、Discord、QQ、DingTalk、LINE、WeCom で PicoClaw と会話できます

| チャネル | セットアップ |
|---------|------------|
| **Telegram** | 簡単（トークンのみ） |
| **Discord** | 簡単（Bot トークン + Intents） |
| **QQ** | 簡単（AppID + AppSecret） |
| **DingTalk** | 普通（アプリ認証情報） |
| **LINE** | 普通（認証情報 + Webhook URL） |
| **WeCom** | 普通（CorpID + Webhook設定） |

<details>
<summary><b>Telegram</b>（推奨）</summary>

**1. Bot を作成**

- Telegram を開き、`@BotFather` を検索
- `/newbot` を送信、プロンプトに従う
- トークンをコピー

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

> ユーザー ID は Telegram の `@userinfobot` から取得できます。

**3. 起動**

```bash
picoclaw gateway
```
</details>


<details>
<summary><b>Discord</b></summary>

**1. Bot を作成**
- https://discord.com/developers/applications にアクセス
- アプリケーションを作成 → Bot → Add Bot
- Bot トークンをコピー

**2. Intents を有効化**
- Bot の設定画面で **MESSAGE CONTENT INTENT** を有効化
- （任意）**SERVER MEMBERS INTENT** も有効化

**3. ユーザー ID を取得**
- Discord 設定 → 詳細設定 → **開発者モード** を有効化
- 自分のアバターを右クリック → **ユーザーIDをコピー**

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
- OAuth2 → URL Generator
- Scopes: `bot`
- Bot Permissions: `Send Messages`, `Read Message History`
- 生成された招待 URL を開き、サーバーに Bot を追加

**6. 起動**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>QQ</b></summary>

**1. Bot を作成**

- [QQ オープンプラットフォーム](https://q.qq.com/#) にアクセス
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

> `allow_from` を空にすると全ユーザーを許可、QQ番号を指定してアクセス制限可能。

**3. 起動**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>DingTalk</b></summary>

**1. Bot を作成**

- [オープンプラットフォーム](https://open.dingtalk.com/) にアクセス
- 内部アプリを作成
- Client ID と Client Secret をコピー

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

> `allow_from` を空にすると全ユーザーを許可、ユーザーIDを指定してアクセス制限可能。

**3. 起動**

```bash
picoclaw gateway
```

</details>

<details>
<summary><b>LINE</b></summary>

**1. LINE 公式アカウントを作成**

- [LINE Developers Console](https://developers.line.biz/) にアクセス
- プロバイダーを作成 → Messaging API チャネルを作成
- **チャネルシークレット** と **チャネルアクセストークン** をコピー

**2. 設定**

```json
{
  "channels": {
    "line": {
      "enabled": true,
      "channel_secret": "YOUR_CHANNEL_SECRET",
      "channel_access_token": "YOUR_CHANNEL_ACCESS_TOKEN",
      "webhook_host": "0.0.0.0",
      "webhook_port": 18791,
      "webhook_path": "/webhook/line",
      "allow_from": []
    }
  }
}
```

**3. Webhook URL を設定**

LINE の Webhook には HTTPS が必要です。リバースプロキシまたはトンネルを使用してください:

```bash
# ngrok の例
ngrok http 18791
```

LINE Developers Console で Webhook URL を `https://あなたのドメイン/webhook/line` に設定し、**Webhook の利用** を有効にしてください。

**4. 起動**

```bash
picoclaw gateway
```

> グループチャットでは @メンション時のみ応答します。返信は元メッセージを引用する形式です。

> **Docker Compose**: `picoclaw-gateway` サービスに `ports: ["18791:18791"]` を追加して Webhook ポートを公開してください。

</details>

<details>
<summary><b>WeCom (企業微信)</b></summary>

PicoClaw は2種類の WeCom 統合をサポートしています：

**オプション1: WeCom Bot (智能ロボット)** - 簡単な設定、グループチャット対応
**オプション2: WeCom App (自作アプリ)** - より多機能、アクティブメッセージング対応

詳細な設定手順は [WeCom App Configuration Guide](docs/wecom-app-configuration.md) を参照してください。

**クイックセットアップ - WeCom Bot:**

**1. ボットを作成**

* WeCom 管理コンソール → グループチャット → グループボットを追加
* Webhook URL をコピー（形式: `https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx`）

**2. 設定**

```json
{
  "channels": {
    "wecom": {
      "enabled": true,
      "token": "YOUR_TOKEN",
      "encoding_aes_key": "YOUR_ENCODING_AES_KEY",
      "webhook_url": "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY",
      "webhook_host": "0.0.0.0",
      "webhook_port": 18793,
      "webhook_path": "/webhook/wecom",
      "allow_from": []
    }
  }
}
```

**クイックセットアップ - WeCom App:**

**1. アプリを作成**

* WeCom 管理コンソール → アプリ管理 → アプリを作成
* **AgentId** と **Secret** をコピー
* "マイ会社" ページで **CorpID** をコピー

**2. メッセージ受信を設定**

* アプリ詳細で "メッセージを受信" → "APIを設定" をクリック
* URL を `http://your-server:18792/webhook/wecom-app` に設定
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
      "webhook_host": "0.0.0.0",
      "webhook_port": 18792,
      "webhook_path": "/webhook/wecom-app",
      "allow_from": []
    }
  }
}
```

**4. 起動**

```bash
picoclaw gateway
```

> **注意**: WeCom App は Webhook コールバック用にポート 18792 を開放する必要があります。本番環境では HTTPS 用のリバースプロキシを使用してください。

</details>

## ⚙️ 設定

設定ファイル: `~/.picoclaw/config.json`

### ワークスペース構成

PicoClaw は設定されたワークスペース（デフォルト: `~/.picoclaw/workspace`）にデータを保存します：

```
~/.picoclaw/workspace/
├── sessions/          # 会話セッションと履歴
├── memory/            # 長期メモリ（MEMORY.md）
├── state/             # 永続状態（最後のチャネルなど）
├── cron/              # スケジュールジョブデータベース
├── skills/            # カスタムスキル
├── AGENTS.md          # エージェントの行動ガイド
├── HEARTBEAT.md       # 定期タスクプロンプト（30分ごとに確認）
├── IDENTITY.md        # エージェントのアイデンティティ
├── SOUL.md            # エージェントのソウル
├── TOOLS.md           # ツールの説明
└── USER.md            # ユーザー設定
```

### 🔒 セキュリティサンドボックス

PicoClaw はデフォルトでサンドボックス環境で実行されます。エージェントは設定されたワークスペース内のファイルにのみアクセスし、コマンドを実行できます。

#### デフォルト設定

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "restrict_to_workspace": true
    }
  }
}
```

| オプション | デフォルト | 説明 |
|-----------|-----------|------|
| `workspace` | `~/.picoclaw/workspace` | エージェントの作業ディレクトリ |
| `restrict_to_workspace` | `true` | ファイル/コマンドアクセスをワークスペースに制限 |

#### 保護対象ツール

`restrict_to_workspace: true` の場合、以下のツールがサンドボックス化されます：

| ツール | 機能 | 制限 |
|-------|------|------|
| `read_file` | ファイル読み込み | ワークスペース内のファイルのみ |
| `write_file` | ファイル書き込み | ワークスペース内のファイルのみ |
| `list_dir` | ディレクトリ一覧 | ワークスペース内のディレクトリのみ |
| `edit_file` | ファイル編集 | ワークスペース内のファイルのみ |
| `append_file` | ファイル追記 | ワークスペース内のファイルのみ |
| `exec` | コマンド実行 | コマンドパスはワークスペース内である必要あり |

#### exec ツールの追加保護

`restrict_to_workspace: false` でも、`exec` ツールは以下の危険なコマンドをブロックします：

- `rm -rf`, `del /f`, `rmdir /s` — 一括削除
- `format`, `mkfs`, `diskpart` — ディスクフォーマット
- `dd if=` — ディスクイメージング
- `/dev/sd[a-z]` への書き込み — 直接ディスク書き込み
- `shutdown`, `reboot`, `poweroff` — システムシャットダウン
- フォークボム `:(){ :|:& };:`

#### エラー例

```
[ERROR] tool: Tool execution failed
{tool=exec, error=Command blocked by safety guard (path outside working dir)}
```

```
[ERROR] tool: Tool execution failed
{tool=exec, error=Command blocked by safety guard (dangerous pattern detected)}
```

#### 制限の無効化（セキュリティリスク）

エージェントにワークスペース外のパスへのアクセスが必要な場合：

**方法1: 設定ファイル**
```json
{
  "agents": {
    "defaults": {
      "restrict_to_workspace": false
    }
  }
}
```

**方法2: 環境変数**
```bash
export PICOCLAW_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE=false
```

> ⚠️ **警告**: この制限を無効にすると、エージェントはシステム上の任意のパスにアクセスできるようになります。制御された環境でのみ慎重に使用してください。

#### セキュリティ境界の一貫性

`restrict_to_workspace` 設定は、すべての実行パスで一貫して適用されます：

| 実行パス | セキュリティ境界 |
|---------|-----------------|
| メインエージェント | `restrict_to_workspace` ✅ |
| サブエージェント / Spawn | 同じ制限を継承 ✅ |
| ハートビートタスク | 同じ制限を継承 ✅ |

すべてのパスで同じワークスペース制限が適用されます — サブエージェントやスケジュールタスクを通じてセキュリティ境界をバイパスする方法はありません。

### ハートビート（定期タスク）

PicoClaw は自動的に定期タスクを実行できます。ワークスペースに `HEARTBEAT.md` ファイルを作成します：

```markdown
# 定期タスク

- 重要なメールをチェック
- 今後の予定を確認
- 天気予報をチェック
```

エージェントは30分ごと（設定可能）にこのファイルを読み込み、利用可能なツールを使ってタスクを実行します。

#### spawn で非同期タスク実行

時間のかかるタスク（Web検索、API呼び出し）には `spawn` ツールを使って**サブエージェント**を作成します：

```markdown
# 定期タスク

## クイックタスク（直接応答）
- 現在時刻を報告

## 長時間タスク（spawn で非同期）
- AIニュースを検索して要約
- メールをチェックして重要なメッセージを報告
```

**主な特徴:**

| 機能 | 説明 |
|------|------|
| **spawn** | 非同期サブエージェントを作成、ハートビートをブロックしない |
| **独立コンテキスト** | サブエージェントは独自のコンテキストを持ち、セッション履歴なし |
| **message ツール** | サブエージェントは message ツールで直接ユーザーと通信 |
| **非ブロッキング** | spawn 後、ハートビートは次のタスクへ継続 |

#### サブエージェントの通信方法

```
ハートビート発動
    ↓
エージェントが HEARTBEAT.md を読む
    ↓
長いタスク: spawn サブエージェント
    ↓                           ↓
次のタスクへ継続          サブエージェントが独立して動作
    ↓                           ↓
全タスク完了              message ツールを使用
    ↓                           ↓
HEARTBEAT_OK 応答         ユーザーが直接結果を受け取る
```

サブエージェントはツール（message、web_search など）にアクセスでき、メインエージェントを経由せずにユーザーと通信できます。

**設定:**

```json
{
  "heartbeat": {
    "enabled": true,
    "interval": 30
  }
}
```

| オプション | デフォルト | 説明 |
|-----------|-----------|------|
| `enabled` | `true` | ハートビートの有効/無効 |
| `interval` | `30` | チェック間隔（分）、最小5分 |

**環境変数:**
- `PICOCLAW_HEARTBEAT_ENABLED=false` で無効化
- `PICOCLAW_HEARTBEAT_INTERVAL=60` で間隔変更

### プロバイダー

> [!NOTE]
> Groq は Whisper による無料の音声文字起こしを提供しています。設定すると、Telegram の音声メッセージが自動的に文字起こしされます。

| プロバイダー | 用途 | API キー取得先 |
| --- | --- | --- |
| `gemini` | LLM（Gemini 直接） | [aistudio.google.com](https://aistudio.google.com) |
| `zhipu` | LLM（Zhipu 直接） | [bigmodel.cn](https://bigmodel.cn) |
| `openrouter`（要テスト） | LLM（推奨、全モデルにアクセス可能） | [openrouter.ai](https://openrouter.ai) |
| `anthropic`（要テスト） | LLM（Claude 直接） | [console.anthropic.com](https://console.anthropic.com) |
| `openai`（要テスト） | LLM（GPT 直接） | [platform.openai.com](https://platform.openai.com) |
| `deepseek`（要テスト） | LLM（DeepSeek 直接） | [platform.deepseek.com](https://platform.deepseek.com) |
| `groq` | LLM + **音声文字起こし**（Whisper） | [console.groq.com](https://console.groq.com) |
| `cerebras` | LLM（Cerebras 直接） | [cerebras.ai](https://cerebras.ai) |

### 基本設定

1.  **設定ファイルの作成:**

    ```bash
    cp config.example.json config/config.json
    ```

2.  **設定の編集:**

    ```json
    {
      "providers": {
        "openrouter": {
          "api_key": "sk-or-v1-..."
        }
      },
      "channels": {
        "discord": {
          "enabled": true,
          "token": "YOUR_DISCORD_BOT_TOKEN"
        }
      }
    }
    ```

3.  **実行**

    ```bash
    picoclaw agent -m "Hello"
    ```
</details>

<details>
<summary><b>完全な設定例</b></summary>

```json
{
  "agents": {
    "defaults": {
      "model": "anthropic/claude-opus-4-5"
    }
  },
  "providers": {
    "openrouter": {
      "api_key": "sk-or-v1-xxx"
    },
    "groq": {
      "api_key": "gsk_xxx"
    }
  },
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "123456:ABC...",
      "allow_from": ["123456789"]
    },
    "discord": {
      "enabled": true,
      "token": "",
      "allow_from": [""]
    },
    "whatsapp": {
      "enabled": false
    },
    "feishu": {
      "enabled": false,
      "app_id": "cli_xxx",
      "app_secret": "xxx",
      "encrypt_key": "",
      "verification_token": "",
      "allow_from": []
    }
  },
  "tools": {
    "web": {
      "search": {
        "api_key": "BSA..."
      }
    },
    "cron": {
      "exec_timeout_minutes": 5
    }
  },
  "heartbeat": {
    "enabled": true,
    "interval": 30
  }
}
```

</details>

### モデル設定 (model_list)

> **新機能！** PicoClaw は現在 **モデル中心** の設定アプローチを採用しています。`ベンダー/モデル` 形式（例: `zhipu/glm-4.7`）を指定するだけで、新しいプロバイダーを追加できます—**コードの変更は一切不要！**

この設計は、柔軟なプロバイダー選択による **マルチエージェントサポート** も可能にします：

- **異なるエージェント、異なるプロバイダー** : 各エージェントは独自の LLM プロバイダーを使用可能
- **フォールバックモデル** : 耐障性のため、プライマリモデルとフォールバックモデルを設定可能
- **ロードバランシング** : 複数のエンドポイントにリクエストを分散
- **集中設定管理** : すべてのプロバイダーを一箇所で管理

#### 📋 サポートされているすべてのベンダー

| ベンダー | `model` プレフィックス | デフォルト API Base | プロトコル | API キー |
|-------------|-----------------|---------------------|----------|---------|
| **OpenAI** | `openai/` | `https://api.openai.com/v1` | OpenAI | [キーを取得](https://platform.openai.com) |
| **Anthropic** | `anthropic/` | `https://api.anthropic.com/v1` | Anthropic | [キーを取得](https://console.anthropic.com) |
| **Zhipu AI (GLM)** | `zhipu/` | `https://open.bigmodel.cn/api/paas/v4` | OpenAI | [キーを取得](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) |
| **DeepSeek** | `deepseek/` | `https://api.deepseek.com/v1` | OpenAI | [キーを取得](https://platform.deepseek.com) |
| **Google Gemini** | `gemini/` | `https://generativelanguage.googleapis.com/v1beta` | OpenAI | [キーを取得](https://aistudio.google.com/api-keys) |
| **Groq** | `groq/` | `https://api.groq.com/openai/v1` | OpenAI | [キーを取得](https://console.groq.com) |
| **Moonshot** | `moonshot/` | `https://api.moonshot.cn/v1` | OpenAI | [キーを取得](https://platform.moonshot.cn) |
| **Qwen (Alibaba)** | `qwen/` | `https://dashscope.aliyuncs.com/compatible-mode/v1` | OpenAI | [キーを取得](https://dashscope.console.aliyun.com) |
| **NVIDIA** | `nvidia/` | `https://integrate.api.nvidia.com/v1` | OpenAI | [キーを取得](https://build.nvidia.com) |
| **Ollama** | `ollama/` | `http://localhost:11434/v1` | OpenAI | ローカル（キー不要） |
| **OpenRouter** | `openrouter/` | `https://openrouter.ai/api/v1` | OpenAI | [キーを取得](https://openrouter.ai/keys) |
| **VLLM** | `vllm/` | `http://localhost:8000/v1` | OpenAI | ローカル |
| **Cerebras** | `cerebras/` | `https://api.cerebras.ai/v1` | OpenAI | [キーを取得](https://cerebras.ai) |
| **Volcengine** | `volcengine/` | `https://ark.cn-beijing.volces.com/api/v3` | OpenAI | [キーを取得](https://console.volcengine.com) |
| **ShengsuanYun** | `shengsuanyun/` | `https://router.shengsuanyun.com/api/v1` | OpenAI | - |
| **Antigravity** | `antigravity/` | Google Cloud | カスタム | OAuthのみ |
| **GitHub Copilot** | `github-copilot/` | `localhost:4321` | gRPC | - |

#### 基本設定

```json
{
  "model_list": [
    {
      "model_name": "gpt-5.2",
      "model": "openai/gpt-5.2",
      "api_key": "sk-your-openai-key"
    },
    {
      "model_name": "claude-sonnet-4.6",
      "model": "anthropic/claude-sonnet-4.6",
      "api_key": "sk-ant-your-key"
    },
    {
      "model_name": "glm-4.7",
      "model": "zhipu/glm-4.7",
      "api_key": "your-zhipu-key"
    }
  ],
  "agents": {
    "defaults": {
      "model": "gpt-5.2"
    }
  }
}
```

#### ベンダー別の例

**OpenAI**
```json
{
  "model_name": "gpt-5.2",
  "model": "openai/gpt-5.2",
  "api_key": "sk-..."
}
```

**Zhipu AI (GLM)**
```json
{
  "model_name": "glm-4.7",
  "model": "zhipu/glm-4.7",
  "api_key": "your-key"
}
```

**Anthropic (OAuth使用)**
```json
{
  "model_name": "claude-sonnet-4.6",
  "model": "anthropic/claude-sonnet-4.6",
  "auth_method": "oauth"
}
```
> OAuth認証を設定するには、`picoclaw auth login --provider anthropic` を実行してください。

#### ロードバランシング

同じモデル名で複数のエンドポイントを設定すると、PicoClaw が自動的にラウンドロビンで分散します：

```json
{
  "model_list": [
    {
      "model_name": "gpt-5.2",
      "model": "openai/gpt-5.2",
      "api_base": "https://api1.example.com/v1",
      "api_key": "sk-key1"
    },
    {
      "model_name": "gpt-5.2",
      "model": "openai/gpt-5.2",
      "api_base": "https://api2.example.com/v1",
      "api_key": "sk-key2"
    }
  ]
}
```

#### 従来の `providers` 設定からの移行

古い `providers` 設定は**非推奨**ですが、後方互換性のためにサポートされています。

**旧設定（非推奨）:**
```json
{
  "providers": {
    "zhipu": {
      "api_key": "your-key",
      "api_base": "https://open.bigmodel.cn/api/paas/v4"
    }
  },
  "agents": {
    "defaults": {
      "provider": "zhipu",
      "model": "glm-4.7"
    }
  }
}
```

**新設定（推奨）:**
```json
{
  "model_list": [
    {
      "model_name": "glm-4.7",
      "model": "zhipu/glm-4.7",
      "api_key": "your-key"
    }
  ],
  "agents": {
    "defaults": {
      "model": "glm-4.7"
    }
  }
}
```

詳細な移行ガイドは、[docs/migration/model-list-migration.md](docs/migration/model-list-migration.md) を参照してください。

## CLI リファレンス

| コマンド | 説明 |
|---------|------|
| `picoclaw onboard` | 設定＆ワークスペースの初期化 |
| `picoclaw agent -m "..."` | エージェントとチャット |
| `picoclaw agent` | インタラクティブチャットモード |
| `picoclaw gateway` | ゲートウェイを起動 |
| `picoclaw status` | ステータスを表示 |

## 🤝 コントリビュート＆ロードマップ

PR 歓迎！コードベースは意図的に小さく読みやすくしています。🤗

Discord: https://discord.gg/V4sAZ9XWpN

<img src="assets/wechat.png" alt="PicoClaw" width="512">


## 🐛 トラブルシューティング

### Web 検索で「API 設定の問題」と表示される

検索 API キーをまだ設定していない場合、これは正常です。PicoClaw は手動検索用の便利なリンクを提供します。

Web 検索を有効にするには：
1. [https://tavily.com](https://tavily.com) (月 1000 クエリ無料) または [https://brave.com/search/api](https://brave.com/search/api) で無料の API キーを取得（月 2000 クエリ無料）
2. `~/.picoclaw/config.json` に追加：
   ```json
   {
     "tools": {
       "web": {
         "brave": {
           "enabled": true,
           "api_key": "YOUR_BRAVE_API_KEY",
           "max_results": 5
         },
         "duckduckgo": {
           "enabled": true,
           "max_results": 5
         }
       }
     }
   }
   ```

### コンテンツフィルタリングエラーが出る

一部のプロバイダー（Zhipu など）にはコンテンツフィルタリングがあります。クエリを言い換えるか、別のモデルを使用してください。

### Telegram Bot で「Conflict: terminated by other getUpdates」と表示される

別のインスタンスが実行中の場合に発生します。`picoclaw gateway` が 1 つだけ実行されていることを確認してください。

---

## 📝 API キー比較

| サービス | 無料枠 | ユースケース |
|---------|--------|------------|
| **OpenRouter** | 月 200K トークン | 複数モデル（Claude, GPT-4 など） |
| **Zhipu** | 月 200K トークン | 中国ユーザー向け最適 |
| **Qwen** | 無料枠あり | 通義千問 (Qwen) |
| **Brave Search** | 月 2000 クエリ | Web 検索機能 |
| **Tavily** | 月 1000 クエリ | AI エージェント検索最適化 |
| **Groq** | 無料枠あり | 高速推論（Llama, Mixtral） |
| **Cerebras** | 無料枠あり | 高速推論（Llama, Qwen など） |
