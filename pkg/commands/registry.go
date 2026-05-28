package commands

type Registry struct {
	defs  []Definition
	index map[string]int
}

// NewRegistry stores the canonical command set used by both dispatch and
// optional platform registration adapters.
func NewRegistry(defs []Definition) *Registry {
	stored := make([]Definition, len(defs))
	copy(stored, defs)

	index := make(map[string]int, len(stored)*2)
	for i, def := range stored {
		registerCommandName(index, def.Name, i)
		for _, alias := range def.Aliases {
			registerCommandName(index, alias, i)
		}
	}

	return &Registry{defs: stored, index: index}
}

// Definitions returns all registered command definitions.
// Command availability is global and no longer channel-scoped.
func (r *Registry) Definitions() []Definition {
	out := make([]Definition, len(r.defs))
	copy(out, r.defs)
	return out
}

// Lookup returns a command definition by normalized command name or alias.
func (r *Registry) Lookup(name string) (Definition, bool) {
	key := normalizeCommandName(name)
	if key == "" {
		return Definition{}, false
	}
	idx, ok := r.index[key]
	if !ok {
		return Definition{}, false
	}
	return r.defs[idx], true
}

func registerCommandName(index map[string]int, name string, defIndex int) {
	key := normalizeCommandName(name)
	if key == "" {
		return
	}
	if _, exists := index[key]; exists {
		return
	}
	index[key] = defIndex
}
