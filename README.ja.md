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

[中文](README.zh.md) | **日本語** | [Português](README.pt-br.md) | [English](README.md)

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

> **セキュリティ**: Docker イメージはデフォルトで非 root ユーザー（`picoclaw`, UID 1000）として実行されます。リソース制限（CPU: 1.0, メモリ: 512MB）は `docker-compose.yml` で設定されています。

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
> Web 検索は **任意** です - 無料の [Brave Search API](https://brave.com/search/api) (月 2000 クエリ無料)

**1. 初期化**

```bash
picoclaw onboard
```

**2. 設定** (`~/.picoclaw/config.json`)

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "model": "glm-4.7",
      "max_tokens": 8192,
      "temperature": 0.7,
      "max_tool_iterations": 20
    }
  },
  "providers": {
    "openrouter": {
      "api_key": "xxx",
      "api_base": "https://openrouter.ai/api/v1"
    }
  },
  "tools": {
    "web": {
      "search": {
        "api_key": "YOUR_BRAVE_API_KEY",
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
- **Web 検索**（任意）: [Brave Search](https://brave.com/search/api) - 無料枠あり（月 2000 リクエスト）

> **注意**: 完全な設定テンプレートは `config.example.json` を参照してください。

**3. チャット**

```bash
picoclaw agent -m "What is 2+2?"
```

これだけです！2 分で AI アシスタントが動きます。

---

## 💬 チャットアプリ

Telegram、Discord、QQ、DingTalk、LINE で PicoClaw と会話できます

| チャネル | セットアップ |
|---------|------------|
| **Telegram** | 簡単（トークンのみ） |
| **Discord** | 簡単（Bot トークン + Intents） |
| **QQ** | 簡単（AppID + AppSecret） |
| **DingTalk** | 普通（アプリ認証情報） |
| **LINE** | 普通（認証情報 + Webhook URL） |

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
      "allowFrom": ["YOUR_USER_ID"]
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
      "allowFrom": ["YOUR_USER_ID"]
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

<details>
<summary><b>Exec 設定</b></summary>

設定で `exec` ツールのセキュリティ動作をカスタマイズできます：

```json
{
  "tools": {
    "exec": {
      "deny_patterns": [],
      "allow_patterns": [],
      "max_timeout": 60
    }
  }
}
```

| オプション | デフォルト | 説明 |
|-----------|-----------|------|
| `deny_patterns` | `[]` | ブロックする追加の正規表現パターン（組み込みルールとマージ） |
| `allow_patterns` | `[]` | 設定時、マッチするコマンド**のみ**許可（ホワイトリストモード） |
| `max_timeout` | `60` | コマンド実行の最大タイムアウト（秒） |

**`deny_patterns` の例** — `pip install` とすべての `docker` コマンドをブロック：

```json
{
  "tools": {
    "exec": {
      "deny_patterns": [
        "\\bpip\\s+install\\b",
        "\\bdocker\\b"
      ]
    }
  }
}
```

**`allow_patterns` の例** — `git`、`ls`、`cat`、`echo` コマンドのみ許可：

```json
{
  "tools": {
    "exec": {
      "allow_patterns": [
        "^git\\b",
        "^ls\\b",
        "^cat\\b",
        "^echo\\b"
      ]
    }
  }
}
```

> **注意**: `deny_patterns` は組み込みルールとマージされます（両方が適用）。`allow_patterns` はホワイトリストとして機能します — 設定時、許可パターンにマッチしないコマンドは deny ルールに関係なくブロックされます。

</details>

#### 組み込み Exec 保護ルール

`security.exec_guard` が `"block"` または `"approve"` に設定されている場合、`exec` ツールには設定で上書きできない組み込み拒否ルールがあります：

| カテゴリ | ブロックパターン | 説明 |
|---------|----------------|------|
| 破壊的 | `rm -rf`, `del /f`, `rmdir /s` | 一括削除 |
| 破壊的 | `format`, `mkfs`, `diskpart` | ディスクフォーマット |
| 破壊的 | `dd if=` | ディスクイメージング |
| 破壊的 | `> /dev/sd[a-z]` | 直接ディスク書き込み |
| システム | `shutdown`, `reboot`, `poweroff` | システム制御 |
| システム | `:(){ :\|:& };:` | フォークボム |
| データ漏洩 | `curl -d/--data/-F/--upload-file` | HTTP データアップロード |
| データ漏洩 | `wget --post-data/--post-file` | HTTP POST アップロード |
| データ漏洩 | `nc <host> <port>`, `ncat` | Netcat 接続 |
| コード注入 | `base64 ... \| sh/bash/zsh` | エンコードされたコマンド実行 |
| リバースシェル | `bash -i >&`, `/dev/tcp/` | リバースシェルパターン |

`restrict_to_workspace: true` の場合、追加の制限が適用されます：

- 機密システムパス（`/etc/`, `/var/`, `/root`, `/home/`, `/proc/`, `/sys/`, `/boot/`）へのアクセスがブロックされます
- シンボリックリンクベースのパストラバーサル攻撃が検出・ブロックされます
- `../` によるパストラバーサルがブロックされます

> 組み込み正規表現パターンの完全なリストは [`pkg/tools/shell.go`](pkg/tools/shell.go#L37-L59) を参照してください。

#### SSRF 保護

`security.ssrf_protection` が `"block"` または `"approve"` に設定されている場合、すべてのアウトバウンド HTTP リクエスト（`web_fetch` ツールおよびファイルダウンロード）は SSRF 攻撃に対して検証されます：

- プライベート IP 範囲（`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`）がブロックされます
- ループバックアドレス（`127.0.0.0/8`, `::1`）がブロックされます
- リンクローカルアドレス（`169.254.0.0/16`）がブロックされます
- クラウドメタデータエンドポイント（`169.254.169.254`）がブロックされます
- `http://` と `https://` スキームのみ許可されます
- リダイレクト先も検証され、リダイレクトベースの SSRF を防止します

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
| Cron スケジュールジョブ | 同じ制限を継承 ✅ |

すべてのパスで同じワークスペース制限が適用されます — サブエージェント、Cron ジョブ、またはスケジュールタスクを通じてセキュリティ境界をバイパスする方法はありません。

#### セキュリティポリシーモード

すべてのセキュリティチェック（コマンドガード、SSRF 保護、パス検証、スキル検証）は、3つの設定可能なモードをサポートしています。デフォルトではすべてのモードが `"off"` です — セキュリティ機能は**オプトイン**であり、明示的に設定しない限り動作は変更されません。

| モード | 動作 |
|--------|------|
| `off` | セキュリティチェック無効（デフォルト）。制限なし。 |
| `block` | 違反が検出されると即座にエラーで拒否。 |
| `approve` | 違反が検出されると実行を一時停止し、IM 経由でユーザーに承認リクエストを送信、返信を待機。 |

<details>
<summary><b>セキュリティ設定</b></summary>

```json
{
  "security": {
    "exec_guard": "off",
    "ssrf_protection": "off",
    "path_validation": "off",
    "skill_validation": "off",
    "approval_timeout": 300
  }
}
```

| オプション | デフォルト | 説明 |
|-----------|-----------|------|
| `exec_guard` | `"off"` | コマンドの拒否/許可パターンチェックのモード |
| `ssrf_protection` | `"off"` | アウトバウンド URL 検証（プライベート IP、メタデータエンドポイント） |
| `path_validation` | `"off"` | シンボリックリンク対応の強化パス制限 |
| `skill_validation` | `"off"` | スキルインストール時のリポジトリ形式チェック |
| `approval_timeout` | `300` | ユーザー承認のタイムアウト（秒）。タイムアウト時は自動拒否 |

環境変数もサポートされています（例: `PICOCLAW_SECURITY_EXEC_GUARD=approve`）。

</details>

#### IM ベースの承認メカニズム

セキュリティチェックが `"approve"` モードに設定されている場合、PicoClaw はコマンドを即座に拒否する代わりに：

1. ツールの実行を**一時停止**
2. 現在の IM チャネル（Telegram、Feishu、DingTalk、Slack など）経由でユーザーに承認リクエストを**送信**
3. ユーザーの承認または拒否キーワードの返信を**待機**
4. 承認された場合は実行を**再開**、拒否またはタイムアウトの場合はエラーを返却

**サポートされている承認キーワード：**

| アクション | 英語 | 中国語 | 日本語 |
|-----------|------|--------|--------|
| 承認 | approve, yes, allow, ok, y | 批准, 允许, 通过, 是 | 承認, 許可, はい |
| 拒否 | deny, no, reject, block, n | 拒绝, 否决, 不 | 拒否, いいえ |

**注意事項：**
- CLI モードでは、非同期 IM チャネルがないため、`"approve"` は `"block"` にフォールバックします。
- Cron ジョブの承認リクエストは、最後にアクティブだった IM チャネルに送信されます。利用可能なチャネルがない場合は `"block"` にフォールバックします。
- 承認待機中にユーザーが送信した承認キーワード以外のメッセージは、通常通りエージェントに渡されます。
- `approval_timeout` 秒以内に返信がない場合、リクエストは自動的に拒否されます。

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
      "apiKey": "sk-or-v1-xxx"
    },
    "groq": {
      "apiKey": "gsk_xxx"
    }
  },
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "123456:ABC...",
      "allowFrom": ["123456789"]
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
      "appId": "cli_xxx",
      "appSecret": "xxx",
      "encryptKey": "",
      "verificationToken": "",
      "allowFrom": []
    }
  },
  "tools": {
    "web": {
      "search": {
        "apiKey": "BSA..."
      }
    },
    "cron": {
      "exec_timeout_minutes": 5
    },
    "exec": {
      "deny_patterns": [],
      "allow_patterns": [],
      "max_timeout": 60
    }
  },
  "heartbeat": {
    "enabled": true,
    "interval": 30
  }
}
```

</details>

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
1. [https://brave.com/search/api](https://brave.com/search/api) で無料の API キーを取得（月 2000 クエリ無料）
2. `~/.picoclaw/config.json` に追加：
   ```json
   {
     "tools": {
       "web": {
         "search": {
           "api_key": "YOUR_BRAVE_API_KEY",
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
| **Brave Search** | 月 2000 クエリ | Web 検索機能 |
| **Groq** | 無料枠あり | 高速推論（Llama, Mixtral） |
