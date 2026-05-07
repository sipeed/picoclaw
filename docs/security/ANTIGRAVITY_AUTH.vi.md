> Quay lại [README](../project/README.vi.md)

# Hướng dẫn Xác thực và Tích hợp Antigravity

## Tổng quan

**Antigravity** (Google Cloud Code Assist) là nhà cung cấp mô hình AI được Google hỗ trợ, cung cấp quyền truy cập vào các mô hình như Claude Opus 4.6 và Gemini thông qua hạ tầng đám mây của Google. Tài liệu này cung cấp hướng dẫn đầy đủ về cách xác thực hoạt động, cách lấy danh sách mô hình và cách triển khai nhà cung cấp mới trong PicoClaw.

---

## Mục lục

1. [Luồng xác thực](#luồng-xác-thực)
2. [Chi tiết triển khai OAuth](#chi-tiết-triển-khai-oauth)
3. [Quản lý token](#quản-lý-token)
4. [Lấy danh sách mô hình](#lấy-danh-sách-mô-hình)
5. [Theo dõi mức sử dụng](#theo-dõi-mức-sử-dụng)
6. [Cấu trúc plugin nhà cung cấp](#cấu-trúc-plugin-nhà-cung-cấp)
7. [Yêu cầu tích hợp](#yêu-cầu-tích-hợp)
8. [Các endpoint API](#các-endpoint-api)
9. [Cấu hình](#cấu-hình)
10. [Tạo nhà cung cấp mới trong PicoClaw](#tạo-nhà-cung-cấp-mới-trong-picoclaw)

---

## Luồng xác thực

### 1. OAuth 2.0 với PKCE

Antigravity sử dụng **OAuth 2.0 với PKCE (Proof Key for Code Exchange)** để xác thực an toàn:

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

### 2. Các bước chi tiết

#### Bước 1: Tạo tham số PKCE
```typescript
function generatePkce(): { verifier: string; challenge: string } {
  const verifier = randomBytes(32).toString("hex");
  const challenge = createHash("sha256").update(verifier).digest("base64url");
  return { verifier, challenge };
}
```

#### Bước 2: Xây dựng URL ủy quyền
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

**Các phạm vi quyền cần thiết:**
```typescript
const SCOPES = [
  "https://www.googleapis.com/auth/cloud-platform",
  "https://www.googleapis.com/auth/userinfo.email",
  "https://www.googleapis.com/auth/userinfo.profile",
  "https://www.googleapis.com/auth/cclog",
  "https://www.googleapis.com/auth/experimentsandconfigs",
];
```

#### Bước 3: Xử lý callback OAuth

**Chế độ tự động (Phát triển cục bộ):**
- Khởi động máy chủ HTTP cục bộ trên cổng 51121
- Chờ chuyển hướng từ Google
- Trích xuất mã ủy quyền từ tham số truy vấn

**Chế độ thủ công (Từ xa/Không có giao diện):**
- Hiển thị URL ủy quyền cho người dùng
- Người dùng hoàn tất xác thực trong trình duyệt
- Người dùng dán URL chuyển hướng đầy đủ vào terminal
- Phân tích mã từ URL đã dán

#### Bước 4: Đổi mã lấy token
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

#### Bước 5: Lấy dữ liệu người dùng bổ sung

**Email người dùng:**
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

**ID dự án (Bắt buộc cho các lệnh gọi API):**
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
  return data.cloudaicompanionProject || "rising-fact-p41fc"; // Giá trị mặc định dự phòng
}
```

---

## Chi tiết triển khai OAuth

### Thông tin xác thực client

**Quan trọng:** Các giá trị này được mã hóa base64 trong mã nguồn để đồng bộ với pi-ai:

```typescript
const decode = (s: string) => Buffer.from(s, "base64").toString();

const CLIENT_ID = decode(
  "MTA3MTAwNjA2MDU5MS10bWhzc2luMmgyMWxjcmUyMzV2dG9sb2poNGc0MDNlcC5hcHBzLmdvb2dsZXVzZXJjb250ZW50LmNvbQ=="
);
const CLIENT_SECRET = decode("R09DU1BYLUs1OEZXUjQ4NkxkTEoxbUxCOHNYQzR6NnFEQWY=");
```

### Các chế độ luồng OAuth

1. **Luồng tự động** (Máy cục bộ có trình duyệt):
   - Tự động mở trình duyệt
   - Máy chủ callback cục bộ bắt chuyển hướng
   - Không cần tương tác người dùng sau xác thực ban đầu

2. **Luồng thủ công** (Từ xa/Không có giao diện/WSL2):
   - Hiển thị URL để sao chép-dán thủ công
   - Người dùng hoàn tất xác thực trong trình duyệt bên ngoài
   - Người dùng dán lại URL chuyển hướng đầy đủ

```typescript
function shouldUseManualOAuthFlow(isRemote: boolean): boolean {
  return isRemote || isWSL2Sync();
}
```

---

## Quản lý token

### Cấu trúc hồ sơ xác thực

```typescript
type OAuthCredential = {
  type: "oauth";
  provider: "google-antigravity";
  access: string;           // Token truy cập
  refresh: string;          // Token làm mới
  expires: number;          // Dấu thời gian hết hạn (ms kể từ epoch)
  email?: string;           // Email người dùng
  projectId?: string;       // ID dự án Google Cloud
};
```

### Làm mới token

Thông tin xác thực bao gồm token làm mới có thể được sử dụng để lấy token truy cập mới khi token hiện tại hết hạn. Thời gian hết hạn được đặt với bộ đệm 5 phút để tránh điều kiện tranh chấp.

---

## Lấy danh sách mô hình

### Lấy các mô hình khả dụng

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
  
  // Trả về các mô hình kèm thông tin hạn mức
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

### Định dạng phản hồi

```typescript
type FetchAvailableModelsResponse = {
  models?: Record<string, {
    displayName?: string;
    quotaInfo?: {
      remainingFraction?: number | string;
      resetTime?: string;      // Dấu thời gian ISO 8601
      isExhausted?: boolean;
    };
  }>;
};
```

---

## Theo dõi mức sử dụng

### Lấy dữ liệu sử dụng

```typescript
export async function fetchAntigravityUsage(
  token: string,
  timeoutMs: number
): Promise<ProviderUsageSnapshot> {
  // 1. Lấy thông tin tín dụng và gói dịch vụ
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

  // Trích xuất thông tin tín dụng
  const { availablePromptCredits, planInfo, currentTier } = data;
  
  // 2. Lấy hạn mức mô hình
  const modelsRes = await fetch(
    `${BASE_URL}/v1internal:fetchAvailableModels`,
    {
      method: "POST",
      headers: { Authorization: `Bearer ${token}` },
      body: JSON.stringify({ project: projectId }),
    }
  );

  // Xây dựng cửa sổ sử dụng
  return {
    provider: "google-antigravity",
    displayName: "Google Antigravity",
    windows: [
      { label: "Credits", usedPercent: calculateUsedPercent(available, monthly) },
      // Hạn mức từng mô hình...
    ],
    plan: currentTier?.name || planType,
  };
}
```

### Cấu trúc phản hồi sử dụng

```typescript
type ProviderUsageSnapshot = {
  provider: "google-antigravity";
  displayName: string;
  windows: UsageWindow[];
  plan?: string;
  error?: string;
};

type UsageWindow = {
  label: string;           // "Credits" hoặc ID mô hình
  usedPercent: number;     // 0-100
  resetAt?: number;        // Dấu thời gian khi hạn mức được đặt lại
};
```

---

## Cấu trúc plugin nhà cung cấp

### Định nghĩa plugin

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
            // Triển khai OAuth tại đây
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
  prompter: WizardPrompter;      // Lời nhắc/thông báo UI
  runtime: RuntimeEnv;           // Ghi log, v.v.
  isRemote: boolean;             // Có đang chạy từ xa không
  openUrl: (url: string) => Promise<void>;  // Mở trình duyệt
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

## Yêu cầu tích hợp

### 1. Môi trường/Phụ thuộc cần thiết

- Go ≥ 1.25
- Mã nguồn PicoClaw (`pkg/providers/` và `pkg/auth/`)
- Các gói thư viện chuẩn `crypto` và `net/http`

### 2. Các header bắt buộc cho lệnh gọi API

```typescript
const REQUIRED_HEADERS = {
  "Authorization": `Bearer ${accessToken}`,
  "Content-Type": "application/json",
  "User-Agent": "antigravity",  // hoặc "google-api-nodejs-client/9.15.1"
  "X-Goog-Api-Client": "google-cloud-sdk vscode_cloudshelleditor/0.1",
};

// Đối với các lệnh gọi loadCodeAssist, cũng bao gồm:
const CLIENT_METADATA = {
  ideType: "ANTIGRAVITY",  // hoặc "IDE_UNSPECIFIED"
  platform: "PLATFORM_UNSPECIFIED",
  pluginType: "GEMINI",
};
```

### 3. Làm sạch schema mô hình

Antigravity sử dụng các mô hình tương thích Gemini, vì vậy schema công cụ phải được làm sạch:

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

// Làm sạch schema trước khi gửi
function cleanToolSchemaForGemini(schema: Record<string, unknown>): unknown {
  // Xóa các từ khóa không được hỗ trợ
  // Đảm bảo cấp cao nhất có type: "object"
  // Làm phẳng các union anyOf/oneOf
}
```

### 4. Xử lý khối suy nghĩ (Mô hình Claude)

Đối với các mô hình Claude qua Antigravity, khối suy nghĩ cần xử lý đặc biệt:

```typescript
const ANTIGRAVITY_SIGNATURE_RE = /^[A-Za-z0-9+/]+={0,2}$/;

export function sanitizeAntigravityThinkingBlocks(
  messages: AgentMessage[]
): AgentMessage[] {
  // Xác thực chữ ký suy nghĩ
  // Chuẩn hóa các trường chữ ký
  // Loại bỏ các khối suy nghĩ chưa ký
}
```

---

## Các endpoint API

### Endpoint xác thực

| Endpoint | Phương thức | Mục đích |
|----------|------------|----------|
| `https://accounts.google.com/o/oauth2/v2/auth` | GET | Ủy quyền OAuth |
| `https://oauth2.googleapis.com/token` | POST | Trao đổi token |
| `https://www.googleapis.com/oauth2/v1/userinfo` | GET | Thông tin người dùng (email) |

### Endpoint Cloud Code Assist

| Endpoint | Phương thức | Mục đích |
|----------|------------|----------|
| `https://cloudcode-pa.googleapis.com/v1internal:loadCodeAssist` | POST | Tải thông tin dự án, tín dụng, gói dịch vụ |
| `https://cloudcode-pa.googleapis.com/v1internal:fetchAvailableModels` | POST | Liệt kê các mô hình khả dụng kèm hạn mức |
| `https://cloudcode-pa.googleapis.com/v1internal:streamGenerateContent?alt=sse` | POST | Endpoint streaming chat |

**Định dạng yêu cầu API (Chat):**
Endpoint `v1internal:streamGenerateContent` yêu cầu một envelope bao bọc yêu cầu Gemini tiêu chuẩn:

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

**Định dạng phản hồi API (SSE):**
Mỗi thông điệp SSE (`data: {...}`) được bao bọc trong trường `response`:

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

## Cấu hình

### Cấu hình config.json

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

### Lưu trữ hồ sơ xác thực

Hồ sơ xác thực được lưu trữ trong `~/.picoclaw/auth.json`:

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

## Tạo nhà cung cấp mới trong PicoClaw

Các nhà cung cấp PicoClaw được triển khai dưới dạng gói Go trong `pkg/providers/`. Để thêm nhà cung cấp mới:

### Triển khai từng bước

#### 1. Tạo file nhà cung cấp

Tạo file Go mới trong `pkg/providers/`:

```
pkg/providers/
└── your_provider.go
```

#### 2. Triển khai interface Provider

Nhà cung cấp của bạn phải triển khai interface `Provider` được định nghĩa trong `pkg/providers/types.go`:

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
    // Triển khai hoàn thành chat với streaming
}
```

#### 3. Đăng ký trong factory

Thêm nhà cung cấp của bạn vào switch giao thức trong `pkg/providers/factory.go`:

```go
case "your-provider":
    return NewYourProvider(sel.apiKey, sel.apiBase, sel.proxy), nil
```

#### 4. Thêm cấu hình mặc định (Tùy chọn)

Thêm mục mặc định trong `pkg/config/defaults.go`:

```go
{
    ModelName: "your-model",
    Model:     "your-provider/model-name",
    APIKey:    "",
},
```

#### 5. Thêm hỗ trợ xác thực (Tùy chọn)

Nếu nhà cung cấp của bạn yêu cầu OAuth hoặc xác thực đặc biệt, thêm case vào `cmd/picoclaw/internal/auth/helpers.go`:

```go
case "your-provider":
    authLoginYourProvider()
```

#### 6. Cấu hình qua `config.json`

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

## Kiểm thử triển khai của bạn

### Lệnh CLI

```bash
# Xác thực với nhà cung cấp
picoclaw auth login --provider your-provider

# Liệt kê mô hình (cho Antigravity)
picoclaw auth models

# Khởi động gateway
picoclaw gateway

# Chạy agent với mô hình cụ thể
picoclaw agent -m "Hello" --model your-model
```

### Biến môi trường cho kiểm thử

```bash
# Ghi đè mô hình mặc định
export PICOCLAW_AGENTS_DEFAULTS_MODEL=your-model

# Ghi đè cài đặt nhà cung cấp
export PICOCLAW_MODEL_LIST='[{"model_name":"your-model","model":"your-provider/model-name","api_key":"..."}]'
```

---

## Tài liệu tham khảo

- **File nguồn:**
  - `pkg/providers/antigravity_provider.go` - Triển khai nhà cung cấp Antigravity
  - `pkg/auth/oauth.go` - Triển khai luồng OAuth
  - `pkg/auth/store.go` - Lưu trữ thông tin xác thực (`~/.picoclaw/auth.json`)
  - `pkg/providers/factory.go` - Factory nhà cung cấp và định tuyến giao thức
  - `pkg/providers/types.go` - Định nghĩa interface nhà cung cấp
  - `cmd/picoclaw/internal/auth/helpers.go` - Lệnh CLI xác thực

- **Tài liệu:**
  - `docs/ANTIGRAVITY_USAGE.md` - Hướng dẫn sử dụng Antigravity
  - `docs/migration/model-list-migration.md` - Hướng dẫn di chuyển

---

## Lưu ý

1. **Dự án Google Cloud:** Antigravity yêu cầu Gemini for Google Cloud được bật trên dự án Google Cloud của bạn
2. **Hạn mức:** Sử dụng hạn mức dự án Google Cloud (không tính phí riêng)
3. **Truy cập mô hình:** Các mô hình khả dụng phụ thuộc vào cấu hình dự án Google Cloud của bạn
4. **Khối suy nghĩ:** Mô hình Claude qua Antigravity yêu cầu xử lý đặc biệt khối suy nghĩ có chữ ký
5. **Làm sạch schema:** Schema công cụ phải được làm sạch để loại bỏ các từ khóa JSON Schema không được hỗ trợ

---

## Xử lý lỗi thường gặp

### 1. Giới hạn tốc độ (HTTP 429)

Antigravity trả về lỗi 429 khi hạn mức dự án/mô hình đã cạn kiệt. Phản hồi lỗi thường chứa `quotaResetDelay` trong trường `details`.

**Ví dụ lỗi 429:**
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

### 2. Phản hồi trống (Mô hình bị hạn chế)

Một số mô hình có thể xuất hiện trong danh sách mô hình khả dụng nhưng trả về phản hồi trống (200 OK nhưng luồng SSE trống). Điều này thường xảy ra với các mô hình xem trước hoặc bị hạn chế mà dự án hiện tại không có quyền sử dụng.

**Cách xử lý:** Coi phản hồi trống là lỗi, thông báo cho người dùng rằng mô hình có thể bị hạn chế hoặc không hợp lệ cho dự án của họ.

---

## Khắc phục sự cố

### "Token expired" (Token đã hết hạn)
- Làm mới token OAuth: `picoclaw auth login --provider antigravity`

### "Gemini for Google Cloud is not enabled" (Gemini for Google Cloud chưa được bật)
- Bật API trong Google Cloud Console của bạn

### "Project not found" (Không tìm thấy dự án)
- Đảm bảo dự án Google Cloud của bạn đã bật các API cần thiết
- Kiểm tra xem ID dự án có được lấy chính xác trong quá trình xác thực không

### Mô hình không xuất hiện trong danh sách
- Xác minh xác thực OAuth đã hoàn tất thành công
- Kiểm tra lưu trữ hồ sơ xác thực: `~/.picoclaw/auth.json`
- Chạy lại `picoclaw auth login --provider antigravity`
