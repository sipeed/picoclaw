package config

import (
	"testing"
)

func TestConfigValidation_Comprehensive(t *testing.T) {
	// Test valid configuration
	validConfig := &Config{
		Agents: AgentsConfig{
			Defaults: AgentDefaults{
				Workspace: "~/test-workspace",
				Model:     "test-model",
			},
		},
		Channels: ChannelsConfig{
			Telegram: TelegramConfig{
				Enabled: false, // Disabled, so no token required
			},
		},
		Gateway: GatewayConfig{
			Host: "localhost",
			Port: 8080,
		},
		ModelList: []ModelConfig{
			{
				ModelName: "test-model",
				Model:     "openai/gpt-test",
			},
		},
	}

	if err := validConfig.Validate(); err != nil {
		t.Errorf("Valid config should not produce validation error, but got: %v", err)
	}

	// Test invalid configuration - missing workspace
	invalidConfig := &Config{
		Agents: AgentsConfig{
			Defaults: AgentDefaults{
				Workspace: "", // Missing required field
			},
		},
		Channels: ChannelsConfig{
			Telegram: TelegramConfig{
				Enabled: false,
			},
		},
		Gateway: GatewayConfig{
			Host: "localhost",
			Port: 8080,
		},
		ModelList: []ModelConfig{
			{
				ModelName: "test-model",
				Model:     "openai/gpt-test",
			},
		},
	}

	if err := invalidConfig.Validate(); err == nil {
		t.Error("Invalid config (missing workspace) should produce validation error, but didn't get one")
	}

	if err := invalidConfig.Validate(); err != nil &&
		(err.Error() == "" || err.Error() == "workspace is required" ||
			err.Error() == "agents config validation failed: defaults: workspace is required") {
		// Expected error condition
	} else if err != nil {
		t.Logf("Got expected error for invalid config: %v", err)
	}
}

func TestConfigValidation_ChannelRequirements(t *testing.T) {
	// Test telegram with enabled but no token
	invalidTelegramConfig := &Config{
		Agents: AgentsConfig{
			Defaults: AgentDefaults{
				Workspace: "~/test-workspace",
				Model:     "test-model",
			},
		},
		Channels: ChannelsConfig{
			Telegram: TelegramConfig{
				Enabled: true, // Enabled but no token
				Token:   "",   // Missing required token
			},
		},
		Gateway: GatewayConfig{
			Host: "localhost",
			Port: 8080,
		},
		ModelList: []ModelConfig{
			{
				ModelName: "test-model",
				Model:     "openai/gpt-test",
			},
		},
	}

	if err := invalidTelegramConfig.Validate(); err == nil {
		t.Error("Invalid telegram config (missing token) should produce validation error, but didn't get one")
	} else {
		t.Logf("Got expected error for invalid telegram config: %v", err)
	}
}
