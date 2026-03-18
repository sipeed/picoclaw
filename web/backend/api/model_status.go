package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

const modelProbeTimeout = 800 * time.Millisecond

var (
	probeTCPServiceFunc            = probeTCPService
	probeOllamaModelFunc           = probeOllamaModel
	probeOpenAICompatibleModelFunc = probeOpenAICompatibleModel
)

func hasModelConfiguration(m config.ModelConfig) bool {
	authMethod := strings.ToLower(strings.TrimSpace(m.AuthMethod))
	apiKey := strings.TrimSpace(m.APIKey)

	if authMethod == "oauth" || authMethod == "token" {
		if provider, ok := oauthProviderForModel(m.Model); ok {
			cred, err := oauthGetCredential(provider)
			if err != nil || cred == nil {
				return false
			}
			return strings.TrimSpace(cred.AccessToken) != "" || strings.TrimSpace(cred.RefreshToken) != ""
		}
		return true
	}

	if requiresRuntimeProbe(m) {
		return true
	}

	return apiKey != ""
}

// isModelConfigured reports whether a model is currently available to use.
// Local models must be reachable; remote/API-key models only need saved config.
func isModelConfigured(m config.ModelConfig) bool {
	if !hasModelConfiguration(m) {
		return false
	}
	if requiresRuntimeProbe(m) {
		return probeLocalModelAvailability(m)
	}
	return true
}

func requiresRuntimeProbe(m config.ModelConfig) bool {
	authMethod := strings.ToLower(strings.TrimSpace(m.AuthMethod))
	if authMethod == "local" {
		return true
	}

	switch modelProtocol(m.Model) {
	case "claude-cli", "claudecli", "codex-cli", "codexcli", "github-copilot", "copilot":
		return true
	case "ollama", "vllm":
		apiBase := strings.TrimSpace(m.APIBase)
		return apiBase == "" || hasLocalAPIBase(apiBase)
	}

	if hasLocalAPIBase(m.APIBase) {
		return true
	}

	return false
}

func probeLocalModelAvailability(m config.ModelConfig) bool {
	apiBase := modelProbeAPIBase(m)
	protocol, modelID := splitModel(m.Model)
	switch protocol {
	case "ollama":
		return probeOllamaModelFunc(apiBase, modelID)
	case "vllm":
		return probeOpenAICompatibleModelFunc(apiBase, modelID)
	case "github-copilot", "copilot":
		return probeTCPServiceFunc(apiBase)
	case "claude-cli", "claudecli", "codex-cli", "codexcli":
		return true
	default:
		if hasLocalAPIBase(apiBase) {
			return probeOpenAICompatibleModelFunc(apiBase, modelID)
		}
		return false
	}
}

func modelProbeAPIBase(m config.ModelConfig) string {
	if apiBase := strings.TrimSpace(m.APIBase); apiBase != "" {
		return normalizeModelProbeAPIBase(apiBase)
	}

	switch modelProtocol(m.Model) {
	case "ollama":
		return "http://localhost:11434/v1"
	case "vllm":
		return "http://localhost:8000/v1"
	case "github-copilot", "copilot":
		return "localhost:4321"
	default:
		return ""
	}
}

func normalizeModelProbeAPIBase(raw string) string {
	u, err := parseAPIBase(raw)
	if err != nil {
		return strings.TrimSpace(raw)
	}

	switch strings.ToLower(u.Hostname()) {
	case "0.0.0.0":
		u.Host = net.JoinHostPort("127.0.0.1", u.Port())
	case "::":
		u.Host = net.JoinHostPort("::1", u.Port())
	default:
		return strings.TrimSpace(raw)
	}

	if u.Port() == "" {
		u.Host = u.Hostname()
	}

	return u.String()
}

func oauthProviderForModel(model string) (string, bool) {
	switch modelProtocol(model) {
	case "openai":
		return oauthProviderOpenAI, true
	case "anthropic":
		return oauthProviderAnthropic, true
	case "antigravity", "google-antigravity":
		return oauthProviderGoogleAntigravity, true
	default:
		return "", false
	}
}

func modelProtocol(model string) string {
	protocol, _ := splitModel(model)
	return protocol
}

func splitModel(model string) (protocol, modelID string) {
	model = strings.ToLower(strings.TrimSpace(model))
	protocol, _, found := strings.Cut(model, "/")
	if !found {
		return "openai", model
	}
	return protocol, strings.TrimSpace(model[strings.Index(model, "/")+1:])
}

func hasLocalAPIBase(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}

	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		u, err = url.Parse("//" + raw)
		if err != nil {
			return false
		}
	}

	switch strings.ToLower(u.Hostname()) {
	case "localhost", "127.0.0.1", "::1", "0.0.0.0":
		return true
	default:
		return false
	}
}

func probeTCPService(raw string) bool {
	hostPort, err := hostPortFromAPIBase(raw)
	if err != nil {
		return false
	}

	conn, err := net.DialTimeout("tcp", hostPort, modelProbeTimeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func probeOllamaModel(apiBase, modelID string) bool {
	root, err := apiRootFromAPIBase(apiBase)
	if err != nil {
		return false
	}

	var resp struct {
		Models []struct {
			Name  string `json:"name"`
			Model string `json:"model"`
		} `json:"models"`
	}
	if err := getJSON(root+"/api/tags", &resp); err != nil {
		return false
	}

	for _, model := range resp.Models {
		if ollamaModelMatches(model.Name, modelID) || ollamaModelMatches(model.Model, modelID) {
			return true
		}
	}
	return false
}

func probeOpenAICompatibleModel(apiBase, modelID string) bool {
	if strings.TrimSpace(apiBase) == "" {
		return false
	}

	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := getJSON(strings.TrimRight(strings.TrimSpace(apiBase), "/")+"/models", &resp); err != nil {
		return false
	}

	for _, model := range resp.Data {
		if strings.EqualFold(strings.TrimSpace(model.ID), modelID) {
			return true
		}
	}
	return false
}

func getJSON(rawURL string, out any) error {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: modelProbeTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func apiRootFromAPIBase(raw string) (string, error) {
	u, err := parseAPIBase(raw)
	if err != nil {
		return "", err
	}
	return (&url.URL{Scheme: u.Scheme, Host: u.Host}).String(), nil
}

func hostPortFromAPIBase(raw string) (string, error) {
	u, err := parseAPIBase(raw)
	if err != nil {
		return "", err
	}

	if port := u.Port(); port != "" {
		return u.Host, nil
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return net.JoinHostPort(u.Hostname(), "443"), nil
	default:
		return net.JoinHostPort(u.Hostname(), "80"), nil
	}
}

func parseAPIBase(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty api base")
	}

	u, err := url.Parse(raw)
	if err == nil && u.Hostname() != "" {
		return u, nil
	}

	u, err = url.Parse("//" + raw)
	if err != nil || u.Hostname() == "" {
		return nil, fmt.Errorf("invalid api base %q", raw)
	}
	if u.Scheme == "" {
		u.Scheme = "http"
	}
	return u, nil
}

func ollamaModelMatches(candidate, want string) bool {
	candidate = strings.TrimSpace(candidate)
	want = strings.TrimSpace(want)
	if candidate == "" || want == "" {
		return false
	}
	if strings.EqualFold(candidate, want) {
		return true
	}

	base, _, _ := strings.Cut(candidate, ":")
	return strings.EqualFold(base, want)
}
