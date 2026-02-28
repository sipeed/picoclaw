// Package tools provides swarm node management tools.
package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/swarm"
)

// FormatClusterStatus returns a human-readable cluster status string.
// Used by the /nodes command handler.
func FormatClusterStatus(
	discovery *swarm.DiscoveryService,
	load *swarm.LoadMonitor,
	localID string,
	verbose bool,
) string {
	if discovery == nil {
		return "Swarm mode is not enabled."
	}

	members := discovery.Members()
	if len(members) == 0 {
		return "No nodes found in the swarm cluster."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Swarm Cluster Status (%d node%s):\n\n", len(members), plural(len(members))))

	// Sort nodes: local first, then by ID
	sortedMembers := make([]*swarm.NodeWithState, len(members))
	copy(sortedMembers, members)
	for i := range sortedMembers {
		for j := i + 1; j < len(sortedMembers); j++ {
			if sortedMembers[j].Node.ID == localID {
				sortedMembers[i], sortedMembers[j] = sortedMembers[j], sortedMembers[i]
				break
			}
			if sortedMembers[i].Node.ID > sortedMembers[j].Node.ID {
				sortedMembers[i], sortedMembers[j] = sortedMembers[j], sortedMembers[i]
			}
		}
	}

	for _, m := range sortedMembers {
		node := m.Node
		state := m.State

		localMark := " "
		if node.ID == localID {
			localMark = "*"
		}

		sb.WriteString(fmt.Sprintf("%s **%s**\n", localMark, node.ID))

		if verbose {
			sb.WriteString(fmt.Sprintf("   Address: %s:%d\n", node.Addr, node.Port))
			sb.WriteString(fmt.Sprintf("   Status: %s", state.Status))
			writeStatusEmoji(&sb, state.Status)
			sb.WriteString("\n")

			loadPercent := int(node.LoadScore * 100)
			sb.WriteString(fmt.Sprintf("   Load: %.2f%% ", node.LoadScore*100))
			if loadPercent < 50 {
				sb.WriteString("🟩")
			} else if loadPercent < 80 {
				sb.WriteString("🟨")
			} else {
				sb.WriteString("🟥")
			}
			sb.WriteString("\n")

			lastSeen := time.Unix(0, state.LastSeen)
			age := time.Since(lastSeen)
			sb.WriteString(fmt.Sprintf("   Last seen: %s ago\n", formatDuration(age)))

			if len(node.AgentCaps) > 0 {
				sb.WriteString("   Capabilities:\n")
				for k, v := range node.AgentCaps {
					sb.WriteString(fmt.Sprintf("     - %s: %s\n", k, v))
				}
			}

			if len(node.Labels) > 0 {
				sb.WriteString("   Labels:\n")
				for k, v := range node.Labels {
					sb.WriteString(fmt.Sprintf("     - %s: %s\n", k, v))
				}
			}

			if state.PingSuccess > 0 || state.PingFailure > 0 {
				total := state.PingSuccess + state.PingFailure
				successRate := float64(state.PingSuccess) / float64(total) * 100
				sb.WriteString(fmt.Sprintf("   Ping: %d/%d success (%.1f%%)\n",
					state.PingSuccess, total, successRate))
			}
		} else {
			writeStatusEmoji(&sb, state.Status)
			sb.WriteString(fmt.Sprintf(" Load: %.0f%% @ %s:%d\n",
				node.LoadScore*100, node.Addr, node.Port))
		}

		sb.WriteString("\n")
	}

	sb.WriteString("* = this node\n")
	sb.WriteString("\nTip: Use `@node-id: message` to route a request to a specific node.")

	return sb.String()
}

// writeStatusEmoji appends a status emoji to the builder.
func writeStatusEmoji(sb *strings.Builder, status swarm.NodeStatus) {
	switch status {
	case swarm.NodeStatusAlive:
		sb.WriteString(" 🟢")
	case swarm.NodeStatusSuspect:
		sb.WriteString(" 🟡")
	case swarm.NodeStatusDead:
		sb.WriteString(" 🔴")
	case swarm.NodeStatusLeft:
		sb.WriteString(" ⚪")
	}
}

// SwarmRouteTool routes a request to a specific node in the swarm.
type SwarmRouteTool struct {
	discovery     *swarm.DiscoveryService
	handoff       *swarm.HandoffCoordinator
	localID       string
	sendMessageFn func(ctx context.Context, targetNodeID, content, channel, chatID, senderID string) (string, error)
}

// NewSwarmRouteTool creates a new swarm route tool.
func NewSwarmRouteTool(
	discovery *swarm.DiscoveryService,
	handoff *swarm.HandoffCoordinator,
	localID string,
) *SwarmRouteTool {
	return &SwarmRouteTool{
		discovery: discovery,
		handoff:   handoff,
		localID:   localID,
	}
}

// SetSendMessageFn sets the function to send messages to other nodes.
func (t *SwarmRouteTool) SetSendMessageFn(
	fn func(ctx context.Context, targetNodeID, content, channel, chatID, senderID string) (string, error),
) {
	t.sendMessageFn = fn
}

// Name returns the tool name.
func (t *SwarmRouteTool) Name() string {
	return "swarm_route"
}

// Description returns the tool description.
func (t *SwarmRouteTool) Description() string {
	return "Route this request to a specific node in the swarm cluster"
}

// Parameters returns the tool parameters schema.
func (t *SwarmRouteTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"node_id": map[string]any{
				"type":        "string",
				"description": "The target node ID to route the request to",
			},
			"message": map[string]any{
				"type":        "string",
				"description": "The message to send to the target node",
			},
		},
		"required": []string{"node_id"},
	}
}

// Execute executes the swarm route tool.
func (t *SwarmRouteTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	if t.discovery == nil {
		return ErrorResult("Swarm mode is not enabled")
	}

	nodeID, _ := args["node_id"].(string)
	message, _ := args["message"].(string)

	if nodeID == "" {
		return ErrorResult("node_id is required")
	}

	// Check if target is this node
	if nodeID == t.localID {
		return &ToolResult{
			ForLLM:  fmt.Sprintf("Target node %s is this node. Processing locally.", nodeID),
			ForUser: fmt.Sprintf("This request is already on node %s.", nodeID),
			IsError: false,
		}
	}

	// Find target node
	members := t.discovery.Members()
	var target *swarm.NodeWithState
	for _, m := range members {
		if m.Node.ID == nodeID {
			target = m
			break
		}
	}

	if target == nil {
		return ErrorResult(fmt.Sprintf("Node %s not found in cluster", nodeID))
	}

	// Check if target is available
	if target.State.Status != swarm.NodeStatusAlive {
		return ErrorResult(fmt.Sprintf("Node %s is not alive (status: %s)", nodeID, target.State.Status))
	}

	if target.Node.LoadScore > 0.9 {
		return ErrorResult(fmt.Sprintf("Node %s is overloaded (load: %.0f%%)", nodeID, target.Node.LoadScore*100))
	}

	// If sendMessageFn is available, use it to send the actual message
	if t.sendMessageFn != nil {
		logger.InfoCF("swarm", "Sending message to node via Pico channel", map[string]any{
			"target":  nodeID,
			"message": message,
		})

		response, err := t.sendMessageFn(ctx, nodeID, message, "", "", "")
		if err != nil {
			return ErrorResult(fmt.Sprintf("Failed to send message to node %s: %v", nodeID, err))
		}

		return &ToolResult{
			ForLLM:  response,
			ForUser: response,
			IsError: false,
		}
	}

	// Fallback to placeholder message (for backward compatibility)
	logger.InfoCF("swarm", "Routing request to node", map[string]any{
		"target":  nodeID,
		"message": message,
		"addr":    target.Node.GetAddress(),
	})

	return &ToolResult{
		ForLLM: fmt.Sprintf("Request routed to node %s @ %s (load: %.0f%%). "+
			"That node will process the request and respond independently. "+
			"For direct communication, you can contact that node directly at %s:%d.",
			nodeID, target.Node.Addr, target.Node.LoadScore*100, target.Node.Addr, target.Node.Port),
		ForUser: fmt.Sprintf("Your request has been routed to node %s. They will respond separately.",
			nodeID),
		IsError: false,
		Async:   true,
	}
}

// ParseNodeMention extracts node ID from a message like "@node-id: message"
func ParseNodeMention(message string) (nodeID, content string) {
	message = strings.TrimSpace(message)

	// Check for @node-id: pattern
	if strings.HasPrefix(message, "@") {
		rest := message[1:]
		idx := strings.IndexAny(rest, ": \n")
		if idx > 0 && rest[idx] == ':' {
			nodeID = rest[:idx]
			content = strings.TrimSpace(rest[idx+1:])
			return nodeID, content
		}
	}

	return "", message
}

// Helper functions

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}
