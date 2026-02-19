package tools

// DefaultToolGroups maps group references (e.g. "group:fs") to tool names.
// Groups provide a convenient shorthand for tool policies so operators
// can allow/deny entire categories without listing every tool name.
var DefaultToolGroups = map[string][]string{
	"group:fs":     {"read_file", "write_file", "edit_file", "append_file", "list_dir"},
	"group:web":    {"web_search", "web_fetch"},
	"group:exec":   {"exec"},
	"group:hw":     {"i2c", "spi"},
	"group:comms":  {"message", "spawn"},
	"group:agents": {"blackboard", "handoff", "list_agents"},
}

// ResolveToolNames expands group refs (e.g. "group:fs") and individual tool
// names into a deduplicated list of concrete tool names.
// Unknown group refs are treated as individual tool names (pass-through).
func ResolveToolNames(refs []string) []string {
	seen := make(map[string]struct{}, len(refs))
	result := make([]string, 0, len(refs))

	for _, ref := range refs {
		if tools, ok := DefaultToolGroups[ref]; ok {
			for _, name := range tools {
				if _, dup := seen[name]; !dup {
					seen[name] = struct{}{}
					result = append(result, name)
				}
			}
		} else {
			if _, dup := seen[ref]; !dup {
				seen[ref] = struct{}{}
				result = append(result, ref)
			}
		}
	}

	return result
}
