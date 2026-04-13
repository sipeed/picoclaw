// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"gopkg.in/yaml.v3"
)

const (
	DefaultPolicyTimeout = 5 * time.Second
	PolicyConfigFile     = ".policy.yml"
)

// PolicyResult represents the outcome of a policy evaluation
type PolicyResult struct {
	Allowed bool                   `json:"allowed"`
	Reason  string                 `json:"reason,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// Intent represents the identified user intent from LLM analysis
type Intent struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description,omitempty"`
	Confidence  float64                `json:"confidence,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ActionPlan represents the planned actions to fulfill an intent
type ActionPlan struct {
	Actions []Action `json:"actions"`
}

// Action represents a single action in the plan
type Action struct {
	Type      string                 `json:"type"`
	Tool      string                 `json:"tool,omitempty"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
	Target    string                 `json:"target,omitempty"`
}

// ToolCall represents a tool invocation request
type ToolCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
	Channel   string                 `json:"channel,omitempty"`
	ChatID    string                 `json:"chat_id,omitempty"`
	SenderID  string                 `json:"sender_id,omitempty"`
}

// Evaluator evaluates policies using configurable rules
type Evaluator struct {
	mu            sync.RWMutex
	config        *Config
	rules         []CompiledRule
	patterns      map[string]*regexp.Regexp
	configPath    string
	defaultResult PolicyResult
	timeout       time.Duration
}

// Config holds policy configuration
type Config struct {
	Enabled           bool              `json:"enabled" yaml:"enabled"`
	Timeout           int               `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	DefaultAllow      bool              `json:"default_allow" yaml:"default_allow"`
	Rules             []Rule            `json:"rules,omitempty" yaml:"rules,omitempty"`
	AllowedTools      []string          `json:"allowed_tools,omitempty" yaml:"allowed_tools,omitempty"`
	DeniedTools       []string          `json:"denied_tools,omitempty" yaml:"denied_tools,omitempty"`
	AllowedIntents    []string          `json:"allowed_intents,omitempty" yaml:"allowed_intents,omitempty"`
	DeniedIntents     []string          `json:"denied_intents,omitempty" yaml:"denied_intents,omitempty"`
	MaxToolArgs       int               `json:"max_tool_args,omitempty" yaml:"max_tool_args,omitempty"`
	RequireApproval   []string          `json:"require_approval,omitempty" yaml:"require_approval,omitempty"`
	ArgumentPatterns  []ArgumentPattern `json:"argument_patterns,omitempty" yaml:"argument_patterns,omitempty"`
	CustomPolicies    map[string]string `json:"custom_policies,omitempty" yaml:"custom_policies,omitempty"`
}

// Rule represents a policy rule
type Rule struct {
	ID          string   `json:"id" yaml:"id"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Condition   string   `json:"condition" yaml:"condition"`
	Action      string   `json:"action" yaml:"action"` // "allow", "deny", "require_approval"
	Tools       []string `json:"tools,omitempty" yaml:"tools,omitempty"`
	Intents     []string `json:"intents,omitempty" yaml:"intents,omitempty"`
	Priority    int      `json:"priority,omitempty" yaml:"priority,omitempty"`
}

// ArgumentPattern defines patterns to match in tool arguments
type ArgumentPattern struct {
	Tool      string `json:"tool" yaml:"tool"`
	Argument  string `json:"argument" yaml:"argument"`
	Pattern   string `json:"pattern" yaml:"pattern"`
	Action    string `json:"action" yaml:"action"`
	Reason    string `json:"reason,omitempty" yaml:"reason,omitempty"`
	Compiled  *regexp.Regexp `json:"-" yaml:"-"`
}

// CompiledRule is a pre-compiled rule for efficient evaluation
type CompiledRule struct {
	Rule            Rule
	ToolPatterns    []*regexp.Regexp
	IntentPatterns  []*regexp.Regexp
	ConditionParsed ConditionExpr
}

// ConditionExpr represents a parsed condition expression
type ConditionExpr struct {
	Field    string
	Operator string
	Value    interface{}
}

// NewEvaluator creates a new policy evaluator
func NewEvaluator(cfg *config.Config, configPath string) (*Evaluator, error) {
	e := &Evaluator{
		patterns:   make(map[string]*regexp.Regexp),
		configPath: configPath,
		timeout:    DefaultPolicyTimeout,
		defaultResult: PolicyResult{
			Allowed: true,
			Reason:  "default allow",
		},
	}

	// Load policy configuration
	policyCfg, err := e.loadPolicyConfig(configPath)
	if err != nil {
		logger.WarnCF("policy", "Failed to load policy config", map[string]any{"error": err.Error()})
		// Continue with default config
		policyCfg = &Config{
			Enabled:      false,
			DefaultAllow: true,
		}
	}

	e.config = policyCfg

	// Apply configuration
	if policyCfg.Timeout > 0 {
		e.timeout = time.Duration(policyCfg.Timeout) * time.Second
	}

	if !policyCfg.DefaultAllow {
		e.defaultResult = PolicyResult{
			Allowed: false,
			Reason:  "default deny",
		}
	}

	// Compile rules
	if err := e.compileRules(); err != nil {
		logger.ErrorCF("policy", "Failed to compile rules", map[string]any{"error": err.Error()})
	}

	return e, nil
}

// loadPolicyConfig loads policy configuration from file
func (e *Evaluator) loadPolicyConfig(configPath string) (*Config, error) {
	policyPath := filepath.Join(filepath.Dir(configPath), PolicyConfigFile)

	data, err := os.ReadFile(policyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{
				Enabled:      false,
				DefaultAllow: true,
			}, nil
		}
		return nil, fmt.Errorf("failed to read policy file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		// Try JSON format
		if err2 := json.Unmarshal(data, &cfg); err2 != nil {
			return nil, fmt.Errorf("failed to parse policy file (YAML/JSON): %w", err)
		}
	}

	return &cfg, nil
}

// compileRules compiles all rules for efficient evaluation
func (e *Evaluator) compileRules() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.rules = make([]CompiledRule, 0, len(e.config.Rules))

	for _, rule := range e.config.Rules {
		compiled := CompiledRule{
			Rule:           rule,
			ToolPatterns:   make([]*regexp.Regexp, 0, len(rule.Tools)),
			IntentPatterns: make([]*regexp.Regexp, 0, len(rule.Intents)),
		}

		// Compile tool patterns
		for _, tool := range rule.Tools {
			if re, err := regexp.Compile(tool); err == nil {
				compiled.ToolPatterns = append(compiled.ToolPatterns, re)
			} else {
				logger.WarnCF("policy", "Invalid tool pattern in rule", map[string]any{
					"rule_id": rule.Rule.ID,
					"pattern": tool,
					"error":   err.Error(),
				})
			}
		}

		// Compile intent patterns
		for _, intent := range rule.Intents {
			if re, err := regexp.Compile(intent); err == nil {
				compiled.IntentPatterns = append(compiled.IntentPatterns, re)
			} else {
				logger.WarnCF("policy", "Invalid intent pattern in rule", map[string]any{
					"rule_id": rule.Rule.ID,
					"pattern": intent,
					"error":   err.Error(),
				})
			}
		}

		// Parse condition
		compiled.ConditionParsed = parseCondition(rule.Condition)

		e.rules = append(e.rules, compiled)
	}

	// Sort rules by priority
	sortRulesByPriority(e.rules)

	// Compile argument patterns
	for i := range e.config.ArgumentPatterns {
		if e.config.ArgumentPatterns[i].Pattern != "" {
			if re, err := regexp.Compile(e.config.ArgumentPatterns[i].Pattern); err == nil {
				e.config.ArgumentPatterns[i].Compiled = re
			} else {
				logger.WarnCF("policy", "Invalid argument pattern", map[string]any{
					"pattern": e.config.ArgumentPatterns[i].Pattern,
					"error":   err.Error(),
				})
			}
		}
	}

	return nil
}

// parseCondition parses a condition string into a ConditionExpr
func parseCondition(condition string) ConditionExpr {
	// Simple condition parser: "field operator value"
	// Supported operators: ==, !=, contains, starts_with, ends_with, >, <, >=, <=
	operators := []string{"==", "!=", "contains", "starts_with", "ends_with", ">=", "<=", ">", "<"}

	for _, op := range operators {
		parts := strings.SplitN(condition, " "+op+" ", 2)
		if len(parts) == 2 {
			return ConditionExpr{
				Field:    strings.TrimSpace(parts[0]),
				Operator: op,
				Value:    strings.TrimSpace(parts[1]),
			}
		}
	}

	return ConditionExpr{}
}

// EvaluateIntent evaluates if an intent is allowed
func (e *Evaluator) EvaluateIntent(ctx context.Context, intent Intent) (PolicyResult, error) {
	if !e.config.Enabled {
		return PolicyResult{Allowed: true, Reason: "policy disabled"}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	done := make(chan PolicyResult, 1)
	go func() {
		result := e.evaluateIntentInternal(intent)
		done <- result
	}()

	select {
	case result := <-done:
		return result, nil
	case <-ctx.Done():
		return PolicyResult{
			Allowed: e.defaultResult.Allowed,
			Reason:  "policy evaluation timeout",
		}, nil
	}
}

func (e *Evaluator) evaluateIntentInternal(intent Intent) PolicyResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Check denied intents first
	for _, deniedPattern := range e.config.DeniedIntents {
		if re, ok := e.patterns[deniedPattern]; ok {
			if re.MatchString(intent.Type) {
				return PolicyResult{
					Allowed: false,
					Reason:  fmt.Sprintf("intent %q matches denied pattern", intent.Type),
				}
			}
		} else if strings.Contains(intent.Type, deniedPattern) {
			return PolicyResult{
				Allowed: false,
				Reason:  fmt.Sprintf("intent %q is denied", intent.Type),
			}
		}
	}

	// Check allowed intents (if specified, only these are allowed)
	if len(e.config.AllowedIntents) > 0 {
		allowed := false
		for _, allowedPattern := range e.config.AllowedIntents {
			if re, ok := e.patterns[allowedPattern]; ok {
				if re.MatchString(intent.Type) {
					allowed = true
					break
				}
			} else if intent.Type == allowedPattern || strings.Contains(intent.Type, allowedPattern) {
				allowed = true
				break
			}
		}
		if !allowed {
			return PolicyResult{
				Allowed: false,
				Reason:  fmt.Sprintf("intent %q is not in allowed list", intent.Type),
			}
		}
	}

	// Evaluate rules
	for _, rule := range e.rules {
		if len(rule.IntentPatterns) > 0 {
			matched := false
			for _, pattern := range rule.IntentPatterns {
				if pattern.MatchString(intent.Type) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		// Check condition
		if rule.ConditionParsed.Field != "" {
			if !evaluateCondition(rule.ConditionParsed, intent) {
				continue
			}
		}

		// Apply rule action
		switch rule.Rule.Action {
		case "allow":
			return PolicyResult{Allowed: true, Reason: fmt.Sprintf("rule %q allows this intent", rule.Rule.ID)}
		case "deny":
			return PolicyResult{Allowed: false, Reason: fmt.Sprintf("rule %q denies this intent", rule.Rule.ID)}
		case "require_approval":
			return PolicyResult{
				Allowed: false,
				Reason:  fmt.Sprintf("rule %q requires approval for this intent", rule.Rule.ID),
				Data:    map[string]interface{}{"requires_approval": true},
			}
		}
	}

	return e.defaultResult
}

// EvaluateActionPlan evaluates if an action plan is allowed
func (e *Evaluator) EvaluateActionPlan(ctx context.Context, plan ActionPlan) (PolicyResult, error) {
	if !e.config.Enabled {
		return PolicyResult{Allowed: true, Reason: "policy disabled"}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	done := make(chan PolicyResult, 1)
	go func() {
		result := e.evaluateActionPlanInternal(plan)
		done <- result
	}()

	select {
	case result := <-done:
		return result, nil
	case <-ctx.Done():
		return PolicyResult{
			Allowed: e.defaultResult.Allowed,
			Reason:  "policy evaluation timeout",
		}, nil
	}
}

func (e *Evaluator) evaluateActionPlanInternal(plan ActionPlan) PolicyResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, action := range plan.Actions {
		// Evaluate each action as a tool call
		toolCall := ToolCall{
			Name:      action.Tool,
			Arguments: action.Arguments,
			Target:    action.Target,
		}
		result := e.evaluateToolCallInternal(toolCall)
		if !result.Allowed {
			return result
		}
	}

	return PolicyResult{Allowed: true, Reason: "all actions in plan are allowed"}
}

// EvaluateToolCall evaluates if a tool call is allowed
func (e *Evaluator) EvaluateToolCall(ctx context.Context, toolCall ToolCall) (PolicyResult, error) {
	if !e.config.Enabled {
		return PolicyResult{Allowed: true, Reason: "policy disabled"}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	done := make(chan PolicyResult, 1)
	go func() {
		result := e.evaluateToolCallInternal(toolCall)
		done <- result
	}()

	select {
	case result := <-done:
		return result, nil
	case <-ctx.Done():
		return PolicyResult{
			Allowed: e.defaultResult.Allowed,
			Reason:  "policy evaluation timeout",
		}, nil
	}
}

func (e *Evaluator) evaluateToolCallInternal(toolCall ToolCall) PolicyResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Check denied tools first
	for _, deniedPattern := range e.config.DeniedTools {
		if re, ok := e.patterns[deniedPattern]; ok {
			if re.MatchString(toolCall.Name) {
				return PolicyResult{
					Allowed: false,
					Reason:  fmt.Sprintf("tool %q matches denied pattern", toolCall.Name),
				}
			}
		} else if toolCall.Name == deniedPattern || strings.Contains(toolCall.Name, deniedPattern) {
			return PolicyResult{
				Allowed: false,
				Reason:  fmt.Sprintf("tool %q is denied", toolCall.Name),
			}
		}
	}

	// Check allowed tools (if specified, only these are allowed)
	if len(e.config.AllowedTools) > 0 {
		allowed := false
		for _, allowedPattern := range e.config.AllowedTools {
			if re, ok := e.patterns[allowedPattern]; ok {
				if re.MatchString(toolCall.Name) {
					allowed = true
					break
				}
			} else if toolCall.Name == allowedPattern || strings.Contains(toolCall.Name, allowedPattern) {
				allowed = true
				break
			}
		}
		if !allowed {
			return PolicyResult{
				Allowed: false,
				Reason:  fmt.Sprintf("tool %q is not in allowed list", toolCall.Name),
			}
		}
	}

	// Check argument patterns
	for _, argPattern := range e.config.ArgumentPatterns {
		if argPattern.Tool != "" && argPattern.Tool != toolCall.Name {
			continue
		}
		if argPattern.Compiled == nil {
			continue
		}

		if args, ok := toolCall.Arguments[argPattern.Argument]; ok {
			argStr := fmt.Sprintf("%v", args)
			if argPattern.Compiled.MatchString(argStr) {
				switch argPattern.Action {
				case "deny":
					reason := argPattern.Reason
					if reason == "" {
						reason = fmt.Sprintf("argument %q matches denied pattern", argPattern.Argument)
					}
					return PolicyResult{Allowed: false, Reason: reason}
				case "require_approval":
					return PolicyResult{
						Allowed: false,
						Reason:  fmt.Sprintf("argument %q requires approval", argPattern.Argument),
						Data:    map[string]interface{}{"requires_approval": true},
					}
				}
			}
		}
	}

	// Check max arguments
	if e.config.MaxToolArgs > 0 && len(toolCall.Arguments) > e.config.MaxToolArgs {
		return PolicyResult{
			Allowed: false,
			Reason:  fmt.Sprintf("tool call exceeds maximum arguments (%d)", e.config.MaxToolArgs),
		}
	}

	// Evaluate rules
	for _, rule := range e.rules {
		if len(rule.ToolPatterns) > 0 {
			matched := false
			for _, pattern := range rule.ToolPatterns {
				if pattern.MatchString(toolCall.Name) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		// Apply rule action
		switch rule.Rule.Action {
		case "allow":
			return PolicyResult{Allowed: true, Reason: fmt.Sprintf("rule %q allows this tool", rule.Rule.ID)}
		case "deny":
			return PolicyResult{Allowed: false, Reason: fmt.Sprintf("rule %q denies this tool", rule.Rule.ID)}
		case "require_approval":
			return PolicyResult{
				Allowed: false,
				Reason:  fmt.Sprintf("rule %q requires approval for this tool", rule.Rule.ID),
				Data:    map[string]interface{}{"requires_approval": true},
			}
		}
	}

	// Check require_approval list
	for _, tool := range e.config.RequireApproval {
		if tool == toolCall.Name || strings.Contains(toolCall.Name, tool) {
			return PolicyResult{
				Allowed: false,
				Reason:  fmt.Sprintf("tool %q requires explicit approval", toolCall.Name),
				Data:    map[string]interface{}{"requires_approval": true},
			}
		}
	}

	return e.defaultResult
}

// Reload reloads policy configuration from disk
func (e *Evaluator) Reload() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	policyCfg, err := e.loadPolicyConfig(e.configPath)
	if err != nil {
		return err
	}

	e.config = policyCfg

	if policyCfg.Timeout > 0 {
		e.timeout = time.Duration(policyCfg.Timeout) * time.Second
	}

	if !policyCfg.DefaultAllow {
		e.defaultResult = PolicyResult{
			Allowed: false,
			Reason:  "default deny",
		}
	} else {
		e.defaultResult = PolicyResult{
			Allowed: true,
			Reason:  "default allow",
		}
	}

	e.mu.Unlock()
	err = e.compileRules()
	e.mu.Lock()
	return err
}

// UpdateConfig updates the evaluator with new policy configuration
func (e *Evaluator) UpdateConfig(policyCfg Config) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.config = &policyCfg

	if policyCfg.Timeout > 0 {
		e.timeout = time.Duration(policyCfg.Timeout) * time.Second
	}

	if !policyCfg.DefaultAllow {
		e.defaultResult = PolicyResult{
			Allowed: false,
			Reason:  "default deny",
		}
	} else {
		e.defaultResult = PolicyResult{
			Allowed: true,
			Reason:  "default allow",
		}
	}

	e.mu.Unlock()
	err := e.compileRules()
	e.mu.Lock()
	return err
}

// SetDefaultResult sets the default policy result when evaluation fails
func (e *Evaluator) SetDefaultResult(result PolicyResult) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.defaultResult = result
}

// IsEnabled returns whether policy evaluation is enabled
func (e *Evaluator) IsEnabled() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.config.Enabled
}

// Helper functions

func sortRulesByPriority(rules []CompiledRule) {
	// Sort by priority (higher priority first) using standard library
	sort.SliceStable(rules, func(i, j int) bool {
		return rules[i].Rule.Priority > rules[j].Rule.Priority
	})
}

func evaluateCondition(cond ConditionExpr, intent Intent) bool {
	var fieldValue interface{}

	switch cond.Field {
	case "intent.type":
		fieldValue = intent.Type
	case "intent.confidence":
		fieldValue = intent.Confidence
	case "intent.description":
		fieldValue = intent.Description
	default:
		if intent.Metadata != nil {
			fieldValue = intent.Metadata[cond.Field]
		}
	}

	if fieldValue == nil {
		return false
	}

	strValue := fmt.Sprintf("%v", fieldValue)
	strCond := fmt.Sprintf("%v", cond.Value)

	switch cond.Operator {
	case "==":
		return strValue == strCond
	case "!=":
		return strValue != strCond
	case "contains":
		return strings.Contains(strValue, strCond)
	case "starts_with":
		return strings.HasPrefix(strValue, strCond)
	case "ends_with":
		return strings.HasSuffix(strValue, strCond)
	case ">":
		// Numeric comparison
		return compareNumbers(fieldValue, cond.Value) > 0
	case "<":
		return compareNumbers(fieldValue, cond.Value) < 0
	case ">=":
		return compareNumbers(fieldValue, cond.Value) >= 0
	case "<=":
		return compareNumbers(fieldValue, cond.Value) <= 0
	}

	return false
}

func compareNumbers(a, b interface{}) int {
	aFloat := toFloat64(a)
	bFloat := toFloat64(b)
	if aFloat > bFloat {
		return 1
	} else if aFloat < bFloat {
		return -1
	}
	return 0
}

func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		var f float64
		fmt.Sscanf(val, "%f", &f)
		return f
	}
	return 0
}
