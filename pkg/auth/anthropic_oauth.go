package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	anthropicClientID = "9d1c250a-e61b-44d9-88ed-5944d1962f5e"

	// OAuth endpoints
	anthropicConsoleAuthorizeURL  = "https://console.anthropic.com/oauth/authorize"
	anthropicClaudeAIAuthorizeURL = "https://claude.ai/oauth/authorize"
	anthropicTokenURL             = "https://console.anthropic.com/v1/oauth/token"
	anthropicCallbackURL          = "https://console.anthropic.com/oauth/code/callback"

	// API endpoints
	anthropicCreateAPIKeyURL = "https://api.anthropic.com/api/oauth/claude_cli/create_api_key"
	anthropicRolesURL        = "https://api.anthropic.com/api/oauth/claude_cli/roles"
	anthropicProfileURL      = "https://api.anthropic.com/api/oauth/profile"

	// Scopes
	anthropicScopesAll    = "org:create_api_key user:profile user:inference"
	anthropicScopesMaxPro = "user:profile user:inference"

	// Beta header required for OAuth access
	AnthropicOAuthBeta = "oauth-2025-04-20"
)

// AnthropicOAuthMode determines which authorization endpoint to use.
type AnthropicOAuthMode string

const (
	// AnthropicOAuthMax uses claude.ai for Claude Max/Pro subscription users.
	// Tokens are used directly with Bearer auth.
	AnthropicOAuthMax AnthropicOAuthMode = "max"

	// AnthropicOAuthConsole uses console.anthropic.com for API key creation.
	// After OAuth, an API key is created and used for subsequent requests.
	AnthropicOAuthConsole AnthropicOAuthMode = "console"
)

// AnthropicMaxOAuthConfig returns the OAuth config for Claude Max/Pro users.
func AnthropicMaxOAuthConfig() OAuthProviderConfig {
	return OAuthProviderConfig{
		Issuer:   anthropicClaudeAIAuthorizeURL,
		ClientID: anthropicClientID,
		TokenURL: anthropicTokenURL,
		Scopes:   anthropicScopesAll,
		Port:     1456,
	}
}

// AnthropicConsoleOAuthConfig returns the OAuth config for API key creation via console.
func AnthropicConsoleOAuthConfig() OAuthProviderConfig {
	return OAuthProviderConfig{
		Issuer:   anthropicConsoleAuthorizeURL,
		ClientID: anthropicClientID,
		TokenURL: anthropicTokenURL,
		Scopes:   anthropicScopesAll,
		Port:     1456,
	}
}

// buildAnthropicAuthorizeURL constructs the Anthropic OAuth authorization URL.
func buildAnthropicAuthorizeURL(mode AnthropicOAuthMode, pkce PKCECodes) string {
	var baseURL string
	switch mode {
	case AnthropicOAuthMax:
		baseURL = anthropicClaudeAIAuthorizeURL
	default:
		baseURL = anthropicConsoleAuthorizeURL
	}

	params := url.Values{
		"code":                  {"true"},
		"client_id":             {anthropicClientID},
		"response_type":         {"code"},
		"redirect_uri":          {anthropicCallbackURL},
		"scope":                 {anthropicScopesAll},
		"code_challenge":        {pkce.CodeChallenge},
		"code_challenge_method": {"S256"},
		"state":                 {pkce.CodeVerifier},
	}

	return baseURL + "?" + params.Encode()
}

// anthropicTokenResponse represents the response from Anthropic's token endpoint.
type anthropicTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
}

// ExchangeAnthropicCode exchanges an authorization code for tokens.
// The code parameter may contain a "#state" suffix (e.g. "authcode#statevalue").
// Both the code and state must be sent in the token exchange request.
func ExchangeAnthropicCode(code, verifier string) (*anthropicTokenResponse, error) {
	// The authorization code comes as "code#state" - split into both parts
	parts := strings.SplitN(code, "#", 2)
	authCode := parts[0]
	state := ""
	if len(parts) > 1 {
		state = parts[1]
	}

	payload := map[string]string{
		"grant_type":    "authorization_code",
		"client_id":     anthropicClientID,
		"redirect_uri":  anthropicCallbackURL,
		"code":          authCode,
		"code_verifier": verifier,
	}
	if state != "" {
		payload["state"] = state
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshaling token request: %w", err)
	}

	resp, err := http.Post(anthropicTokenURL, "application/json", strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("token exchange request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var tokenResp anthropicTokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return nil, fmt.Errorf("parsing token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("no access token in response")
	}

	return &tokenResp, nil
}

// RefreshAnthropicToken refreshes an Anthropic OAuth access token.
func RefreshAnthropicToken(refreshToken string) (*anthropicTokenResponse, error) {
	payload := map[string]string{
		"grant_type":    "refresh_token",
		"client_id":     anthropicClientID,
		"refresh_token": refreshToken,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshaling refresh request: %w", err)
	}

	resp, err := http.Post(anthropicTokenURL, "application/json", strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("token refresh request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var tokenResp anthropicTokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return nil, fmt.Errorf("parsing refresh response: %w", err)
	}

	return &tokenResp, nil
}

// RefreshAnthropicCredential refreshes an expired Anthropic OAuth credential,
// preserving all metadata (email, plan, org, scopes), and saves it to the store.
func RefreshAnthropicCredential(cred *AuthCredential) error {
	if cred.RefreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}
	tokenResp, err := RefreshAnthropicToken(cred.RefreshToken)
	if err != nil {
		return err
	}
	cred.AccessToken = tokenResp.AccessToken
	if tokenResp.RefreshToken != "" {
		cred.RefreshToken = tokenResp.RefreshToken
	}
	if tokenResp.ExpiresIn > 0 {
		cred.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}
	if tokenResp.Scope != "" {
		cred.Scopes = tokenResp.Scope
	}
	return SetCredential("anthropic", cred)
}

// CreateAnthropicAPIKey creates an API key using an OAuth access token.
func CreateAnthropicAPIKey(accessToken string) (string, error) {
	req, err := http.NewRequest("POST", anthropicCreateAPIKeyURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating API key request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API key creation request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API key creation failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		RawKey string `json:"raw_key"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parsing API key response: %w", err)
	}

	if result.RawKey == "" {
		return "", fmt.Errorf("no API key in response")
	}

	return result.RawKey, nil
}

// FetchAnthropicProfile fetches the user's Anthropic profile using an OAuth access token.
func FetchAnthropicProfile(accessToken string) (*AnthropicProfile, error) {
	req, err := http.NewRequest("GET", anthropicProfileURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating profile request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("profile request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("profile request failed (status %d): %s", resp.StatusCode, string(body))
	}

	var profile AnthropicProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, fmt.Errorf("parsing profile response: %w", err)
	}

	return &profile, nil
}

// AnthropicProfile represents the user's Anthropic account profile.
type AnthropicProfile struct {
	Account struct {
		UUID        string `json:"uuid"`
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
	} `json:"account"`
	Organization struct {
		UUID             string `json:"uuid"`
		OrganizationType string `json:"organization_type"`
		RateLimitTier    string `json:"rate_limit_tier"`
	} `json:"organization"`
}

// SubscriptionType returns a normalized subscription type string.
func (p *AnthropicProfile) SubscriptionType() string {
	switch p.Organization.OrganizationType {
	case "claude_max":
		return "max"
	case "claude_pro":
		return "pro"
	case "claude_enterprise":
		return "enterprise"
	case "claude_team":
		return "team"
	default:
		return ""
	}
}

// LoginAnthropicOAuth performs the Anthropic OAuth flow.
func LoginAnthropicOAuth(mode AnthropicOAuthMode) (*AuthCredential, error) {
	pkce, err := GeneratePKCE()
	if err != nil {
		return nil, fmt.Errorf("generating PKCE: %w", err)
	}

	authURL := buildAnthropicAuthorizeURL(mode, pkce)

	fmt.Printf("\nOpen this URL in your browser to authenticate:\n\n  %s\n\n", authURL)

	if err := openBrowser(authURL); err != nil {
		fmt.Println("Could not open browser automatically.")
	}

	fmt.Println("After authorizing, you'll be redirected to a page with a code.")
	fmt.Println("Paste the full URL or just the authorization code here:")
	fmt.Print("\n> ")

	var input string
	fmt.Scanln(&input)
	input = strings.TrimSpace(input)

	if input == "" {
		return nil, fmt.Errorf("no authorization code provided")
	}

	// Extract code from URL if it's a full URL
	code := input
	if strings.Contains(input, "?") || strings.Contains(input, "#") {
		u, err := url.Parse(input)
		if err == nil {
			if c := u.Query().Get("code"); c != "" {
				code = c
			}
		}
	}

	tokenResp, err := ExchangeAnthropicCode(code, pkce.CodeVerifier)
	if err != nil {
		return nil, fmt.Errorf("exchanging code: %w", err)
	}

	var expiresAt time.Time
	if tokenResp.ExpiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	cred := &AuthCredential{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    expiresAt,
		Provider:     "anthropic",
		AuthMethod:   "oauth",
		Scopes:       tokenResp.Scope,
	}

	profile, err := FetchAnthropicProfile(tokenResp.AccessToken)
	if err != nil {
		fmt.Printf("Warning: could not fetch profile: %v\n", err)
	} else {
		cred.Email = profile.Account.Email
		cred.AccountID = profile.Account.UUID
		cred.OrgID = profile.Organization.UUID
		cred.SubscriptionType = profile.SubscriptionType()
		if profile.Account.Email != "" {
			fmt.Printf("Email: %s\n", profile.Account.Email)
		}
		if profile.SubscriptionType() != "" {
			fmt.Printf("Plan: %s\n", profile.SubscriptionType())
		}
	}

	if mode == AnthropicOAuthConsole {
		fmt.Println("Creating API key...")
		apiKey, err := CreateAnthropicAPIKey(tokenResp.AccessToken)
		if err != nil {
			return nil, fmt.Errorf("creating API key: %w", err)
		}
		cred.APIKey = apiKey
		cred.SubscriptionType = "api"
		fmt.Println("API key created successfully!")
	} else {
		if cred.SubscriptionType == "" {
			cred.SubscriptionType = "max"
		}
	}

	return cred, nil
}

// GetAnthropicAccessToken returns a valid access token for Anthropic OAuth credentials.
func GetAnthropicAccessToken(cred *AuthCredential) (string, error) {
	if cred == nil {
		return "", fmt.Errorf("no credentials")
	}

	if cred.APIKey != "" {
		return cred.APIKey, nil
	}

	if cred.AuthMethod == "token" {
		return cred.AccessToken, nil
	}

	if !cred.NeedsRefresh() {
		return cred.AccessToken, nil
	}

	if cred.RefreshToken == "" {
		return cred.AccessToken, nil
	}

	tokenResp, err := RefreshAnthropicToken(cred.RefreshToken)
	if err != nil {
		return cred.AccessToken, nil
	}

	cred.AccessToken = tokenResp.AccessToken
	if tokenResp.RefreshToken != "" {
		cred.RefreshToken = tokenResp.RefreshToken
	}
	if tokenResp.ExpiresIn > 0 {
		cred.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	if err := SetCredential("anthropic", cred); err != nil {
		fmt.Printf("Warning: could not save refreshed token: %v\n", err)
	}

	return cred.AccessToken, nil
}

// IsAnthropicMaxOAuth returns true if the credential is for the Claude Max/Pro OAuth flow.
func IsAnthropicMaxOAuth(cred *AuthCredential) bool {
	if cred == nil {
		return false
	}
	return cred.Provider == "anthropic" &&
		cred.AuthMethod == "oauth" &&
		cred.APIKey == "" &&
		(cred.SubscriptionType == "max" || cred.SubscriptionType == "pro")
}
