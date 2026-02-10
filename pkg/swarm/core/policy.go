package core

import "strings"

// RolePolicy defines permissions for a role
type RolePolicy struct {
	AllowedTools []string // Whitelist (if empty, nothing allowed unless wildcard)
	DeniedTools  []string // Blacklist (overrides whitelist)
}

type PolicyChecker struct {
	policies map[string]RolePolicy
}

func NewPolicyChecker(policies map[string]RolePolicy) *PolicyChecker {
	return &PolicyChecker{
		policies: policies,
	}
}

// CanUseTool checks if a role is allowed to use a specific tool
func (pc *PolicyChecker) CanUseTool(roleName, toolName string) bool {
	policy, ok := pc.policies[roleName]
	if !ok {
		// Default restrictive policy for unknown roles
		return false
	}

	// 1. Check Denied List first (Safety First)
	for _, denied := range policy.DeniedTools {
		if matchWildcard(denied, toolName) {
			return false
		}
	}

	// 2. Check Allowed List
	for _, allowed := range policy.AllowedTools {
		if matchWildcard(allowed, toolName) {
			return true
		}
	}

	return false
}

func matchWildcard(pattern, subject string) bool {
	if pattern == "*" {
		return true
	}
	return strings.EqualFold(pattern, subject)
}