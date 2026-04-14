package policy

import (
	"context"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/agent"
)

// Config defines the security policy for tool execution.
type Config struct {
	// RequiresApproval maps a tool name to a boolean.
	// If true, the tool will always return Approved=false with a "requires human approval" reason.
	RequiresApproval map[string]bool `json:"requires_approval"`

	// DisallowedTools maps a tool name to a boolean.
	// If true, the tool will be rejected without any human-in-the-loop option.
	DisallowedTools map[string]bool `json:"disallowed_tools"`

	// AllowedTools maps a tool name to a boolean.
	// If set (non-empty), only tools in this map are allowed.
	AllowedTools map[string]bool `json:"allowed_tools"`
}

// Checker implements the agent.ToolApprover interface.
type Checker struct {
	Config Config
}

// Ensure Checker implements ToolApprover.
var _ agent.ToolApprover = (*Checker)(nil)

// NewChecker creates a new policy checker.
func NewChecker(cfg Config) *Checker {
	return &Checker{Config: cfg}
}

func (c *Checker) ApproveTool(ctx context.Context, req *agent.ToolApprovalRequest) (agent.ApprovalDecision, error) {
	if req == nil {
		return agent.ApprovalDecision{Approved: false, Reason: "request is nil"}, nil
	}

	// 1. Explicit Disallow
	if c.Config.DisallowedTools[req.Tool] {
		return agent.ApprovalDecision{
			Approved: false,
			Reason:   fmt.Sprintf("Tool %q is globally disallowed by security policy", req.Tool),
		}, nil
	}

	// 2. Whitelisting (if enabled)
	if len(c.Config.AllowedTools) > 0 {
		allowed := false
		if c.Config.AllowedTools[req.Tool] {
			allowed = true
		} else {
			// Check for prefix matches (e.g. "github" matches "mcp_github_...")
			// Match logic consistent with ToolRegistry.Filter
			for w, ok := range c.Config.AllowedTools {
				if !ok {
					continue
				}
				if strings.HasPrefix(req.Tool, "mcp_"+w+"_") ||
					strings.HasPrefix(req.Tool, "tool_"+w+"_") ||
					strings.HasPrefix(req.Tool, w+"_") {
					allowed = true
					break
				}
			}
		}

		if !allowed {
			return agent.ApprovalDecision{
				Approved: false,
				Reason:   fmt.Sprintf("Tool %q is not in the allowed tools whitelist", req.Tool),
			}, nil
		}
	}

	// 3. Human Approval Required
	if c.Config.RequiresApproval[req.Tool] {
		return agent.ApprovalDecision{
			Approved: false,
			Reason:   fmt.Sprintf("Tool %q requires explicit human approval", req.Tool),
		}, nil
	}

	return agent.ApprovalDecision{Approved: true}, nil
}
