package doctor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/auth"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// Severity classifies how bad a problem is.
type Severity int

const (
	SeverityInfo Severity = iota
	SeverityWarn
	SeverityError
)

func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarn:
		return "warn"
	case SeverityError:
		return "ERROR"
	default:
		return "?"
	}
}

func (s Severity) Icon() string {
	switch s {
	case SeverityInfo:
		return "i"
	case SeverityWarn:
		return "!"
	case SeverityError:
		return "x"
	default:
		return "?"
	}
}

// Finding is a single problem or observation.
type Finding struct {
	Check    string
	Severity Severity
	Message  string
	Fix      string // non-empty if auto-fixable
	FixFunc  func() error
}

// Result is what a check function returns.
type Result struct {
	Findings []Finding
}

func (r *Result) Add(check string, sev Severity, msg string) {
	r.Findings = append(r.Findings, Finding{Check: check, Severity: sev, Message: msg})
}

func (r *Result) AddFixable(check string, sev Severity, msg, fix string, fn func() error) {
	r.Findings = append(r.Findings, Finding{Check: check, Severity: sev, Message: msg, Fix: fix, FixFunc: fn})
}

func (r *Result) OK(check, msg string) {
	r.Add(check, SeverityInfo, msg)
}

func (r *Result) Warn(check, msg string) {
	r.Add(check, SeverityWarn, msg)
}

func (r *Result) Error(check, msg string) {
	r.Add(check, SeverityError, msg)
}

// Options controls doctor behavior.
type Options struct {
	Fix       bool   // attempt auto-fixes
	ConfigDir string // ~/.picoclaw
}

// Run executes all checks and returns findings.
func Run(opts Options) []Finding {
	if opts.ConfigDir == "" {
		home, _ := os.UserHomeDir()
		opts.ConfigDir = filepath.Join(home, ".picoclaw")
	}

	var all []Finding
	checks := []func(Options) Result{
		checkWorkspace,
		checkConfig,
		checkSessions,
		checkAuth,
	}
	for _, check := range checks {
		r := check(opts)
		all = append(all, r.Findings...)
	}
	return all
}

// ---------------------------------------------------------------------------
// Check: workspace structure
// ---------------------------------------------------------------------------

func checkWorkspace(opts Options) Result {
	var r Result
	check := "workspace"

	configDir := opts.ConfigDir
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		r.Error(check, fmt.Sprintf("config directory missing: %s", configDir))
		return r
	}
	r.OK(check, fmt.Sprintf("config directory exists: %s", configDir))

	configFile := filepath.Join(configDir, "config.json")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		r.Error(check, "config.json missing — run 'picoclaw onboard'")
		return r
	}
	r.OK(check, "config.json exists")

	// Load config to find workspace path
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		r.Error(check, fmt.Sprintf("config.json parse error: %v", err))
		return r
	}

	ws := cfg.WorkspacePath()
	if _, err := os.Stat(ws); os.IsNotExist(err) {
		r.Warn(check, fmt.Sprintf("workspace directory missing: %s", ws))
	} else {
		r.OK(check, fmt.Sprintf("workspace directory exists: %s", ws))
	}

	sessionsDir := filepath.Join(ws, "sessions")
	if _, err := os.Stat(sessionsDir); os.IsNotExist(err) {
		r.OK(check, "sessions directory does not exist yet (will be created on first use)")
	} else {
		r.OK(check, fmt.Sprintf("sessions directory exists: %s", sessionsDir))
	}

	// Check auth.json permissions
	authFile := filepath.Join(configDir, "auth.json")
	if info, err := os.Stat(authFile); err == nil {
		perm := info.Mode().Perm()
		if perm&0077 != 0 {
			r.AddFixable(check, SeverityWarn,
				fmt.Sprintf("auth.json has loose permissions: %o (should be 600)", perm),
				"chmod 600 auth.json",
				func() error { return os.Chmod(authFile, 0600) },
			)
		} else {
			r.OK(check, "auth.json permissions OK (600)")
		}
	}

	return r
}

// ---------------------------------------------------------------------------
// Check: config validation
// ---------------------------------------------------------------------------

func checkConfig(opts Options) Result {
	var r Result
	check := "config"

	configFile := filepath.Join(opts.ConfigDir, "config.json")
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		r.Error(check, fmt.Sprintf("cannot load config: %v", err))
		return r
	}

	defaultModel := cfg.Agents.Defaults.Model
	if defaultModel == "" {
		r.Error(check, "no default model configured")
		return r
	}
	r.OK(check, fmt.Sprintf("default model: %s", defaultModel))

	if len(cfg.ModelList) == 0 {
		r.Error(check, "model_list is empty — no models configured")
		return r
	}
	r.OK(check, fmt.Sprintf("%d model(s) in model_list", len(cfg.ModelList)))

	// Check each model entry
	foundDefault := false
	for i, m := range cfg.ModelList {
		if err := m.Validate(); err != nil {
			r.Error(check, fmt.Sprintf("model_list[%d] (%s): %v", i, m.ModelName, err))
			continue
		}

		// Check provider prefix
		parts := strings.SplitN(m.Model, "/", 2)
		if len(parts) < 2 {
			r.Warn(check, fmt.Sprintf("model_list[%d] (%s): model identifier %q missing provider/ prefix", i, m.ModelName, m.Model))
		}

		// Check auth: needs either api_key or auth_method
		if m.APIKey == "" && m.AuthMethod == "" {
			r.Warn(check, fmt.Sprintf("model_list[%d] (%s): no api_key or auth_method set", i, m.ModelName))
		}

		if m.ModelName == defaultModel {
			foundDefault = true
		}
	}

	if !foundDefault {
		r.Error(check, fmt.Sprintf("default model %q not found in model_list — agent will fail to start", defaultModel))
	} else {
		r.OK(check, fmt.Sprintf("default model %q found in model_list", defaultModel))
	}

	return r
}

// ---------------------------------------------------------------------------
// Check: session integrity
// ---------------------------------------------------------------------------

// sessionFile is the raw JSON structure we load for inspection.
type sessionFile struct {
	Key      string              `json:"key"`
	Messages []providers.Message `json:"messages"`
	Summary  string              `json:"summary,omitempty"`
	Created  time.Time           `json:"created"`
	Updated  time.Time           `json:"updated"`
}

func checkSessions(opts Options) Result {
	var r Result
	check := "sessions"

	configFile := filepath.Join(opts.ConfigDir, "config.json")
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		r.Error(check, fmt.Sprintf("cannot load config to find workspace: %v", err))
		return r
	}

	sessionsDir := filepath.Join(cfg.WorkspacePath(), "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			r.OK(check, "no sessions directory — nothing to check")
			return r
		}
		r.Error(check, fmt.Sprintf("cannot read sessions directory: %v", err))
		return r
	}

	sessionCount := 0
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		sessionCount++

		filePath := filepath.Join(sessionsDir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			r.Error(check, fmt.Sprintf("%s: cannot read: %v", entry.Name(), err))
			continue
		}

		var sess sessionFile
		if err := json.Unmarshal(data, &sess); err != nil {
			r.Error(check, fmt.Sprintf("%s: invalid JSON: %v", entry.Name(), err))
			continue
		}

		problems := checkSessionMessages(sess.Messages)
		if len(problems) == 0 {
			r.OK(check, fmt.Sprintf("%s: %d messages, OK", entry.Name(), len(sess.Messages)))
		} else {
			for _, p := range problems {
				r.AddFixable(check, SeverityError,
					fmt.Sprintf("%s: %s", entry.Name(), p),
					"repair corrupt session (inject synthetic tool results, drop orphans)",
					makeSessionRepairFunc(filePath, sess),
				)
			}
		}
	}

	if sessionCount == 0 {
		r.OK(check, "no session files found")
	}

	return r
}

// checkSessionMessages inspects a message array for common corruption patterns.
func checkSessionMessages(msgs []providers.Message) []string {
	var problems []string

	// Build a set of tool_call IDs that have a corresponding tool result.
	toolResultIDs := map[string]bool{}
	for _, m := range msgs {
		if m.Role == "tool" && m.ToolCallID != "" {
			toolResultIDs[m.ToolCallID] = true
		}
	}

	for i, m := range msgs {
		// Check: assistant message with tool_calls must be followed by tool results
		if m.Role == "assistant" && len(m.ToolCalls) > 0 {
			for _, tc := range m.ToolCalls {
				id := tc.ID
				if id == "" {
					// Use Function.Name for identification
					name := tc.Name
					if tc.Function != nil {
						name = tc.Function.Name
					}
					problems = append(problems, fmt.Sprintf("message[%d]: tool_call has empty ID (tool: %s)", i, name))
					continue
				}
				if !toolResultIDs[id] {
					name := tc.Name
					if tc.Function != nil {
						name = tc.Function.Name
					}
					problems = append(problems, fmt.Sprintf("message[%d]: orphan tool_call %q (tool: %s) — no matching tool result", i, id, name))
				}
			}
		}

		// Check: tool result must have a corresponding tool_call
		if m.Role == "tool" && m.ToolCallID != "" {
			found := false
			for _, prev := range msgs[:i] {
				if prev.Role != "assistant" {
					continue
				}
				for _, tc := range prev.ToolCalls {
					if tc.ID == m.ToolCallID {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				problems = append(problems, fmt.Sprintf("message[%d]: orphan tool_result %q — no matching tool_call", i, m.ToolCallID))
			}
		}

		// Check: empty content on non-tool messages is suspicious (not always an error)
		if m.Role == "assistant" && m.Content == "" && len(m.ToolCalls) == 0 {
			problems = append(problems, fmt.Sprintf("message[%d]: assistant message with empty content and no tool_calls", i))
		}

		// Check: consecutive same-role messages (user, user) — some providers reject this
		if i > 0 && m.Role == msgs[i-1].Role && m.Role == "user" {
			problems = append(problems, fmt.Sprintf("message[%d]: consecutive user messages (some providers reject this)", i))
		}
	}

	return problems
}

// repairSessionMessages fixes orphaned tool_use/tool_result pairs in a message slice.
func repairSessionMessages(msgs []providers.Message) []providers.Message {
	if len(msgs) == 0 {
		return msgs
	}

	// Collect all tool_call IDs
	toolCallIDs := map[string]bool{}
	for _, m := range msgs {
		if m.Role == "assistant" {
			for _, tc := range m.ToolCalls {
				if tc.ID != "" {
					toolCallIDs[tc.ID] = true
				}
			}
		}
	}

	// Collect all tool_result IDs
	toolResultIDs := map[string]bool{}
	for _, m := range msgs {
		if m.Role == "tool" && m.ToolCallID != "" {
			toolResultIDs[m.ToolCallID] = true
		}
	}

	// Build repaired slice: drop orphan results, inject missing results
	repaired := make([]providers.Message, 0, len(msgs))
	for _, m := range msgs {
		// Drop orphaned tool_result
		if m.Role == "tool" && m.ToolCallID != "" && !toolCallIDs[m.ToolCallID] {
			continue
		}
		repaired = append(repaired, m)

		// Inject missing tool_results after assistant+tool_calls
		if m.Role == "assistant" && len(m.ToolCalls) > 0 {
			for _, tc := range m.ToolCalls {
				if tc.ID == "" {
					continue
				}
				if !toolResultIDs[tc.ID] {
					repaired = append(repaired, providers.Message{
						Role:       "tool",
						ToolCallID: tc.ID,
						Content:    "[tool result unavailable - session was repaired by picoclaw doctor]",
					})
					toolResultIDs[tc.ID] = true
				}
			}
		}
	}
	return repaired
}

func makeSessionRepairFunc(path string, sess sessionFile) func() error {
	return func() error {
		sess.Messages = repairSessionMessages(sess.Messages)
		data, err := json.MarshalIndent(sess, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling repaired session: %w", err)
		}
		return os.WriteFile(path, data, 0o644)
	}
}

// ---------------------------------------------------------------------------
// Check: auth credential health
// ---------------------------------------------------------------------------

func checkAuth(opts Options) Result {
	var r Result
	check := "auth"

	store, err := auth.LoadStore()
	if err != nil {
		r.Error(check, fmt.Sprintf("cannot load auth store: %v", err))
		return r
	}

	if len(store.Credentials) == 0 {
		r.Warn(check, "no credentials stored — run 'picoclaw auth login'")
		return r
	}

	r.OK(check, fmt.Sprintf("%d credential(s) found", len(store.Credentials)))

	for provider, cred := range store.Credentials {
		prefix := fmt.Sprintf("[%s]", provider)

		if cred.AccessToken == "" {
			r.Error(check, fmt.Sprintf("%s access_token is empty", prefix))
			continue
		}

		switch cred.AuthMethod {
		case "oauth":
			if cred.ExpiresAt.IsZero() {
				r.Warn(check, fmt.Sprintf("%s OAuth token has no expiry set", prefix))
			} else if cred.IsExpired() {
				if cred.RefreshToken != "" {
					r.Warn(check, fmt.Sprintf("%s OAuth token expired at %s (refresh token available)", prefix, cred.ExpiresAt.Format(time.RFC3339)))
					// Try a refresh
					if provider == "anthropic" {
						r.AddFixable(check, SeverityWarn,
							fmt.Sprintf("%s token expired — can attempt refresh", prefix),
							"refresh Anthropic OAuth token",
							func() error { return tryRefreshAnthropic(cred) },
						)
					}
				} else {
					r.Error(check, fmt.Sprintf("%s OAuth token expired at %s (no refresh token)", prefix, cred.ExpiresAt.Format(time.RFC3339)))
				}
			} else {
				remaining := time.Until(cred.ExpiresAt).Truncate(time.Minute)
				r.OK(check, fmt.Sprintf("%s OAuth token valid (expires in %s)", prefix, remaining))

				if cred.NeedsRefresh() {
					r.Warn(check, fmt.Sprintf("%s token expires within 5 minutes — will need refresh soon", prefix))
				}
			}

			if cred.Email != "" {
				r.OK(check, fmt.Sprintf("%s email: %s", prefix, cred.Email))
			}
			if cred.SubscriptionType != "" {
				r.OK(check, fmt.Sprintf("%s plan: %s", prefix, cred.SubscriptionType))
			}

		case "token", "":
			// Paste token — just check it looks non-empty
			r.OK(check, fmt.Sprintf("%s API key/token present (length %d)", prefix, len(cred.AccessToken)))

		default:
			r.Warn(check, fmt.Sprintf("%s unknown auth_method: %s", prefix, cred.AuthMethod))
		}

		// Check: can we actually reach the provider's API?
		if provider == "anthropic" {
			checkAnthropicReachable(&r, check, prefix)
		} else if provider == "openai" {
			checkOpenAIReachable(&r, check, prefix)
		}
	}

	return r
}

// tryRefreshAnthropic attempts to refresh an expired Anthropic OAuth token.
func tryRefreshAnthropic(cred *auth.AuthCredential) error {
	return auth.RefreshAnthropicCredential(cred)
}

func checkAnthropicReachable(r *Result, check, prefix string) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://api.anthropic.com/v1/models")
	if err != nil {
		r.Warn(check, fmt.Sprintf("%s cannot reach api.anthropic.com: %v", prefix, err))
		return
	}
	resp.Body.Close()
	// 401 is expected without auth — it means the endpoint is reachable
	if resp.StatusCode == 401 || resp.StatusCode == 200 || resp.StatusCode == 403 {
		r.OK(check, fmt.Sprintf("%s api.anthropic.com reachable", prefix))
	} else {
		r.Warn(check, fmt.Sprintf("%s api.anthropic.com returned unexpected status: %d", prefix, resp.StatusCode))
	}
}

func checkOpenAIReachable(r *Result, check, prefix string) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://api.openai.com/v1/models")
	if err != nil {
		r.Warn(check, fmt.Sprintf("%s cannot reach api.openai.com: %v", prefix, err))
		return
	}
	resp.Body.Close()
	if resp.StatusCode == 401 || resp.StatusCode == 200 || resp.StatusCode == 403 {
		r.OK(check, fmt.Sprintf("%s api.openai.com reachable", prefix))
	} else {
		r.Warn(check, fmt.Sprintf("%s api.openai.com returned unexpected status: %d", prefix, resp.StatusCode))
	}
}
