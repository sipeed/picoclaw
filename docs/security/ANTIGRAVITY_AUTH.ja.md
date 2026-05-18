> [README](../project/README.ja.md) に戻る

# Antigravity 認証・統合ガイド

## 概要

**Antigravity**（Google Cloud Code Assist）は、Google が提供する AI モデルプロバイダーで、Google のクラウドインフラストラクチャを通じて Claude Opus 4.6 や Gemini などのモデルへのアクセスを提供します。本ドキュメントでは、認証の仕組み、モデルの取得方法、PicoClaw での新しいプロバイダーの実装方法について完全なガイドを提供します。

---

## 目次

1. [認証フロー](#認証フロー)
2. [OAuth 実装の詳細](#oauth-実装の詳細)
3. [トークン管理](#トークン管理)
4. [モデルリストの取得](#モデルリストの取得)
5. [使用量トラッキング](#使用量トラッキング)
6. [プロバイダープラグイン構造](#プロバイダープラグイン構造)
7. [統合要件](#統合要件)
8. [API エンドポイント](#api-エンドポイント)
9. [設定](#設定)
10. [PicoClaw での新しいプロバイダーの作成](#picoclaw-での新しいプロバイダーの作成)

---

## 認証フロー

### 1. PKCE 付き OAuth 2.0

Antigravity はセキュアな認証のために **OAuth 2.0 with PKCE（Proof Key for Code Exchange）** を使用します：

```
┌─────────────┐                                    ┌─────────────────┐
│   Client    │ ───(1) Generate PKCE Pair────────> │                 │
│             │ ───(2) Open Auth URL─────────────> │  Google OAuth   │
│             │                                    │    Server       │
│             │ <──(3) Redirect with Code───────── │                 │
│             │                                    └─────────────────┘
│             │ ───(4) Exchange Code for Tokens──> │   Token URL     │
│             │                                    │                 │
│             │ <──(5) Access + Refresh Tokens──── │                 │
└─────────────┘                                    └─────────────────┘
```

### 2. 詳細手順

#### ステップ 1：PKCE パラメータの生成
```typescript
function generatePkce(): { verifier: string; challenge: string } {
  const verifier = randomBytes(32).toString("hex");
  const challenge = createHash("sha256").update(verifier).digest("base64url");
  return { verifier, challenge };
}
```

#### ステップ 2：認可 URL の構築
```typescript
const AUTH_URL = "https://accounts.google.com/o/oauth2/v2/auth";
const REDIRECT_URI = "http://localhost:51121/oauth-callback";

function buildAuthUrl(params: { challenge: string; state: string }): string {
  const url = new URL(AUTH_URL);
  url.searchParams.set("client_id", CLIENT_ID);
  url.searchParams.set("response_type", "code");
  url.searchParams.set("redirect_uri", REDIRECT_URI);
  url.searchParams.set("scope", SCOPES.join(" "));
  url.searchParams.set("code_challenge", params.challenge);
  url.searchParams.set("code_challenge_method", "S256");
  url.searchParams.set("state", params.state);
  url.searchParams.set("access_type", "offline");
  url.searchParams.set("prompt", "consent");
  return url.toString();
}
```

**必要なスコープ：**
```typescript
const SCOPES = [
  "https://www.googleapis.com/auth/cloud-platform",
  "https://www.googleapis.com/auth/userinfo.email",
  "https://www.googleapis.com/auth/userinfo.profile",
  "https://www.googleapis.com/auth/cclog",
  "https://www.googleapis.com/auth/experimentsandconfigs",
];
```

#### ステップ 3：OAuth コールバックの処理

**自動モード（ローカル開発）：**
- ポート 51121 でローカル HTTP サーバーを起動
- Google からのリダイレクトを待機
- クエリパラメータから認可コードを抽出

**手動モード（リモート/ヘッドレス）：**
- ユーザーに認可 URL を表示
- ユーザーがブラウザで認証を完了
- ユーザーが完全なリダイレクト URL をターミナルに貼り付け
- 貼り付けられた URL からコードを解析

#### ステップ 4：コードをトークンに交換
```typescript
const TOKEN_URL = "https://oauth2.googleapis.com/token";

async function exchangeCode(params: {
  code: string;
  verifier: string;
}): Promise<{ access: string; refresh: string; expires: number }> {
  const response = await fetch(TOKEN_URL, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({
      client_id: CLIENT_ID,
      client_secret: CLIENT_SECRET,
      code: params.code,
      grant_type: "authorization_code",
      redirect_uri: REDIRECT_URI,
      code_verifier: params.verifier,
    }),
  });

  const data = await response.json();
  
  return {
    access: data.access_token,
    refresh: data.refresh_token,
    expires: Date.now() + data.expires_in * 1000 - 5 * 60 * 1000, // 5 min buffer
  };
}
```

#### ステップ 5：追加のユーザーデータの取得

**ユーザーメール：**
```typescript
async function fetchUserEmail(accessToken: string): Promise<string | undefined> {
  const response = await fetch(
    "https://www.googleapis.com/oauth2/v1/userinfo?alt=json",
    { headers: { Authorization: `Bearer ${accessToken}` } }
  );
  const data = await response.json();
  return data.email;
}
```

**プロジェクト ID（API 呼び出しに必須）：**
```typescript
async function fetchProjectId(accessToken: string): Promise<string> {
  const headers = {
    Authorization: `Bearer ${accessToken}`,
    "Content-Type": "application/json",
    "User-Agent": "google-api-nodejs-client/9.15.1",
    "X-Goog-Api-Client": "google-cloud-sdk vscode_cloudshelleditor/0.1",
    "Client-Metadata": JSON.stringify({
      ideType: "IDE_UNSPECIFIED",
      platform: "PLATFORM_UNSPECIFIED",
      pluginType: "GEMINI",
    }),
  };

  const response = await fetch(
    "https://cloudcode-pa.googleapis.com/v1internal:loadCodeAssist",
    {
      method: "POST",
      headers,
      body: JSON.stringify({
        metadata: {
          ideType: "IDE_UNSPECIFIED",
          platform: "PLATFORM_UNSPECIFIED",
          pluginType: "GEMINI",
        },
      }),
    }
  );

  const data = await response.json();
  return data.cloudaicompanionProject || "rising-fact-p41fc"; // デフォルトのフォールバック
}
```

---

## OAuth 実装の詳細

### クライアント認証情報

**重要：** これらは pi-ai との同期のためにソースコード内で base64 エンコードされています：

```typescript
const decode = (s: string) => Buffer.from(s, "base64").toString();

const CLIENT_ID = decode(
  "MTA3MTAwNjA2MDU5MS10bWhzc2luMmgyMWxjcmUyMzV2dG9sb2poNGc0MDNlcC5hcHBzLmdvb2dsZXVzZXJjb250ZW50LmNvbQ=="
);
const CLIENT_SECRET = decode("R09DU1BYLUs1OEZXUjQ4NkxkTEoxbUxCOHNYQzR6NnFEQWY=");
```

### OAuth フローモード

1. **自動フロー**（ブラウザのあるローカルマシン）：
   - ブラウザを自動的に開く
   - ローカルコールバックサーバーがリダイレクトをキャプチャ
   - 初回認証後はユーザー操作不要

2. **手動フロー**（リモート/ヘッドレス/WSL2）：
   - 手動コピー＆ペースト用の URL を表示
   - ユーザーが外部ブラウザで認証を完了
   - ユーザーが完全なリダイレクト URL を貼り付け

```typescript
function shouldUseManualOAuthFlow(isRemote: boolean): boolean {
  return isRemote || isWSL2Sync();
}
```

---

## トークン管理

### 認証プロファイル構造

```typescript
type OAuthCredential = {
  type: "oauth";
  provider: "google-antigravity";
  access: string;           // アクセストークン
  refresh: string;          // リフレッシュトークン
  expires: number;          // 有効期限タイムスタンプ（エポックからのミリ秒）
  email?: string;           // ユーザーメール
  projectId?: string;       // Google Cloud プロジェクト ID
};
```

### トークンの更新

認証情報にはリフレッシュトークンが含まれており、現在のアクセストークンが期限切れになった際に新しいアクセストークンを取得するために使用できます。有効期限は競合状態を防ぐために 5 分のバッファを設けています。

---

## モデルリストの取得

### 利用可能なモデルの取得

```typescript
const BASE_URL = "https://cloudcode-pa.googleapis.com";

async function fetchAvailableModels(
  accessToken: string,
  projectId: string
): Promise<Model[]> {
  const headers = {
    Authorization: `Bearer ${accessToken}`,
    "Content-Type": "application/json",
    "User-Agent": "antigravity",
    "X-Goog-Api-Client": "google-cloud-sdk vscode_cloudshelleditor/0.1",
  };

  const response = await fetch(
    `${BASE_URL}/v1internal:fetchAvailableModels`,
    {
      method: "POST",
      headers,
      body: JSON.stringify({ project: projectId }),
    }
  );

  const data = await response.json();
  
  // クォータ情報付きのモデルを返す
  return Object.entries(data.models).map(([modelId, modelInfo]) => ({
    id: modelId,
    displayName: modelInfo.displayName,
    quotaInfo: {
      remainingFraction: modelInfo.quotaInfo?.remainingFraction,
      resetTime: modelInfo.quotaInfo?.resetTime,
      isExhausted: modelInfo.quotaInfo?.isExhausted,
    },
  }));
}
```

### レスポンス形式

```typescript
type FetchAvailableModelsResponse = {
  models?: Record<string, {
    displayName?: string;
    quotaInfo?: {
      remainingFraction?: number | string;
      resetTime?: string;      // ISO 8601 タイムスタンプ
      isExhausted?: boolean;
    };
  }>;
};
```

---

## 使用量トラッキング

### 使用量データの取得

```typescript
export async function fetchAntigravityUsage(
  token: string,
  timeoutMs: number
): Promise<ProviderUsageSnapshot> {
  // 1. クレジットとプラン情報を取得
  const loadCodeAssistRes = await fetch(
    `${BASE_URL}/v1internal:loadCodeAssist`,
    {
      method: "POST",
      headers: {
        Authorization: `Bearer ${token}`,
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        metadata: {
          ideType: "ANTIGRAVITY",
          platform: "PLATFORM_UNSPECIFIED",
          pluginType: "GEMINI",
        },
      }),
    }
  );

  // クレジット情報を抽出
  const { availablePromptCredits, planInfo, currentTier } = data;
  
  // 2. モデルクォータを取得
  const modelsRes = await fetch(
    `${BASE_URL}/v1internal:fetchAvailableModels`,
    {
      method: "POST",
      headers: { Authorization: `Bearer ${token}` },
      body: JSON.stringify({ project: projectId }),
    }
  );

  // 使用量ウィンドウを構築
  return {
    provider: "google-antigravity",
    displayName: "Google Antigravity",
    windows: [
      { label: "Credits", usedPercent: calculateUsedPercent(available, monthly) },
      // 個別モデルクォータ...
    ],
    plan: currentTier?.name || planType,
  };
}
```

### 使用量レスポンス構造

```typescript
type ProviderUsageSnapshot = {
  provider: "google-antigravity";
  displayName: string;
  windows: UsageWindow[];
  plan?: string;
  error?: string;
};

type UsageWindow = {
  label: string;           // "Credits" またはモデル ID
  usedPercent: number;     // 0-100
  resetAt?: number;        // クォータがリセットされるタイムスタンプ
};
```

---

## プロバイダープラグイン構造

### プラグイン定義

```typescript
const antigravityPlugin = {
  id: "google-antigravity-auth",
  name: "Google Antigravity Auth",
  description: "OAuth flow for Google Antigravity (Cloud Code Assist)",
  configSchema: emptyPluginConfigSchema(),
  
  register(api: PicoClawPluginApi) {
    api.registerProvider({
      id: "google-antigravity",
      label: "Google Antigravity",
      docsPath: "/providers/models",
      aliases: ["antigravity"],
      
      auth: [
        {
          id: "oauth",
          label: "Google OAuth",
          hint: "PKCE + localhost callback",
          kind: "oauth",
          run: async (ctx: ProviderAuthContext) => {
            // OAuth 実装はここに記述
          },
        },
      ],
    });
  },
};
```

### ProviderAuthContext

```typescript
type ProviderAuthContext = {
  config: PicoClawConfig;
  agentDir?: string;
  workspaceDir?: string;
  prompter: WizardPrompter;      // UI プロンプト/通知
  runtime: RuntimeEnv;           // ログなど
  isRemote: boolean;             // リモート実行かどうか
  openUrl: (url: string) => Promise<void>;  // ブラウザオープナー
  oauth: {
    createVpsAwareHandlers: Function;
  };
};
```

### ProviderAuthResult

```typescript
type ProviderAuthResult = {
  profiles: Array<{
    profileId: string;
    credential: AuthProfileCredential;
  }>;
  configPatch?: Partial<PicoClawConfig>;
  defaultModel?: string;
  notes?: string[];
};
```

---

## 統合要件

### 1. 必要な環境/依存関係

- Go ≥ 1.25
- PicoClaw コードベース（`pkg/providers/` および `pkg/auth/`）
- `crypto` および `net/http` 標準ライブラリパッケージ

### 2. API 呼び出しに必要なヘッダー

```typescript
const REQUIRED_HEADERS = {
  "Authorization": `Bearer ${accessToken}`,
  "Content-Type": "application/json",
  "User-Agent": "antigravity",  // または "google-api-nodejs-client/9.15.1"
  "X-Goog-Api-Client": "google-cloud-sdk vscode_cloudshelleditor/0.1",
};

// loadCodeAssist 呼び出しには以下も含める：
const CLIENT_METADATA = {
  ideType: "ANTIGRAVITY",  // または "IDE_UNSPECIFIED"
  platform: "PLATFORM_UNSPECIFIED",
  pluginType: "GEMINI",
};
```

### 3. モデルスキーマのサニタイズ

Antigravity は Gemini 互換モデルを使用するため、ツールスキーマのサニタイズが必要です：

```typescript
const GOOGLE_SCHEMA_UNSUPPORTED_KEYWORDS = new Set([
  "patternProperties",
  "additionalProperties",
  "$schema",
  "$id",
  "$ref",
  "$defs",
  "definitions",
  "examples",
  "minLength",
  "maxLength",
  "minimum",
  "maximum",
  "multipleOf",
  "pattern",
  "format",
  "minItems",
  "maxItems",
  "uniqueItems",
  "minProperties",
  "maxProperties",
]);

// 送信前にスキーマをクリーンアップ
function cleanToolSchemaForGemini(schema: Record<string, unknown>): unknown {
  // サポートされていないキーワードを削除
  // トップレベルに type: "object" があることを確認
  // anyOf/oneOf ユニオンをフラット化
}
```

### 4. 思考ブロックの処理（Claude モデル）

Antigravity の Claude モデルでは、思考ブロックに特別な処理が必要です：

```typescript
const ANTIGRAVITY_SIGNATURE_RE = /^[A-Za-z0-9+/]+={0,2}$/;

export function sanitizeAntigravityThinkingBlocks(
  messages: AgentMessage[]
): AgentMessage[] {
  // 思考シグネチャを検証
  // シグネチャフィールドを正規化
  // 署名されていない思考ブロックを破棄
}
```

---

## API エンドポイント

### 認証エンドポイント

| エンドポイント | メソッド | 用途 |
|---------------|---------|------|
| `https://accounts.google.com/o/oauth2/v2/auth` | GET | OAuth 認可 |
| `https://oauth2.googleapis.com/token` | POST | トークン交換 |
| `https://www.googleapis.com/oauth2/v1/userinfo` | GET | ユーザー情報（メール） |

### Cloud Code Assist エンドポイント

| エンドポイント | メソッド | 用途 |
|---------------|---------|------|
| `https://cloudcode-pa.googleapis.com/v1internal:loadCodeAssist` | POST | プロジェクト情報、クレジット、プランの読み込み |
| `https://cloudcode-pa.googleapis.com/v1internal:fetchAvailableModels` | POST | クォータ付き利用可能モデルの一覧 |
| `https://cloudcode-pa.googleapis.com/v1internal:streamGenerateContent?alt=sse` | POST | チャットストリーミングエンドポイント |

**API リクエスト形式（チャット）：**
`v1internal:streamGenerateContent` エンドポイントは、標準の Gemini リクエストをラップするエンベロープ形式を期待します：

```json
{
  "project": "your-project-id",
  "model": "model-id",
  "request": {
    "contents": [...],
    "systemInstruction": {...},
    "generationConfig": {...},
    "tools": [...]
  },
  "requestType": "agent",
  "userAgent": "antigravity",
  "requestId": "agent-timestamp-random"
}
```

**API レスポンス形式（SSE）：**
各 SSE メッセージ（`data: {...}`）は `response` フィールドでラップされます：

```json
{
  "response": {
    "candidates": [...],
    "usageMetadata": {...},
    "modelVersion": "...",
    "responseId": "..."
  },
  "traceId": "...",
  "metadata": {}
}
```

---

## 設定

### config.json の設定

```json
{
  "model_list": [
    {
      "model_name": "gemini-flash",
      "model": "antigravity/gemini-3-flash",
      "auth_method": "oauth"
    }
  ],
  "agents": {
    "defaults": {
      "model_name": "gemini-flash"
    }
  }
}
```

### 認証プロファイルの保存

認証プロファイルは `~/.picoclaw/auth.json` に保存されます：

```json
{
  "credentials": {
    "google-antigravity": {
      "access_token": "ya29...",
      "refresh_token": "1//...",
      "expires_at": "2026-01-01T00:00:00Z",
      "provider": "google-antigravity",
      "auth_method": "oauth",
      "email": "user@example.com",
      "project_id": "my-project-id"
    }
  }
}
```

---

## PicoClaw での新しいプロバイダーの作成

PicoClaw のプロバイダーは `pkg/providers/` 配下の Go パッケージとして実装されます。新しいプロバイダーを追加するには：

### ステップバイステップの実装

#### 1. プロバイダーファイルの作成

`pkg/providers/` に新しい Go ファイルを作成します：

```
pkg/providers/
└── your_provider.go
```

#### 2. Provider インターフェースの実装

プロバイダーは `pkg/providers/types.go` で定義された `Provider` インターフェースを実装する必要があります：

```go
package providers

type YourProvider struct {
    apiKey  string
    apiBase string
}

func NewYourProvider(apiKey, apiBase, proxy string) *YourProvider {
    if apiBase == "" {
        apiBase = "https://api.your-provider.com/v1"
    }
    return &YourProvider{apiKey: apiKey, apiBase: apiBase}
}

func (p *YourProvider) Chat(ctx context.Context, messages []Message, tools []Tool, cb StreamCallback) error {
    // ストリーミング付きチャット補完を実装
}
```

#### 3. ファクトリーへの登録

`pkg/providers/factory.go` のプロトコルスイッチにプロバイダーを追加します：

```go
case "your-provider":
    return NewYourProvider(sel.apiKey, sel.apiBase, sel.proxy), nil
```

#### 4. デフォルト設定の追加（オプション）

`pkg/config/defaults.go` にデフォルトエントリを追加します：

```go
{
    ModelName: "your-model",
    Model:     "your-provider/model-name",
    APIKey:    "",
},
```

#### 5. 認証サポートの追加（オプション）

プロバイダーが OAuth や特別な認証を必要とする場合、`cmd/picoclaw/internal/auth/helpers.go` にケースを追加します：

```go
case "your-provider":
    authLoginYourProvider()
```

#### 6. `config.json` での設定

```json
{
  "model_list": [
    {
      "model_name": "your-model",
      "model": "your-provider/model-name",
      "api_keys": ["your-api-key"],
      "api_base": "https://api.your-provider.com/v1"
    }
  ]
}
```

---

## 実装のテスト

### CLI コマンド

```bash
# プロバイダーで認証
picoclaw auth login --provider your-provider

# モデルの一覧表示（Antigravity 用）
picoclaw auth models

# ゲートウェイの起動
picoclaw gateway

# 特定のモデルでエージェントを実行
picoclaw agent -m "Hello" --model your-model
```

### テスト用環境変数

```bash
# デフォルトモデルの上書き
export PICOCLAW_AGENTS_DEFAULTS_MODEL=your-model

# プロバイダー設定の上書き
export PICOCLAW_MODEL_LIST='[{"model_name":"your-model","model":"your-provider/model-name","api_keys":["..."]}]'
```

---

## 参考資料

- **ソースファイル：**
  - `pkg/providers/antigravity_provider.go` - Antigravity プロバイダー実装
  - `pkg/auth/oauth.go` - OAuth フロー実装
  - `pkg/auth/store.go` - 認証情報ストレージ（`~/.picoclaw/auth.json`）
  - `pkg/providers/factory.go` - プロバイダーファクトリーとプロトコルルーティング
  - `pkg/providers/types.go` - プロバイダーインターフェース定義
  - `cmd/picoclaw/internal/auth/helpers.go` - 認証 CLI コマンド

- **ドキュメント：**
  - `docs/ANTIGRAVITY_USAGE.md` - Antigravity 使用ガイド
  - `docs/migration/model-list-migration.md` - 移行ガイド

---

## 注意事項

1. **Google Cloud プロジェクト：** Antigravity は Google Cloud プロジェクトで Gemini for Google Cloud が有効になっている必要があります
2. **クォータ：** Google Cloud プロジェクトのクォータを使用します（個別の課金ではありません）
3. **モデルアクセス：** 利用可能なモデルは Google Cloud プロジェクトの設定に依存します
4. **思考ブロック：** Antigravity 経由の Claude モデルは、署名付き思考ブロックの特別な処理が必要です
5. **スキーマサニタイズ：** ツールスキーマはサポートされていない JSON Schema キーワードを削除するためにサニタイズが必要です

---

---

## 一般的なエラー処理

### 1. レート制限（HTTP 429）

プロジェクト/モデルのクォータが枯渇すると、Antigravity は 429 エラーを返します。エラーレスポンスには通常、`details` フィールドに `quotaResetDelay` が含まれます。

**429 エラーの例：**
```json
{
  "error": {
    "code": 429,
    "message": "You have exhausted your capacity on this model. Your quota will reset after 4h30m28s.",
    "status": "RESOURCE_EXHAUSTED",
    "details": [
      {
        "@type": "type.googleapis.com/google.rpc.ErrorInfo",
        "metadata": {
          "quotaResetDelay": "4h30m28.060903746s"
        }
      }
    ]
  }
}
```

### 2. 空のレスポンス（制限付きモデル）

一部のモデルは利用可能モデルリストに表示されますが、空のレスポンスを返す場合があります（200 OK だが SSE ストリームが空）。これは通常、現在のプロジェクトに使用権限がないプレビュー版または制限付きモデルで発生します。

**対処法：** 空のレスポンスをエラーとして扱い、そのモデルがプロジェクトに対して制限されているか無効である可能性があることをユーザーに通知します。

---

## トラブルシューティング

### "Token expired"（トークン期限切れ）
- OAuth トークンを更新：`picoclaw auth login --provider antigravity`

### "Gemini for Google Cloud is not enabled"（Gemini for Google Cloud が有効になっていない）
- Google Cloud Console で API を有効にしてください

### "Project not found"（プロジェクトが見つからない）
- Google Cloud プロジェクトで必要な API が有効になっていることを確認してください
- 認証中にプロジェクト ID が正しく取得されているか確認してください

### モデルがリストに表示されない
- OAuth 認証が正常に完了したことを確認してください
- 認証プロファイルストレージを確認：`~/.picoclaw/auth.json`
- `picoclaw auth login --provider antigravity` を再実行してください
