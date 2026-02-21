# Antigravity Provider Implementation

This document provides technical details on the Antigravity (Google Cloud Code Assist) provider implementation in PicoClaw. It covers OAuth authentication, API integration, and how to extend the provider.

## Overview

**Antigravity** (Google Cloud Code Assist) is a Google-backed AI model provider that offers access to models like Claude Opus 4.6 and Gemini through Google's Cloud infrastructure.

---

## Authentication Flow

### OAuth 2.0 with PKCE

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

### PKCE Parameters

```go
func generatePkce() (verifier string, challenge string) {
    verifier = randomHex(32)
    challenge = sha256Base64URL(verifier)
    return
}
```

### Authorization URL

```go
const (
    authURL     = "https://accounts.google.com/o/oauth2/v2/auth"
    redirectURI = "http://localhost:51121/oauth-callback"
)

func buildAuthURL(challenge, state string) string {
    params := url.Values{}
    params.Set("client_id", clientID)
    params.Set("response_type", "code")
    params.Set("redirect_uri", redirectURI)
    params.Set("scope", strings.Join(scopes, " "))
    params.Set("code_challenge", challenge)
    params.Set("code_challenge_method", "S256")
    params.Set("state", state)
    params.Set("access_type", "offline")
    params.Set("prompt", "consent")
    return authURL + "?" + params.Encode()
}
```

### Required OAuth Scopes

```go
var scopes = []string{
    "https://www.googleapis.com/auth/cloud-platform",
    "https://www.googleapis.com/auth/userinfo.email",
    "https://www.googleapis.com/auth/userinfo.profile",
    "https://www.googleapis.com/auth/cclog",
    "https://www.googleapis.com/auth/experimentsandconfigs",
}
```

### Token Exchange

```go
const tokenURL = "https://oauth2.googleapis.com/token"

func exchangeCode(code, verifier string) (*TokenResponse, error) {
    data := url.Values{
        "client_id":     {clientID},
        "client_secret": {clientSecret},
        "code":          {code},
        "grant_type":    {"authorization_code"},
        "redirect_uri":  {redirectURI},
        "code_verifier": {verifier},
    }

    resp, err := http.PostForm(tokenURL, data)
    // Parse response...
}
```

---

## Token Management

### Credential Structure

Credentials are stored in `~/.picoclaw/auth.json`:

```go
type OAuthCredential struct {
    Type       string    `json:"type"`              // "oauth"
    Provider   string    `json:"provider"`          // "google-antigravity"
    Access     string    `json:"access_token"`      // Access token
    Refresh    string    `json:"refresh_token"`     // Refresh token
    Expires    time.Time `json:"expires_at"`        // Expiration timestamp
    Email      string    `json:"email,omitempty"`   // User email
    ProjectID  string    `json:"project_id,omitempty"` // Google Cloud project ID
}
```

### Automatic Token Refresh

Access tokens expire and are automatically refreshed using the refresh token:

```go
func (c *OAuthCredential) Refresh() error {
    data := url.Values{
        "client_id":     {clientID},
        "client_secret": {clientSecret},
        "refresh_token": {c.Refresh},
        "grant_type":    {"refresh_token"},
    }

    resp, err := http.PostForm(tokenURL, data)
    // Update access token and expiration...
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

### Required Headers

```go
var requiredHeaders = map[string]string{
    "Authorization":    "Bearer " + accessToken,
    "Content-Type":     "application/json",
    "User-Agent":       "antigravity",
    "X-Goog-Api-Client": "google-cloud-sdk vscode_cloudshelleditor/0.1",
}
```

### Chat Request Format

The `v1internal:streamGenerateContent` endpoint expects an envelope:

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

### SSE Response Format

Each SSE message is wrapped in a `response` field:

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

## Model Management

### Fetch Available Models

```go
func fetchAvailableModels(accessToken, projectID string) ([]Model, error) {
    req := map[string]string{"project": projectID}

    resp, err := post(
        "https://cloudcode-pa.googleapis.com/v1internal:fetchAvailableModels",
        req,
        headers(accessToken),
    )

    // Parse response with quota information
    // ...
}
```

### Model Response Structure

```go
type Model struct {
    ID          string
    DisplayName string
    QuotaInfo   struct {
        RemainingFraction float64
        ResetTime         time.Time
        IsExhausted       bool
    }
}
```

---

## Usage Tracking

### Fetch Usage Data

```go
func fetchUsage(accessToken string) (*UsageSnapshot, error) {
    // 1. Load credits and plan info
    loadResp, _ := post(
        "https://cloudcode-pa.googleapis.com/v1internal:loadCodeAssist",
        map[string]interface{}{
            "metadata": map[string]string{
                "ideType":   "ANTIGRAVITY",
                "platform":  "PLATFORM_UNSPECIFIED",
                "pluginType": "GEMINI",
            },
        },
        headers(accessToken),
    )

    // Extract: availablePromptCredits, planInfo, currentTier

    // 2. Fetch model quotas
    modelsResp, _ := post(
        "https://cloudcode-pa.googleapis.com/v1internal:fetchAvailableModels",
        map[string]string{"project": projectID},
        headers(accessToken),
    )

    // Build usage windows
    // ...
}
```

### Usage Snapshot Structure

```go
type UsageSnapshot struct {
    Provider    string        `json:"provider"`     // "google-antigravity"
    DisplayName string        `json:"displayName"`  // "Google Antigravity"
    Windows     []UsageWindow `json:"windows"`      // Quota windows
    Plan        string        `json:"plan"`         // Current plan
}

type UsageWindow struct {
    Label       string `json:"label"`        // "Credits" or model ID
    UsedPercent int    `json:"usedPercent"`  // 0-100
    ResetAt     int64  `json:"resetAt"`      // Reset timestamp
}
```

---

## Schema Sanitization

Antigravity uses Gemini-compatible models, so tool schemas must be sanitized:

```go
var unsupportedKeywords = map[string]bool{
    "patternProperties": true,
    "additionalProperties": true,
    "$schema": true,
    "$id": true,
    "$ref": true,
    "$defs": true,
    "definitions": true,
    "examples": true,
    "minLength": true,
    "maxLength": true,
    "minimum": true,
    "maximum": true,
    "multipleOf": true,
    "pattern": true,
    "format": true,
    "minItems": true,
    "maxItems": true,
    "uniqueItems": true,
    "minProperties": true,
    "maxProperties": true,
}

func cleanSchemaForGemini(schema map[string]interface{}) map[string]interface{} {
    // Remove unsupported keywords
    // Ensure top-level has type: "object"
    // Flatten anyOf/oneOf unions
    // ...
}
```

---

## Thinking Block Handling

For Antigravity Claude models, thinking blocks require special handling:

```go
var signatureRegex = regexp.MustCompile(`^[A-Za-z0-9+/]+={0,2}$`)

func sanitizeThinkingBlocks(messages []Message) []Message {
    // Validate thinking signatures
    // Normalize signature fields
    // Discard unsigned thinking blocks
    // ...
}
```

---

## Error Handling

### Rate Limiting (HTTP 429)

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

### Empty Responses

Some models return 200 OK with an empty SSE stream. This usually indicates the model is restricted or the project lacks permission.

---

## Source Files

| File | Purpose |
|------|---------|
| `pkg/providers/antigravity_provider.go` | Provider implementation |
| `pkg/auth/oauth.go` | OAuth flow implementation |
| `pkg/auth/store.go` | Credential storage |
| `pkg/providers/factory.go` | Provider factory |
| `pkg/providers/types.go` | Interface definitions |
| `cmd/picoclaw/cmd_auth.go` | Auth CLI commands |

---

## Configuration

### config.json

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

---

## Notes

1. **Google Cloud Project**: Requires Gemini for Google Cloud enabled
2. **Quotas**: Uses Google Cloud project quotas (not separate billing)
3. **Model Access**: Available models depend on project configuration
4. **Thinking Blocks**: Claude models require special handling
5. **Schema Sanitization**: Tool schemas must be cleaned for Gemini compatibility
