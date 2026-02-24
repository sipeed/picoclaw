# Antigravity Authentication & Integration Guide

## Overview

**Antigravity** (Google Cloud Code Assist) is a Google-backed AI model provider that offers access to models like Claude Opus 4.6 and Gemini through Google's Cloud infrastructure. This document provides a complete guide on how authentication works, how to fetch models, and how to implement a new provider in PicoClaw.

---

## Table of Contents

1. [Authentication Flow](#authentication-flow)
2. [OAuth Implementation Details](#oauth-implementation-details)
3. [Token Management](#token-management)
4. [Models List Fetching](#models-list-fetching)
5. [Usage Tracking](#usage-tracking)
6. [Provider Plugin Structure](#provider-plugin-structure)
7. [Integration Requirements](#integration-requirements)
8. [API Endpoints](#api-endpoints)
9. [Configuration](#configuration)
10. [Creating a New Provider in PicoClaw](#creating-a-new-provider-in-picoclaw)

---

## Authentication Flow

### 1. OAuth 2.0 with PKCE

Antigravity uses **OAuth 2.0 with PKCE (Proof Key for Code Exchange)** for secure authentication:

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

### 2. Detailed Steps

#### Step 1: Generate PKCE Parameters
```typescript
function generatePkce(): { verifier: string; challenge: string } {
  const verifier = randomBytes(32).toString("hex");
  const challenge = createHash("sha256").update(verifier).digest("base64url");
  return { verifier, challenge };
}
```

#### Step 2: Build Authorization URL
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

**Required Scopes:**
```typescript
const SCOPES = [
  "https://www.googleapis.com/auth/cloud-platform",
  "https://www.googleapis.com/auth/userinfo.email",
  "https://www.googleapis.com/auth/userinfo.profile",
  "https://www.googleapis.com/auth/cclog",
  "https://www.googleapis.com/auth/experimentsandconfigs",
];
```

#### Step 3: Handle OAuth Callback

**Automatic Mode (Local Development):**
- Start a local HTTP server on port 51121
- Wait for the redirect from Google
- Extract the authorization code from the query parameters

**Manual Mode (Remote/Headless):**
- Display the authorization URL to the user
- User completes authentication in their browser
- User pastes the full redirect URL back into the terminal
- Parse the code from the pasted URL

#### Step 4: Exchange Code for Tokens
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

#### Step 5: Fetch Additional User Data

**User Email:**
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

**Project ID (Required for API calls):**
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
  return data.cloudaicompanionProject || "rising-fact-p41fc"; // Default fallback
}
```

---

## OAuth Implementation Details

### Client Credentials

**Important:** These are base64-encoded in the source code for sync with pi-ai:

```typescript
const decode = (s: string) => Buffer.from(s, "base64").toString();

const CLIENT_ID = decode(
  "MTA3MTAwNjA2MDU5MS10bWhzc2luMmgyMWxjcmUyMzV2dG9sb2poNGc0MDNlcC5hcHBzLmdvb2dsZXVzZXJjb250ZW50LmNvbQ=="
);
const CLIENT_SECRET = decode("R09DU1BYLUs1OEZXUjQ4NkxkTEoxbUxCOHNYQzR6NnFEQWY=");
```

### OAuth Flow Modes

1. **Automatic Flow** (Local machines with browser):
   - Opens browser automatically
   - Local callback server captures redirect
   - No user interaction required after initial auth

2. **Manual Flow** (Remote/headless/WSL2):
   - URL displayed for manual copy-paste
   - User completes auth in external browser
   - User pastes full redirect URL back

```typescript
function shouldUseManualOAuthFlow(isRemote: boolean): boolean {
  return isRemote || isWSL2Sync();
}
```

---

## Token Management

### Auth Profile Structure

```typescript
type OAuthCredential = {
  type: "oauth";
  provider: "google-antigravity";
  access: string;           // Access token
  refresh: string;          // Refresh token
  expires: number;          // Expiration timestamp (ms since epoch)
  email?: string;           // User email
  projectId?: string;       // Google Cloud project ID
};
```

### Token Refresh

The credential includes a refresh token that can be used to obtain new access tokens when the current one expires. The expiration is set with a 5-minute buffer to prevent race conditions.

---

## Models List Fetching

### Fetch Available Models

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
  
  // Returns models with quota information
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

### Response Format

```typescript
type FetchAvailableModelsResponse = {
  models?: Record<string, {
    displayName?: string;
    quotaInfo?: {
      remainingFraction?: number | string;
      resetTime?: string;      // ISO 8601 timestamp
      isExhausted?: boolean;
    };
  }>;
};
```

---

## Usage Tracking

### Fetch Usage Data

```typescript
export async function fetchAntigravityUsage(
  token: string,
  timeoutMs: number
): Promise<ProviderUsageSnapshot> {
  // 1. Fetch credits and plan info
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

  // Extract credits info
  const { availablePromptCredits, planInfo, currentTier } = data;
  
  // 2. Fetch model quotas
  const modelsRes = await fetch(
    `${BASE_URL}/v1internal:fetchAvailableModels`,
    {
      method: "POST",
      headers: { Authorization: `Bearer ${token}` },
      body: JSON.stringify({ project: projectId }),
    }
  );

  // Build usage windows
  return {
    provider: "google-antigravity",
    displayName: "Google Antigravity",
    windows: [
      { label: "Credits", usedPercent: calculateUsedPercent(available, monthly) },
      // Individual model quotas...
    ],
    plan: currentTier?.name || planType,
  };
}
```

### Usage Response Structure

```typescript
type ProviderUsageSnapshot = {
  provider: "google-antigravity";
  displayName: string;
  windows: UsageWindow[];
  plan?: string;
  error?: string;
};

type UsageWindow = {
  label: string;           // "Credits" or model ID
  usedPercent: number;     // 0-100
  resetAt?: number;        // Timestamp when quota resets
};
```

---

## Provider Plugin Structure

### Plugin Definition

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
            // OAuth implementation here
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
  prompter: WizardPrompter;      // UI prompts/notifications
  runtime: RuntimeEnv;           // Logging, etc.
  isRemote: boolean;             // Whether running remotely
  openUrl: (url: string) => Promise<void>;  // Browser opener
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

## Integration Requirements

### 1. Required Environment/Dependencies

- Go ≥ 1.21
- PicoClaw codebase (`pkg/providers/` and `pkg/auth/`)
- `crypto` and `net/http` standard library packages

### 2. Required Headers for API Calls

```typescript
const REQUIRED_HEADERS = {
  "Authorization": `Bearer ${accessToken}`,
  "Content-Type": "application/json",
  "User-Agent": "antigravity",  // or "google-api-nodejs-client/9.15.1"
  "X-Goog-Api-Client": "google-cloud-sdk vscode_cloudshelleditor/0.1",
};

// For loadCodeAssist calls, also include:
const CLIENT_METADATA = {
  ideType: "ANTIGRAVITY",  // or "IDE_UNSPECIFIED"
  platform: "PLATFORM_UNSPECIFIED",
  pluginType: "GEMINI",
};
```

### 3. Model Schema Sanitization

Antigravity uses Gemini-compatible models, so tool schemas must be sanitized:

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

// Clean schema before sending
function cleanToolSchemaForGemini(schema: Record<string, unknown>): unknown {
  // Remove unsupported keywords
  // Ensure top-level has type: "object"
  // Flatten anyOf/oneOf unions
}
```

### 4. Thinking Block Handling (Claude Models)

For Antigravity Claude models, thinking blocks require special handling:

```typescript
const ANTIGRAVITY_SIGNATURE_RE = /^[A-Za-z0-9+/]+={0,2}$/;

export function sanitizeAntigravityThinkingBlocks(
  messages: AgentMessage[]
): AgentMessage[] {
  // Validate thinking signatures
  // Normalize signature fields
  // Discard unsigned thinking blocks
}
```

---

## API Endpoints

### Authentication Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `https://accounts.google.com/o/oauth2/v2/auth` | GET | OAuth authorization |
| `https://oauth2.googleapis.com/token` | POST | Token exchange |
| `https://www.googleapis.com/oauth2/v1/userinfo` | GET | User info (email) |

### Cloud Code Assist Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `https://cloudcode-pa.googleapis.com/v1internal:loadCodeAssist` | POST | Load project info, credits, plan |
| `https://cloudcode-pa.googleapis.com/v1internal:fetchAvailableModels` | POST | List available models with quotas |
| `https://cloudcode-pa.googleapis.com/v1internal:streamGenerateContent?alt=sse` | POST | Chat streaming endpoint |

**API Request Format (Chat):**
The `v1internal:streamGenerateContent` endpoint expects an envelope wrapping the standard Gemini request:

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

**API Response Format (SSE):**
Each SSE message (`data: {...}`) is wrapped in a `response` field:

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

## Configuration

### config.json Configuration

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
      "model": "gemini-flash"
    }
  }
}
```

### Auth Profile Storage

Auth profiles are stored in `~/.picoclaw/auth.json`:

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

## Creating a New Provider in PicoClaw

PicoClaw providers are implemented as Go packages under `pkg/providers/`. To add a new provider:

### Step-by-Step Implementation

#### 1. Create Provider File

Create a new Go file in `pkg/providers/`:

```
pkg/providers/
└── your_provider.go
```

#### 2. Implement the Provider Interface

Your provider must implement the `Provider` interface defined in `pkg/providers/types.go`:

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
    // Implement chat completion with streaming
}
```

#### 3. Register in the Factory

Add your provider to the protocol switch in `pkg/providers/factory.go`:

```go
case "your-provider":
    return NewYourProvider(sel.apiKey, sel.apiBase, sel.proxy), nil
```

#### 4. Add Default Config (Optional)

Add a default entry in `pkg/config/defaults.go`:

```go
{
    ModelName: "your-model",
    Model:     "your-provider/model-name",
    APIKey:    "",
},
```

#### 5. Add Auth Support (Optional)

If your provider requires OAuth or special authentication, add a case to `cmd/picoclaw/cmd_auth.go`:

```go
case "your-provider":
    authLoginYourProvider()
```

#### 6. Configure via `config.json`

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

## Testing Your Implementation

### CLI Commands

```bash
# Authenticate with a provider
picoclaw auth login --provider your-provider

# List models (for Antigravity)
picoclaw auth models

# Start the gateway
picoclaw gateway

# Run an agent with a specific model
picoclaw agent -m "Hello" --model your-model
```

### Environment Variables for Testing

```bash
# Override default model
export PICOCLAW_AGENTS_DEFAULTS_MODEL=your-model

# Override provider settings
export PICOCLAW_MODEL_LIST='[{"model_name":"your-model","model":"your-provider/model-name","api_key":"..."}]'
```

---

## References

- **Source Files:**
  - `pkg/providers/antigravity_provider.go` - Antigravity provider implementation
  - `pkg/auth/oauth.go` - OAuth flow implementation
  - `pkg/auth/store.go` - Auth credential storage (`~/.picoclaw/auth.json`)
  - `pkg/providers/factory.go` - Provider factory and protocol routing
  - `pkg/providers/types.go` - Provider interface definitions
  - `cmd/picoclaw/cmd_auth.go` - Auth CLI commands

- **Documentation:**
  - `docs/ANTIGRAVITY_USAGE.md` - Antigravity usage guide
  - `docs/migration/model-list-migration.md` - Migration guide

---

## Notes

1. **Google Cloud Project:** Antigravity requires Gemini for Google Cloud to be enabled on your Google Cloud project
2. **Quotas:** Uses Google Cloud project quotas (not separate billing)
3. **Model Access:** Available models depend on your Google Cloud project configuration
4. **Thinking Blocks:** Claude models via Antigravity require special handling of thinking blocks with signatures
5. **Schema Sanitization:** Tool schemas must be sanitized to remove unsupported JSON Schema keywords

---

---

## Common Error Handling

### 1. Rate Limiting (HTTP 429)

Antigravity returns a 429 error when project/model quotas are exhausted. The error response often contains a `quotaResetDelay` in the `details` field.

**Example 429 Error:**
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

### 2. Empty Responses (Restricted Models)

Some models might show up in the available models list but return an empty response (200 OK but empty SSE stream). This usually happens for preview or restricted models that the current project doesn't have permission to use.

**Treatment:** Treat empty responses as errors informing the user that the model might be restricted or invalid for their project.

---

## Troubleshooting

### "Token expired"
- Refresh OAuth tokens: `picoclaw auth login --provider antigravity`

### "Gemini for Google Cloud is not enabled"
- Enable the API in your Google Cloud Console

### "Project not found"
- Ensure your Google Cloud project has the necessary APIs enabled
- Check that the project ID is correctly fetched during authentication

### Models not appearing in list
- Verify OAuth authentication completed successfully
- Check auth profile storage: `~/.picoclaw/auth.json`
- Re-run `picoclaw auth login --provider antigravity`
