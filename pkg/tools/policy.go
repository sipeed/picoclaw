package tools

// ToolPolicy represents one layer of allow/deny filtering.
type ToolPolicy struct {
	Allow []string
	Deny  []string
}

// ApplyPolicy filters a registry in-place.
// Allow (if non-empty): only listed tools survive.
// Deny: listed tools removed from whatever remains.
func ApplyPolicy(reg *ToolRegistry, policy ToolPolicy) {
	allowNames := ResolveToolNames(policy.Allow)
	denyNames := ResolveToolNames(policy.Deny)

	// Allow-list: if non-empty, remove everything not in the allow set
	if len(allowNames) > 0 {
		allowSet := make(map[string]struct{}, len(allowNames))
		for _, name := range allowNames {
			allowSet[name] = struct{}{}
		}
		for _, name := range reg.List() {
			if _, ok := allowSet[name]; !ok {
				reg.Remove(name)
			}
		}
	}

	// Deny-list: remove listed tools
	for _, name := range denyNames {
		reg.Remove(name)
	}
}

// DepthDenyList returns tools to deny at a given depth.
// depth 0: nil (main agent, full access)
// depth >= maxDepth: spawn/handoff/list_agents denied (leaf, no further chaining)
// between 0 and maxDepth: nil (mid-chain, full access)
func DepthDenyList(depth, maxDepth int) []string {
	if depth >= maxDepth {
		return []string{"spawn", "handoff", "list_agents"}
	}
	return nil
}
