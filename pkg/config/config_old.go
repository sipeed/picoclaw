// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package config

type agentDefaultsV0 struct {
	Workspace                 string         `json:"workspace"                       env:"PICOCLAW_AGENTS_DEFAULTS_WORKSPACE"`
	RestrictToWorkspace       bool           `json:"restrict_to_workspace"           env:"PICOCLAW_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE"`
	AllowReadOutsideWorkspace bool           `json:"allow_read_outside_workspace"    env:"PICOCLAW_AGENTS_DEFAULTS_ALLOW_READ_OUTSIDE_WORKSPACE"`
	Provider                  string         `json:"provider"                        env:"PICOCLAW_AGENTS_DEFAULTS_PROVIDER"`
	ModelName                 string         `json:"model_name,omitempty"            env:"PICOCLAW_AGENTS_DEFAULTS_MODEL_NAME"`
	Model                     string         `json:"model"                           env:"PICOCLAW_AGENTS_DEFAULTS_MODEL"` // Deprecated: use model_name instead
	ModelFallbacks            []string       `json:"model_fallbacks,omitempty"`
	ImageModel                string         `json:"image_model,omitempty"           env:"PICOCLAW_AGENTS_DEFAULTS_IMAGE_MODEL"`
	ImageModelFallbacks       []string       `json:"image_model_fallbacks,omitempty"`
	MaxTokens                 int            `json:"max_tokens"                      env:"PICOCLAW_AGENTS_DEFAULTS_MAX_TOKENS"`
	Temperature               *float64       `json:"temperature,omitempty"           env:"PICOCLAW_AGENTS_DEFAULTS_TEMPERATURE"`
	MaxToolIterations         int            `json:"max_tool_iterations"             env:"PICOCLAW_AGENTS_DEFAULTS_MAX_TOOL_ITERATIONS"`
	SummarizeMessageThreshold int            `json:"summarize_message_threshold"     env:"PICOCLAW_AGENTS_DEFAULTS_SUMMARIZE_MESSAGE_THRESHOLD"`
	SummarizeTokenPercent     int            `json:"summarize_token_percent"         env:"PICOCLAW_AGENTS_DEFAULTS_SUMMARIZE_TOKEN_PERCENT"`
	MaxMediaSize              int            `json:"max_media_size,omitempty"        env:"PICOCLAW_AGENTS_DEFAULTS_MAX_MEDIA_SIZE"`
	Routing                   *RoutingConfig `json:"routing,omitempty"`
}

// GetModelName returns the effective model name for the agent defaults.
// It prefers the new "model_name" field but falls back to "model" for backward compatibility.
func (d *agentDefaultsV0) GetModelName() string {
	if d.ModelName != "" {
		return d.ModelName
	}
	return d.Model
}

type agentsConfigV0 struct {
	Defaults agentDefaultsV0 `json:"defaults"`
	List     []AgentConfig   `json:"list,omitempty"`
}

// configV0 represents the config structure before versioning was introduced.
// This struct is used for loading legacy config files (version 0).
// It is unexported since it's only used internally for migration.
type configV0 struct {
	Agents    agentsConfigV0  `json:"agents"`
	Bindings  []AgentBinding  `json:"bindings,omitempty"`
	Session   SessionConfig   `json:"session,omitempty"`
	Channels  ChannelsConfig  `json:"channels"`
	Providers ProvidersConfig `json:"providers,omitempty"`
	ModelList []ModelConfig   `json:"model_list"`
	Gateway   GatewayConfig   `json:"gateway"`
	Tools     ToolsConfig     `json:"tools"`
	Heartbeat HeartbeatConfig `json:"heartbeat"`
	Devices   DevicesConfig   `json:"devices"`
}

func (c *configV0) migrateChannelConfigs() {
	// Discord: mention_only -> group_trigger.mention_only
	if c.Channels.Discord.MentionOnly && !c.Channels.Discord.GroupTrigger.MentionOnly {
		c.Channels.Discord.GroupTrigger.MentionOnly = true
	}

	// OneBot: group_trigger_prefix -> group_trigger.prefixes
	if len(c.Channels.OneBot.GroupTriggerPrefix) > 0 &&
		len(c.Channels.OneBot.GroupTrigger.Prefixes) == 0 {
		c.Channels.OneBot.GroupTrigger.Prefixes = c.Channels.OneBot.GroupTriggerPrefix
	}
}

func (c *configV0) Migrate() (*Config, error) {
	// Migrate legacy channel config fields to new unified structures
	cfg := DefaultConfig()

	// Always copy user's Agents config to preserve settings like Provider, Model, MaxTokens
	cfg.Agents.List = c.Agents.List
	cfg.Agents.Defaults.Workspace = c.Agents.Defaults.Workspace
	cfg.Agents.Defaults.RestrictToWorkspace = c.Agents.Defaults.RestrictToWorkspace
	cfg.Agents.Defaults.AllowReadOutsideWorkspace = c.Agents.Defaults.AllowReadOutsideWorkspace
	cfg.Agents.Defaults.Provider = c.Agents.Defaults.Provider
	cfg.Agents.Defaults.ModelName = c.Agents.Defaults.GetModelName()
	cfg.Agents.Defaults.ModelFallbacks = c.Agents.Defaults.ModelFallbacks
	cfg.Agents.Defaults.ImageModel = c.Agents.Defaults.ImageModel
	cfg.Agents.Defaults.ImageModelFallbacks = c.Agents.Defaults.ImageModelFallbacks
	cfg.Agents.Defaults.MaxTokens = c.Agents.Defaults.MaxTokens
	cfg.Agents.Defaults.Temperature = c.Agents.Defaults.Temperature
	cfg.Agents.Defaults.MaxToolIterations = c.Agents.Defaults.MaxToolIterations
	cfg.Agents.Defaults.SummarizeMessageThreshold = c.Agents.Defaults.SummarizeMessageThreshold
	cfg.Agents.Defaults.SummarizeTokenPercent = c.Agents.Defaults.SummarizeTokenPercent
	cfg.Agents.Defaults.MaxMediaSize = c.Agents.Defaults.MaxMediaSize
	cfg.Agents.Defaults.Routing = c.Agents.Defaults.Routing

	// Copy other top-level fields
	cfg.Bindings = c.Bindings
	cfg.Session = c.Session
	cfg.Channels = c.Channels
	cfg.Gateway = c.Gateway
	cfg.Tools = c.Tools
	cfg.Heartbeat = c.Heartbeat
	cfg.Devices = c.Devices

	// Only override ModelList if user provided values
	if len(c.ModelList) > 0 {
		cfg.ModelList = c.ModelList
	}

	cfg.Version = CurrentVersion
	return cfg, nil
}
