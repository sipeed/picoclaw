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
  
  register(api: OpenClawPluginApi) {
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
  config: OpenClawConfig;
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
  configPatch?: Partial<OpenClawConfig>;
  defaultModel?: string;
  notes?: string[];
};
```

---

## Integration Requirements

### 1. Required Environment/Dependencies

- Node.js ≥ 22
- OpenClaw plugin-sdk
- crypto module (built-in)
- http module (built-in)

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

### openclaw.json Configuration

```json5
{
  agents: {
    defaults: {
      model: {
        primary: "google-antigravity/claude-opus-4-6-thinking",
      },
    },
  },
}
```

### Auth Profile Storage

Auth profiles are stored in `~/.openclaw/agent/auth-profiles.json`:

```json
{
  "version": 1,
  "profiles": {
    "google-antigravity:user@example.com": {
      "type": "oauth",
      "provider": "google-antigravity",
      "access": "ya29...",
      "refresh": "1//...",
      "expires": 1704067200000,
      "email": "user@example.com",
      "projectId": "my-project-id"
    }
  }
}
```

---

## Creating a New Provider in PicoClaw

### Step-by-Step Implementation

#### 1. Create Plugin Structure

```
extensions/
└── your-provider-auth/
    ├── openclaw.plugin.json
    ├── package.json
    ├── README.md
    └── index.ts
```

#### 2. Define Plugin Manifest

**openclaw.plugin.json:**
```json
{
  "id": "your-provider-auth",
  "providers": ["your-provider"],
  "configSchema": {
    "type": "object",
    "additionalProperties": false,
    "properties": {}
  }
}
```

**package.json:**
```json
{
  "name": "@openclaw/your-provider-auth",
  "version": "1.0.0",
  "private": true,
  "description": "Your Provider OAuth plugin",
  "type": "module"
}
```

#### 3. Implement OAuth Flow

```typescript
import {
  buildOauthProviderAuthResult,
  emptyPluginConfigSchema,
  type OpenClawPluginApi,
  type ProviderAuthContext,
} from "openclaw/plugin-sdk";

const YOUR_CLIENT_ID = "your-client-id";
const YOUR_CLIENT_SECRET = "your-client-secret";
const AUTH_URL = "https://provider.com/oauth/authorize";
const TOKEN_URL = "https://provider.com/oauth/token";
const REDIRECT_URI = "http://localhost:PORT/oauth-callback";

async function loginYourProvider(params: {
  isRemote: boolean;
  openUrl: (url: string) => Promise<void>;
  prompt: (message: string) => Promise<string>;
  note: (message: string, title?: string) => Promise<void>;
  log: (message: string) => void;
  progress: { update: (msg: string) => void; stop: (msg?: string) => void };
}) {
  // 1. Generate PKCE
  const { verifier, challenge } = generatePkce();
  const state = randomBytes(16).toString("hex");
  
  // 2. Build auth URL
  const authUrl = buildAuthUrl({ challenge, state });
  
  // 3. Start callback server (if not remote)
  const callbackServer = !params.isRemote 
    ? await startCallbackServer({ timeoutMs: 5 * 60 * 1000 })
    : null;
  
  // 4. Open browser or show URL
  if (callbackServer) {
    await params.openUrl(authUrl);
    const callback = await callbackServer.waitForCallback();
    code = callback.searchParams.get("code");
  } else {
    await params.note(`Auth URL: ${authUrl}`, "OAuth");
    const input = await params.prompt("Paste redirect URL:");
    const parsed = parseCallbackInput(input);
    code = parsed.code;
  }
  
  // 5. Exchange code for tokens
  const tokens = await exchangeCode({ code, verifier });
  
  // 6. Fetch additional user data
  const email = await fetchUserEmail(tokens.access);
  
  return { ...tokens, email };
}
```

#### 4. Register Provider

```typescript
const yourProviderPlugin = {
  id: "your-provider-auth",
  name: "Your Provider Auth",
  description: "OAuth for Your Provider",
  configSchema: emptyPluginConfigSchema(),
  
  register(api: OpenClawPluginApi) {
    api.registerProvider({
      id: "your-provider",
      label: "Your Provider",
      docsPath: "/providers/models",
      aliases: ["yp"],
      
      auth: [
        {
          id: "oauth",
          label: "OAuth Login",
          hint: "Browser-based authentication",
          kind: "oauth",
          
          run: async (ctx: ProviderAuthContext) => {
            const spin = ctx.prompter.progress("Starting OAuth...");
            
            try {
              const result = await loginYourProvider({
                isRemote: ctx.isRemote,
                openUrl: ctx.openUrl,
                prompt: async (msg) => String(await ctx.prompter.text({ message: msg })),
                note: ctx.prompter.note,
                log: (msg) => ctx.runtime.log(msg),
                progress: spin,
              });
              
              return buildOauthProviderAuthResult({
                providerId: "your-provider",
                defaultModel: "your-provider/model-name",
                access: result.access,
                refresh: result.refresh,
                expires: result.expires,
                email: result.email,
                notes: ["Provider-specific notes"],
              });
            } catch (err) {
              spin.stop("OAuth failed");
              throw err;
            }
          },
        },
      ],
    });
  },
};

export default yourProviderPlugin;
```

#### 5. Implement Usage Tracking (Optional)

```typescript
// src/infra/provider-usage.fetch.your-provider.ts
export async function fetchYourProviderUsage(
  token: string,
  timeoutMs: number,
  fetchFn: typeof fetch
): Promise<ProviderUsageSnapshot> {
  // Fetch usage data from provider API
  const response = await fetchFn("https://api.provider.com/usage", {
    headers: { Authorization: `Bearer ${token}` },
  });
  
  const data = await response.json();
  
  return {
    provider: "your-provider",
    displayName: "Your Provider",
    windows: [
      { label: "Credits", usedPercent: data.usedPercent },
    ],
    plan: data.planName,
  };
}
```

#### 6. Register Usage Fetcher

```typescript
// src/infra/provider-usage.load.ts
case "your-provider":
  return await fetchYourProviderUsage(auth.token, timeoutMs, fetchFn);
```

#### 7. Add Provider to Type Definitions

```typescript
// src/infra/provider-usage.types.ts
export type SupportedProvider =
  | "anthropic"
  | "github-copilot"
  | "google-gemini-cli"
  | "google-antigravity"
  | "your-provider"  // Add here
  | "minimax"
  | "openai-codex";
```

#### 8. Add Auth Choice Handler

```typescript
// src/commands/auth-choice.apply.your-provider.ts
import { applyAuthChoicePluginProvider } from "./auth-choice.apply.plugin-provider.js";

export async function applyAuthChoiceYourProvider(
  params: ApplyAuthChoiceParams
): Promise<ApplyAuthChoiceResult | null> {
  return await applyAuthChoicePluginProvider(params, {
    authChoice: "your-provider",
    pluginId: "your-provider-auth",
    providerId: "your-provider",
    methodId: "oauth",
    label: "Your Provider",
  });
}
```

#### 9. Export from Main Index

```typescript
// src/commands/auth-choice.apply.ts
import { applyAuthChoiceYourProvider } from "./auth-choice.apply.your-provider.js";

// In the switch statement:
case "your-provider":
  return await applyAuthChoiceYourProvider(params);
```

### Helper Utilities

#### PKCE Generation
```typescript
function generatePkce(): { verifier: string; challenge: string } {
  const verifier = randomBytes(32).toString("hex");
  const challenge = createHash("sha256").update(verifier).digest("base64url");
  return { verifier, challenge };
}
```

#### Callback Server
```typescript
async function startCallbackServer(params: { timeoutMs: number }) {
  const port = 51121; // Your port
  
  const server = createServer((request, response) => {
    const url = new URL(request.url!, `http://localhost:${port}`);
    
    if (url.pathname === "/oauth-callback") {
      response.writeHead(200, { "Content-Type": "text/html" });
      response.end("<h1>Authentication complete</h1>");
      resolveCallback(url);
      server.close();
    }
  });
  
  await new Promise((resolve, reject) => {
    server.listen(port, "127.0.0.1", resolve);
    server.once("error", reject);
  });
  
  return {
    waitForCallback: () => callbackPromise,
    close: () => new Promise((resolve) => server.close(resolve)),
  };
}
```

---

## Testing Your Implementation

### CLI Commands

```bash
# Enable the plugin
openclaw plugins enable your-provider-auth

# Restart gateway
openclaw gateway restart

# Authenticate
openclaw models auth login --provider your-provider --set-default

# List models
openclaw models list

# Set model
openclaw models set your-provider/model-name

# Check usage
openclaw models usage
```

### Environment Variables for Testing

```bash
# Test specific providers only
export OPENCLAW_LIVE_PROVIDERS="your-provider,google-antigravity"

# Test with specific models
export OPENCLAW_LIVE_GATEWAY_MODELS="your-provider/model-name"
```

---

## References

- **Source Files:**
  - `extensions/google-antigravity-auth/index.ts` - Full OAuth implementation
  - `src/infra/provider-usage.fetch.antigravity.ts` - Usage fetching
  - `src/agents/pi-embedded-runner/google.ts` - Model sanitization
  - `src/agents/model-forward-compat.ts` - Forward compatibility
  - `src/plugin-sdk/provider-auth-result.ts` - Auth result builder
  - `src/plugins/types.ts` - Plugin type definitions

- **Documentation:**
  - `docs/concepts/model-providers.md` - Provider overview
  - `docs/concepts/usage-tracking.md` - Usage tracking

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
- Refresh OAuth tokens: `openclaw models auth login --provider google-antigravity`

### "Gemini for Google Cloud is not enabled"
- Enable the API in your Google Cloud Console

### "Project not found"
- Ensure your Google Cloud project has the necessary APIs enabled
- Check that the project ID is correctly fetched during authentication

### Models not appearing in list
- Verify OAuth authentication completed successfully
- Check auth profile storage: `~/.openclaw/agent/auth-profiles.json`
- Ensure the plugin is enabled: `openclaw plugins list`
