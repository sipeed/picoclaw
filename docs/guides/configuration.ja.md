# ⚙️ 設定ガイド

> [README](../project/README.ja.md) に戻る

## ⚙️ 設定詳細

設定ファイルパス: `~/.picoclaw/config.json`

### 環境変数

環境変数を使用してデフォルトパスを上書きできます。ポータブルインストール、コンテナ化デプロイ、または picoclaw をシステムサービスとして実行する場合に便利です。これらの変数は独立しており、異なるパスを制御します。

| 変数              | 説明                                                                                                                             | デフォルトパス            |
|-------------------|-----------------------------------------------------------------------------------------------------------------------------------------|---------------------------|
| `PICOCLAW_CONFIG` | 設定ファイルのパスを上書きします。picoclaw がどの `config.json` を読み込むかを直接指定し、他のすべての場所を無視します。 | `~/.picoclaw/config.json` |
| `PICOCLAW_HOME`   | picoclaw データのルートディレクトリを上書きします。`workspace` やその他のデータディレクトリのデフォルト場所を変更します。          | `~/.picoclaw`             |

**例：**

```bash
# 特定の設定ファイルで picoclaw を実行
# ワークスペースパスはその設定ファイル内から読み込まれます
PICOCLAW_CONFIG=/etc/picoclaw/production.json picoclaw gateway

# /opt/picoclaw にすべてのデータを保存して picoclaw を実行
# 設定はデフォルトの ~/.picoclaw/config.json から読み込まれます
# ワークスペースは /opt/picoclaw/workspace に作成されます
PICOCLAW_HOME=/opt/picoclaw picoclaw agent

# 両方を使用して完全にカスタマイズ
PICOCLAW_HOME=/srv/picoclaw PICOCLAW_CONFIG=/srv/picoclaw/main.json picoclaw gateway
```

### Gateway ログレベル

`gateway.log_level` は Gateway のログ詳細度を制御します。`config.json` で設定できます：

```json
{
  "gateway": {
    "log_level": "warn"
  }
}
```

デフォルト値は `warn` です。サポートされる値：`debug`、`info`、`warn`、`error`、`fatal`。

環境変数でも上書き可能です：`PICOCLAW_LOG_LEVEL=info`

### ワークスペースレイアウト

PicoClaw は設定されたワークスペース（デフォルト: `~/.picoclaw/workspace`）にデータを保存します：

```
~/.picoclaw/workspace/
├── sessions/          # 会話セッションと履歴
├── memory/           # 長期記憶 (MEMORY.md)
├── state/            # 永続化状態 (最後のチャネルなど)
├── cron/             # スケジュールジョブデータベース
├── skills/           # カスタムスキル
├── AGENT.md          # Agent 動作ガイド
├── HEARTBEAT.md      # 定期タスクプロンプト (30 分ごとにチェック)
├── IDENTITY.md       # Agent アイデンティティ
├── SOUL.md           # Agent ソウル/性格
└── USER.md           # ユーザー設定
```

> **注意：** `AGENT.md`、`SOUL.md`、`USER.md` および `memory/MEMORY.md` への変更は、ファイル更新時刻（mtime）の追跡により実行時に自動検出されます。これらのファイルを編集した後に **gateway を再起動する必要はありません** — Agent は次のリクエスト時に最新の内容を自動的に読み込みます。

### スキルソース

デフォルトでは、スキルは以下の順序で読み込まれます：

1. `~/.picoclaw/workspace/skills`（ワークスペース）
2. `~/.picoclaw/skills`（グローバル）
3. `<ビルド時埋め込みパス>/skills`（ビルトイン）

高度な/テスト用セットアップでは、以下の環境変数でビルトインスキルのルートを上書きできます：

```bash
export PICOCLAW_BUILTIN_SKILLS=/path/to/skills
```

### チャットチャネルからスキルとコマンドを使う

スキルをインストールすると、チャットチャネルから直接確認したり明示的に適用したりできます：

- `/list skills` は現在の Agent から見えるインストール済みスキル名を表示します。
- `/use <skill> <message>` は 1 回のリクエストだけそのスキルを強制します。
- `/use <skill>` は同じチャット内の次のメッセージにそのスキルを予約します。
- `/use clear` は `/use <skill>` で設定した保留中のスキル上書きを解除します。
- `/btw <question>` は現在のセッション履歴を変更せずに即時の横道の質問を送ります。`/btw` はツールなしの直接質問として処理され、通常のツール実行フローには入りません。

例：

```text
/list skills
/use git 直近 3 つのコミットを squash する方法を教えて
/btw さっきのデプロイ方針の結論だけもう一度教えて
/use italiapersonalfinance
dammi le ultime news
```

### 統一コマンド実行ポリシー

- 汎用スラッシュコマンドは `pkg/agent/loop.go` 内の `commands.Executor` を通じて統一的に実行されます。
- チャネルアダプターはローカルで汎用コマンドを消費しなくなりました。受信テキストを bus/agent パスに転送するだけです。Telegram は起動時に `/start`、`/help`、`/show`、`/list`、`/use`、`/btw` などのサポート済みコマンドを自動登録します。
- 未登録のスラッシュコマンド（例: `/foo`）は通常の LLM 処理にパススルーされます。
- 登録済みだが現在のチャネルでサポートされていないコマンド（例: WhatsApp での `/show`）は、明示的なユーザー向けエラーを返し、以降の処理を停止します。

### 🔒 セキュリティサンドボックス

PicoClaw はデフォルトでサンドボックス環境で実行されます。Agent は設定されたワークスペース内のファイルアクセスとコマンド実行のみが可能です。

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

| オプション              | デフォルト値            | 説明                                  |
| ----------------------- | ----------------------- | ------------------------------------- |
| `workspace`             | `~/.picoclaw/workspace` | Agent の作業ディレクトリ              |
| `restrict_to_workspace` | `true`                  | ファイル/コマンドアクセスをワークスペース内に制限 |

#### 保護されたツール

`restrict_to_workspace: true` の場合、以下のツールがサンドボックス化されます：

| ツール        | 機能             | 制限                               |
| ------------- | ---------------- | ---------------------------------- |
| `read_file`   | ファイル読み取り | ワークスペース内のファイルのみ     |
| `write_file`  | ファイル書き込み | ワークスペース内のファイルのみ     |
| `list_dir`    | ディレクトリ一覧 | ワークスペース内のディレクトリのみ |
| `edit_file`   | ファイル編集     | ワークスペース内のファイルのみ     |
| `append_file` | ファイル追記     | ワークスペース内のファイルのみ     |
| `exec`        | コマンド実行     | コマンドパスはワークスペース内必須 |

#### 追加の Exec 保護

`restrict_to_workspace: false` の場合でも、`exec` ツールは以下の危険なコマンドをブロックします：

* `rm -rf`、`del /f`、`rmdir /s` — 一括削除
* `format`、`mkfs`、`diskpart` — ディスクフォーマット
* `dd if=` — ディスクイメージング
* `/dev/sd[a-z]` への書き込み — 直接ディスク書き込み
* `shutdown`、`reboot`、`poweroff` — システムシャットダウン
* Fork bomb `:(){ :|:& };:`

### ファイルアクセス制御

| 設定キー | 型 | デフォルト値 | 説明 |
|----------|------|-------------|------|
| `tools.allow_read_paths` | string[] | `[]` | ワークスペース外で読み取りを許可する追加パス |
| `tools.allow_write_paths` | string[] | `[]` | ワークスペース外で書き込みを許可する追加パス |

### Exec セキュリティ設定

| 設定キー | 型 | デフォルト値 | 説明 |
|----------|------|-------------|------|
| `tools.exec.allow_remote` | bool | `false` | リモートチャネル（Telegram/Discord など）からの exec ツール実行を許可 |
| `tools.exec.enable_deny_patterns` | bool | `true` | 危険なコマンドのインターセプトを有効化 |
| `tools.exec.custom_deny_patterns` | string[] | `[]` | カスタムブロック正規表現パターン |
| `tools.exec.custom_allow_patterns` | string[] | `[]` | カスタム許可正規表現パターン |

> **セキュリティ注意:** Symlink 保護はデフォルトで有効です。すべてのファイルパスはホワイトリストマッチング前に `filepath.EvalSymlinks` で解決され、シンボリックリンクエスケープ攻撃を防止します。

#### 既知の制限：ビルドツールの子プロセス

exec セキュリティガードは PicoClaw が直接起動するコマンドラインのみを検査します。`make`、`go run`、`cargo`、`npm run`、またはカスタムビルドスクリプトなどの開発ツールが生成する子プロセスは再帰的に検査しません。

つまり、トップレベルのコマンドが初期ガードチェックを通過した後、他のバイナリをコンパイルまたは起動できます。実際には、ビルドスクリプト、Makefile、パッケージスクリプト、生成されたバイナリを、直接のシェルコマンドと同等レベルの実行可能コードとしてレビューする必要があります。

高リスク環境の場合：

* 実行前にビルドスクリプトをレビューしてください。
* コンパイル・実行ワークフローには承認/手動レビューを優先してください。
* ビルトインガードより強力な分離が必要な場合は、コンテナまたは VM 内で PicoClaw を実行してください。

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

Agent がワークスペース外のパスにアクセスする必要がある場合：

**方法 1: 設定ファイル**

```json
{
  "agents": {
    "defaults": {
      "restrict_to_workspace": false
    }
  }
}
```

**方法 2: 環境変数**

```bash
export PICOCLAW_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE=false
```

> ⚠️ **警告**: この制限を無効にすると、Agent がシステム上の任意のパスにアクセスできるようになります。管理された環境でのみ慎重に使用してください。

#### セキュリティ境界の一貫性

`restrict_to_workspace` 設定はすべての実行パスで一貫して適用されます：

| 実行パス         | セキュリティ境界             |
| ---------------- | ---------------------------- |
| メイン Agent     | `restrict_to_workspace` ✅   |
| サブ Agent / Spawn | 同じ制限を継承 ✅           |
| ハートビートタスク | 同じ制限を継承 ✅           |

すべてのパスは同じワークスペース制限を共有しており、サブ Agent やスケジュールタスクを通じてセキュリティ境界を回避することはできません。

### ハートビート（定期タスク）

PicoClaw は定期タスクを自動実行できます。ワークスペースに `HEARTBEAT.md` ファイルを作成してください：

```markdown
# Periodic Tasks

- Check my email for important messages
- Review my calendar for upcoming events
- Check the weather forecast
```

Agent は 30 分ごと（設定可能）にこのファイルを読み取り、利用可能なツールを使用してタスクを実行します。

#### Spawn を使用した非同期タスク

長時間実行タスク（Web 検索、API 呼び出し）には、`spawn` ツールを使用して**サブ Agent (subagent)** を作成します：

```markdown
# Periodic Tasks

## Quick Tasks (respond directly)

- Report current time

## Long Tasks (use spawn for async)

- Search the web for AI news and summarize
- Check email and report important messages
```

**主な動作：**

| 特性             | 説明                                         |
| ---------------- | -------------------------------------------- |
| **spawn**        | 非同期サブ Agent を作成、メインハートビートをブロックしない |
| **独立コンテキスト** | サブ Agent は独自のコンテキストを持ち、セッション履歴なし |
| **message tool** | サブ Agent は message ツールでユーザーと直接通信 |
| **ノンブロッキング** | spawn 後、ハートビートは次のタスクに進む     |

**設定：**

```json
{
  "heartbeat": {
    "enabled": true,
    "interval": 30
  }
}
```

| オプション | デフォルト値 | 説明                           |
| ---------- | ------------ | ------------------------------ |
| `enabled`  | `true`       | ハートビートの有効/無効        |
| `interval` | `30`         | チェック間隔（分単位、最小: 5）|

**環境変数:**

- `PICOCLAW_HEARTBEAT_ENABLED=false` で無効化
- `PICOCLAW_HEARTBEAT_INTERVAL=60` で間隔を変更

#### サブ Agent の通信フロー

```
ハートビート起動
    ↓
Agent が HEARTBEAT.md を読む
    ↓
長時間タスク：spawn サブ Agent
    ↓                           ↓
次のタスクへ継続           サブ Agent が独立して動作
    ↓                           ↓
全タスク完了               サブ Agent が "message" ツールを使用
    ↓                           ↓
HEARTBEAT_OK を返信        ユーザーが直接結果を受信
```

### Providers

> [!NOTE]
> Groq は Whisper による無料音声文字起こしを提供します。設定すると、任意のチャンネルの音声メッセージが Agent レベルで自動的に文字起こしされます。

| Provider     | 用途                                    | API キー取得                                                 |
| ------------ | --------------------------------------- | ------------------------------------------------------------ |
| `gemini`     | LLM（Gemini 直接）                      | [aistudio.google.com](https://aistudio.google.com)           |
| `zhipu`      | LLM（Zhipu 直接）                       | [bigmodel.cn](https://bigmodel.cn)                           |
| `volcengine` | LLM（Volcengine 直接）                  | [volcengine.com](https://www.volcengine.com/activity/codingplan?utm_campaign=PicoClaw&utm_content=PicoClaw&utm_medium=devrel&utm_source=OWO&utm_term=PicoClaw) |
| `openrouter` | LLM（推奨、全モデルにアクセス可能）     | [openrouter.ai](https://openrouter.ai)                       |
| `anthropic`  | LLM（Claude 直接）                      | [console.anthropic.com](https://console.anthropic.com)       |
| `openai`     | LLM（GPT 直接）                         | [platform.openai.com](https://platform.openai.com)           |
| `deepseek`   | LLM（DeepSeek 直接）                    | [platform.deepseek.com](https://platform.deepseek.com)       |
| `qwen`       | LLM（Qwen 直接）                        | [dashscope.console.aliyun.com](https://dashscope.console.aliyun.com) |
| `groq`       | LLM + **音声文字起こし**（Whisper）     | [console.groq.com](https://console.groq.com)                 |
| `cerebras`   | LLM（Cerebras 直接）                    | [cerebras.ai](https://cerebras.ai)                           |
| `vivgrid`    | LLM（Vivgrid 直接）                     | [vivgrid.com](https://vivgrid.com)                           |

### モデル設定 (model_list)

> **新機能：** PicoClaw は**モデル中心**の設定アプローチを採用しました。`vendor/model` 形式（例：`zhipu/glm-4.7`）を指定するだけで新しい Provider を追加できます — **コード変更不要！**

#### サポートされている全 Vendor

| Vendor                  | `model` プレフィックス | デフォルト API Base                                 | プロトコル | API Key                                                          |
| ----------------------- | ---------------------- | --------------------------------------------------- | ---------- | ---------------------------------------------------------------- |
| **OpenAI**              | `openai/`              | `https://api.openai.com/v1`                         | OpenAI     | [取得](https://platform.openai.com)                              |
| **Anthropic**           | `anthropic/`           | `https://api.anthropic.com/v1`                      | Anthropic  | [取得](https://console.anthropic.com)                            |
| **智谱 AI (GLM)**       | `zhipu/`               | `https://open.bigmodel.cn/api/paas/v4`              | OpenAI     | [取得](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys)    |
| **DeepSeek**            | `deepseek/`            | `https://api.deepseek.com/v1`                       | OpenAI     | [取得](https://platform.deepseek.com)                            |
| **Google Gemini**       | `gemini/`              | `https://generativelanguage.googleapis.com/v1beta`  | Gemini     | [取得](https://aistudio.google.com/api-keys)                     |
| **Groq**                | `groq/`                | `https://api.groq.com/openai/v1`                    | OpenAI     | [取得](https://console.groq.com)                                 |
| **通義千問 (Qwen)**     | `qwen/`                | `https://dashscope.aliyuncs.com/compatible-mode/v1` | OpenAI     | [取得](https://dashscope.console.aliyun.com)                     |
| **Ollama**              | `ollama/`              | `http://localhost:11434/v1`                         | OpenAI     | ローカル（キー不要）                                             |
| **OpenRouter**          | `openrouter/`          | `https://openrouter.ai/api/v1`                      | OpenAI     | [取得](https://openrouter.ai/keys)                               |
| **VolcEngine (Doubao)** | `volcengine/`          | `https://ark.cn-beijing.volces.com/api/v3`          | OpenAI     | [取得](https://www.volcengine.com/activity/codingplan?utm_campaign=PicoClaw&utm_content=PicoClaw&utm_medium=devrel&utm_source=OWO&utm_term=PicoClaw) |
| **Antigravity**         | `antigravity/`         | Google Cloud                                        | Custom     | OAuth のみ                                                       |

#### ロードバランシング

同じモデル名に複数のエンドポイントを設定すると、PicoClaw が自動的にラウンドロビンします：

```json
{
  "model_list": [
    { "model_name": "gpt-5.4", "model": "openai/gpt-5.4", "api_base": "https://api1.example.com/v1", "api_keys": ["sk-key1"] },
    { "model_name": "gpt-5.4", "model": "openai/gpt-5.4", "api_base": "https://api2.example.com/v1", "api_keys": ["sk-key2"] }
  ]
}
```

#### 旧 `providers` 設定からの移行

旧 `providers` 設定は**非推奨**となり、V2 で削除されました。既存の V0/V1 設定は自動的に移行されます。[docs/migration/model-list-migration.md](../migration/model-list-migration.md) を参照してください。

### Provider アーキテクチャ

PicoClaw はプロトコルファミリーで Provider をルーティングします：

- **OpenAI 互換**：OpenRouter、Groq、Zhipu、vLLM スタイルのエンドポイントなど。
- **Gemini ネイティブ**：Google Gemini のネイティブ `models/*:generateContent` / `models/*:streamGenerateContent` エンドポイント。
- **Anthropic**：Claude ネイティブ API の動作。
- **Codex/OAuth**：OpenAI OAuth/トークン認証ルート。

これによりランタイムを軽量に保ちつつ、新しい OpenAI 互換バックエンドの追加をほぼ設定操作（`api_base` + `api_keys`）のみで実現します。

### スケジュールタスク / リマインダー

PicoClaw は `cron` ツールを通じて cron スタイルのスケジュールタスクをサポートします。

```json
{
  "tools": {
    "cron": {
      "enabled": true,
      "exec_timeout_minutes": 5
    }
  }
}
```

スケジュールタスクは再起動後も `~/.picoclaw/workspace/cron/` に保存されます。

### 高度なトピック

| トピック | 説明 |
| -------- | ---- |
| [Hook システム](../architecture/hooks/README.md) | イベント駆動 Hook：オブザーバー、インターセプター、承認 Hook |
| [Steering](../architecture/steering.md) | 実行中の Agent ループにメッセージを注入 |
| [SubTurn](../architecture/subturn.md) | サブ Agent の調整、並行制御、ライフサイクル |
| [コンテキスト管理](../architecture/agent-refactor/context.md) | コンテキスト境界検出、圧縮戦略 |
