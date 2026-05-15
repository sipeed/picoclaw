// Package tools integrates Tirith pre-exec security scanning.
//
// Tirith (https://github.com/sheeki03/tirith) is a terminal security tool
// that scans commands for content-level threats: homograph/punycode URLs,
// pipe-to-interpreter patterns, base64 decode-execute chains, terminal
// injection, suspicious packages/URLs, insecure transport, and local
// threat-intelligence matches when configured in Tirith.
//
// PicoClaw invokes a locally installed Tirith binary when explicitly enabled.
// It does not download, install, vendor, or bundle Tirith, and it disables
// Tirith auto-update / live enrichment for this pre-exec check.
package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	tirithDefaultBin       = "tirith"
	tirithDefaultTimeout   = 5
	tirithStdoutLimitBytes = 256 * 1024
	tirithStderrLimitBytes = 8 * 1024
	tirithMaxFindings      = 50
	tirithMaxSummaryRunes  = 500
)

var tirithScrubbedEnvVars = map[string]struct{}{
	"TIRITH":                       {},
	"TIRITH_POLICY_ROOT":           {},
	"TIRITH_SERVER_URL":            {},
	"TIRITH_API_KEY":               {},
	"TIRITH_ALLOW_HTTP":            {},
	"GOOGLE_SAFE_BROWSING_API_KEY": {},
	"GOOGLE_SAFE_BROWSING_KEY":     {},
	"SAFE_BROWSING_API_KEY":        {},
	"ABUSECH_AUTH_KEY":             {},
	"ABUSE_CH_AUTH_KEY":            {},
	"URLHAUS_AUTH_KEY":             {},
	"THREATFOX_AUTH_KEY":           {},
}

const tirithLocalOnlyPolicy = `policy_server_url: null
policy_server_api_key: null
allow_bypass_env: false
allow_bypass_env_noninteractive: false
threat_intel:
  auto_update_hours: 0
  osv_enabled: false
  deps_dev_enabled: false
  google_safe_browsing_key: null
  abusech_auth_key: null
  phishing_army_enabled: false
`

// TirithConfig holds Tirith security scanner settings (tools-internal).
// Mapped from config.TirithConfig at ExecTool construction time.
type TirithConfig struct {
	Enabled  bool
	BinPath  string
	Timeout  int
	FailOpen bool
}

var tirithWarningCache = struct {
	sync.Mutex
	seen map[string]struct{}
}{
	seen: make(map[string]struct{}),
}

// tirithGuard scans a command with tirith for content-level threats.
// Call AFTER guardCommand() because cheap regex guards reject known-bad commands first.
// Returns empty string if allowed, error message if blocked.
func tirithGuard(ctx context.Context, command, cwd string, cfg TirithConfig) string {
	if !cfg.Enabled {
		return ""
	}

	binPath, cacheKey, err := resolveTirithPath(cfg.BinPath)
	if err != nil {
		return tirithScannerFailure(cacheKey, cfg, err)
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = tirithDefaultTimeout
	}

	policyRoot, cleanupPolicy, err := createTirithLocalOnlyPolicyRoot()
	if err != nil {
		return tirithScannerFailure(cacheKey, cfg, err)
	}
	defer cleanupPolicy()

	parentCtx := ctx
	ctx, cancel := context.WithTimeout(parentCtx, time.Duration(timeout)*time.Second)
	defer cancel()

	shellFlag := "posix"
	if runtime.GOOS == "windows" {
		shellFlag = "powershell"
	}

	cmd := exec.Command(binPath, "check", "--format", "json", "--non-interactive",
		"--no-daemon", "--shell", shellFlag, "--", command)
	if cwd != "" {
		cmd.Dir = cwd
	}
	cmd.Env = tirithSubprocessEnv(policyRoot)
	var stdout cappedBuffer
	var stderr cappedBuffer
	stdout.limit = tirithStdoutLimitBytes
	stderr.limit = tirithStderrLimitBytes
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = runTirithCommand(ctx, cmd)
	if err != nil {
		if parentCtx.Err() != nil {
			return "Tirith security scan canceled"
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return tirithScannerFailure(cacheKey, cfg, fmt.Errorf("timed out after %ds", timeout))
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			return "Tirith security scan canceled"
		}

		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return tirithHandleExit(exitErr.ExitCode(), stdout.Bytes(), cfg)
		}

		if stderr.Len() > 0 {
			err = fmt.Errorf("%w: %s", err, tirithSanitizeText(stderr.String()))
		}
		return tirithScannerFailure(cacheKey, cfg, err)
	}

	return tirithHandleExit(0, stdout.Bytes(), cfg)
}

func runTirithCommand(ctx context.Context, cmd *exec.Cmd) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	prepareCommandForTermination(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		_ = terminateProcessTree(cmd)
		err := <-done
		if err != nil {
			return err
		}
		return ctx.Err()
	}
}

func createTirithLocalOnlyPolicyRoot() (string, func(), error) {
	root, err := os.MkdirTemp("", "picoclaw-tirith-policy-*")
	if err != nil {
		return "", func() {}, fmt.Errorf("create local-only Tirith policy root: %w", err)
	}

	cleanup := func() {
		_ = os.RemoveAll(root)
	}

	policyDir := filepath.Join(root, ".tirith")
	if err := os.MkdirAll(policyDir, 0o700); err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("create local-only Tirith policy directory: %w", err)
	}

	policyPath := filepath.Join(policyDir, "policy.yaml")
	if err := os.WriteFile(policyPath, []byte(tirithLocalOnlyPolicy), 0o600); err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("write local-only Tirith policy: %w", err)
	}

	return root, cleanup, nil
}

func tirithSubprocessEnv(policyRoot string) []string {
	env := make([]string, 0, len(os.Environ())+1)
	for _, entry := range os.Environ() {
		name, _, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if tirithShouldScrubEnv(name) {
			continue
		}
		env = append(env, entry)
	}
	env = append(env, "TIRITH_POLICY_ROOT="+policyRoot)
	return env
}

func tirithShouldScrubEnv(name string) bool {
	_, ok := tirithScrubbedEnvVars[strings.ToUpper(name)]
	return ok
}

func tirithHandleExit(exitCode int, jsonOutput []byte, cfg TirithConfig) string {
	switch exitCode {
	case 0:
		return ""
	case 1:
		return fmt.Sprintf("Command blocked by Tirith security scan: %s", tirithSummarize(jsonOutput))
	case 2:
		log.Printf("[tirith] warning: %s", tirithSummarize(jsonOutput))
		return ""
	case 3:
		log.Printf(
			"[tirith] warning acknowledgement required; allowing in non-interactive mode: %s",
			tirithSummarize(jsonOutput),
		)
		return ""
	default:
		return tirithScannerFailure(
			normalizeTirithBin(cfg.BinPath),
			cfg,
			fmt.Errorf("unexpected exit code %d", exitCode),
		)
	}
}

func tirithScannerFailure(cacheKey string, cfg TirithConfig, err error) string {
	message := tirithSanitizeText(err.Error())
	if cfg.FailOpen {
		tirithLogOncef(cacheKey, "scanner-failure", "[tirith] unavailable (fail-open): %s", message)
		return ""
	}
	return fmt.Sprintf("Tirith security scan failed (fail-closed): %s", message)
}

func tirithLogOncef(cacheKey, category, format string, args ...any) {
	cacheKey = normalizeTirithBin(cacheKey)
	key := cacheKey + "\x00" + category + "\x00" + fmt.Sprintf(format, args...)

	tirithWarningCache.Lock()
	defer tirithWarningCache.Unlock()
	if _, ok := tirithWarningCache.seen[key]; ok {
		return
	}
	tirithWarningCache.seen[key] = struct{}{}
	log.Printf(format, args...)
}

func tirithSummarize(jsonOutput []byte) string {
	if len(jsonOutput) == 0 {
		return "security issue detected"
	}
	var data struct {
		Summary  string `json:"summary"`
		Findings []struct {
			Severity    string `json:"severity"`
			Title       string `json:"title"`
			Message     string `json:"message"`
			Description string `json:"description"`
		} `json:"findings"`
	}
	if err := json.Unmarshal(jsonOutput, &data); err != nil {
		return "security issue detected (details unavailable)"
	}

	if summary := tirithTruncateRunes(tirithSanitizeText(data.Summary), tirithMaxSummaryRunes); summary != "" {
		return summary
	}

	limit := len(data.Findings)
	if limit > tirithMaxFindings {
		limit = tirithMaxFindings
	}
	parts := make([]string, 0, limit)
	for _, f := range data.Findings[:limit] {
		title := f.Title
		if title == "" {
			title = f.Message
		}
		if title == "" {
			title = f.Description
		}
		title = tirithSanitizeText(title)
		if title == "" {
			continue
		}
		severity := tirithSanitizeText(f.Severity)
		if severity == "" {
			parts = append(parts, title)
		} else {
			parts = append(parts, fmt.Sprintf("[%s] %s", severity, title))
		}
	}
	if len(parts) == 0 {
		return "security issue detected"
	}
	if len(data.Findings) > limit {
		return tirithJoinFindingSummary(parts, len(data.Findings)-limit)
	}
	return tirithTruncateRunes(strings.Join(parts, "; "), tirithMaxSummaryRunes)
}

func tirithJoinFindingSummary(parts []string, more int) string {
	summary := strings.Join(parts, "; ")
	if more <= 0 {
		return tirithTruncateRunes(summary, tirithMaxSummaryRunes)
	}

	suffix := fmt.Sprintf("; ...and %d more", more)
	summaryRunes := []rune(summary)
	suffixRunes := []rune(suffix)
	if len(summaryRunes)+len(suffixRunes) <= tirithMaxSummaryRunes {
		return summary + suffix
	}
	prefixLimit := tirithMaxSummaryRunes - len(suffixRunes)
	if prefixLimit <= 0 {
		return tirithTruncateRunes(strings.TrimPrefix(suffix, "; "), tirithMaxSummaryRunes)
	}
	prefix := strings.TrimSpace(string(summaryRunes[:prefixLimit]))
	prefix = strings.TrimSuffix(prefix, ";")
	if prefix == "" {
		return strings.TrimPrefix(suffix, "; ")
	}
	return prefix + suffix
}

// resolveTirithPath resolves only a user-installed Tirith binary. It deliberately
// does not cache failed lookups so installing Tirith after a miss works immediately.
func resolveTirithPath(configured string) (string, string, error) {
	normalized := normalizeTirithBin(configured)
	if isExplicitTirithPath(normalized) {
		if err := validateTirithPath(normalized); err != nil {
			return "", normalized, err
		}
		return normalized, normalized, nil
	}

	path, err := exec.LookPath(normalized)
	if err != nil {
		return "", normalized, err
	}
	if err := validateTirithPath(path); err != nil {
		return "", normalized, err
	}
	return path, normalized, nil
}

func normalizeTirithBin(configured string) string {
	bin := strings.TrimSpace(configured)
	if bin == "" {
		bin = tirithDefaultBin
	}
	if bin == "~" || strings.HasPrefix(bin, "~/") || strings.HasPrefix(bin, `~\`) {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			if bin == "~" {
				bin = home
			} else {
				bin = filepath.Join(home, bin[2:])
			}
		}
	}
	if isExplicitTirithPath(bin) {
		cleaned := filepath.Clean(bin)
		if filepath.IsAbs(cleaned) {
			return cleaned
		}
		if abs, err := filepath.Abs(cleaned); err == nil {
			return abs
		}
		return cleaned
	}
	return bin
}

func isExplicitTirithPath(path string) bool {
	return filepath.IsAbs(path) ||
		strings.Contains(path, "/") ||
		strings.Contains(path, `\`)
}

func validateTirithPath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory", path)
	}
	if runtime.GOOS != "windows" && info.Mode()&0o111 == 0 {
		return fmt.Errorf("%s is not executable", path)
	}
	return nil
}

type cappedBuffer struct {
	limit int
	buf   bytes.Buffer
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	written := len(p)
	if b.limit <= 0 || b.buf.Len() >= b.limit {
		return written, nil
	}
	remaining := b.limit - b.buf.Len()
	if len(p) > remaining {
		p = p[:remaining]
	}
	_, _ = b.buf.Write(p)
	return written, nil
}

func (b *cappedBuffer) Bytes() []byte {
	return b.buf.Bytes()
}

func (b *cappedBuffer) String() string {
	return b.buf.String()
}

func (b *cappedBuffer) Len() int {
	return b.buf.Len()
}

func tirithSanitizeText(input string) string {
	input = tirithStripANSI(input)
	var b strings.Builder
	b.Grow(len(input))
	for _, r := range input {
		if tirithDropRune(r) {
			continue
		}
		b.WriteRune(r)
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func tirithDropRune(r rune) bool {
	if unicode.IsControl(r) {
		return true
	}
	switch {
	case r >= '\u202a' && r <= '\u202e':
		return true
	case r >= '\u2066' && r <= '\u2069':
		return true
	case r >= '\u200b' && r <= '\u200f':
		return true
	case r == '\u2060' || r == '\ufeff':
		return true
	case r >= '\ufe00' && r <= '\ufe0f':
		return true
	case r >= '\U000e0100' && r <= '\U000e01ef':
		return true
	case r >= '\U000e0000' && r <= '\U000e007f':
		return true
	default:
		return false
	}
}

func tirithStripANSI(input string) string {
	var b strings.Builder
	for i := 0; i < len(input); {
		r, size := utf8.DecodeRuneInString(input[i:])
		if r != '\x1b' {
			b.WriteString(input[i : i+size])
			i += size
			continue
		}

		i += size
		if i >= len(input) {
			continue
		}
		switch input[i] {
		case '[':
			i++
			for i < len(input) {
				c := input[i]
				i++
				if c >= 0x40 && c <= 0x7e {
					break
				}
			}
		case ']':
			i++
			for i < len(input) {
				if input[i] == 0x07 {
					i++
					break
				}
				if input[i] == '\x1b' && i+1 < len(input) && input[i+1] == '\\' {
					i += 2
					break
				}
				i++
			}
		default:
			i++
		}
	}
	return b.String()
}

func tirithTruncateRunes(s string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	return string(runes[:limit]) + "..."
}
