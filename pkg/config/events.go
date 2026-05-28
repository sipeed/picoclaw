package config

// EventsConfig groups runtime event configuration.
type EventsConfig struct {
	Logging EventLoggingConfig `json:"logging,omitempty" envPrefix:"PICOCLAW_EVENTS_LOGGING_"`
}

// EventLoggingConfig controls centralized runtime event logging.
type EventLoggingConfig struct {
	// Enabled controls whether runtime events are printed by the built-in logger.
	Enabled bool `json:"enabled" env:"ENABLED"`
	// Include contains exact event kinds or glob patterns such as "agent.*" or "*".
	Include []string `json:"include,omitempty" env:"INCLUDE"`
	// Exclude contains exact event kinds or glob patterns to suppress after Include matches.
	Exclude []string `json:"exclude,omitempty" env:"EXCLUDE"`
	// MinSeverity filters out events below the configured severity: debug, info, warn, or error.
	MinSeverity string `json:"min_severity,omitempty" env:"MIN_SEVERITY"`
	// IncludePayload adds the raw payload to logs. Leave disabled unless detailed diagnostics are needed.
	IncludePayload bool `json:"include_payload,omitempty" env:"INCLUDE_PAYLOAD"`
}

// DefaultEventLoggingInclude keeps the pre-existing behavior where agent events
// are printed, while non-agent runtime events are published for subscribers only.
var DefaultEventLoggingInclude = []string{"agent.*"}

// EffectiveEventLoggingConfig returns a logging config with stable defaults.
func EffectiveEventLoggingConfig(cfg *Config) EventLoggingConfig {
	if cfg == nil {
		return defaultEventLoggingConfig()
	}

	out := cfg.Events.Logging
	if out.MinSeverity == "" {
		out.MinSeverity = "info"
	}
	if len(out.Include) == 0 {
		out.Include = append([]string(nil), DefaultEventLoggingInclude...)
	}
	return out
}

func defaultEventLoggingConfig() EventLoggingConfig {
	return EventLoggingConfig{
		Enabled:     true,
		Include:     append([]string(nil), DefaultEventLoggingInclude...),
		MinSeverity: "info",
	}
}
