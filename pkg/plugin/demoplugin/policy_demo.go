package demoplugin

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/hooks"
	"github.com/sipeed/picoclaw/pkg/plugin"
)

// PolicyDemoConfig controls the demo plugin behavior.
type PolicyDemoConfig struct {
	BlockedTools         []string
	RedactPrefixes       []string
	ChannelToolAllowlist map[string][]string
	DenyOutboundPatterns []string
	MaxToolTimeoutSecond int
}

// PolicyDemoStats provides basic evidence that hook paths were executed.
type PolicyDemoStats struct {
	BeforeToolCalls   int
	BlockedToolCalls  int
	MessageSends      int
	RedactedMessages  int
	BlockedMessages   int
	SessionStarts     int
	SessionEnds       int
	AfterToolCalls    int
	TotalToolDuration time.Duration
}

// PolicyDemoPlugin demonstrates why plugins are needed: it enforces runtime policy
// at tool-call and outbound-message lifecycle points and collects audit metrics.
type PolicyDemoPlugin struct {
	blockedTools     map[string]struct{}
	prefixes         []string
	channelAllowlist map[string]map[string]struct{}
	denyPatterns     []string
	maxTimeout       int

	mu    sync.Mutex
	stats PolicyDemoStats
}

func NewPolicyDemoPlugin(cfg PolicyDemoConfig) *PolicyDemoPlugin {
	blocked := make(map[string]struct{}, len(cfg.BlockedTools))
	for _, t := range cfg.BlockedTools {
		t = normalizeLower(t)
		if t == "" {
			continue
		}
		blocked[t] = struct{}{}
	}

	prefixes := make([]string, 0, len(cfg.RedactPrefixes))
	for _, p := range cfg.RedactPrefixes {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		prefixes = append(prefixes, p)
	}

	allowlist := make(map[string]map[string]struct{}, len(cfg.ChannelToolAllowlist))
	for channel, tools := range cfg.ChannelToolAllowlist {
		channel = normalizeLower(channel)
		if channel == "" {
			continue
		}
		toolSet := make(map[string]struct{}, len(tools))
		for _, t := range tools {
			t = normalizeLower(t)
			if t == "" {
				continue
			}
			toolSet[t] = struct{}{}
		}
		allowlist[channel] = toolSet
	}

	patterns := make([]string, 0, len(cfg.DenyOutboundPatterns))
	for _, p := range cfg.DenyOutboundPatterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		patterns = append(patterns, p)
	}

	maxTimeout := cfg.MaxToolTimeoutSecond
	if maxTimeout < 0 {
		maxTimeout = 0
	}

	return &PolicyDemoPlugin{
		blockedTools:     blocked,
		prefixes:         prefixes,
		channelAllowlist: allowlist,
		denyPatterns:     patterns,
		maxTimeout:       maxTimeout,
	}
}

func (p *PolicyDemoPlugin) Name() string {
	return "policy-demo"
}

func (p *PolicyDemoPlugin) APIVersion() string {
	return plugin.APIVersion
}

func (p *PolicyDemoPlugin) Snapshot() PolicyDemoStats {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stats
}

func (p *PolicyDemoPlugin) Register(r *hooks.HookRegistry) error {
	r.OnBeforeToolCall("policy-demo-tool-policy", 100, func(_ context.Context, e *hooks.BeforeToolCallEvent) error {
		tool := normalizeLower(e.ToolName)
		p.incBeforeToolCalls()

		if _, blocked := p.blockedTools[tool]; blocked {
			e.Cancel = true
			e.CancelReason = "blocked by policy-demo plugin"
			p.incBlockedToolCalls()
			return nil
		}

		channel := normalizeLower(e.Channel)
		if allow, ok := p.channelAllowlist[channel]; ok {
			if _, allowed := allow[tool]; !allowed {
				e.Cancel = true
				e.CancelReason = fmt.Sprintf("tool %q is not allowed on channel %q", e.ToolName, e.Channel)
				p.incBlockedToolCalls()
				return nil
			}
		}

		if p.maxTimeout > 0 {
			clampArgNumber(e.Args, "timeout", p.maxTimeout)
			clampArgNumber(e.Args, "timeout_seconds", p.maxTimeout)
		}
		return nil
	})

	r.OnMessageSending("policy-demo-redact-and-guard", 50, func(_ context.Context, e *hooks.MessageSendingEvent) error {
		p.incMessageSends()

		for _, pattern := range p.denyPatterns {
			if strings.Contains(e.Content, pattern) {
				e.Cancel = true
				e.CancelReason = "blocked by policy-demo outbound guard"
				p.incBlockedMessages()
				return nil
			}
		}

		content := e.Content
		redacted := false
		for _, prefix := range p.prefixes {
			next := strings.ReplaceAll(content, prefix, "[redacted]-")
			if next != content {
				redacted = true
			}
			content = next
		}
		e.Content = content
		if redacted {
			p.incRedactedMessages()
		}
		return nil
	})

	r.OnSessionStart("policy-demo-session-start-audit", 0, func(_ context.Context, _ *hooks.SessionEvent) error {
		p.incSessionStarts()
		return nil
	})

	r.OnSessionEnd("policy-demo-session-end-audit", 0, func(_ context.Context, _ *hooks.SessionEvent) error {
		p.incSessionEnds()
		return nil
	})

	r.OnAfterToolCall("policy-demo-after-tool-audit", 0, func(_ context.Context, e *hooks.AfterToolCallEvent) error {
		p.incAfterToolCall(e.Duration)
		return nil
	})

	return nil
}

func normalizeLower(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func clampArgNumber(args map[string]any, key string, max int) {
	if args == nil || max <= 0 {
		return
	}
	v, ok := args[key]
	if !ok {
		return
	}
	n, ok := toInt(v)
	if !ok {
		return
	}
	if n > max {
		args[key] = max
	}
}

func toInt(v any) (int, bool) {
	maxInt := int(^uint(0) >> 1)
	maxIntU64 := uint64(maxInt)

	switch n := v.(type) {
	case int:
		return n, true
	case int8:
		return int(n), true
	case int16:
		return int(n), true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case uint:
		if uint64(n) > maxIntU64 {
			return 0, false
		}
		return int(n), true
	case uint8:
		return int(n), true
	case uint16:
		return int(n), true
	case uint32:
		if uint64(n) > maxIntU64 {
			return 0, false
		}
		return int(n), true
	case uint64:
		if n > maxIntU64 {
			return 0, false
		}
		return int(n), true
	case float32:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}

func (p *PolicyDemoPlugin) incBeforeToolCalls() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stats.BeforeToolCalls++
}

func (p *PolicyDemoPlugin) incBlockedToolCalls() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stats.BlockedToolCalls++
}

func (p *PolicyDemoPlugin) incMessageSends() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stats.MessageSends++
}

func (p *PolicyDemoPlugin) incRedactedMessages() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stats.RedactedMessages++
}

func (p *PolicyDemoPlugin) incBlockedMessages() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stats.BlockedMessages++
}

func (p *PolicyDemoPlugin) incSessionStarts() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stats.SessionStarts++
}

func (p *PolicyDemoPlugin) incSessionEnds() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stats.SessionEnds++
}

func (p *PolicyDemoPlugin) incAfterToolCall(d time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stats.AfterToolCalls++
	p.stats.TotalToolDuration += d
}
