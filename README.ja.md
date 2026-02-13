<div align="center">
<img src="assets/logo.jpg" alt="PicoClaw" width="512">

<h1>PicoClaw: Go で書かれた超効率 AI アシスタント</h1>

<h3>$10 ハードウェア · 10MB RAM · 1秒起動 · 皮皮虾，我们走！</h3>
<h3></h3>

<p>
<img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
<img src="https://img.shields.io/badge/Arch-x86__64%2C%20ARM64%2C%20RISC--V-blue" alt="Hardware">
<img src="https://img.shields.io/badge/license-MIT-green" alt="License">
</p>

**日本語** | [English](README.md)

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
2026-02-09 🎉 PicoClaw リリース！$10 ハードウェアで 10MB 未満の RAM で動く AI エージェントを 1 日で構築。🦐 皮皮虾，我们走！

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

Telegram、Discord、QQ、DingTalk で PicoClaw と会話できます

| チャネル | セットアップ |
|---------|------------|
| **Telegram** | 簡単（トークンのみ） |
| **Discord** | 簡単（Bot トークン + Intents） |
| **QQ** | 簡単（AppID + AppSecret） |
| **DingTalk** | 普通（アプリ認証情報） |

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

- [QQ オープンプラットフォーム](https://connect.qq.com/) にアクセス
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

### Web 検索で「API 配置问题」と表示される

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
