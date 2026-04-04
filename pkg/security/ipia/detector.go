package ipia

import (
	"context"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/agent"
)

var injectionPatterns = []string{
	"ignore previous instructions",
	"ignore all previous instructions",
	"ignore the above instructions",
	"system prompt:",
	"you are now an admin",
	"new mission:",
	"forget your safety guidelines",
	"stay in character as",
	"dan mode",
}

// Detector implements the agent.ToolInterceptor interface to detect indirect prompt injection.
type Detector struct {
	Enabled bool
}

// Ensure Detector implements ToolInterceptor.
var _ agent.ToolInterceptor = (*Detector)(nil)

// NewDetector creates a new IPIA detector.
func NewDetector(enabled bool) *Detector {
	return &Detector{Enabled: enabled}
}

func (d *Detector) scan(text string) (bool, string) {
	lower := strings.ToLower(text)
	for _, pattern := range injectionPatterns {
		if strings.Contains(lower, pattern) {
			return true, pattern
		}
	}
	return false, ""
}

func (d *Detector) BeforeTool(ctx context.Context, call *agent.ToolCallHookRequest) (*agent.ToolCallHookRequest, agent.HookDecision, error) {
	return call, agent.HookDecision{Action: agent.HookActionContinue}, nil
}

func (d *Detector) AfterTool(ctx context.Context, resp *agent.ToolResultHookResponse) (*agent.ToolResultHookResponse, agent.HookDecision, error) {
	if !d.Enabled || resp == nil || resp.Result == nil {
		return resp, agent.HookDecision{Action: agent.HookActionContinue}, nil
	}

	if found, pattern := d.scan(resp.Result.ForLLM); found {
		return resp, agent.HookDecision{
			Action: agent.HookActionAbortTurn,
			Reason: fmt.Sprintf("Indirect prompt injection detected in tool output (pattern: %q)", pattern),
		}, nil
	}

	if found, pattern := d.scan(resp.Result.ForUser); found {
		return resp, agent.HookDecision{
			Action: agent.HookActionAbortTurn,
			Reason: fmt.Sprintf("Indirect prompt injection detected in tool output (pattern: %q)", pattern),
		}, nil
	}

	return resp, agent.HookDecision{Action: agent.HookActionContinue}, nil
}
