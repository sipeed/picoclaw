> 返回 [README](../../README.zh.md)

# Antigravity 认证与集成指南

## 概述

**Antigravity**（Google Cloud Code Assist）是由 Google 支持的 AI 模型提供商，通过 Google 的云基础设施提供对 Claude Opus 4.6 和 Gemini 等模型的访问。本文档提供了关于认证工作原理、如何获取模型以及如何在 PicoClaw 中实现新提供商的完整指南。

---

## 目录

1. [认证流程](#认证流程)
2. [OAuth 实现细节](#oauth-实现细节)
3. [令牌管理](#令牌管理)
4. [模型列表获取](#模型列表获取)
5. [用量追踪](#用量追踪)
6. [提供商插件结构](#提供商插件结构)
7. [集成要求](#集成要求)
8. [API 端点](#api-端点)
9. [配置](#配置)
10. [在 PicoClaw 中创建新提供商](#在-picoclaw-中创建新提供商)

---

## 认证流程

### 1. 带 PKCE 的 OAuth 2.0

Antigravity 使用 **OAuth 2.0 with PKCE（Proof Key for Code Exchange）** 进行安全认证：

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

### 2. 详细步骤

#### 步骤 1：生成 PKCE 参数
```typescript
function generatePkce(): { verifier: string; challenge: string } {
  const verifier = randomBytes(32).toString("hex");
  const challenge = createHash("sha256").update(verifier).digest("base64url");
  return { verifier, challenge };
}
```

#### 步骤 2：构建授权 URL
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

**所需权限范围：**
```typescript
const SCOPES = [
  "https://www.googleapis.com/auth/cloud-platform",
  "https://www.googleapis.com/auth/userinfo.email",
  "https://www.googleapis.com/auth/userinfo.profile",
  "https://www.googleapis.com/auth/cclog",
  "https://www.googleapis.com/auth/experimentsandconfigs",
];
```

#### 步骤 3：处理 OAuth 回调

**自动模式（本地开发）：**
- 在端口 51121 上启动本地 HTTP 服务器
- 等待来自 Google 的重定向
- 从查询参数中提取授权码

**手动模式（远程/无头环境）：**
- 向用户显示授权 URL
- 用户在浏览器中完成认证
- 用户将完整的重定向 URL 粘贴回终端
- 从粘贴的 URL 中解析授权码

#### 步骤 4：用授权码交换令牌
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

#### 步骤 5：获取额外的用户数据

**用户邮箱：**
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

**项目 ID（API 调用必需）：**
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
  return data.cloudaicompanionProject || "rising-fact-p41fc"; // 默认回退值
}
```

---

## OAuth 实现细节

### 客户端凭据

**重要：** 这些凭据在源代码中以 base64 编码存储，用于与 pi-ai 同步：

```typescript
const decode = (s: string) => Buffer.from(s, "base64").toString();

const CLIENT_ID = decode(
  "MTA3MTAwNjA2MDU5MS10bWhzc2luMmgyMWxjcmUyMzV2dG9sb2poNGc0MDNlcC5hcHBzLmdvb2dsZXVzZXJjb250ZW50LmNvbQ=="
);
const CLIENT_SECRET = decode("R09DU1BYLUs1OEZXUjQ4NkxkTEoxbUxCOHNYQzR6NnFEQWY=");
```

### OAuth 流程模式

1. **自动流程**（有浏览器的本地机器）：
   - 自动打开浏览器
   - 本地回调服务器捕获重定向
   - 初始认证后无需用户交互

2. **手动流程**（远程/无头/WSL2 环境）：
   - 显示 URL 供手动复制粘贴
   - 用户在外部浏览器中完成认证
   - 用户将完整的重定向 URL 粘贴回来

```typescript
function shouldUseManualOAuthFlow(isRemote: boolean): boolean {
  return isRemote || isWSL2Sync();
}
```

---

## 令牌管理

### 认证配置文件结构

```typescript
type OAuthCredential = {
  type: "oauth";
  provider: "google-antigravity";
  access: string;           // 访问令牌
  refresh: string;          // 刷新令牌
  expires: number;          // 过期时间戳（毫秒，自 epoch 起）
  email?: string;           // 用户邮箱
  projectId?: string;       // Google Cloud 项目 ID
};
```

### 令牌刷新

凭据包含一个刷新令牌，可在当前访问令牌过期时用于获取新的访问令牌。过期时间设置了 5 分钟的缓冲区以防止竞态条件。

---

## 模型列表获取

### 获取可用模型

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
  
  // 返回带有配额信息的模型
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

### 响应格式

```typescript
type FetchAvailableModelsResponse = {
  models?: Record<string, {
    displayName?: string;
    quotaInfo?: {
      remainingFraction?: number | string;
      resetTime?: string;      // ISO 8601 时间戳
      isExhausted?: boolean;
    };
  }>;
};
```

---

## 用量追踪

### 获取用量数据

```typescript
export async function fetchAntigravityUsage(
  token: string,
  timeoutMs: number
): Promise<ProviderUsageSnapshot> {
  // 1. 获取额度和计划信息
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

  // 提取额度信息
  const { availablePromptCredits, planInfo, currentTier } = data;
  
  // 2. 获取模型配额
  const modelsRes = await fetch(
    `${BASE_URL}/v1internal:fetchAvailableModels`,
    {
      method: "POST",
      headers: { Authorization: `Bearer ${token}` },
      body: JSON.stringify({ project: projectId }),
    }
  );

  // 构建用量窗口
  return {
    provider: "google-antigravity",
    displayName: "Google Antigravity",
    windows: [
      { label: "Credits", usedPercent: calculateUsedPercent(available, monthly) },
      // 各模型配额...
    ],
    plan: currentTier?.name || planType,
  };
}
```

### 用量响应结构

```typescript
type ProviderUsageSnapshot = {
  provider: "google-antigravity";
  displayName: string;
  windows: UsageWindow[];
  plan?: string;
  error?: string;
};

type UsageWindow = {
  label: string;           // "Credits" 或模型 ID
  usedPercent: number;     // 0-100
  resetAt?: number;        // 配额重置的时间戳
};
```

---

## 提供商插件结构

### 插件定义

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
            // OAuth 实现在此处
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
  prompter: WizardPrompter;      // UI 提示/通知
  runtime: RuntimeEnv;           // 日志等
  isRemote: boolean;             // 是否在远程运行
  openUrl: (url: string) => Promise<void>;  // 浏览器打开器
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

## 集成要求

### 1. 所需环境/依赖

- Go ≥ 1.25
- PicoClaw 代码库（`pkg/providers/` 和 `pkg/auth/`）
- `crypto` 和 `net/http` 标准库包

### 2. API 调用所需的请求头

```typescript
const REQUIRED_HEADERS = {
  "Authorization": `Bearer ${accessToken}`,
  "Content-Type": "application/json",
  "User-Agent": "antigravity",  // 或 "google-api-nodejs-client/9.15.1"
  "X-Goog-Api-Client": "google-cloud-sdk vscode_cloudshelleditor/0.1",
};

// 对于 loadCodeAssist 调用，还需包含：
const CLIENT_METADATA = {
  ideType: "ANTIGRAVITY",  // 或 "IDE_UNSPECIFIED"
  platform: "PLATFORM_UNSPECIFIED",
  pluginType: "GEMINI",
};
```

### 3. 模型 Schema 清理

Antigravity 使用兼容 Gemini 的模型，因此工具 schema 必须进行清理：

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

// 发送前清理 schema
function cleanToolSchemaForGemini(schema: Record<string, unknown>): unknown {
  // 移除不支持的关键字
  // 确保顶层有 type: "object"
  // 展平 anyOf/oneOf 联合类型
}
```

### 4. 思维块处理（Claude 模型）

对于 Antigravity 的 Claude 模型，思维块需要特殊处理：

```typescript
const ANTIGRAVITY_SIGNATURE_RE = /^[A-Za-z0-9+/]+={0,2}$/;

export function sanitizeAntigravityThinkingBlocks(
  messages: AgentMessage[]
): AgentMessage[] {
  // 验证思维签名
  // 规范化签名字段
  // 丢弃未签名的思维块
}
```

---

## API 端点

### 认证端点

| 端点 | 方法 | 用途 |
|------|------|------|
| `https://accounts.google.com/o/oauth2/v2/auth` | GET | OAuth 授权 |
| `https://oauth2.googleapis.com/token` | POST | 令牌交换 |
| `https://www.googleapis.com/oauth2/v1/userinfo` | GET | 用户信息（邮箱） |

### Cloud Code Assist 端点

| 端点 | 方法 | 用途 |
|------|------|------|
| `https://cloudcode-pa.googleapis.com/v1internal:loadCodeAssist` | POST | 加载项目信息、额度、计划 |
| `https://cloudcode-pa.googleapis.com/v1internal:fetchAvailableModels` | POST | 列出可用模型及配额 |
| `https://cloudcode-pa.googleapis.com/v1internal:streamGenerateContent?alt=sse` | POST | 聊天流式端点 |

**API 请求格式（聊天）：**
`v1internal:streamGenerateContent` 端点期望一个包装标准 Gemini 请求的信封格式：

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

**API 响应格式（SSE）：**
每条 SSE 消息（`data: {...}`）被包装在 `response` 字段中：

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

## 配置

### config.json 配置

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

### 认证配置文件存储

认证配置文件存储在 `~/.picoclaw/auth.json` 中：

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

## 在 PicoClaw 中创建新提供商

PicoClaw 提供商以 Go 包的形式实现，位于 `pkg/providers/` 下。要添加新提供商：

### 分步实现

#### 1. 创建提供商文件

在 `pkg/providers/` 中创建新的 Go 文件：

```
pkg/providers/
└── your_provider.go
```

#### 2. 实现 Provider 接口

你的提供商必须实现 `pkg/providers/types.go` 中定义的 `Provider` 接口：

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
    // 实现带流式传输的聊天补全
}
```

#### 3. 在工厂中注册

将你的提供商添加到 `pkg/providers/factory.go` 中的协议分支：

```go
case "your-provider":
    return NewYourProvider(sel.apiKey, sel.apiBase, sel.proxy), nil
```

#### 4. 添加默认配置（可选）

在 `pkg/config/defaults.go` 中添加默认条目：

```go
{
    ModelName: "your-model",
    Model:     "your-provider/model-name",
    APIKey:    "",
},
```

#### 5. 添加认证支持（可选）

如果你的提供商需要 OAuth 或特殊认证，在 `cmd/picoclaw/internal/auth/helpers.go` 中添加分支：

```go
case "your-provider":
    authLoginYourProvider()
```

#### 6. 通过 `config.json` 配置

```json
{
  "model_list": [
    {
      "model_name": "your-model",
      "model": "your-provider/model-name",
      "api_key": "your-api-key",
      "api_base": "https://api.your-provider.com/v1"
    }
  ]
}
```

---

## 测试你的实现

### CLI 命令

```bash
# 使用提供商进行认证
picoclaw auth login --provider your-provider

# 列出模型（用于 Antigravity）
picoclaw auth models

# 启动网关
picoclaw gateway

# 使用指定模型运行代理
picoclaw agent -m "Hello" --model your-model
```

### 测试用环境变量

```bash
# 覆盖默认模型
export PICOCLAW_AGENTS_DEFAULTS_MODEL=your-model

# 覆盖提供商设置
export PICOCLAW_MODEL_LIST='[{"model_name":"your-model","model":"your-provider/model-name","api_key":"..."}]'
```

---

## 参考资料

- **源文件：**
  - `pkg/providers/antigravity_provider.go` - Antigravity 提供商实现
  - `pkg/auth/oauth.go` - OAuth 流程实现
  - `pkg/auth/store.go` - 认证凭据存储（`~/.picoclaw/auth.json`）
  - `pkg/providers/factory.go` - 提供商工厂和协议路由
  - `pkg/providers/types.go` - 提供商接口定义
  - `cmd/picoclaw/internal/auth/helpers.go` - 认证 CLI 命令

- **文档：**
  - `docs/ANTIGRAVITY_USAGE.md` - Antigravity 使用指南
  - `docs/migration/model-list-migration.md` - 迁移指南

---

## 注意事项

1. **Google Cloud 项目：** Antigravity 要求在你的 Google Cloud 项目上启用 Gemini for Google Cloud
2. **配额：** 使用 Google Cloud 项目配额（非独立计费）
3. **模型访问：** 可用模型取决于你的 Google Cloud 项目配置
4. **思维块：** 通过 Antigravity 使用的 Claude 模型需要对带签名的思维块进行特殊处理
5. **Schema 清理：** 工具 schema 必须清理以移除不支持的 JSON Schema 关键字

---

---

## 常见错误处理

### 1. 速率限制（HTTP 429）

当项目/模型配额耗尽时，Antigravity 会返回 429 错误。错误响应通常在 `details` 字段中包含 `quotaResetDelay`。

**429 错误示例：**
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

### 2. 空响应（受限模型）

某些模型可能出现在可用模型列表中，但返回空响应（200 OK 但 SSE 流为空）。这通常发生在当前项目没有权限使用的预览版或受限模型上。

**处理方式：** 将空响应视为错误，通知用户该模型可能对其项目受限或无效。

---

## 故障排除

### "Token expired"（令牌已过期）
- 刷新 OAuth 令牌：`picoclaw auth login --provider antigravity`

### "Gemini for Google Cloud is not enabled"（Gemini for Google Cloud 未启用）
- 在 Google Cloud Console 中启用该 API

### "Project not found"（项目未找到）
- 确保你的 Google Cloud 项目已启用必要的 API
- 检查认证过程中项目 ID 是否正确获取

### 模型未出现在列表中
- 验证 OAuth 认证是否成功完成
- 检查认证配置文件存储：`~/.picoclaw/auth.json`
- 重新运行 `picoclaw auth login --provider antigravity`
