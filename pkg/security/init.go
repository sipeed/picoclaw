package security

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/security/behavior"
	"github.com/sipeed/picoclaw/pkg/security/canary"
	"github.com/sipeed/picoclaw/pkg/security/ipia"
	"github.com/sipeed/picoclaw/pkg/security/pii"
	"github.com/sipeed/picoclaw/pkg/security/policy"
)

// Init registers all security hooks as built-in hooks.
// This should be called once at application startup.
func Init() {
	_ = agent.RegisterBuiltinHook("security_canary", func(ctx context.Context, spec config.BuiltinHookConfig) (any, error) {
		if !spec.Enabled {
			return nil, nil // Or a disabled hook, but nil is fine if enable check is in loop
		}
		return canary.NewHook()
	})

	_ = agent.RegisterBuiltinHook("security_pii", func(ctx context.Context, spec config.BuiltinHookConfig) (any, error) {
		return pii.NewRedactor(spec.Enabled), nil
	})

	_ = agent.RegisterBuiltinHook("security_ipia", func(ctx context.Context, spec config.BuiltinHookConfig) (any, error) {
		return ipia.NewDetector(spec.Enabled), nil
	})

	_ = agent.RegisterBuiltinHook("security_policy", func(ctx context.Context, spec config.BuiltinHookConfig) (any, error) {
		var pcfg policy.Config
		if len(spec.Config) > 0 {
			if err := json.Unmarshal(spec.Config, &pcfg); err != nil {
				return nil, fmt.Errorf("failed to unmarshal security_policy config: %w", err)
			}
		}
		return policy.NewChecker(pcfg), nil
	})

	_ = agent.RegisterBuiltinHook("security_behavior", func(ctx context.Context, spec config.BuiltinHookConfig) (any, error) {
		type bcfg struct {
			MaxToolCalls  int   `json:"max_tool_calls"`
			MaxTotalBytes int64 `json:"max_total_bytes"`
		}
		var bc bcfg
		if len(spec.Config) > 0 {
			if err := json.Unmarshal(spec.Config, &bc); err != nil {
				return nil, fmt.Errorf("failed to unmarshal security_behavior config: %w", err)
			}
		}
		return behavior.NewMonitor(bc.MaxToolCalls, bc.MaxTotalBytes), nil
	})
}
