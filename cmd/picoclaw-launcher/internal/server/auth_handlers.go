package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/auth"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// oauthSession stores in-flight OAuth state for browser-based flows.
type oauthSession struct {
	Provider    string
	PKCE        auth.PKCECodes
	State       string
	RedirectURI string
	OAuthCfg    auth.OAuthProviderConfig
	ConfigPath  string
}

// deviceCodeSession stores in-flight device code flow state.
type deviceCodeSession struct {
	mu         sync.Mutex
	Provider   string
	Info       *auth.DeviceCodeInfo
	OAuthCfg   auth.OAuthProviderConfig
	ConfigPath string
	Status     string // "pending", "success", "error"
	Error      string
	Done       bool
}

var (
	oauthSessions   = map[string]*oauthSession{} // keyed by state
	oauthSessionsMu sync.Mutex

	activeDeviceSession   *deviceCodeSession
	activeDeviceSessionMu sync.Mutex
)

// handleOpenAILogin starts the OpenAI device code flow and returns device code info to the frontend.
func handleOpenAILogin(w http.ResponseWriter, configPath string) {
	// Check if there's already a pending device code session
	activeDeviceSessionMu.Lock()
	if activeDeviceSession != nil {
		activeDeviceSession.mu.Lock()
		if !activeDeviceSession.Done {
			resp := map[string]any{
				"status":     "pending",
				"device_url": activeDeviceSession.Info.VerifyURL,
				"user_code":  activeDeviceSession.Info.UserCode,
				"message":    "Device code flow already in progress. Enter the code in your browser.",
			}
			activeDeviceSession.mu.Unlock()
			activeDeviceSessionMu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		activeDeviceSession.mu.Unlock()
	}
	activeDeviceSessionMu.Unlock()

	// Request a device code
	oauthCfg := auth.OpenAIOAuthConfig()
	info, err := auth.RequestDeviceCode(oauthCfg)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to request device code: %v", err), http.StatusInternalServerError)
		return
	}

	session := &deviceCodeSession{
		Provider:   "openai",
		Info:       info,
		OAuthCfg:   oauthCfg,
		ConfigPath: configPath,
		Status:     "pending",
	}

	activeDeviceSessionMu.Lock()
	activeDeviceSession = session
	activeDeviceSessionMu.Unlock()

	// Start background polling
	go func() {
		deadline := time.After(15 * time.Minute)
		ticker := time.NewTicker(time.Duration(info.Interval) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-deadline:
				session.mu.Lock()
				session.Status = "error"
				session.Error = "Authentication timed out after 15 minutes"
				session.Done = true
				session.mu.Unlock()
				return
			case <-ticker.C:
				cred, err := auth.PollDeviceCodeOnce(oauthCfg, info.DeviceAuthID, info.UserCode)
				if err != nil {
					continue // Still pending
				}
				if cred != nil {
					if saveErr := auth.SetCredential("openai", cred); saveErr != nil {
						session.mu.Lock()
						session.Status = "error"
						session.Error = saveErr.Error()
						session.Done = true
						session.mu.Unlock()
						return
					}
					updateConfigAfterLogin(configPath, "openai", cred)
					session.mu.Lock()
					session.Status = "success"
					session.Done = true
					session.mu.Unlock()
					log.Printf("OpenAI device code login successful (account: %s)", cred.AccountID)
					return
				}
			}
		}
	}()

	// Return device code info to frontend
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":     "pending",
		"device_url": info.VerifyURL,
		"user_code":  info.UserCode,
		"message":    "Open the URL and enter the code to authenticate.",
	})
}

// handleAnthropicLogin saves a pasted API token for Anthropic.
func handleAnthropicLogin(w http.ResponseWriter, token, configPath string) {
	if token == "" {
		http.Error(w, "Token is required for Anthropic login", http.StatusBadRequest)
		return
	}

	cred := &auth.AuthCredential{
		AccessToken: token,
		Provider:    "anthropic",
		AuthMethod:  "token",
	}

	if err := auth.SetCredential("anthropic", cred); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save credentials: %v", err), http.StatusInternalServerError)
		return
	}

	updateConfigAfterLogin(configPath, "anthropic", cred)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Anthropic token saved",
	})
}

// handleGoogleAntigravityLogin generates a PKCE + auth URL and returns it to the frontend.
func handleGoogleAntigravityLogin(w http.ResponseWriter, r *http.Request, configPath string) {
	oauthCfg := auth.GoogleAntigravityOAuthConfig()

	pkce, err := auth.GeneratePKCE()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate PKCE: %v", err), http.StatusInternalServerError)
		return
	}

	state, err := auth.GenerateState()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate state: %v", err), http.StatusInternalServerError)
		return
	}

	// Build redirect URI pointing to picoclaw-launcher's own callback
	scheme := "http"
	redirectURI := fmt.Sprintf("%s://%s/auth/callback", scheme, r.Host)

	authURL := auth.BuildAuthorizeURL(oauthCfg, pkce, state, redirectURI)

	// Store session for callback
	oauthSessionsMu.Lock()
	oauthSessions[state] = &oauthSession{
		Provider:    "google-antigravity",
		PKCE:        pkce,
		State:       state,
		RedirectURI: redirectURI,
		OAuthCfg:    oauthCfg,
		ConfigPath:  configPath,
	}
	oauthSessionsMu.Unlock()

	// Clean up stale sessions after 10 minutes
	go func() {
		time.Sleep(10 * time.Minute)
		oauthSessionsMu.Lock()
		delete(oauthSessions, state)
		oauthSessionsMu.Unlock()
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "redirect",
		"auth_url": authURL,
		"message":  "Open the URL to authenticate with Google.",
	})
}

// handleOAuthCallback processes the OAuth callback from Google Antigravity.
func handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")

	oauthSessionsMu.Lock()
	session, ok := oauthSessions[state]
	if ok {
		delete(oauthSessions, state)
	}
	oauthSessionsMu.Unlock()

	if !ok {
		http.Error(w, "Invalid or expired OAuth state", http.StatusBadRequest)
		return
	}

	if code == "" {
		errMsg := r.URL.Query().Get("error")
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(
			w,
			`<html><body><h2>Authentication failed</h2><p>%s</p><p>You can close this window.</p></body></html>`,
			errMsg,
		)
		return
	}

	cred, err := auth.ExchangeCodeForTokens(session.OAuthCfg, code, session.PKCE.CodeVerifier, session.RedirectURI)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(
			w,
			`<html><body><h2>Authentication failed</h2><p>%s</p><p>You can close this window.</p></body></html>`,
			err.Error(),
		)
		return
	}

	cred.Provider = session.Provider

	// Fetch user info for Google Antigravity
	if session.Provider == "google-antigravity" {
		if email, err := fetchGoogleUserEmail(cred.AccessToken); err == nil {
			cred.Email = email
		}
		if projectID, err := providers.FetchAntigravityProjectID(cred.AccessToken); err == nil {
			cred.ProjectID = projectID
		}
	}

	if err := auth.SetCredential(session.Provider, cred); err != nil {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<html><body><h2>Failed to save credentials</h2><p>%s</p></body></html>`, err.Error())
		return
	}

	updateConfigAfterLogin(session.ConfigPath, session.Provider, cred)

	// Redirect back to picoclaw-launcher UI
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<html><body>
		<h2>Authentication successful!</h2>
		<p>Redirecting back to Config Editor...</p>
		<script>setTimeout(function(){ window.location.href = '/#auth'; }, 1000);</script>
	</body></html>`)
}

// fetchGoogleUserEmail retrieves the user's email from Google's userinfo endpoint.
func fetchGoogleUserEmail(accessToken string) (string, error) {
	req, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("userinfo request failed: %s", string(body))
	}

	var userInfo struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return "", err
	}
	return userInfo.Email, nil
}
