package command

import (
	"context"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/bus"
)

type ShowCommand struct{}

func (c *ShowCommand) Name() string {
	return "/show"
}

func (c *ShowCommand) Description() string {
	return "Show current configuration (model, channel)"
}

func (c *ShowCommand) Execute(ctx context.Context, agent AgentState, args []string, msg bus.InboundMessage) (string, error) {
	if len(args) < 1 {
		return "Usage: /show [model|channel]", nil
	}
	switch args[0] {
	case "model":
		return fmt.Sprintf("Current model: %s", agent.GetModel()), nil
	case "channel":
		return fmt.Sprintf("Current channel: %s", msg.Channel), nil
	default:
		return fmt.Sprintf("Unknown show target: %s", args[0]), nil
	}
}

type ListCommand struct{}

func (c *ListCommand) Name() string {
	return "/list"
}

func (c *ListCommand) Description() string {
	return "List available resources (models, channels)"
}

func (c *ListCommand) Execute(ctx context.Context, agent AgentState, args []string, msg bus.InboundMessage) (string, error) {
	if len(args) < 1 {
		return "Usage: /list [models|channels]", nil
	}
	switch args[0] {
	case "models":
		// TODO: Fetch available models dynamically if possible
		return "Available models: glm-4.7, claude-3-5-sonnet, gpt-4o (configured in config.json/env)", nil
	case "channels":
		cm := agent.GetChannelManager()
		if cm == nil {
			return "Channel manager not initialized", nil
		}

		// Use reflection or interface assertion to access GetEnabledChannels
		// Since we use interface{} to avoid circular deps, we need to assert a local interface or use reflection
		// For simplicity, let's assume the caller injected something that has GetEnabledChannels
		type ChannelLister interface {
			GetEnabledChannels() []string
		}

		if lister, ok := cm.(ChannelLister); ok {
			channels := lister.GetEnabledChannels()
			if len(channels) == 0 {
				return "No channels enabled", nil
			}
			return fmt.Sprintf("Enabled channels: %s", strings.Join(channels, ", ")), nil
		}
		return "Channel manager does not support listing channels", nil

	default:
		return fmt.Sprintf("Unknown list target: %s", args[0]), nil
	}
}

type SwitchCommand struct{}

func (c *SwitchCommand) Name() string {
	return "/switch"
}

func (c *SwitchCommand) Description() string {
	return "Switch configuration context"
}

func (c *SwitchCommand) Execute(ctx context.Context, agent AgentState, args []string, msg bus.InboundMessage) (string, error) {
	if len(args) < 3 || args[1] != "to" {
		return "Usage: /switch [model|channel] to <name>", nil
	}
	target := args[0]
	value := args[2]

	switch target {
	case "model":
		oldModel := agent.GetModel()
		agent.SetModel(value)
		return fmt.Sprintf("Switched model from %s to %s", oldModel, value), nil
	case "channel":
		cm := agent.GetChannelManager()
		if cm == nil {
			return "Channel manager not initialized", nil
		}

		type ChannelGetter interface {
			GetChannel(name string) (interface{}, bool)
		}

		if getter, ok := cm.(ChannelGetter); ok {
			if _, exists := getter.GetChannel(value); !exists && value != "cli" {
				return fmt.Sprintf("Channel '%s' not found or not enabled", value), nil
			}
			return fmt.Sprintf("Switched target channel to %s (Note: this currently only validates existence)", value), nil
		}
		return "Channel manager check failed", nil

	default:
		return fmt.Sprintf("Unknown switch target: %s", target), nil
	}
}

type StartCommand struct{}

func (c *StartCommand) Name() string {
	return "/start"
}

func (c *StartCommand) Description() string {
	return "Start the bot and get a welcome message"
}

func (c *StartCommand) Execute(ctx context.Context, agent AgentState, args []string, msg bus.InboundMessage) (string, error) {
	return "Hello! I am PicoClaw 🦞\nI am your personal AI agent. Type /help to see what I can do.", nil
}

type HelpCommand struct {
	Registry *Registry
}

func (c *HelpCommand) Name() string {
	return "/help"
}

func (c *HelpCommand) Description() string {
	return "Show available commands"
}

func (c *HelpCommand) Execute(ctx context.Context, agent AgentState, args []string, msg bus.InboundMessage) (string, error) {
	if c.Registry == nil {
		return "Error: Command registry not available", nil
	}

	var sb strings.Builder
	sb.WriteString("Available commands:\n")

	for name, cmd := range c.Registry.ListCommands() {
		sb.WriteString(fmt.Sprintf("%s - %s\n", name, cmd.Description()))
	}

	return sb.String(), nil
}
