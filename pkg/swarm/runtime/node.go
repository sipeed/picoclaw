package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/sipeed/picoclaw/pkg/swarm/config"
	"github.com/sipeed/picoclaw/pkg/swarm/core"
	"github.com/sipeed/picoclaw/pkg/tools"
)

type NodeActor struct {
	Data   *core.Node
	Bus    core.EventBus
	Store  core.SwarmStore
	LLM    core.LLMClient
	Tools  *tools.ToolRegistry
	Policy *core.PolicyChecker
	Config config.SwarmConfig

	peerInsights []string // Buffer for thoughts heard from peers
}

func NewNodeActor(data *core.Node, bus core.EventBus, store core.SwarmStore, llm core.LLMClient, toolRegistry *tools.ToolRegistry, policy *core.PolicyChecker, cfg config.SwarmConfig) *NodeActor {
	return &NodeActor{
		Data:         data,
		Bus:          bus,
		Store:        store,
		LLM:          llm,
		Tools:        toolRegistry,
		Policy:       policy,
		Config:       cfg,
		peerInsights: make([]string, 0),
	}
}

func (n *NodeActor) Start(ctx context.Context) {
	n.Data.Status = core.NodeStatusRunning
	if err := n.Store.UpdateNode(ctx, n.Data); err != nil {
		slog.Error("Failed to update node status to running", "node", n.Data.ID, "error", err)
	}

	// Subscribe to peer events
	sub, _ := n.Bus.Subscribe("node.events", func(e core.Event) {
		if e.SwarmID == n.Data.SwarmID && e.NodeID != n.Data.ID {
			if e.Type == core.EventNodeThinking {
				content, _ := e.Payload["content"].(string)
				if len(content) > 10 { // Only record significant thoughts
					n.peerInsights = append(n.peerInsights, fmt.Sprintf("Peer %s: %s", e.NodeID[:4], content))
				}
			}
		}
	})

	go func() {
		defer sub.Unsubscribe()
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Node panic recovered", "node", n.Data.ID, "err", r)
			}
		}()
		n.run(ctx)
	}()
}

func (n *NodeActor) run(ctx context.Context) {
	messages := []core.Message{
		{Role: "system", Content: n.Data.Role.SystemPrompt},
		{Role: "user", Content: n.Data.Task},
	}

	allowedTools := n.getToolsForRole()
	model := n.Data.Role.Model

	maxIterations := n.Config.Limits.MaxIterations
	if maxIterations <= 0 {
		maxIterations = 10
	}

	summarizeThreshold := n.Config.Limits.SummarizeThreshold
	if summarizeThreshold <= 0 {
		summarizeThreshold = 20
	}

	for i := 0; i < maxIterations; i++ {
		// Progressive Summarization: If history > threshold messages, compress the middle part
		if len(messages) > summarizeThreshold {
			slog.Info("Context threshold reached, performing progressive summarization", "node", n.Data.ID)

			// 1. Identify context to summarize
			// Header: System (0) + User Goal (1)
			// Tail: Last N messages
			keepCount := n.Config.Limits.PruningMsgKeep
			if keepCount <= 0 {
				keepCount = 6
			}

			if len(messages) > keepCount+2 {
				toSummarize := messages[2 : len(messages)-keepCount]

				summaryPrompt := "Briefly summarize the key findings, data points, and progress from the conversation above. Focus on facts discovered so far. Be very concise."
				summaryMessages := append(toSummarize, core.Message{Role: "user", Content: summaryPrompt})

				summaryResp, err := n.LLM.Chat(ctx, summaryMessages, nil, model)
				if err == nil {
					// 2. Reconstruct history: [Header] + [Summary] + [Tail]
					tail := messages[len(messages)-keepCount:]
					newHistory := make([]core.Message, 0, keepCount+3)
					newHistory = append(newHistory, messages[0:2]...) // Keep System + Goal
					newHistory = append(newHistory, core.Message{
						Role:    "system",
						Content: fmt.Sprintf("Previous context summary: %s", summaryResp.Content),
					})
					messages = append(newHistory, tail...)

					n.Bus.Publish("node.events", core.Event{
						Type: core.EventNodeThinking, SwarmID: n.Data.SwarmID, NodeID: n.Data.ID,
						Payload: map[string]any{"content": "🧠 I've summarized my previous findings to keep my memory sharp."},
					})
				}
			}
		}

		// Inject Peer Insights
		if len(n.peerInsights) > 0 {
			insightMsg := "Peer insights so far:\n"
			for _, in := range n.peerInsights {
				insightMsg += "- " + in + "\n"
			}
			messages = append(messages, core.Message{Role: "system", Content: insightMsg})
			n.peerInsights = nil
		}

		resp, err := n.LLM.Chat(ctx, messages, allowedTools, model)
		if err != nil {
			n.fail(ctx, err)
			return
		}

		n.Data.Stats.Iterations++
		n.Data.Stats.TokensInput += resp.Usage.Input
		n.Data.Stats.TokensOutput += resp.Usage.Output

		messages = append(messages, core.Message{Role: "assistant", Content: resp.Content})

		if resp.Content != "" {
			n.Bus.Publish("node.events", core.Event{
				Type: core.EventNodeThinking, SwarmID: n.Data.SwarmID, NodeID: n.Data.ID,
				Payload: map[string]any{"content": resp.Content},
			})
		}

		if len(resp.ToolCalls) == 0 {
			n.complete(ctx, resp.Content)
			return
		}

		// Process Tools
		for _, tc := range resp.ToolCalls {
			result := n.execTool(ctx, tc)
			messages = append(messages, core.Message{Role: "tool", Content: result, ToolCallID: tc.ID})
		}
	}
	n.fail(ctx, fmt.Errorf("max iterations reached"))
}

func (n *NodeActor) execTool(ctx context.Context, tc core.ToolCall) string {
	if !n.Policy.CanUseTool(n.Data.Role.Name, tc.Name) {
		return "Error: Unauthorized tool"
	}

	var args map[string]any
	json.Unmarshal(tc.Arguments, &args)

	out, err := n.Tools.Execute(ctx, tc.Name, args)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return out
}

func (n *NodeActor) getToolsForRole() []core.ToolDef {
	defs := n.Tools.GetDefinitions()
	var allowed []core.ToolDef
	for _, d := range defs {
		f := d["function"].(map[string]any)
		name := f["name"].(string)
		if n.Policy.CanUseTool(n.Data.Role.Name, name) {
			allowed = append(allowed, core.ToolDef{
				Name: name, Description: f["description"].(string), Parameters: f["parameters"],
			})
		}
	}
	return allowed
}

func (n *NodeActor) complete(ctx context.Context, out string) {
	n.Data.Output = out
	n.Data.Status = core.NodeStatusCompleted
	if err := n.Store.UpdateNode(ctx, n.Data); err != nil {
		slog.Error("Failed to update node status to completed", "node", n.Data.ID, "error", err)
	}
	n.Bus.Publish("node.events", core.Event{
		Type: core.EventNodeCompleted, SwarmID: n.Data.SwarmID, NodeID: n.Data.ID,
		Payload: map[string]any{"output": out},
	})
}

func (n *NodeActor) fail(ctx context.Context, err error) {
	n.Data.Status = core.NodeStatusFailed
	if errUpdate := n.Store.UpdateNode(ctx, n.Data); errUpdate != nil {
		slog.Error("Failed to update node status to failed", "node", n.Data.ID, "error", errUpdate)
	}
	n.Bus.Publish("node.events", core.Event{
		Type: core.EventNodeFailed, SwarmID: n.Data.SwarmID, NodeID: n.Data.ID,
		Payload: map[string]any{"error": err.Error()},
	})
}