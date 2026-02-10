package config

import "time"

type SwarmConfig struct {
	Limits     LimitsConfig     `json:"limits"`
	Resilience ResilienceConfig `json:"resilience"`
	Memory     MemoryConfig     `json:"memory"` // Added memory config
	Roles      map[string]Role  `json:"roles"`
	Policies   []Policy         `json:"policies"`
}

type MemoryConfig struct {
	SharedKnowledge bool `json:"shared_knowledge"` // If true, search across all swarms
}

type LimitsConfig struct {
	MaxDepth       int           `json:"max_depth" env:"PICOCLAW_SWARM_MAX_DEPTH"`
	MaxNodes       int           `json:"max_nodes" env:"PICOCLAW_SWARM_MAX_NODES"`
	GlobalTimeout  time.Duration `json:"global_timeout" env:"PICOCLAW_SWARM_GLOBAL_TIMEOUT"`
	MaxRetries     int           `json:"max_retries" env:"PICOCLAW_SWARM_MAX_RETRIES"`
	MaxIterations  int           `json:"max_iterations" env:"PICOCLAW_SWARM_MAX_ITERATIONS"`
	PruningMsgKeep int           `json:"pruning_msg_keep" env:"PICOCLAW_SWARM_PRUNING_MSG_KEEP"`
}

type ResilienceConfig struct {
	RetryBackoff time.Duration `json:"retry_backoff" env:"PICOCLAW_SWARM_RETRY_BACKOFF"`
}

type Role struct {
	Description  string   `json:"description"`
	SystemPrompt string   `json:"system_prompt"`
	Tools        []string `json:"tools"`
	Model        string   `json:"model"`
	MaxCost      float64  `json:"max_cost"`
}

type Policy struct {
	Role    string   `json:"role"`
	Allowed []string `json:"allowed"`
	Denied  []string `json:"denied"`
}

func DefaultSwarmConfig() SwarmConfig {
	return SwarmConfig{
		Limits: LimitsConfig{
			MaxDepth:       3,
			MaxNodes:       10,
			GlobalTimeout:  10 * time.Minute,
			MaxRetries:     3,
			MaxIterations:  10,
			PruningMsgKeep: 6, // Keep last 6 messages when pruning
		},
		Resilience: ResilienceConfig{
			RetryBackoff: 2 * time.Second,
		},
		Memory: MemoryConfig{
			SharedKnowledge: true, // Default to enabled for "Hive Mind" effect
		},
		Roles: map[string]Role{
			"Manager": {
				Description:  "Orchestrates and delegates tasks",
				SystemPrompt: "You are the MANAGER of this swarm...",
				Tools:        []string{"delegate_task", "save_memory", "search_memory"},
			},
			"Researcher": {
				Description:  "Deep research and verification",
				SystemPrompt: "You are a RESEARCHER...",
				Tools:        []string{"web_search", "web_fetch", "save_memory", "search_memory", "read_file"},
			},
			"Analyst": {
				Description:  "Analyze provided data",
				SystemPrompt: "You are an ANALYST...",
				Tools:        []string{"web_search", "web_fetch", "read_file", "list_dir", "save_memory", "search_memory"},
			},
			"Writer": {
				Description:  "Write content to files",
				SystemPrompt: "You are a WRITER...",
				Tools:        []string{"read_file", "write_file", "list_dir", "search_memory"},
			},
			"Critic": {
				Description:  "Red team and validate",
				SystemPrompt: "You are a CRITIC...",
				Tools:        []string{"web_search", "read_file", "search_memory"},
			},
		},
		Policies: []Policy{
			{Role: "Manager", Allowed: []string{"*"}},
			{Role: "Researcher", Allowed: []string{"web_search", "web_fetch", "read_file", "save_memory", "search_memory"}, Denied: []string{"exec", "write_file"}},
			{Role: "Analyst", Allowed: []string{"web_search", "web_fetch", "read_file", "list_dir", "save_memory", "search_memory"}, Denied: []string{"write_file", "exec"}},
			{Role: "Writer", Allowed: []string{"read_file", "write_file", "list_dir", "search_memory"}, Denied: []string{"exec"}},
			{Role: "Critic", Allowed: []string{"web_search", "read_file", "search_memory"}},
		},
	}
}
