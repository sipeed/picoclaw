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
      "api_base": "https://open.bigmodel.cn/api/paas/v4"
    }
  },
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

Telegram で PicoClaw と会話できます

| チャネル | セットアップ |
|---------|------------|
| **Telegram** | 簡単（トークンのみ） |
| **Discord** | 簡単（Bot トークン + Intents） |

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

## 設定 (Configuration)

PicoClaw は設定に `config.json` を使用します。

1.  **設定ファイルの作成:**

    サンプル設定ファイルをコピーします:

    ```bash
    cp config.example.json config/config.json
    ```

2.  **設定の編集:**

    `config/config.json` を開き、APIキーや設定を記述します。

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

**3. 実行**

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
| `picoclaw gateway --install-daemon` | システムサービスとしてインストール |
| `picoclaw gateway --uninstall-daemon` | システムサービスをアンインストール |
| `picoclaw gateway --start-daemon` | インストール済みサービスを起動 |
| `picoclaw gateway --stop-daemon` | インストール済みサービスを停止 |
| `picoclaw gateway --daemon-status` | サービスの状態を表示 |
| `picoclaw status` | ステータスを表示 |

### システムサービス（デーモン）として実行

PicoClaw は [kardianos/service](https://github.com/kardianos/service) を使用してシステムサービスとしてインストールできます。**Linux (systemd)**、**macOS (launchd)**、**Windows (SCM)** に対応しています。

```bash
# システムサービスとしてインストール（Linux では root/sudo が必要）
sudo picoclaw gateway --install-daemon

# サービスを起動
sudo picoclaw gateway --start-daemon

# サービスの状態を確認
picoclaw gateway --daemon-status

# サービスを停止
sudo picoclaw gateway --stop-daemon

# サービスをアンインストール
sudo picoclaw gateway --uninstall-daemon
```

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
