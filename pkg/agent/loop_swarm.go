// PicoClaw - Ultra-lightweight personal AI agent
// Swarm mode support for multi-agent coordination
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels/pico"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	picolib "github.com/sipeed/picoclaw/pkg/pico"
	"github.com/sipeed/picoclaw/pkg/swarm"
	"github.com/sipeed/picoclaw/pkg/tools"
)

// registerPicoNodeHandler registers the incoming node message handler on the Pico channel.
func (al *AgentLoop) registerPicoNodeHandler() {
	if al.channelManager == nil {
		return
	}
	ch, ok := al.channelManager.GetChannel("pico")
	if !ok {
		logger.WarnC("swarm", "Pico channel not available, inter-node request handler not registered")
		return
	}
	picoCh, ok := ch.(*pico.PicoChannel)
	if !ok {
		logger.WarnC("swarm", "Pico channel is not a PicoChannel type, inter-node request handler not registered")
		return
	}
	picoCh.SetNodeRequestHandler(func(payload map[string]any) (map[string]any, error) {
		p := picolib.NodePayload(payload)

		switch p.Action() {
		case picolib.NodeActionHandoffRequest:
			return al.handleIncomingHandoffRequest(p)
		case picolib.NodeActionMessage, "":
			// Default action: direct message (backward compatible)
			response, err := al.handleIncomingNodeMessage(&picolib.DirectMessage{
				MessageID:    p.RequestID(),
				SourceNodeID: p.SourceNodeID(),
				Content:      p.Content(),
				Channel:      p.Channel(),
				ChatID:       p.ChatID(),
				SenderID:     p.SenderID(),
				Metadata:     p.Metadata(),
			})
			if err != nil {
				return picolib.ErrorReply(err.Error()), nil
			}
			return picolib.ResponseReply(response), nil
		default:
			return picolib.ErrorReply(fmt.Sprintf("unknown action: %s", p.Action())), nil
		}
	})
	logger.InfoC("swarm", "Node request handler registered on Pico channel")
}

// handleNodeRouting handles routing a message to a specific node in the swarm.
func (al *AgentLoop) handleNodeRouting(
	ctx context.Context,
	msg bus.InboundMessage,
	targetNodeID, content string,
) (string, error) {
	// Check if target is this node
	if targetNodeID == al.swarmDiscovery.LocalNode().ID {
		logger.InfoCF("swarm", "Target is this node, processing locally", map[string]any{"node_id": targetNodeID})
		// Update content and continue processing locally.
		// Note: content has already been stripped of @node-id: prefix by ParseNodeMention,
		// so calling processMessage is safe and will not cause infinite recursion.
		msg.Content = content
		return al.processMessage(ctx, msg)
	}

	// Find target node
	members := al.swarmDiscovery.Members()
	var target *swarm.NodeWithState
	for _, m := range members {
		if m.Node.ID == targetNodeID {
			target = m
			break
		}
	}

	if target == nil {
		return fmt.Sprintf("❌ Node '%s' not found in cluster. Use /nodes to see available nodes.", targetNodeID), nil
	}

	// Check if target is available
	if target.State.Status != swarm.NodeStatusAlive {
		return fmt.Sprintf("❌ Node '%s' is not alive (status: %s)", targetNodeID, target.State.Status), nil
	}

	if target.Node.LoadScore > 0.9 {
		return fmt.Sprintf("❌ Node '%s' is overloaded (load: %.0f%%)", targetNodeID, target.Node.LoadScore*100), nil
	}

	// Use PicoNodeClient to send message to target node
	if al.swarmPicoClient == nil {
		return fmt.Sprintf("❌ Node-to-node communication not initialized"), nil
	}

	// Construct target Pico address using the node's HTTP port
	if target.Node.HTTPPort == 0 {
		return fmt.Sprintf("❌ Node '%s' does not have an HTTP port configured", targetNodeID), nil
	}
	picoAddr := picolib.BuildNodeAddr(target.Node.Addr, target.Node.HTTPPort)

	logger.InfoCF("swarm", "Sending message to remote node via Pico channel", map[string]any{
		"target":    targetNodeID,
		"pico_addr": picoAddr,
		"load":      target.Node.LoadScore,
	})

	// Send the message using PicoNodeClient
	response, err := al.swarmPicoClient.SendMessage(
		ctx,
		picoAddr,
		targetNodeID,
		content,
		msg.Channel,
		msg.ChatID,
		msg.SenderID,
	)
	if err != nil {
		return fmt.Sprintf("❌ Failed to send message to node '%s': %v", targetNodeID, err), nil
	}

	return response, nil
}

// initSwarm initializes the swarm mode components.
func (al *AgentLoop) initSwarm() {
	logger.InfoC("swarm", "Initializing swarm mode")

	// Create and start discovery service
	swarmConfig := al.convertToSwarmConfig(al.cfg.Swarm)
	discovery, err := swarm.NewDiscoveryService(swarmConfig)
	if err != nil {
		logger.ErrorCF("swarm", "Failed to create discovery service", map[string]any{"error": err.Error()})
		al.swarmInitError = fmt.Errorf("discovery service creation failed: %w", err)
		return
	}
	if err := discovery.Start(); err != nil {
		logger.ErrorCF("swarm", "Failed to start discovery service", map[string]any{"error": err.Error()})
		al.swarmInitError = fmt.Errorf("discovery service start failed: %w", err)
		return
	}
	al.swarmDiscovery = discovery

	// Create handoff coordinator
	al.swarmHandoff = swarm.NewHandoffCoordinator(discovery, al.convertHandoffConfig())

	// Create and start load monitor
	al.swarmLoad = swarm.NewLoadMonitor(al.convertLoadMonitorConfig())
	if al.cfg.Swarm.LoadMonitor.Enabled {
		al.swarmLoad.Start()
		al.swarmLoad.OnThreshold(func(score float64) {
			discovery.UpdateLoad(score)
		})
	}

	al.swarmEnabled = true

	// Initialize leader election if enabled
	if al.cfg.Swarm.LeaderElection.Enabled {
		al.initSwarmLeaderElection(discovery)
	}

	// Initialize inter-node communication via Pico channel
	al.initSwarmPicoClient(discovery)

	// Register swarm tools
	al.registerSwarmTools(discovery)

	// Subscribe to node events for logging
	al.subscribeSwarmEvents(discovery)

	logger.InfoCF("swarm", "Swarm mode initialized", map[string]any{
		"node_id":   discovery.LocalNode().ID,
		"bind_addr": al.cfg.Swarm.BindAddr,
		"bind_port": al.cfg.Swarm.BindPort,
		"handoff":   al.cfg.Swarm.Handoff.Enabled,
	})
}

// initSwarmLeaderElection initializes the leader election module.
func (al *AgentLoop) initSwarmLeaderElection(discovery *swarm.DiscoveryService) {
	// Get membership manager from discovery service
	membership := discovery.GetMembershipManager()
	if membership == nil {
		logger.WarnCF("swarm", "Membership manager not available, leader election disabled", nil)
		return
	}

	// Convert config.SwarmLeaderElectionConfig to swarm.LeaderElectionConfig
	leaderElectionConfig := swarm.LeaderElectionConfig{
		Enabled: al.cfg.Swarm.LeaderElection.Enabled,
		ElectionInterval: swarm.Duration{
			Duration: time.Duration(al.cfg.Swarm.LeaderElection.ElectionInterval) * time.Second,
		},
		LeaderHeartbeatTimeout: swarm.Duration{
			Duration: time.Duration(al.cfg.Swarm.LeaderElection.LeaderHeartbeatTimeout) * time.Second,
		},
	}

	// Create leader election instance
	leaderElection := swarm.NewLeaderElection(
		discovery.LocalNode().ID,
		membership,
		leaderElectionConfig,
	)

	// Start leader election
	leaderElection.Start()
	al.swarmLeaderElection = leaderElection

	logger.InfoCF("swarm", "Leader election initialized", map[string]any{
		"node_id": discovery.LocalNode().ID,
		"enabled": al.cfg.Swarm.LeaderElection.Enabled,
	})
}

// initSwarmPicoClient sets up the PicoNodeClient for inter-node communication
// and wires the handoff coordinator to use it.
func (al *AgentLoop) initSwarmPicoClient(discovery *swarm.DiscoveryService) {
	picoToken := al.cfg.Channels.Pico.Token
	if picoToken == "" {
		logger.WarnCF("swarm", "Pico token not configured, inter-node communication disabled", nil)
		return
	}

	al.swarmPicoClient = picolib.NewPicoNodeClient(discovery.LocalNode().ID, picoToken)
	logger.InfoCF("swarm", "PicoNodeClient initialized for inter-node communication", nil)

	al.swarmHandoff.SetSendFunc(
		func(ctx context.Context, targetAddr string, req *swarm.HandoffRequest) (*swarm.HandoffResponse, error) {
			return al.sendHandoffViaPico(ctx, targetAddr, req)
		},
	)
}

// sendHandoffViaPico sends a handoff request to a remote node via the Pico channel
// and parses the response. Uses a mapstructure decoder to handle type mismatches gracefully.
func (al *AgentLoop) sendHandoffViaPico(
	ctx context.Context,
	targetAddr string,
	req *swarm.HandoffRequest,
) (*swarm.HandoffResponse, error) {
	payload := swarm.NewHandoffRequestPayload(req)
	replyPayload, err := al.swarmPicoClient.SendNodeAction(ctx, targetAddr, payload)
	if err != nil {
		return nil, fmt.Errorf("handoff request via Pico failed: %w", err)
	}

	respData, ok := replyPayload.RawValue(picolib.PayloadKeyHandoffResp)
	if !ok {
		return nil, fmt.Errorf("missing handoff_response in reply")
	}

	// Type-assert respData to map[string]any first
	respMap, ok := respData.(map[string]any)
	if !ok {
		// Fall back to JSON marshal/unmarshal if not a map
		respJSON, err := json.Marshal(respData)
		if err != nil {
			return nil, fmt.Errorf("failed to convert handoff response to JSON: %w", err)
		}
		var resp swarm.HandoffResponse
		if err := json.Unmarshal(respJSON, &resp); err != nil {
			return nil, fmt.Errorf("failed to parse handoff response: %w", err)
		}
		return &resp, nil
	}

	// Direct struct assignment from map with type coercion
	resp := &swarm.HandoffResponse{
		RequestID:  toString(respMap["request_id"]),
		Accepted:   toBool(respMap["accepted"]),
		NodeID:     toString(respMap["node_id"]),
		Reason:     toString(respMap["reason"]),
		SessionKey: toString(respMap["session_key"]),
		Timestamp:  toInt64(respMap["timestamp"]),
		State:      swarm.HandoffState(toString(respMap["state"])),
	}

	return resp, nil
}

// Helper functions for type coercion to handle json.Number and other types
func toString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case json.Number:
		return val.String()
	default:
		return ""
	}
}

func toBool(v any) bool {
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val == "true" || val == "1"
	case float64:
		return val != 0
	default:
		return false
	}
}

func toInt64(v any) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case float64:
		return int64(val)
	case json.Number:
		i, _ := val.Int64()
		return i
	case string:
		i, _ := strconv.ParseInt(val, 10, 64)
		return i
	default:
		return 0
	}
}

// sendMessageToNode resolves a target node by ID from the discovery service
// and sends a message to it via the Pico channel.
func (al *AgentLoop) sendMessageToNode(
	ctx context.Context,
	targetNodeID, content, channel, chatID, senderID string,
) (string, error) {
	members := al.swarmDiscovery.Members()
	for _, m := range members {
		if m.Node.ID == targetNodeID {
			if m.Node.HTTPPort == 0 {
				return "", fmt.Errorf("node %s does not have an HTTP port configured", targetNodeID)
			}
			picoAddr := picolib.BuildNodeAddr(m.Node.Addr, m.Node.HTTPPort)
			return al.swarmPicoClient.SendMessage(ctx, picoAddr, targetNodeID, content, channel, chatID, senderID)
		}
	}
	return "", fmt.Errorf("node %s not found", targetNodeID)
}

// registerSwarmTools registers swarm-related tools (handoff, routing).
// Note: /nodes is registered as a command (not a tool) via registerSwarmCommands.
func (al *AgentLoop) registerSwarmTools(discovery *swarm.DiscoveryService) {
	localNodeID := discovery.LocalNode().ID

	if !al.cfg.Swarm.Handoff.Enabled {
		return
	}

	handoffTool := tools.NewHandoffTool(al.swarmHandoff)
	al.RegisterTool(handoffTool)

	routeTool := tools.NewSwarmRouteTool(discovery, al.swarmHandoff, localNodeID)
	if al.swarmPicoClient != nil {
		routeTool.SetSendMessageFn(al.sendMessageToNode)
	}
	al.RegisterTool(routeTool)
}

// registerSwarmCommands registers swarm slash commands on the channel command registry.
// Called from SetChannelManager after both swarm and channel manager are initialized.
func (al *AgentLoop) registerSwarmCommands() {
	if al.commandRegistry == nil || al.swarmDiscovery == nil {
		return
	}

	localNodeID := al.swarmDiscovery.LocalNode().ID

	al.commandRegistry.Register("nodes", "List swarm cluster nodes", func(
		ctx context.Context, args string, msg bus.InboundMessage,
	) (string, error) {
		verbose := strings.Contains(args, "verbose") || strings.Contains(args, "-v")
		return tools.FormatClusterStatus(al.swarmDiscovery, al.swarmLoad, localNodeID, verbose), nil
	})
}

// subscribeSwarmEvents subscribes to discovery events and logs node state changes.
func (al *AgentLoop) subscribeSwarmEvents(discovery *swarm.DiscoveryService) {
	discovery.Subscribe(func(event *swarm.NodeEvent) {
		switch event.Event {
		case swarm.EventJoin:
			logger.InfoCF("swarm", "Node joined", map[string]any{"node_id": event.Node.ID})
		case swarm.EventLeave:
			logger.InfoCF("swarm", "Node left", map[string]any{"node_id": event.Node.ID})
		case swarm.EventUpdate:
			logger.DebugCF("swarm", "Node updated", map[string]any{
				"node_id":    event.Node.ID,
				"load_score": event.Node.LoadScore,
			})
		}
	})
}

// convertHandoffConfig converts config.SwarmHandoffConfig to swarm.HandoffConfig.
func (al *AgentLoop) convertHandoffConfig() swarm.HandoffConfig {
	cfg := al.cfg.Swarm.Handoff
	return swarm.HandoffConfig{
		Enabled:       cfg.Enabled,
		LoadThreshold: cfg.LoadThreshold,
		Timeout:       swarm.Duration{Duration: time.Duration(cfg.Timeout) * time.Second},
		MaxRetries:    cfg.MaxRetries,
		RetryDelay:    swarm.Duration{Duration: time.Duration(cfg.RetryDelay) * time.Second},
	}
}

// convertLoadMonitorConfig converts config.SwarmLoadMonitorConfig to *swarm.LoadMonitorConfig.
func (al *AgentLoop) convertLoadMonitorConfig() *swarm.LoadMonitorConfig {
	cfg := al.cfg.Swarm.LoadMonitor
	return &swarm.LoadMonitorConfig{
		Enabled:       cfg.Enabled,
		Interval:      swarm.Duration{Duration: time.Duration(cfg.Interval) * time.Second},
		SampleSize:    cfg.SampleSize,
		CPUWeight:     cfg.CPUWeight,
		MemoryWeight:  cfg.MemoryWeight,
		SessionWeight: cfg.SessionWeight,
	}
}

// convertToSwarmConfig converts the config.SwarmConfig to swarm.Config.
func (al *AgentLoop) convertToSwarmConfig(cfg config.SwarmConfig) *swarm.Config {
	// Helper function to apply defaults for duration configs
	applyDurationDefault := func(val int, defaultVal time.Duration) time.Duration {
		if val <= 0 {
			return defaultVal
		}
		return time.Duration(val) * time.Second
	}

	return &swarm.Config{
		Enabled:       cfg.Enabled,
		NodeID:        cfg.NodeID,
		BindAddr:      cfg.BindAddr,
		BindPort:      cfg.BindPort,
		AdvertiseAddr: cfg.AdvertiseAddr,
		Discovery: swarm.DiscoveryConfig{
			JoinAddrs: cfg.Discovery.JoinAddrs,
			GossipInterval: swarm.Duration{
				Duration: applyDurationDefault(cfg.Discovery.GossipInterval, swarm.DefaultGossipInterval),
			},
			PushPullInterval: swarm.Duration{
				Duration: applyDurationDefault(cfg.Discovery.PushPullInterval, swarm.DefaultPushPullInterval),
			},
			NodeTimeout: swarm.Duration{
				Duration: applyDurationDefault(cfg.Discovery.NodeTimeout, swarm.DefaultNodeTimeout),
			},
			DeadNodeTimeout: swarm.Duration{
				Duration: applyDurationDefault(cfg.Discovery.DeadNodeTimeout, swarm.DefaultDeadNodeTimeout),
			},
		},
		Handoff: swarm.HandoffConfig{
			Enabled:       cfg.Handoff.Enabled,
			LoadThreshold: cfg.Handoff.LoadThreshold,
			Timeout: swarm.Duration{
				Duration: applyDurationDefault(cfg.Handoff.Timeout, swarm.DefaultHandoffTimeout),
			},
			MaxRetries: cfg.Handoff.MaxRetries,
			RetryDelay: swarm.Duration{
				Duration: applyDurationDefault(cfg.Handoff.RetryDelay, swarm.DefaultHandoffRetryDelay),
			},
		},
		RPC: swarm.RPCConfig{
			Port: cfg.RPC.Port,
			Timeout: swarm.Duration{
				Duration: applyDurationDefault(cfg.RPC.Timeout, 10*time.Second),
			},
		},
		LoadMonitor: swarm.LoadMonitorConfig{
			Enabled: cfg.LoadMonitor.Enabled,
			Interval: swarm.Duration{
				Duration: applyDurationDefault(cfg.LoadMonitor.Interval, swarm.DefaultLoadSampleInterval),
			},
			SampleSize:    cfg.LoadMonitor.SampleSize,
			CPUWeight:     cfg.LoadMonitor.CPUWeight,
			MemoryWeight:  cfg.LoadMonitor.MemoryWeight,
			SessionWeight: cfg.LoadMonitor.SessionWeight,
		},
		LeaderElection: swarm.LeaderElectionConfig{
			Enabled: cfg.LeaderElection.Enabled,
			ElectionInterval: swarm.Duration{
				Duration: applyDurationDefault(cfg.LeaderElection.ElectionInterval, 5*time.Second),
			},
			LeaderHeartbeatTimeout: swarm.Duration{
				Duration: applyDurationDefault(cfg.LeaderElection.LeaderHeartbeatTimeout, 10*time.Second),
			},
		},
		HTTPPort: al.cfg.Gateway.Port,
	}
}

// handleIncomingHandoffRequest handles a handoff_request action from a remote node.
func (al *AgentLoop) handleIncomingHandoffRequest(payload picolib.NodePayload) (map[string]any, error) {
	if al.swarmHandoff == nil {
		return picolib.ErrorReply("handoff coordinator not available"), nil
	}

	// Parse request from payload
	reqData, ok := payload.RawValue(picolib.PayloadKeyRequest)
	if !ok {
		return picolib.ErrorReply("missing request in handoff payload"), nil
	}

	reqJSON, err := json.Marshal(reqData)
	if err != nil {
		return picolib.ErrorReply(fmt.Sprintf("failed to marshal request: %v", err)), nil
	}

	var req swarm.HandoffRequest
	if err := json.Unmarshal(reqJSON, &req); err != nil {
		return picolib.ErrorReply(fmt.Sprintf("failed to parse handoff request: %v", err)), nil
	}

	resp := al.swarmHandoff.HandleIncomingHandoff(&req)

	return swarm.HandoffResponseReply(resp), nil
}

// handleIncomingNodeMessage handles a message received from another node via Pico channel.
func (al *AgentLoop) handleIncomingNodeMessage(msg *picolib.DirectMessage) (string, error) {
	logger.InfoCF("swarm", "Processing incoming node message", map[string]any{
		"from":       msg.SourceNodeID,
		"message_id": msg.MessageID,
		"content":    msg.Content[:min(50, len(msg.Content))],
	})

	// Create an inbound message from the node message
	inboundMsg := bus.InboundMessage{
		Content:  msg.Content,
		Channel:  msg.Channel,
		ChatID:   msg.ChatID,
		SenderID: msg.SenderID,
	}

	// Process the message through the agent loop
	ctx := context.Background()
	response, err := al.processMessage(ctx, inboundMsg)
	if err != nil {
		return "", fmt.Errorf("failed to process message: %w", err)
	}

	return response, nil
}

// shouldHandoff determines if the current request should be handed off to another node.
func (al *AgentLoop) shouldHandoff(agent *AgentInstance, opts processOptions) bool {
	if !al.swarmEnabled || al.swarmHandoff == nil {
		return false
	}

	// Check if load is too high
	if al.swarmLoad != nil && al.swarmLoad.ShouldOffload() {
		logger.InfoCF("swarm", "Load threshold exceeded, considering handoff", map[string]any{
			"load_score": al.swarmLoad.GetCurrentLoad().Score,
		})
		return true
	}

	return false
}

// UpdateSwarmLoad updates the current load score reported to the swarm.
func (al *AgentLoop) UpdateSwarmLoad(sessionCount int) {
	if al.swarmLoad != nil {
		al.swarmLoad.SetSessionCount(sessionCount)
	}
}

// IncrementSwarmSessions increments the active session count.
func (al *AgentLoop) IncrementSwarmSessions() {
	if al.swarmLoad != nil {
		al.swarmLoad.IncrementSessions()
	}
}

// DecrementSwarmSessions decrements the active session count.
func (al *AgentLoop) DecrementSwarmSessions() {
	if al.swarmLoad != nil {
		al.swarmLoad.DecrementSessions()
	}
}

// GetSwarmStatus returns the current swarm status.
func (al *AgentLoop) GetSwarmStatus() map[string]any {
	if !al.swarmEnabled {
		return map[string]any{"enabled": false}
	}

	status := map[string]any{
		"enabled": true,
		"node_id": al.swarmDiscovery.LocalNode().ID,
		"handoff": al.cfg.Swarm.Handoff.Enabled,
	}

	if al.swarmLoad != nil {
		metrics := al.swarmLoad.GetCurrentLoad()
		status["load"] = map[string]any{
			"score":           metrics.Score,
			"cpu_usage":       metrics.CPUUsage,
			"memory_usage":    metrics.MemoryUsage,
			"active_sessions": metrics.ActiveSessions,
			"goroutines":      metrics.Goroutines,
			"trend":           al.swarmLoad.GetTrend(),
		}
	}

	if al.swarmDiscovery != nil {
		members := al.swarmDiscovery.Members()
		status["members"] = len(members)
	}

	return status
}

// ShutdownSwarm gracefully shuts down the swarm components.
func (al *AgentLoop) ShutdownSwarm() error {
	if !al.swarmEnabled {
		return nil
	}

	var errs []string

	if al.swarmLoad != nil {
		al.swarmLoad.Stop()
	}

	if al.swarmLeaderElection != nil {
		al.swarmLeaderElection.Stop()
	}

	if al.swarmHandoff != nil {
		if err := al.swarmHandoff.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("handoff: %v", err))
		}
	}

	if al.swarmDiscovery != nil {
		if err := al.swarmDiscovery.Stop(); err != nil {
			errs = append(errs, fmt.Sprintf("discovery: %v", err))
		}
	}

	al.swarmEnabled = false

	if len(errs) > 0 {
		return fmt.Errorf("swarm shutdown errors: %s", strings.Join(errs, ", "))
	}
	return nil
}

// initiateSwarmHandoff initiates a handoff to another node.
func (al *AgentLoop) initiateSwarmHandoff(
	ctx context.Context,
	agent *AgentInstance,
	sessionKey string,
	msg bus.InboundMessage,
) (*swarm.HandoffResponse, error) {
	if al.swarmHandoff == nil {
		return nil, swarm.ErrDiscoveryDisabled
	}

	// Build session history for handoff
	sessionMessages := make([]picolib.SessionMessage, 0)
	history := agent.Sessions.GetHistory(sessionKey)

	for _, m := range history {
		if m.Role == "user" || m.Role == "assistant" {
			sessionMessages = append(sessionMessages, picolib.SessionMessage{
				Role:    m.Role,
				Content: m.Content,
			})
		}
	}

	// Create handoff request
	req := &swarm.HandoffRequest{
		Reason:          swarm.ReasonOverloaded,
		SessionKey:      sessionKey,
		SessionMessages: sessionMessages,
		Context: map[string]any{
			"channel":  msg.Channel,
			"chat_id":  msg.ChatID,
			"sender":   msg.SenderID,
			"agent_id": agent.ID,
		},
		Metadata: map[string]string{
			"original_channel": msg.Channel,
			"original_chat_id": msg.ChatID,
		},
	}

	logger.InfoCF("swarm", "Initiating handoff", map[string]any{
		"session_key": sessionKey,
		"reason":      req.Reason,
		"history_len": len(sessionMessages),
	})

	// Execute handoff
	resp, err := al.swarmHandoff.InitiateHandoff(ctx, req)

	if resp != nil {
		logger.InfoCF("swarm", "Handoff response received", map[string]any{
			"accepted": resp.Accepted,
			"node_id":  resp.NodeID,
			"state":    resp.State,
		})
	}

	return resp, err
}
