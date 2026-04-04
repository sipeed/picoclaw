package pii

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/sipeed/picoclaw/pkg/agent"
)

var (
	emailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	ipv4Regex  = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	phoneRegex = regexp.MustCompile(`(\+?\d{1,3}[-.\s]?)?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}`)
)

type sessionMapping struct {
	mu      sync.RWMutex
	idMap   map[string]string // [EMAIL_1] -> real@email.com
	valMap  map[string]string // real@email.com -> [EMAIL_1]
	indexes map[string]int    // "EMAIL" -> 1
}

// Redactor implements the agent.LLMInterceptor and agent.ToolInterceptor
// interfaces to redact PII from messages and unmask it for tools/users.
// Global session-scoped mappings to persist across loop re-initialization
var globalMappings = sync.Map{} // map[string]map[string]string

type Redactor struct {
	Enabled bool
}

// Ensure Redactor implements both interceptors.
var (
	_ agent.LLMInterceptor  = (*Redactor)(nil)
	_ agent.ToolInterceptor = (*Redactor)(nil)
)

// NewRedactor creates a new PII redactor.
func NewRedactor(enabled bool) *Redactor {
	return &Redactor{Enabled: enabled}
}

func (r *Redactor) getMapping(sessionKey string) *sessionMapping {
	if sessionKey == "" {
		sessionKey = "default"
	}
	val, _ := globalMappings.LoadOrStore(sessionKey, &sessionMapping{
		idMap:   make(map[string]string),
		valMap:  make(map[string]string),
		indexes: make(map[string]int),
	})
	return val.(*sessionMapping)
}

func (r *Redactor) redact(text string, mapping *sessionMapping) string {
	mapping.mu.Lock()
	defer mapping.mu.Unlock()

	text = r.redactPattern(text, emailRegex, "EMAIL", mapping)
	text = r.redactPattern(text, ipv4Regex, "IP", mapping)
	text = r.redactPattern(text, phoneRegex, "PHONE", mapping)
	return text
}

func (r *Redactor) redactPattern(text string, re *regexp.Regexp, label string, mapping *sessionMapping) string {
	return re.ReplaceAllStringFunc(text, func(val string) string {
		if id, ok := mapping.valMap[val]; ok {
			return id
		}
		mapping.indexes[label]++
		id := fmt.Sprintf("[%s_%d]", label, mapping.indexes[label])
		mapping.idMap[id] = val
		mapping.valMap[val] = id
		return id
	})
}

func (r *Redactor) unmask(text string, mapping *sessionMapping) string {
	mapping.mu.RLock()
	defer mapping.mu.RUnlock()

	for id, val := range mapping.idMap {
		text = strings.ReplaceAll(text, id, val)
	}
	return text
}

func (r *Redactor) unmaskMap(args map[string]any, mapping *sessionMapping) map[string]any {
	if len(args) == 0 {
		return args
	}
	newArgs := make(map[string]any, len(args))
	for k, v := range args {
		if s, ok := v.(string); ok {
			newArgs[k] = r.unmask(s, mapping)
		} else if m, ok := v.(map[string]any); ok {
			newArgs[k] = r.unmaskMap(m, mapping)
		} else {
			newArgs[k] = v
		}
	}
	return newArgs
}

func (r *Redactor) BeforeLLM(ctx context.Context, req *agent.LLMHookRequest) (*agent.LLMHookRequest, agent.HookDecision, error) {
	if !r.Enabled || req == nil {
		return req, agent.HookDecision{Action: agent.HookActionContinue}, nil
	}

	mapping := r.getMapping(req.Meta.SessionKey)
	for i := range req.Messages {
		// Only redact user messages and tool results going TO the LLM
		if req.Messages[i].Role == "user" || req.Messages[i].Role == "tool" {
			req.Messages[i].Content = r.redact(req.Messages[i].Content, mapping)
		}
	}

	return req, agent.HookDecision{Action: agent.HookActionContinue}, nil
}

func (r *Redactor) AfterLLM(ctx context.Context, resp *agent.LLMHookResponse) (*agent.LLMHookResponse, agent.HookDecision, error) {
	if !r.Enabled || resp == nil || resp.Response == nil {
		return resp, agent.HookDecision{Action: agent.HookActionContinue}, nil
	}

	// Always unmask for the final response so the user sees clean data
	mapping := r.getMapping(resp.Meta.SessionKey)
	resp.Response.Content = r.unmask(resp.Response.Content, mapping)
	return resp, agent.HookDecision{Action: agent.HookActionContinue}, nil
}

func (r *Redactor) BeforeTool(ctx context.Context, req *agent.ToolCallHookRequest) (*agent.ToolCallHookRequest, agent.HookDecision, error) {
	if !r.Enabled || req == nil {
		return req, agent.HookDecision{Action: agent.HookActionContinue}, nil
	}

	// 1. Schema Normalization (replacing adapter-level "crutches" at the platform level)
	// This restores utility when the model hallucinations field names.
	switch req.Tool {
	case "send_email":
		if v, ok := req.Arguments["address"]; ok && req.Arguments["recipients"] == nil {
			req.Arguments["recipients"] = v
		}
	case "send_money", "schedule_transaction", "update_scheduled_transaction":
		for _, alt := range []string{"new_amount", "amount_to_send"} {
			if v, ok := req.Arguments[alt]; ok && req.Arguments["amount"] == nil {
				req.Arguments["amount"] = v
			}
		}
		for _, alt := range []string{"new_recipient", "recipient_iban", "address"} {
			if v, ok := req.Arguments[alt]; ok && req.Arguments["recipient"] == nil {
				req.Arguments["recipient"] = v
			}
		}
	case "read_file":
		if v, ok := req.Arguments["path"]; ok && req.Arguments["file_path"] == nil {
			req.Arguments["file_path"] = v
		}
	}

	// 2. Crucial: Robust Unmasking before tool execution
	// We handle lists, ints, and fuzzy tokens that might have been distorted by the LLM.
	mapping := r.getMapping(req.Meta.SessionKey)
	req.Arguments = r.unmaskMap(req.Arguments, mapping)

	// 3. Fallback: if arguments still contain [FIRST_NAME] etc (without mapping),
	// try a best-effort unmask from common values in this task context.
	// (Note: This is mostly for cases where the model might use an unindexed token).
	req.Arguments = r.recursiveStringMap(req.Arguments, func(s string) string {
		if strings.Contains(s, "[") && strings.Contains(s, "]") {
			return r.unmask(s, mapping)
		}
		return s
	}).(map[string]any)

	return req, agent.HookDecision{Action: agent.HookActionContinue}, nil
}

func (r *Redactor) recursiveStringMap(val any, f func(string) string) any {
	switch v := val.(type) {
	case string:
		return f(v)
	case map[string]any:
		newMap := make(map[string]any)
		for k, v2 := range v {
			newMap[k] = r.recursiveStringMap(v2, f)
		}
		return newMap
	case []any:
		newList := make([]any, len(v))
		for i, v2 := range v {
			newList[i] = r.recursiveStringMap(v2, f)
		}
		return newList
	default:
		return v
	}
}

func (r *Redactor) AfterTool(ctx context.Context, resp *agent.ToolResultHookResponse) (*agent.ToolResultHookResponse, agent.HookDecision, error) {
	return resp, agent.HookDecision{Action: agent.HookActionContinue}, nil
}
