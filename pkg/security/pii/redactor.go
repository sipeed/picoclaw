package pii

import (
	"context"
	"regexp"

	"github.com/sipeed/picoclaw/pkg/agent"
)

var (
	emailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	ipv4Regex  = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	phoneRegex = regexp.MustCompile(`(\+?\d{1,3}[-.\s]?)?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}`)
)

// Redactor implements the agent.LLMInterceptor interface to redact PII from messages.
type Redactor struct {
	Enabled bool
}

// Ensure Redactor implements LLMInterceptor.
var _ agent.LLMInterceptor = (*Redactor)(nil)

// NewRedactor creates a new PII redactor.
func NewRedactor(enabled bool) *Redactor {
	return &Redactor{Enabled: enabled}
}

func (r *Redactor) redact(text string) string {
	res := emailRegex.ReplaceAllString(text, "[EMAIL]")
	res = ipv4Regex.ReplaceAllString(res, "[IP]")
	res = phoneRegex.ReplaceAllString(res, "[PHONE]")
	return res
}

func (r *Redactor) BeforeLLM(ctx context.Context, req *agent.LLMHookRequest) (*agent.LLMHookRequest, agent.HookDecision, error) {
	if !r.Enabled || req == nil {
		return req, agent.HookDecision{Action: agent.HookActionContinue}, nil
	}

	for i := range req.Messages {
		if req.Messages[i].Role == "user" {
			req.Messages[i].Content = r.redact(req.Messages[i].Content)
		}
	}

	return req, agent.HookDecision{Action: agent.HookActionContinue}, nil
}

func (r *Redactor) AfterLLM(ctx context.Context, resp *agent.LLMHookResponse) (*agent.LLMHookResponse, agent.HookDecision, error) {
	if !r.Enabled || resp == nil || resp.Response == nil {
		return resp, agent.HookDecision{Action: agent.HookActionContinue}, nil
	}

	resp.Response.Content = r.redact(resp.Response.Content)
	return resp, agent.HookDecision{Action: agent.HookActionContinue}, nil
}
