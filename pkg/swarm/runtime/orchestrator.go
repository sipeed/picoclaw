package runtime

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sipeed/picoclaw/pkg/swarm/config"
	"github.com/sipeed/picoclaw/pkg/swarm/core"
	"github.com/sipeed/picoclaw/pkg/tools"
)

type Orchestrator struct {
	store         core.SwarmStore
	bus           core.EventBus
	llm           core.LLMClient
	tools         *tools.ToolRegistry
	memory        core.SharedMemory
	activeSwarms  map[core.SwarmID]context.CancelFunc
	config        config.SwarmConfig
	policyChecker *core.PolicyChecker
	defaultModel  string
	mu            sync.RWMutex
}

func NewOrchestrator(store core.SwarmStore, bus core.EventBus, llm core.LLMClient, reg *tools.ToolRegistry, cfg config.SwarmConfig, model string) *Orchestrator {
	p := make(map[string]core.RolePolicy)
	for _, pol := range cfg.Policies {
		p[pol.Role] = core.RolePolicy{AllowedTools: pol.Allowed, DeniedTools: pol.Denied}
	}
	return &Orchestrator{
		store: store, bus: bus, llm: llm, tools: reg,
		activeSwarms: make(map[core.SwarmID]context.CancelFunc),
		config:       cfg, policyChecker: core.NewPolicyChecker(p), defaultModel: model,
	}
}

func (o *Orchestrator) SetSharedMemory(m core.SharedMemory) { o.memory = m }

func (o *Orchestrator) SpawnSwarm(ctx context.Context, goal, channel, chatID string) (core.SwarmID, error) {
	id := core.SwarmID(uuid.New().String())
	if err := o.store.CreateSwarm(ctx, &core.Swarm{
		ID: id, Goal: goal, Status: core.SwarmStatusActive, CreatedAt: time.Now(),
		OriginChannel: channel, OriginChatID: chatID,
	}); err != nil {
		return "", fmt.Errorf("failed to create swarm in store: %w", err)
	}

	o.mu.Lock()
	sCtx, cancel := context.WithCancel(ctx)
	o.activeSwarms[id] = cancel
	o.mu.Unlock()

	go func() {
		defer o.StopSwarm(id)
		o.RunSubTask(sCtx, id, "", "Manager", goal)
		
		// Update swarm status to completed
		sw, err := o.store.GetSwarm(context.Background(), id)
		if err == nil {
			sw.Status = core.SwarmStatusCompleted
			o.store.UpdateSwarm(context.Background(), sw)
		}
	}()
	return id, nil
}

func (o *Orchestrator) RunSubTask(ctx context.Context, sid core.SwarmID, pid core.NodeID, roleName, task string) (string, error) {
	rc, ok := o.config.Roles[roleName]
	if !ok { return "", fmt.Errorf("role %s missing", roleName) }

	model := rc.Model
	if model == "" { model = o.defaultModel }

	node := &core.Node{
		ID: core.NodeID(uuid.New().String()), SwarmID: sid, ParentID: pid,
		Role: core.Role{Name: roleName, SystemPrompt: rc.SystemPrompt, Model: model},
		Task: task, Status: core.NodeStatusPending,
	}
	if err := o.store.CreateNode(ctx, node); err != nil {
		return "", fmt.Errorf("failed to create node in store: %w", err)
	}

	nt := o.tools.Clone()
	if roleName == "Manager" { nt.Register(NewDelegateTool(o, sid, node.ID)) }
	if o.memory != nil {
		nt.Register(NewMemorySaveTool(o.memory, sid))
		nt.Register(NewMemorySearchTool(o.memory, sid, o.config.Memory.SharedKnowledge))
	}

	done := make(chan string, 1)
	fail := make(chan error, 1)
	sub, _ := o.bus.Subscribe("node.events", func(e core.Event) {
		if e.NodeID == node.ID {
			if e.Type == core.EventNodeCompleted { 
				output, _ := e.Payload["output"].(string)
				done <- output 
			}
			if e.Type == core.EventNodeFailed { fail <- fmt.Errorf("%v", e.Payload["error"]) }
		}
	})
	defer sub.Unsubscribe()

	NewNodeActor(node, o.bus, o.store, o.llm, nt, o.policyChecker, o.config).Start(ctx)

	timeout := o.config.Limits.GlobalTimeout
	if timeout <= 0 {
		timeout = 15 * time.Minute
	}

	select {
	case out := <-done: return out, nil
	case err := <-fail: return "", err
	case <-ctx.Done(): return "", ctx.Err()
	case <-time.After(timeout): return "", core.ErrTaskTimeout
	}
}

func (o *Orchestrator) StopSwarm(id core.SwarmID) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if c, ok := o.activeSwarms[id]; ok {
		c()
		delete(o.activeSwarms, id)
	}
}

func (o *Orchestrator) StopAll() {
	o.mu.Lock()
	defer o.mu.Unlock()
	for id, cancel := range o.activeSwarms {
		cancel()
		delete(o.activeSwarms, id)
	}
}
