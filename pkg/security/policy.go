package security

import (
	"context"
	"fmt"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

// PolicyMode represents the security enforcement mode.
type PolicyMode string

const (
	ModeOff     PolicyMode = "off"     // Security check disabled
	ModeBlock   PolicyMode = "block"   // Reject on violation
	ModeApprove PolicyMode = "approve" // Pause and request IM approval
)

// IsOff returns true when the mode means "no enforcement".
// Both the zero value ("") and the explicit "off" string are treated as off.
func (m PolicyMode) IsOff() bool {
	return m == "" || m == ModeOff
}

// Violation describes a security event detected by a guard.
type Violation struct {
	Category string // e.g. "exec_guard", "ssrf", "path_validation", "skill_validation"
	Tool     string // tool name that triggered the violation
	Action   string // the action that was attempted (command, URL, path, etc.)
	Reason   string // human-readable explanation
	RuleName string // name/pattern of the matched rule
}

// PolicyEngine centralises security policy decisions.
type PolicyEngine struct {
	config *config.SecurityConfig
	bus    *bus.MessageBus
}

// NewPolicyEngine creates a PolicyEngine from configuration and message bus.
func NewPolicyEngine(cfg *config.SecurityConfig, msgBus *bus.MessageBus) *PolicyEngine {
	return &PolicyEngine{
		config: cfg,
		bus:    msgBus,
	}
}

// GetMode returns the configured PolicyMode for a given security category.
func (pe *PolicyEngine) GetMode(category string) PolicyMode {
	var raw string
	switch category {
	case "exec_guard":
		raw = pe.config.ExecGuard
	case "ssrf":
		raw = pe.config.SSRFProtection
	case "path_validation":
		raw = pe.config.PathValidation
	case "skill_validation":
		raw = pe.config.SkillValidation
	default:
		return ModeOff
	}
	switch PolicyMode(raw) {
	case ModeBlock:
		return ModeBlock
	case ModeApprove:
		return ModeApprove
	default:
		return ModeOff
	}
}

// Evaluate checks a violation against the given mode and returns nil to allow
// or an error to deny. In "approve" mode it sends an IM approval request and
// blocks until the user responds or the timeout expires.
func (pe *PolicyEngine) Evaluate(ctx context.Context, mode PolicyMode, v Violation, channel, chatID string) error {
	switch {
	case mode.IsOff():
		return nil
	case mode == ModeBlock:
		return fmt.Errorf("blocked by security policy [%s]: %s", v.Category, v.Reason)
	case mode == ModeApprove:
		// CLI channel has no async IM listener; fall back to block
		if channel == "" || channel == "cli" {
			return fmt.Errorf("blocked by security policy [%s]: %s (approve mode unavailable in CLI)", v.Category, v.Reason)
		}
		return pe.requestApproval(ctx, v, channel, chatID)
	default:
		return nil
	}
}
