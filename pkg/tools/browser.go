//go:build cdp

package tools

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
)

// pageVisit records a single page navigation for browsing history.
type pageVisit struct {
	URL       string
	Title     string
	Timestamp time.Time
}

const maxHistorySize = 50 // keep last 50 pages to bound memory

// BrowserTool provides browser automation capabilities via Chrome DevTools Protocol.
// It enables AI agents to navigate web pages, interact with elements, and extract data.
//
// Lock ordering: cdpMu must always be acquired before stateMu to avoid deadlock.
type BrowserTool struct {
	cfg        config.BrowserToolConfig
	cdp        *CDPClient
	cdpMu      sync.Mutex // guards lazy cdp connection and cdp pointer
	stateMu    sync.Mutex // guards mutable session state: history, mediaStore, tempFiles
	chromePath string
	stealthJS  string
	mediaStore media.MediaStore
	history    []pageVisit // browsing history, most recent last
	tempFiles  []string    // temp files to clean up on close
}

// NewBrowserTool creates a new BrowserTool. It verifies that Chrome is available
// on the system. The actual CDP connection is deferred to the first Execute call
// (lazy connect via connectIfNeeded). Returns an error if Chrome is not found.
func NewBrowserTool(cfg config.BrowserToolConfig) (*BrowserTool, error) {
	chromePath, err := FindChromePath()
	if err != nil {
		return nil, fmt.Errorf(
			"Chrome/Chromium required for browser tool but not found. "+
				"Install Chrome and restart, or set CHROME_PATH env var. "+
				"Error: %w", err)
	}

	logger.InfoCF("tool", "Chrome found for browser tool",
		map[string]any{"path": chromePath})

	var stealthJS string
	if cfg.Stealth {
		stealthJS = generateStealthJS()
	}

	return &BrowserTool{
		cfg:        cfg,
		chromePath: chromePath,
		stealthJS:  stealthJS,
	}, nil
}

// connectIfNeeded lazily connects to the CDP endpoint.
// It is safe for concurrent use.
func (t *BrowserTool) connectIfNeeded() error {
	t.cdpMu.Lock()
	defer t.cdpMu.Unlock()

	if t.cdp != nil {
		return nil
	}

	endpoint := t.cfg.CDPEndpoint
	if endpoint == "" {
		endpoint = "http://127.0.0.1:9222"
	}

	cdp, err := NewCDPClient(endpoint)
	if err != nil {
		return fmt.Errorf(
			"failed to connect to Chrome CDP at %s. "+
				"Make sure Chrome is running with: %s --remote-debugging-port=9222. "+
				"Error: %w", endpoint, t.chromePath, err)
	}

	// Enable required CDP domains
	if err := cdp.EnablePage(); err != nil {
		cdp.Close()
		return fmt.Errorf("failed to enable Page domain: %w", err)
	}
	if err := cdp.EnableRuntime(); err != nil {
		cdp.Close()
		return fmt.Errorf("failed to enable Runtime domain: %w", err)
	}
	if err := cdp.EnableDOM(); err != nil {
		cdp.Close()
		return fmt.Errorf("failed to enable DOM domain: %w", err)
	}

	// Inject stealth JS if configured
	if t.stealthJS != "" {
		if err := cdp.InjectScript(context.Background(), t.stealthJS); err != nil {
			logger.WarnCF("tool", "Failed to inject stealth JS",
				map[string]any{"error": err.Error()})
		}
	}

	t.cdp = cdp
	return nil
}

func (t *BrowserTool) Name() string { return "browser" }

func (t *BrowserTool) Description() string {
	return `Browser automation via CDP. Actions: navigate, state, click, type, fill, select, screenshot, get_text, scroll, keys, evaluate, close.
Workflow: navigate url → state (get [N] indices) → click/type [N] → state (verify). Always run state after navigate or click.`
}

func (t *BrowserTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"description": "Browser action to perform",
				"enum":        []string{"navigate", "state", "click", "type", "fill", "select", "screenshot", "get_text", "scroll", "keys", "evaluate", "close"},
			},
			"url": map[string]any{
				"type":        "string",
				"description": "URL for navigate action",
			},
			"index": map[string]any{
				"type":        "integer",
				"description": "Element index [N] from state output for click/type/fill/select/get_text",
			},
			"text": map[string]any{
				"type":        "string",
				"description": "Text for type/fill actions, option value for select, key name for keys",
			},
			"direction": map[string]any{
				"type":        "string",
				"description": "Scroll direction: up or down",
				"enum":        []string{"up", "down"},
			},
			"code": map[string]any{
				"type":        "string",
				"description": "JavaScript code for evaluate action (requires allow_evaluate=true in config)",
			},
		},
		"required": []string{"action"},
	}
}

func (t *BrowserTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	action, _ := args["action"].(string)
	if action == "" {
		return ErrorResult("action is required")
	}

	// Apply configured timeout before any work so connectIfNeeded also respects it
	timeout := time.Duration(t.cfg.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Connect lazily on first use
	if action != "close" {
		if err := t.connectIfNeeded(); err != nil {
			return ErrorResult(err.Error())
		}
	}

	switch action {
	case "navigate":
		return t.executeNavigate(ctx, args)
	case "state":
		return t.executeState(ctx)
	case "click":
		return t.executeClick(ctx, args)
	case "type":
		return t.executeType(ctx, args)
	case "fill":
		return t.executeFill(ctx, args)
	case "select":
		return t.executeSelect(ctx, args)
	case "screenshot":
		return t.executeScreenshot(ctx)
	case "get_text":
		return t.executeGetText(ctx, args)
	case "scroll":
		return t.executeScroll(ctx, args)
	case "keys":
		return t.executeKeys(ctx, args)
	case "evaluate":
		return t.executeEvaluate(ctx, args)
	case "close":
		return t.executeClose()
	default:
		return ErrorResult(fmt.Sprintf("unknown browser action: %s", action))
	}
}

// SetMediaStore implements the mediaStoreAware interface.
func (t *BrowserTool) SetMediaStore(store media.MediaStore) {
	t.stateMu.Lock()
	defer t.stateMu.Unlock()
	t.mediaStore = store
}

// trackTempFile records a temp file for cleanup on close.
func (t *BrowserTool) trackTempFile(path string) {
	t.stateMu.Lock()
	defer t.stateMu.Unlock()
	t.tempFiles = append(t.tempFiles, path)
}

func (t *BrowserTool) executeClose() *ToolResult {
	t.cdpMu.Lock()
	defer t.cdpMu.Unlock()
	t.stateMu.Lock()
	defer t.stateMu.Unlock()

	if t.cdp != nil {
		t.cdp.Close()
		t.cdp = nil
	}
	t.history = nil

	// Clean up temp files created by screenshots without MediaStore
	for _, f := range t.tempFiles {
		os.Remove(f)
	}
	t.tempFiles = nil

	return SilentResult("Browser session closed.")
}

// --- Browsing history helpers ---

// recordVisit adds a page to the browsing history.
func (t *BrowserTool) recordVisit(pageURL, title string) {
	t.stateMu.Lock()
	defer t.stateMu.Unlock()

	// Avoid duplicate consecutive entries (e.g. navigate then state on same page)
	if len(t.history) > 0 {
		last := &t.history[len(t.history)-1]
		if last.URL == pageURL {
			if title != "" {
				last.Title = title
			}
			return
		}
	}

	t.history = append(t.history, pageVisit{
		URL:       pageURL,
		Title:     title,
		Timestamp: time.Now(),
	})

	// Trim to max size
	if len(t.history) > maxHistorySize {
		t.history = t.history[len(t.history)-maxHistorySize:]
	}
}

// updateCurrentTitle updates the title of the most recent history entry.
// Useful when a SPA changes title after initial navigation.
func (t *BrowserTool) updateCurrentTitle(title string) {
	t.stateMu.Lock()
	defer t.stateMu.Unlock()

	if len(t.history) > 0 && title != "" {
		t.history[len(t.history)-1].Title = title
	}
}

// historySummary returns a compact browsing history for LLM context.
// Only shown when there are 2+ pages visited (no point showing history for the first page).
func (t *BrowserTool) historySummary() string {
	t.stateMu.Lock()
	defer t.stateMu.Unlock()

	if len(t.history) < 2 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Browsing history (%d pages):\n", len(t.history)))

	// Show all entries compactly: just index, title, URL
	for i, v := range t.history {
		marker := "  "
		if i == len(t.history)-1 {
			marker = "> " // current page marker
		}
		title := v.Title
		if title == "" {
			title = "(untitled)"
		}
		sb.WriteString(fmt.Sprintf("%s%d. %s — %s\n", marker, i+1, title, v.URL))
	}

	return sb.String()
}

// --- Helpers ---

// validateBrowserURL checks that a URL is safe to navigate to.
// It blocks private networks, loopback, metadata endpoints, and non-HTTP schemes.
//
// Note on DNS rebinding: validateBrowserURL resolves DNS at check time, but Chrome
// navigates later. An attacker's DNS could return a public IP during validation and
// a private IP during connection. This is an inherent limitation of the CDP approach
// since we cannot intercept Chrome's actual network connections. For higher-security
// deployments, use network-level controls (e.g., firewall rules on the Chrome process).
func validateBrowserURL(urlStr string) error {
	if urlStr == "" {
		return fmt.Errorf("empty URL")
	}

	parsed, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("only http/https URLs are allowed (got %s://)", parsed.Scheme)
	}

	if parsed.Host == "" {
		return fmt.Errorf("missing host in URL")
	}

	hostname := parsed.Hostname()

	// Block obvious private/local hosts (string check)
	if hostname == "localhost" || hostname == "0.0.0.0" ||
		hostname == "metadata.google.internal" {
		return fmt.Errorf("navigation to %s is not allowed", hostname)
	}

	// Reject numeric-only hostnames that net.ParseIP doesn't handle but Chrome may
	// interpret as IPs (e.g., "0", "127.1", "2130706433", "0x7f000001").
	// These bypass net.ParseIP (returns nil) but can resolve to loopback/private IPs.
	if isNumericHost(hostname) {
		return fmt.Errorf("numeric hostname %q is not allowed (potential IP bypass)", hostname)
	}

	// Parse as IP to catch standard representations
	ip := net.ParseIP(hostname)
	if ip != nil {
		if err := checkPrivateIP(ip, hostname); err != nil {
			return err
		}
	} else {
		// Hostname (not literal IP) — resolve DNS with bounded timeout to check for private IPs
		resolver := &net.Resolver{}
		dnsCtx, dnsCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer dnsCancel()
		addrs, err := resolver.LookupHost(dnsCtx, hostname)
		if err == nil {
			for _, addr := range addrs {
				resolved := net.ParseIP(addr)
				if resolved == nil {
					continue
				}
				if err := checkPrivateIP(resolved, hostname); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// checkPrivateIP returns an error if the IP belongs to a private, loopback,
// link-local, unspecified, metadata, CGNAT, or benchmark address range.
func checkPrivateIP(ip net.IP, label string) error {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return fmt.Errorf("navigation to private/local IP %s is not allowed (resolved from %s)", ip, label)
	}

	// Block cloud metadata IPs
	if ip.Equal(net.ParseIP("169.254.169.254")) {
		return fmt.Errorf("navigation to cloud metadata endpoint is not allowed (resolved from %s)", label)
	}

	// Block CGNAT range (100.64.0.0/10) — may host internal services in cloud environments
	cgnat := &net.IPNet{
		IP:   net.ParseIP("100.64.0.0"),
		Mask: net.CIDRMask(10, 32),
	}
	if cgnat.Contains(ip) {
		return fmt.Errorf("navigation to CGNAT IP %s is not allowed (resolved from %s)", ip, label)
	}

	// Block benchmark/test range (198.18.0.0/15)
	benchmark := &net.IPNet{
		IP:   net.ParseIP("198.18.0.0"),
		Mask: net.CIDRMask(15, 32),
	}
	if benchmark.Contains(ip) {
		return fmt.Errorf("navigation to benchmark IP %s is not allowed (resolved from %s)", ip, label)
	}

	return nil
}

// isNumericHost returns true if the hostname is purely numeric, hex-prefixed,
// or uses dot-separated numeric/hex/octal segments that could be interpreted
// as an IP address in non-standard formats (e.g., "127.1", "0x7f000001", "2130706433").
func isNumericHost(host string) bool {
	if host == "" {
		return false
	}

	// Remove IPv6 brackets if present (already parsed by url.Hostname)
	host = strings.TrimPrefix(host, "[")
	host = strings.TrimSuffix(host, "]")

	// Check each segment separated by dots
	for _, seg := range strings.Split(host, ".") {
		if seg == "" {
			continue
		}
		// Hex prefix (0x...)
		if strings.HasPrefix(seg, "0x") || strings.HasPrefix(seg, "0X") {
			return true
		}
		// Pure digits (decimal or octal)
		allDigits := true
		for _, c := range seg {
			if c < '0' || c > '9' {
				allDigits = false
				break
			}
		}
		if !allDigits {
			return false
		}
	}
	return true
}

// getIntArg extracts a non-negative integer from args, handling both float64 (JSON) and int types.
// Rejects negative values and non-integer floats (e.g. 1.9).
func getIntArg(args map[string]any, key string) (int, bool) {
	v, ok := args[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		if n < 0 || n != float64(int(n)) {
			return 0, false
		}
		return int(n), true
	case int:
		if n < 0 {
			return 0, false
		}
		return n, true
	case int64:
		if n < 0 {
			return 0, false
		}
		return int(n), true
	}
	return 0, false
}
