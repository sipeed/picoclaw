// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"sync"
	"sync/atomic"

	"jane/pkg/bus"
	"jane/pkg/channels"
	"jane/pkg/commands"
	"jane/pkg/config"
	"jane/pkg/media"
	"jane/pkg/providers"
	"jane/pkg/state"
	"jane/pkg/voice"
)

type AgentLoop struct {
	bus            *bus.MessageBus
	cfg            *config.Config
	registry       *AgentRegistry
	state          *state.Manager
	running        atomic.Bool
	summarizing    sync.Map
	fallback       *providers.FallbackChain
	channelManager *channels.Manager
	mediaStore     media.MediaStore
	transcriber    voice.Transcriber
	cmdRegistry    *commands.Registry
	mcp            mcpRuntime
}

// processOptions configures how a message is processed
type processOptions struct {
	SessionKey      string   // Session identifier for history/context
	Channel         string   // Target channel for tool execution
	ChatID          string   // Target chat ID for tool execution
	UserMessage     string   // User message content (may include prefix)
	Media           []string // media:// refs from inbound message
	DefaultResponse string   // Response when LLM returns empty
	EnableSummary   bool     // Whether to trigger summarization
	SendResponse    bool     // Whether to send response via bus
	NoHistory       bool     // If true, don't load session history (for heartbeat)
}

const (
	defaultResponse           = "I've completed processing but have no response to give. Increase `max_tool_iterations` in config.json."
	sessionKeyAgentPrefix     = "agent:"
	metadataKeyAccountID      = "account_id"
	metadataKeyGuildID        = "guild_id"
	metadataKeyTeamID         = "team_id"
	metadataKeyParentPeerKind = "parent_peer_kind"
	metadataKeyParentPeerID   = "parent_peer_id"
)
