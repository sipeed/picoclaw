package swarm

import (
	"context"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/swarm/adapters"
	"github.com/sipeed/picoclaw/pkg/swarm/bus"
	"github.com/sipeed/picoclaw/pkg/swarm/config"
	"github.com/sipeed/picoclaw/pkg/swarm/core"
	"github.com/sipeed/picoclaw/pkg/swarm/memory"
	"github.com/sipeed/picoclaw/pkg/swarm/runtime"
	"github.com/sipeed/picoclaw/pkg/tools"
)

type Service struct {
	Orchestrator *runtime.Orchestrator
	Store        core.SwarmStore
	Bus          core.EventBus
	Outbound     chan string
}

func NewService(dbPath string, provider providers.LLMProvider, registry *tools.ToolRegistry, cfg config.SwarmConfig, model string) (*Service, error) {
	store, _ := memory.NewSQLiteStore(dbPath)
	eventBus := bus.NewChannelBus()
	adapter := adapters.NewLLMAdapter(provider)

	var sharedMem core.SharedMemory = store
	if m, err := memory.NewChromemStore(context.Background(), adapter); err == nil {
		sharedMem = m
	}

	orch := runtime.NewOrchestrator(store, eventBus, adapter, registry, cfg, model)
	orch.SetSharedMemory(sharedMem)

	s := &Service{Orchestrator: orch, Store: store, Bus: eventBus, Outbound: make(chan string, 100)}
	s.listen()
	return s, nil
}

func (s *Service) listen() {
	if _, err := s.Bus.Subscribe("node.events", func(e core.Event) {
		msg := ""
		switch e.Type {
		case core.EventNodeThinking: msg = fmt.Sprintf("🤖 [%s]: %s", e.NodeID[:4], e.Payload["content"])
		case core.EventNodeCompleted: msg = fmt.Sprintf("✅ [%s] Done.", e.NodeID[:4])
		case core.EventNodeFailed: msg = fmt.Sprintf("❌ [%s] Failed: %v", e.NodeID[:4], e.Payload["error"])
		}
		if msg != "" {
			select { case s.Outbound <- msg: default: }
		}
	}); err != nil {
		fmt.Printf("Warning: Swarm event subscription failed: %v\n", err)
	}
}

func (s *Service) HandleCommand(ctx context.Context, input string) string {
	args := strings.Fields(input)
	if len(args) < 2 { return "Usage: /swarm <spawn|list|stop|status|viz> [goal/id]" }

	switch args[1] {
	case "spawn":
		goal := strings.Join(args[2:], " ")
		id, err := s.Orchestrator.SpawnSwarm(ctx, goal)
		if err != nil {
			return fmt.Sprintf("❌ Failed to spawn swarm: %v", err)
		}
		return fmt.Sprintf("🚀 Swarm ID: %s", id)
	case "list":
		swarms, err := s.Store.ListSwarms(ctx, core.SwarmStatusActive)
		if err != nil {
			return fmt.Sprintf("❌ Failed to list swarms: %v", err)
		}
		out := "Active Swarms:\n"
		for _, sw := range swarms { out += fmt.Sprintf("- %s: %s\n", sw.ID, sw.Goal) }
		return out
	case "stop":
		if len(args) < 3 { return "ID required" }
		s.Orchestrator.StopSwarm(core.SwarmID(args[2]))
		return "Stopped."
	case "status":
		if len(args) < 3 { return "ID required" }
		sid := core.SwarmID(args[2])
		sw, err := s.Store.GetSwarm(ctx, sid)
		if err != nil { return "Swarm not found" }
		nodes, _ := s.Store.GetSwarmNodes(ctx, sid)
		
		out := fmt.Sprintf("📊 Swarm Status: %s\nGoal: %s\nNodes (%d):\n", sw.Status, sw.Goal, len(nodes))
		for _, n := range nodes {
			out += fmt.Sprintf("- [%s] %s: %s (%s)\n", n.ID[:4], n.Role.Name, n.Status, n.Task)
		}
		return out
	case "viz":
		if len(args) < 3 { return "ID required" }
		sid := core.SwarmID(args[2])
		nodes, _ := s.Store.GetSwarmNodes(ctx, sid)
		if len(nodes) == 0 { return "No nodes found" }

		out := "```mermaid\ngraph TD\n"
		for _, n := range nodes {
			nodeLabel := fmt.Sprintf("%s[%s: %s]", n.ID[:8], n.Role.Name, n.Status)
			out += fmt.Sprintf("  %s\n", nodeLabel)
			if n.ParentID != "" {
				out += fmt.Sprintf("  %s --> %s\n", n.ParentID[:8], n.ID[:8])
			}
		}
		out += "```"
		return out
	}
	return "Unknown command"
}

func (s *Service) Stop() {
	if s.Orchestrator != nil {
		s.Orchestrator.StopAll()
	}
}