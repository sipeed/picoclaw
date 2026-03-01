package commands

type Registry struct {
	defs []Definition
}

func NewRegistry(defs []Definition) *Registry {
	return &Registry{defs: defs}
}

func (r *Registry) ForChannel(channel string) []Definition {
	out := make([]Definition, 0, len(r.defs))
	for _, d := range r.defs {
		if len(d.Channels) == 0 {
			out = append(out, d)
			continue
		}
		for _, ch := range d.Channels {
			if ch == channel {
				out = append(out, d)
				break
			}
		}
	}
	return out
}
