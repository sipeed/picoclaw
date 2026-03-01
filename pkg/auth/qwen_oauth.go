package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Qwen Portal OAuth constants (extracted from openclaw/openclaw extensions/qwen-portal-auth).
// Reference: https://github.com/openclaw/openclaw/tree/main/extensions/qwen-portal-auth
const (
	qwenOAuthBaseURL      = "https://chat.qwen.ai"
	qwenClientID          = "f0304373b74a44d2b584a3fb70ca9e56"
	qwenOAuthScope        = "openid profile email model.completion"
	qwenDeviceGrantType   = "urn:ietf:params:oauth:grant-type:device_code"
)

// qwenEndpointFuncs holds customizable endpoint functions for testing.
var (
	qwenDeviceCodeEndpointFunc = func() string { return qwenOAuthBaseURL + "/api/v1/oauth2/device/code" }
	qwenTokenEndpointFunc      = func() string { return qwenOAuthBaseURL + "/api/v1/oauth2/token" }
)

// SetQwenTestEndpoints sets custom endpoints for testing.
// This function is exported for testing purposes only.
func SetQwenTestEndpoints(deviceCodeURL, tokenURL string) {
	qwenDeviceCodeEndpointFunc = func() string { return deviceCodeURL }
	qwenTokenEndpointFunc = func() string { return tokenURL }
}

// qwenDeviceAuthorization is returned by the device/code endpoint.
type qwenDeviceAuthorization struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// qwenTokenResponse is returned by the token polling endpoint.
type qwenTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	// Error fields for pending/error states
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	// Resource URL (base URL for API calls)
	ResourceURL string `json:"resource_url"`
}

// generatePKCE generates a PKCE (RFC 7636) verifier and S256 challenge pair.
func generatePKCE() (verifier, challenge string, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("generating PKCE verifier: %w", err)
	}
	verifier = base64.RawURLEncoding.EncodeToString(raw)

	h := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(h[:])
	return verifier, challenge, nil
}

// requestQwenDeviceCode requests a device authorization from chat.qwen.ai.
func requestQwenDeviceCode(challenge string) (*qwenDeviceAuthorization, error) {
	body := url.Values{}
	body.Set("client_id", qwenClientID)
	body.Set("scope", qwenOAuthScope)
	body.Set("code_challenge", challenge)
	body.Set("code_challenge_method", "S256")

	req, err := http.NewRequest("POST", qwenDeviceCodeEndpointFunc(), strings.NewReader(body.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-request-id", uuid.New().String())

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("device code request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device code request failed (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var da qwenDeviceAuthorization
	if err := json.Unmarshal(respBody, &da); err != nil {
		return nil, fmt.Errorf("parsing device code response: %w", err)
	}
	if da.DeviceCode == "" || da.UserCode == "" {
		return nil, fmt.Errorf("invalid device code response: missing device_code or user_code")
	}
	return &da, nil
}

// pollQwenToken polls the token endpoint until the user authorizes or the code expires.
func pollQwenToken(deviceCode, verifier string, interval, expiresIn int) (*qwenTokenResponse, error) {
	body := url.Values{}
	body.Set("grant_type", qwenDeviceGrantType)
	body.Set("client_id", qwenClientID)
	body.Set("device_code", deviceCode)
	body.Set("code_verifier", verifier)

	client := &http.Client{Timeout: 15 * time.Second}
	deadline := time.Now().Add(time.Duration(expiresIn) * time.Second)
	pollInterval := time.Duration(interval) * time.Second
	if pollInterval < 3*time.Second {
		pollInterval = 3 * time.Second
	}

	for time.Now().Before(deadline) {
		time.Sleep(pollInterval)

		req, err := http.NewRequest("POST", qwenTokenEndpointFunc(), strings.NewReader(body.Encode()))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			// transient network error — keep polling
			continue
		}
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var tok qwenTokenResponse
		if err := json.Unmarshal(respBody, &tok); err != nil {
			continue
		}

		switch tok.Error {
		case "":
			// Success — must have access_token
			if tok.AccessToken != "" {
				return &tok, nil
			}
			return nil, fmt.Errorf("token response missing access_token")
		case "authorization_pending":
			// User has not yet authorized; keep polling
			continue
		case "slow_down":
			// Server asking us to back off
			pollInterval += 5 * time.Second
			continue
		case "expired_token":
			return nil, fmt.Errorf("device code expired — please run the login command again")
		case "access_denied":
			return nil, fmt.Errorf("authorization denied by user")
		default:
			desc := tok.ErrorDescription
			if desc == "" {
				desc = tok.Error
			}
			return nil, fmt.Errorf("OAuth error: %s", desc)
		}
	}
	return nil, fmt.Errorf("timed out waiting for authorization")
}

// LoginQwenQRCode performs the Qwen Portal OAuth device-code flow.
// It prints a verification URL for the user to open and authorize, then polls
// until the token is granted.
func LoginQwenQRCode() (*AuthCredential, error) {
	fmt.Println()
	//nolint:gosmopolitan // intentional Chinese characters for user-facing message
	fmt.Println("=== Qwen (通义千问) OAuth Login ===")
	fmt.Println()

	// 1. Generate PKCE
	verifier, challenge, err := generatePKCE()
	if err != nil {
		return nil, err
	}

	// 2. Request device code
	fmt.Println("Requesting authorization code from chat.qwen.ai...")
	da, err := requestQwenDeviceCode(challenge)
	if err != nil {
		return nil, fmt.Errorf("requesting device code: %w", err)
	}

	// 3. Show the user how to authorize
	fmt.Println()
	fmt.Println("──────────────────────────────────────────────────")

	verifyURL := da.VerificationURIComplete
	if verifyURL == "" {
		verifyURL = da.VerificationURI
	}
	if verifyURL != "" {
		fmt.Printf("  1. Open this URL in your browser:\n\n     %s\n\n", verifyURL)
	}
	if da.UserCode != "" && da.VerificationURIComplete == "" {
		fmt.Printf("  2. Enter the code: %s\n\n", da.UserCode)
	}
	fmt.Println("  3. Log in with your Qwen / Alibaba Cloud account")
	fmt.Println("  4. Click \"Authorize\" in the browser")
	fmt.Println()
	fmt.Println("──────────────────────────────────────────────────")

	// Try to display a simple QR code hint
	if verifyURL != "" {
		printSimpleQRHint(verifyURL)
	}

	fmt.Println()
	fmt.Printf("Waiting for authorization (expires in %ds)...\n", da.ExpiresIn)

	// 4. Poll for token
	tok, err := pollQwenToken(da.DeviceCode, verifier, da.Interval, da.ExpiresIn)
	if err != nil {
		return nil, err
	}

	// 5. Build credential
	expiresAt := time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)

	cred := &AuthCredential{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		ExpiresAt:    expiresAt,
		Provider:     "qwen",
		AuthMethod:   "oauth",
	}
	return cred, nil
}

// RefreshQwenCredentials exchanges a refresh_token for a new access_token.
func RefreshQwenCredentials(cred *AuthCredential) (*AuthCredential, error) {
	if cred == nil || cred.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token available — please run: picoclaw auth login --provider qwen")
	}

	body := url.Values{}
	body.Set("grant_type", "refresh_token")
	body.Set("refresh_token", cred.RefreshToken)
	body.Set("client_id", qwenClientID)

	req, err := http.NewRequest("POST", qwenTokenEndpointFunc(), strings.NewReader(body.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refreshing Qwen token: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusBadRequest {
		return nil, fmt.Errorf("Qwen refresh token expired — please run: picoclaw auth login --provider qwen")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var tok qwenTokenResponse
	if err := json.Unmarshal(respBody, &tok); err != nil {
		return nil, fmt.Errorf("parsing refresh response: %w", err)
	}
	if tok.AccessToken == "" {
		return nil, fmt.Errorf("refresh response missing access_token")
	}

	newCred := *cred
	newCred.AccessToken = tok.AccessToken
	if tok.RefreshToken != "" {
		newCred.RefreshToken = tok.RefreshToken
	}
	if tok.ExpiresIn > 0 {
		newCred.ExpiresAt = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	}
	return &newCred, nil
}

// CreateQwenTokenSource returns a closure that provides a valid Qwen OAuth
// access token, automatically refreshing it when expired.
func CreateQwenTokenSource() func() (string, error) {
	return func() (string, error) {
		cred, err := GetCredential("qwen")
		if err != nil {
			return "", fmt.Errorf("loading qwen credentials: %w", err)
		}
		if cred == nil || cred.AccessToken == "" {
			return "", fmt.Errorf("not authenticated with Qwen — run: picoclaw auth login --provider qwen")
		}

		// Auto-refresh if token expires within 5 minutes
		if !cred.ExpiresAt.IsZero() && time.Until(cred.ExpiresAt) < 5*time.Minute {
			newCred, refreshErr := RefreshQwenCredentials(cred)
			if refreshErr == nil {
				_ = SetCredential("qwen", newCred)
				return newCred.AccessToken, nil
			}
			// Refresh failed but token may still be valid; fall through
		}
		return cred.AccessToken, nil
	}
}

// printSimpleQRHint prints a minimal hint to help users open the URL.
func printSimpleQRHint(verifyURL string) {
	// Print a text-based QR code placeholder — real QR rendering would require
	// an external library; here we just give a clear visual cue.
	fmt.Println("  ┌─────────────────────────────────────────┐")
	fmt.Println("  │  Scan or open the URL above in browser  │")
	fmt.Println("  └─────────────────────────────────────────┘")
	_ = verifyURL
}

// IsQwenOAuthModel reports whether a model string belongs to the qwen-oauth protocol.
func IsQwenOAuthModel(model string) bool {
	return strings.HasPrefix(model, "qwen-oauth/") || model == "qwen-oauth"
}
