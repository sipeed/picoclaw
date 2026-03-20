# 🔧 ツール設定

> [README](../../README.ja.md) に戻る

PicoClaw のツール設定は `config.json` の `tools` フィールドにあります。

## ディレクトリ構造

```json
{
  "tools": {
    "web": {
      ...
    },
    "mcp": {
      ...
    },
    "exec": {
      ...
    },
    "cron": {
      ...
    },
    "skills": {
      ...
    }
  }
}
```

## Web ツール

Web ツールはウェブ検索とフェッチに使用されます。

### Web Fetcher
ウェブページコンテンツの取得と処理に関する一般設定。

| 設定項目            | 型     | デフォルト    | 説明                                                                                   |
|---------------------|--------|---------------|----------------------------------------------------------------------------------------|
| `enabled`           | bool   | true          | ウェブページ取得機能を有効にする。                                                     |
| `fetch_limit_bytes` | int    | 10485760      | 取得するウェブページペイロードの最大サイズ（バイト単位、デフォルトは10MB）。            |
| `format`            | string | "plaintext"   | 取得コンテンツの出力形式。オプション：`plaintext` または `markdown`（推奨）。           |

### Brave

| 設定項目      | 型     | デフォルト | 説明                  |
|---------------|--------|------------|-----------------------|
| `enabled`     | bool   | false      | Brave 検索を有効にする |
| `api_key`     | string | -          | Brave Search API キー  |
| `max_results` | int    | 5          | 最大結果数            |

### DuckDuckGo

| 設定項目      | 型   | デフォルト | 説明                      |
|---------------|------|------------|---------------------------|
| `enabled`     | bool | true       | DuckDuckGo 検索を有効にする |
| `max_results` | int  | 5          | 最大結果数                |

### Perplexity

| 設定項目      | 型     | デフォルト | 説明                      |
|---------------|--------|------------|---------------------------|
| `enabled`     | bool   | false      | Perplexity 検索を有効にする |
| `api_key`     | string | -          | Perplexity API キー        |
| `max_results` | int    | 5          | 最大結果数                |

## Exec ツール

Exec ツールはシェルコマンドの実行に使用されます。

| 設定項目               | 型    | デフォルト | 説明                               |
|------------------------|-------|------------|------------------------------------|
| `enabled`              | bool  | true       | Exec ツールを有効にする            |
| `enable_deny_patterns` | bool  | true       | デフォルトの危険コマンドブロックを有効にする |
| `custom_deny_patterns` | array | []         | カスタム拒否パターン（正規表現）   |

### Exec ツールの無効化

`exec` ツールを完全に無効にするには、`enabled` を `false` に設定します：

**設定ファイル経由：**
```json
{
  "tools": {
    "exec": {
      "enabled": false
    }
  }
}
```

**環境変数経由：**
```bash
PICOCLAW_TOOLS_EXEC_ENABLED=false
```

> **注意：** 無効にすると、エージェントはシェルコマンドを実行できなくなります。これは Cron ツールがスケジュールされたシェルコマンドを実行する能力にも影響します。

### 機能

- **`enable_deny_patterns`**：`false` に設定すると、デフォルトの危険コマンドブロックパターンを完全に無効にします
- **`custom_deny_patterns`**：カスタム拒否正規表現パターンを追加します。一致するコマンドはブロックされます

### デフォルトでブロックされるコマンドパターン

デフォルトで、PicoClaw は以下の危険なコマンドをブロックします：

- 削除コマンド：`rm -rf`、`del /f/q`、`rmdir /s`
- ディスク操作：`format`、`mkfs`、`diskpart`、`dd if=`、`/dev/sd*` への書き込み
- システム操作：`shutdown`、`reboot`、`poweroff`
- コマンド置換：`$()`、`${}`、バッククォート
- シェルへのパイプ：`| sh`、`| bash`
- 権限昇格：`sudo`、`chmod`、`chown`
- プロセス制御：`pkill`、`killall`、`kill -9`
- リモート操作：`curl | sh`、`wget | sh`、`ssh`
- パッケージ管理：`apt`、`yum`、`dnf`、`npm install -g`、`pip install --user`
- コンテナ：`docker run`、`docker exec`
- Git：`git push`、`git force`
- その他：`eval`、`source *.sh`

### 既知のアーキテクチャ上の制限

exec ガードは PicoClaw に送信されたトップレベルのコマンドのみを検証します。そのコマンドの実行開始後にビルドツールやスクリプトが生成する子プロセスを再帰的に検査することは**ありません**。

初期コマンドが許可された後、直接コマンドガードをバイパスできるワークフローの例：

- `make run`
- `go run ./cmd/...`
- `cargo run`
- `npm run build`

これは、明らかに危険な直接コマンドのブロックには有用ですが、未レビューのビルドパイプラインに対する完全なサンドボックスでは**ありません**。脅威モデルにワークスペース内の信頼できないコードが含まれる場合は、コンテナ、VM、またはビルド・実行コマンドに対する承認フローなど、より強力な分離を使用してください。

### 設定例

```json
{
  "tools": {
    "exec": {
      "enable_deny_patterns": true,
      "custom_deny_patterns": [
        "\\brm\\s+-r\\b",
        "\\bkillall\\s+python"
      ]
    }
  }
}
```

## Cron ツール

Cron ツールは定期タスクのスケジューリングに使用されます。

| 設定項目               | 型  | デフォルト | 説明                                    |
|------------------------|-----|------------|-----------------------------------------|
| `exec_timeout_minutes` | int | 5          | 実行タイムアウト（分）、0 は無制限      |

## MCP ツール

MCP ツールは外部の Model Context Protocol サーバーとの統合を可能にします。

### ツールディスカバリ（遅延読み込み）

複数の MCP サーバーに接続する場合、数百のツールを同時に公開すると LLM のコンテキストウィンドウを使い果たし、API コストが増加する可能性があります。**Discovery** 機能は、MCP ツールをデフォルトで*非表示*にすることでこの問題を解決します。

すべてのツールを読み込む代わりに、LLM には軽量な検索ツール（BM25 キーワードマッチングまたは正規表現を使用）が提供されます。LLM が特定の機能を必要とする場合、非表示のライブラリを検索します。一致するツールは一時的に「アンロック」され、設定されたターン数（`ttl`）の間コンテキストに注入されます。

### グローバル設定

| 設定項目    | 型     | デフォルト | 説明                                 |
|-------------|--------|------------|--------------------------------------|
| `enabled`   | bool   | false      | MCP 統合をグローバルに有効にする     |
| `discovery` | object | `{}`       | ツールディスカバリ設定（下記参照）   |
| `servers`   | object | `{}`       | サーバー名からサーバー設定へのマップ |

### Discovery 設定（`discovery`）

| 設定項目             | 型   | デフォルト | 説明                                                                                                          |
|----------------------|------|------------|---------------------------------------------------------------------------------------------------------------|
| `enabled`            | bool | false      | true の場合、MCP ツールは非表示になり、検索を通じてオンデマンドで読み込まれます。false の場合、すべてのツールが読み込まれます |
| `ttl`                | int  | 5          | 発見されたツールがアンロック状態を維持する会話ターン数                                                        |
| `max_search_results` | int  | 5          | 検索クエリごとに返されるツールの最大数                                                                        |
| `use_bm25`           | bool | true       | 自然言語/キーワード検索ツール（`tool_search_tool_bm25`）を有効にする。**警告**：正規表現検索よりリソースを消費します |
| `use_regex`          | bool | false      | 正規表現パターン検索ツール（`tool_search_tool_regex`）を有効にする                                            |

> **注意：** `discovery.enabled` が `true` の場合、少なくとも1つの検索エンジン（`use_bm25` または `use_regex`）を有効にする**必要があります**。
> そうしないとアプリケーションの起動に失敗します。

### サーバーごとの設定

| 設定項目   | 型     | 必須     | 説明                                   |
|------------|--------|----------|----------------------------------------|
| `enabled`  | bool   | はい     | この MCP サーバーを有効にする          |
| `type`     | string | いいえ   | トランスポートタイプ：`stdio`、`sse`、`http` |
| `command`  | string | stdio    | stdio トランスポートの実行コマンド     |
| `args`     | array  | いいえ   | stdio トランスポートのコマンド引数     |
| `env`      | object | いいえ   | stdio プロセスの環境変数               |
| `env_file` | string | いいえ   | stdio プロセスの環境ファイルパス       |
| `url`      | string | sse/http | `sse`/`http` トランスポートのエンドポイント URL |
| `headers`  | object | いいえ   | `sse`/`http` トランスポートの HTTP ヘッダー |

### トランスポートの動作

- `type` を省略した場合、トランスポートは自動検出されます：
    - `url` が設定されている → `sse`
    - `command` が設定されている → `stdio`
- `http` と `sse` はどちらも `url` + オプションの `headers` を使用します。
- `env` と `env_file` は `stdio` サーバーにのみ適用されます。

### 設定例

#### 1) Stdio MCP サーバー

```json
{
  "tools": {
    "mcp": {
      "enabled": true,
      "servers": {
        "filesystem": {
          "enabled": true,
          "command": "npx",
          "args": [
            "-y",
            "@modelcontextprotocol/server-filesystem",
            "/tmp"
          ]
        }
      }
    }
  }
}
```

#### 2) リモート SSE/HTTP MCP サーバー

```json
{
  "tools": {
    "mcp": {
      "enabled": true,
      "servers": {
        "remote-mcp": {
          "enabled": true,
          "type": "sse",
          "url": "https://example.com/mcp",
          "headers": {
            "Authorization": "Bearer YOUR_TOKEN"
          }
        }
      }
    }
  }
}
```

#### 3) ツールディスカバリを有効にした大規模 MCP セットアップ

*この例では、LLM は `tool_search_tool_bm25` のみを認識します。ユーザーからリクエストがあった場合にのみ、Github や Postgres のツールを動的に検索してアンロックします。*

```json
{
  "tools": {
    "mcp": {
      "enabled": true,
      "discovery": {
        "enabled": true,
        "ttl": 5,
        "max_search_results": 5,
        "use_bm25": true,
        "use_regex": false
      },
      "servers": {
        "github": {
          "enabled": true,
          "command": "npx",
          "args": [
            "-y",
            "@modelcontextprotocol/server-github"
          ],
          "env": {
            "GITHUB_PERSONAL_ACCESS_TOKEN": "YOUR_GITHUB_TOKEN"
          }
        },
        "postgres": {
          "enabled": true,
          "command": "npx",
          "args": [
            "-y",
            "@modelcontextprotocol/server-postgres",
            "postgresql://user:password@localhost/dbname"
          ]
        },
        "slack": {
          "enabled": true,
          "command": "npx",
          "args": [
            "-y",
            "@modelcontextprotocol/server-slack"
          ],
          "env": {
            "SLACK_BOT_TOKEN": "YOUR_SLACK_BOT_TOKEN",
            "SLACK_TEAM_ID": "YOUR_SLACK_TEAM_ID"
          }
        }
      }
    }
  }
}
```

## Skills ツール

Skills ツールは ClawHub などのレジストリを通じたスキルの発見とインストールを設定します。

### レジストリ

| 設定項目                           | 型     | デフォルト           | 説明                                         |
|------------------------------------|--------|----------------------|----------------------------------------------|
| `registries.clawhub.enabled`       | bool   | true                 | ClawHub レジストリを有効にする               |
| `registries.clawhub.base_url`      | string | `https://clawhub.ai` | ClawHub ベース URL                           |
| `registries.clawhub.auth_token`    | string | `""`                 | より高いレート制限のためのオプションの Bearer トークン |
| `registries.clawhub.search_path`   | string | `/api/v1/search`     | 検索 API パス                                |
| `registries.clawhub.skills_path`   | string | `/api/v1/skills`     | Skills API パス                              |
| `registries.clawhub.download_path` | string | `/api/v1/download`   | ダウンロード API パス                        |

### 設定例

```json
{
  "tools": {
    "skills": {
      "registries": {
        "clawhub": {
          "enabled": true,
          "base_url": "https://clawhub.ai",
          "auth_token": "",
          "search_path": "/api/v1/search",
          "skills_path": "/api/v1/skills",
          "download_path": "/api/v1/download"
        }
      }
    }
  }
}
```

## 環境変数

すべての設定オプションは `PICOCLAW_TOOLS_<SECTION>_<KEY>` 形式の環境変数で上書きできます：

例：

- `PICOCLAW_TOOLS_WEB_BRAVE_ENABLED=true`
- `PICOCLAW_TOOLS_EXEC_ENABLED=false`
- `PICOCLAW_TOOLS_EXEC_ENABLE_DENY_PATTERNS=false`
- `PICOCLAW_TOOLS_CRON_EXEC_TIMEOUT_MINUTES=10`
- `PICOCLAW_TOOLS_MCP_ENABLED=true`

注意：ネストされたマップ形式の設定（例：`tools.mcp.servers.<name>.*`）は環境変数ではなく `config.json` で設定します。
