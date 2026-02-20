// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/auth"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/spf13/cobra"
)

const supportedProvidersMsg = "Supported providers: openai, anthropic, google-antigravity"

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication (login, logout, status)",
	}
	cmd.AddCommand(
		newAuthLoginCmd(),
		newAuthLogoutCmd(),
		newAuthStatusCmd(),
		newAuthModelsCmd(),
	)
	return cmd
}

func newAuthLoginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login via OAuth or paste token",
		RunE:  runAuthLogin,
	}
	cmd.Flags().StringP("provider", "p", "", "Provider (openai, anthropic, google-antigravity)")
	cmd.MarkFlagRequired("provider")
	cmd.Flags().Bool("device-code", false, "Use device code flow (for headless environments)")
	return cmd
}

func newAuthLogoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Remove stored credentials",
		RunE:  runAuthLogout,
	}
	cmd.Flags().StringP("provider", "p", "", "Provider to logout from (omit for all)")
	return cmd
}

func newAuthStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current auth status",
		RunE:  runAuthStatus,
	}
}

func newAuthModelsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "models",
		Short: "List available Antigravity models",
		RunE:  runAuthModels,
	}
}

func runAuthLogin(cmd *cobra.Command, args []string) error {
	provider, _ := cmd.Flags().GetString("provider")
	useDeviceCode, _ := cmd.Flags().GetBool("device-code")

	switch provider {
	case "openai":
		authLoginOpenAI(useDeviceCode)
	case "anthropic":
		authLoginPasteToken(provider)
	case "google-antigravity", "antigravity":
		authLoginGoogleAntigravity()
	default:
		fmt.Printf("Unsupported provider: %s\n", provider)
		fmt.Println(supportedProvidersMsg)
	}
	return nil
}

func runAuthLogout(cmd *cobra.Command, args []string) error {
	provider, _ := cmd.Flags().GetString("provider")

	if provider != "" {
		if err := auth.DeleteCredential(provider); err != nil {
			fmt.Printf("Failed to remove credentials: %v\n", err)
			os.Exit(1)
		}

		appCfg, err := loadConfig()
		if err == nil {
			for i := range appCfg.ModelList {
				switch provider {
				case "openai":
					if isOpenAIModel(appCfg.ModelList[i].Model) {
						appCfg.ModelList[i].AuthMethod = ""
					}
				case "anthropic":
					if isAnthropicModel(appCfg.ModelList[i].Model) {
						appCfg.ModelList[i].AuthMethod = ""
					}
				case "google-antigravity", "antigravity":
					if isAntigravityModel(appCfg.ModelList[i].Model) {
						appCfg.ModelList[i].AuthMethod = ""
					}
				}
			}
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
			for i := range appCfg.ModelList {
				appCfg.ModelList[i].AuthMethod = ""
			}
			appCfg.Providers.OpenAI.AuthMethod = ""
			appCfg.Providers.Anthropic.AuthMethod = ""
			appCfg.Providers.Antigravity.AuthMethod = ""
			config.SaveConfig(getConfigPath(), appCfg)
		}

		fmt.Println("Logged out from all providers")
	}
	return nil
}

func runAuthStatus(cmd *cobra.Command, args []string) error {
	store, err := auth.LoadStore()
	if err != nil {
		fmt.Printf("Error loading auth store: %v\n", err)
		return nil
	}

	if len(store.Credentials) == 0 {
		fmt.Println("No authenticated providers.")
		fmt.Println("Run: picoclaw auth login --provider <name>")
		return nil
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
	return nil
}

func runAuthModels(cmd *cobra.Command, args []string) error {
	cred, err := auth.GetCredential("google-antigravity")
	if err != nil || cred == nil {
		fmt.Println("Not logged in to Google Antigravity.")
		fmt.Println("Run: picoclaw auth login --provider google-antigravity")
		return nil
	}

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
		return nil
	}

	fmt.Printf("Fetching models for project: %s\n\n", projectID)

	models, err := providers.FetchAntigravityModels(cred.AccessToken, projectID)
	if err != nil {
		fmt.Printf("Error fetching models: %v\n", err)
		return nil
	}

	if len(models) == 0 {
		fmt.Println("No models available.")
		return nil
	}

	fmt.Println("Available Antigravity Models:")
	fmt.Println("-----------------------------")
	for _, m := range models {
		status := "\u2713"
		if m.IsExhausted {
			status = "\u2717 (quota exhausted)"
		}
		name := m.ID
		if m.DisplayName != "" {
			name = fmt.Sprintf("%s (%s)", m.ID, m.DisplayName)
		}
		fmt.Printf("  %s %s\n", status, name)
	}
	return nil
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

		foundOpenAI := false
		for i := range appCfg.ModelList {
			if isOpenAIModel(appCfg.ModelList[i].Model) {
				appCfg.ModelList[i].AuthMethod = "oauth"
				foundOpenAI = true
				break
			}
		}

		if !foundOpenAI {
			appCfg.ModelList = append(appCfg.ModelList, config.ModelConfig{
				ModelName:  "gpt-5.2",
				Model:      "openai/gpt-5.2",
				AuthMethod: "oauth",
			})
		}

		appCfg.Agents.Defaults.Model = "gpt-5.2"

		if err := config.SaveConfig(getConfigPath(), appCfg); err != nil {
			fmt.Printf("Warning: could not update config: %v\n", err)
		}
	}

	fmt.Println("Login successful!")
	if cred.AccountID != "" {
		fmt.Printf("Account: %s\n", cred.AccountID)
	}
	fmt.Println("Default model set to: gpt-5.2")
}

func authLoginGoogleAntigravity() {
	cfg := auth.GoogleAntigravityOAuthConfig()

	cred, err := auth.LoginBrowser(cfg)
	if err != nil {
		fmt.Printf("Login failed: %v\n", err)
		os.Exit(1)
	}

	cred.Provider = "google-antigravity"

	email, err := fetchGoogleUserEmail(cred.AccessToken)
	if err != nil {
		fmt.Printf("Warning: could not fetch email: %v\n", err)
	} else {
		cred.Email = email
		fmt.Printf("Email: %s\n", email)
	}

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

		foundAntigravity := false
		for i := range appCfg.ModelList {
			if isAntigravityModel(appCfg.ModelList[i].Model) {
				appCfg.ModelList[i].AuthMethod = "oauth"
				foundAntigravity = true
				break
			}
		}

		if !foundAntigravity {
			appCfg.ModelList = append(appCfg.ModelList, config.ModelConfig{
				ModelName:  "gemini-flash",
				Model:      "antigravity/gemini-3-flash",
				AuthMethod: "oauth",
			})
		}

		appCfg.Agents.Defaults.Model = "gemini-flash"

		if err := config.SaveConfig(getConfigPath(), appCfg); err != nil {
			fmt.Printf("Warning: could not update config: %v\n", err)
		}
	}

	fmt.Println("\n\u2713 Google Antigravity login successful!")
	fmt.Println("Default model set to: gemini-flash")
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
			found := false
			for i := range appCfg.ModelList {
				if isAnthropicModel(appCfg.ModelList[i].Model) {
					appCfg.ModelList[i].AuthMethod = "token"
					found = true
					break
				}
			}
			if !found {
				appCfg.ModelList = append(appCfg.ModelList, config.ModelConfig{
					ModelName:  "claude-sonnet-4.6",
					Model:      "anthropic/claude-sonnet-4.6",
					AuthMethod: "token",
				})
			}
			appCfg.Agents.Defaults.Model = "claude-sonnet-4.6"
		case "openai":
			appCfg.Providers.OpenAI.AuthMethod = "token"
			found := false
			for i := range appCfg.ModelList {
				if isOpenAIModel(appCfg.ModelList[i].Model) {
					appCfg.ModelList[i].AuthMethod = "token"
					found = true
					break
				}
			}
			if !found {
				appCfg.ModelList = append(appCfg.ModelList, config.ModelConfig{
					ModelName:  "gpt-5.2",
					Model:      "openai/gpt-5.2",
					AuthMethod: "token",
				})
			}
			appCfg.Agents.Defaults.Model = "gpt-5.2"
		}
		if err := config.SaveConfig(getConfigPath(), appCfg); err != nil {
			fmt.Printf("Warning: could not update config: %v\n", err)
		}
	}

	fmt.Printf("Token saved for %s!\n", provider)
	fmt.Printf("Default model set to: %s\n", appCfg.Agents.Defaults.Model)
}

// isAntigravityModel checks if a model string belongs to antigravity provider
func isAntigravityModel(model string) bool {
	return model == "antigravity" ||
		model == "google-antigravity" ||
		strings.HasPrefix(model, "antigravity/") ||
		strings.HasPrefix(model, "google-antigravity/")
}

// isOpenAIModel checks if a model string belongs to openai provider
func isOpenAIModel(model string) bool {
	return model == "openai" ||
		strings.HasPrefix(model, "openai/")
}

// isAnthropicModel checks if a model string belongs to anthropic provider
func isAnthropicModel(model string) bool {
	return model == "anthropic" ||
		strings.HasPrefix(model, "anthropic/")
}
