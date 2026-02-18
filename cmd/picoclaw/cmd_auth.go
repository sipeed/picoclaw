// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/sipeed/picoclaw/pkg/auth"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

func authCmd() {
	if len(os.Args) < 3 {
		authHelp()
		return
	}

	switch os.Args[2] {
	case "login":
		authLoginCmd()
	case "logout":
		authLogoutCmd()
	case "status":
		authStatusCmd()
	case "models":
		authModelsCmd()
	default:
		fmt.Printf("Unknown auth command: %s\n", os.Args[2])
		authHelp()
	}
}

func authHelp() {
	fmt.Println("\nAuth commands:")
	fmt.Println("  login       Login via OAuth or paste token")
	fmt.Println("  logout      Remove stored credentials")
	fmt.Println("  status      Show current auth status")
	fmt.Println("  models      List available Antigravity models")
	fmt.Println()
	fmt.Println("Login options:")
	fmt.Println("  --provider <name>    Provider to login with (openai, anthropic, google-antigravity)")
	fmt.Println("  --device-code        Use device code flow (for headless environments)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  picoclaw auth login --provider openai")
	fmt.Println("  picoclaw auth login --provider openai --device-code")
	fmt.Println("  picoclaw auth login --provider anthropic")
	fmt.Println("  picoclaw auth login --provider google-antigravity")
	fmt.Println("  picoclaw auth models")
	fmt.Println("  picoclaw auth logout --provider openai")
	fmt.Println("  picoclaw auth status")
}

func authLoginCmd() {
	provider := ""
	useDeviceCode := false

	args := os.Args[3:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--provider", "-p":
			if i+1 < len(args) {
				provider = args[i+1]
				i++
			}
		case "--device-code":
			useDeviceCode = true
		}
	}

	if provider == "" {
		fmt.Println("Error: --provider is required")
		fmt.Println("Supported providers: openai, anthropic, google-antigravity")
		return
	}

	switch provider {
	case "openai":
		authLoginOpenAI(useDeviceCode)
	case "anthropic":
		authLoginPasteToken(provider)
	case "google-antigravity", "antigravity":
		authLoginGoogleAntigravity()
	default:
		fmt.Printf("Unsupported provider: %s\n", provider)
		fmt.Println("Supported providers: openai, anthropic, google-antigravity")
	}
}

func authLoginOpenAI(useDeviceCode bool) {
	cfg := auth.OpenAIOAuthConfig()

	var cred *auth.AuthCredential
	var err error

	if useDeviceCode {
		cred, err = auth.LoginDeviceCode(cfg)
	} else {
		cred, err = auth.LoginBrowser(cfg)
	}

	if err != nil {
		fmt.Printf("Login failed: %v\n", err)
		os.Exit(1)
	}

	if err := auth.SetCredential("openai", cred); err != nil {
		fmt.Printf("Failed to save credentials: %v\n", err)
		os.Exit(1)
	}

	appCfg, err := loadConfig()
	if err == nil {
		appCfg.Providers.OpenAI.AuthMethod = "oauth"
		if err := config.SaveConfig(getConfigPath(), appCfg); err != nil {
			fmt.Printf("Warning: could not update config: %v\n", err)
		}
	}

	fmt.Println("Login successful!")
	if cred.AccountID != "" {
		fmt.Printf("Account: %s\n", cred.AccountID)
	}
}

func authLoginGoogleAntigravity() {
	cfg := auth.GoogleAntigravityOAuthConfig()

	cred, err := auth.LoginBrowser(cfg)
	if err != nil {
		fmt.Printf("Login failed: %v\n", err)
		os.Exit(1)
	}

	cred.Provider = "google-antigravity"

	// Fetch user email from Google userinfo
	email, err := fetchGoogleUserEmail(cred.AccessToken)
	if err != nil {
		fmt.Printf("Warning: could not fetch email: %v\n", err)
	} else {
		cred.Email = email
		fmt.Printf("Email: %s\n", email)
	}

	// Fetch Cloud Code Assist project ID
	projectID, err := providers.FetchAntigravityProjectID(cred.AccessToken)
	if err != nil {
		fmt.Printf("Warning: could not fetch project ID: %v\n", err)
		fmt.Println("You may need Google Cloud Code Assist enabled on your account.")
	} else {
		cred.ProjectID = projectID
		fmt.Printf("Project: %s\n", projectID)
	}

	if err := auth.SetCredential("google-antigravity", cred); err != nil {
		fmt.Printf("Failed to save credentials: %v\n", err)
		os.Exit(1)
	}

	appCfg, err := loadConfig()
	if err == nil {
		appCfg.Providers.Antigravity.AuthMethod = "oauth"
		if appCfg.Agents.Defaults.Provider == "" {
			appCfg.Agents.Defaults.Provider = "antigravity"
		}
		if appCfg.Agents.Defaults.Provider == "antigravity" || appCfg.Agents.Defaults.Provider == "google-antigravity" {
			appCfg.Agents.Defaults.Model = "gemini-3-flash"
		}
		if err := config.SaveConfig(getConfigPath(), appCfg); err != nil {
			fmt.Printf("Warning: could not update config: %v\n", err)
		}
	}

	fmt.Println("\n✓ Google Antigravity login successful!")
	fmt.Println("Config updated: provider=antigravity, model=gemini-3-flash")
	fmt.Println("Try it: picoclaw agent -m \"Hello world\"")
}

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

func authLoginPasteToken(provider string) {
	cred, err := auth.LoginPasteToken(provider, os.Stdin)
	if err != nil {
		fmt.Printf("Login failed: %v\n", err)
		os.Exit(1)
	}

	if err := auth.SetCredential(provider, cred); err != nil {
		fmt.Printf("Failed to save credentials: %v\n", err)
		os.Exit(1)
	}

	appCfg, err := loadConfig()
	if err == nil {
		switch provider {
		case "anthropic":
			appCfg.Providers.Anthropic.AuthMethod = "token"
		case "openai":
			appCfg.Providers.OpenAI.AuthMethod = "token"
		}
		if err := config.SaveConfig(getConfigPath(), appCfg); err != nil {
			fmt.Printf("Warning: could not update config: %v\n", err)
		}
	}

	fmt.Printf("Token saved for %s!\n", provider)
}

func authLogoutCmd() {
	provider := ""

	args := os.Args[3:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--provider", "-p":
			if i+1 < len(args) {
				provider = args[i+1]
				i++
			}
		}
	}

	if provider != "" {
		if err := auth.DeleteCredential(provider); err != nil {
			fmt.Printf("Failed to remove credentials: %v\n", err)
			os.Exit(1)
		}

		appCfg, err := loadConfig()
		if err == nil {
			switch provider {
			case "openai":
				appCfg.Providers.OpenAI.AuthMethod = ""
			case "anthropic":
				appCfg.Providers.Anthropic.AuthMethod = ""
			case "google-antigravity", "antigravity":
				appCfg.Providers.Antigravity.AuthMethod = ""
			}
			config.SaveConfig(getConfigPath(), appCfg)
		}

		fmt.Printf("Logged out from %s\n", provider)
	} else {
		if err := auth.DeleteAllCredentials(); err != nil {
			fmt.Printf("Failed to remove credentials: %v\n", err)
			os.Exit(1)
		}

		appCfg, err := loadConfig()
		if err == nil {
			appCfg.Providers.OpenAI.AuthMethod = ""
			appCfg.Providers.Anthropic.AuthMethod = ""
			appCfg.Providers.Antigravity.AuthMethod = ""
			config.SaveConfig(getConfigPath(), appCfg)
		}

		fmt.Println("Logged out from all providers")
	}
}

func authStatusCmd() {
	store, err := auth.LoadStore()
	if err != nil {
		fmt.Printf("Error loading auth store: %v\n", err)
		return
	}

	if len(store.Credentials) == 0 {
		fmt.Println("No authenticated providers.")
		fmt.Println("Run: picoclaw auth login --provider <name>")
		return
	}

	fmt.Println("\nAuthenticated Providers:")
	fmt.Println("------------------------")
	for provider, cred := range store.Credentials {
		status := "active"
		if cred.IsExpired() {
			status = "expired"
		} else if cred.NeedsRefresh() {
			status = "needs refresh"
		}

		fmt.Printf("  %s:\n", provider)
		fmt.Printf("    Method: %s\n", cred.AuthMethod)
		fmt.Printf("    Status: %s\n", status)
		if cred.AccountID != "" {
			fmt.Printf("    Account: %s\n", cred.AccountID)
		}
		if cred.Email != "" {
			fmt.Printf("    Email: %s\n", cred.Email)
		}
		if cred.ProjectID != "" {
			fmt.Printf("    Project: %s\n", cred.ProjectID)
		}
		if !cred.ExpiresAt.IsZero() {
			fmt.Printf("    Expires: %s\n", cred.ExpiresAt.Format("2006-01-02 15:04"))
		}
	}
}

func authModelsCmd() {
	cred, err := auth.GetCredential("google-antigravity")
	if err != nil || cred == nil {
		fmt.Println("Not logged in to Google Antigravity.")
		fmt.Println("Run: picoclaw auth login --provider google-antigravity")
		return
	}

	// Refresh token if needed
	if cred.NeedsRefresh() && cred.RefreshToken != "" {
		oauthCfg := auth.GoogleAntigravityOAuthConfig()
		refreshed, refreshErr := auth.RefreshAccessToken(cred, oauthCfg)
		if refreshErr == nil {
			cred = refreshed
			_ = auth.SetCredential("google-antigravity", cred)
		}
	}

	projectID := cred.ProjectID
	if projectID == "" {
		fmt.Println("No project ID stored. Try logging in again.")
		return
	}

	fmt.Printf("Fetching models for project: %s\n\n", projectID)

	models, err := providers.FetchAntigravityModels(cred.AccessToken, projectID)
	if err != nil {
		fmt.Printf("Error fetching models: %v\n", err)
		return
	}

	if len(models) == 0 {
		fmt.Println("No models available.")
		return
	}

	fmt.Println("Available Antigravity Models:")
	fmt.Println("-----------------------------")
	for _, m := range models {
		status := "✓"
		if m.IsExhausted {
			status = "✗ (quota exhausted)"
		}
		name := m.ID
		if m.DisplayName != "" {
			name = fmt.Sprintf("%s (%s)", m.ID, m.DisplayName)
		}
		fmt.Printf("  %s %s\n", status, name)
	}
}
