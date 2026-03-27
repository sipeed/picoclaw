package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/channels/zalo"
)

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	appIDFlag := flag.String("app-id", envOrDefault("ZALO_APP_ID", ""), "Zalo App ID (or set ZALO_APP_ID)")
	appSecretFlag := flag.String("app-secret", envOrDefault("ZALO_APP_SECRET", ""), "Zalo App Secret (or set ZALO_APP_SECRET)")
	redirectURIFlag := flag.String("redirect-uri", envOrDefault("ZALO_REDIRECT_URI", ""), "OAuth redirect URI (or set ZALO_REDIRECT_URI)")
	configPathFlag := flag.String("config", envOrDefault("ZALO_CONFIG_PATH", "docker/data/config.json"), "Path to config.json")
	listenAddrFlag := flag.String("listen", envOrDefault("ZALO_LISTEN_ADDR", "127.0.0.1:9999"), "Callback server listen address")
	flag.Parse()

	appID := *appIDFlag
	appSecret := *appSecretFlag
	redirectURI := *redirectURIFlag
	configPath := *configPathFlag
	listenAddr := *listenAddrFlag

	if appID == "" || appSecret == "" || redirectURI == "" {
		fmt.Fprintln(os.Stderr, "Error: app-id, app-secret, and redirect-uri are required.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  go run ./cmd/zalo-auth/ --app-id=XXX --app-secret=XXX --redirect-uri=https://...")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Or set environment variables: ZALO_APP_ID, ZALO_APP_SECRET, ZALO_REDIRECT_URI")
		os.Exit(1)
	}

	pkce, err := zalo.GeneratePKCE()
	if err != nil {
		log.Fatal("GeneratePKCE:", err)
	}

	// Save verifier
	if err := os.WriteFile("/tmp/zalo_verifier.txt", []byte(pkce.Verifier), 0o600); err != nil {
		log.Fatal("write verifier:", err)
	}

	authURL := fmt.Sprintf(
		"https://oauth.zaloapp.com/v4/oa/permission?app_id=%s&redirect_uri=%s&code_challenge=%s&code_challenge_method=S256",
		appID, url.QueryEscape(redirectURI), pkce.Challenge,
	)

	fmt.Println("══════════════════════════════════════════════════════════")
	fmt.Println("  Zalo OAuth — Open this URL in your browser:")
	fmt.Println()
	fmt.Println(" ", authURL)
	fmt.Println()
	fmt.Println("  Waiting for callback on", listenAddr, "...")
	fmt.Println("══════════════════════════════════════════════════════════")

	srv := &http.Server{Addr: listenAddr}

	http.HandleFunc("/auth/zalo/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprint(w, "Error: no 'code' in callback. Query: "+r.URL.RawQuery)
			return
		}

		fmt.Println("\n[+] Received authorization code:", code[:min(20, len(code))]+"...")

		accessToken, refreshToken, err := exchangeToken(appID, appSecret, code, pkce.Verifier)
		if err != nil {
			msg := fmt.Sprintf("Token exchange failed: %v", err)
			fmt.Println("[-]", msg)
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprint(w, msg)
			return
		}

		fmt.Println("[+] Access Token:", accessToken[:min(30, len(accessToken))]+"...")
		fmt.Println("[+] Refresh Token:", refreshToken[:min(30, len(refreshToken))]+"...")

		if err := updateConfig(configPath, accessToken, refreshToken); err != nil {
			fmt.Println("[-] Config update failed:", err)
		} else {
			fmt.Println("[+] Config updated:", configPath)
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "<h1>✅ Zalo OAuth Success</h1><p>Access token saved. Gateway restarting...</p>")

		restartGateway()

		// Shutdown server after response
		go func() {
			time.Sleep(500 * time.Millisecond)
			srv.Shutdown(context.Background())
		}()
	})

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}

	fmt.Println("\n[+] Done! Zalo channel should now be fully operational.")
}

func exchangeToken(appID, appSecret, code, codeVerifier string) (accessToken, refreshToken string, err error) {
	data := url.Values{
		"app_id":        {appID},
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"code_verifier": {codeVerifier},
	}

	req, err := http.NewRequest(http.MethodPost, "https://oauth.zaloapp.com/v4/oa/access_token",
		strings.NewReader(data.Encode()))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("secret_key", appSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Println("[*] Token response:", string(body))

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Error        int    `json:"error"`
		ErrorDesc    string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", fmt.Errorf("parse: %w", err)
	}
	if result.Error != 0 {
		return "", "", fmt.Errorf("zalo error %d: %s", result.Error, result.ErrorDesc)
	}

	return result.AccessToken, result.RefreshToken, nil
}

func updateConfig(configPath, accessToken, refreshToken string) error {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return err
	}

	channels, _ := cfg["channels"].(map[string]any)
	if channels == nil {
		return fmt.Errorf("no channels in config")
	}
	zaloCfg, _ := channels["zalo"].(map[string]any)
	if zaloCfg == nil {
		return fmt.Errorf("no zalo in channels config")
	}

	zaloCfg["access_token"] = accessToken
	zaloCfg["refresh_token"] = refreshToken

	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, append(out, '\n'), 0o644)
}

func restartGateway() {
	fmt.Println("[*] Restarting gateway...")
	cmd := exec.Command("docker", "compose", "-f", "docker/docker-compose.yml",
		"--profile", "gateway", "restart")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("[-] Restart failed:", err)
		fmt.Println("    Run manually: cd ~/picoclaw && docker compose -f docker/docker-compose.yml --profile gateway restart")
	} else {
		fmt.Println("[+] Gateway restarted")
	}
}
