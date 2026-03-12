package config

// SubagentsConfig holds fork-specific subagent orchestration settings.
type SubagentsConfig struct {
	Enabled     bool              `json:"enabled,omitempty"`
	AllowAgents []string          `json:"allow_agents,omitempty"`
	Model       *AgentModelConfig `json:"model,omitempty"`
}

// PeerMatch identifies a peer by kind (direct/group) and ID.
type PeerMatch struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}
