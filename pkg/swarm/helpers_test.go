// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// mockLLMProvider implements providers.LLMProvider for testing.
type mockLLMProvider struct {
	chatFn func(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string, options map[string]interface{}) (*providers.LLMResponse, error)
	model  string
}

func (m *mockLLMProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string, options map[string]interface{}) (*providers.LLMResponse, error) {
	if m.chatFn != nil {
		return m.chatFn(ctx, messages, tools, model, options)
	}
	return &providers.LLMResponse{Content: "mock response", FinishReason: "stop"}, nil
}

func (m *mockLLMProvider) GetDefaultModel() string {
	if m.model != "" {
		return m.model
	}
	return "test-model"
}

// mockManager is a minimal implementation of the manager interface used for testing
type mockManager struct{}

func (m *mockManager) GetNodeInfo() *NodeInfo {
	return &NodeInfo{
		ID:     "mock-manager",
		Role:   RoleCoordinator,
		Status: StatusOnline,
	}
}

func (m *mockManager) PromoteToCoordinator() error {
	return nil
}

func (m *mockManager) DemoteToWorker() error {
	return nil
}

// startTestNATS starts an embedded NATS server on a random available port.
// It returns the embedded server, the client URL, and a cleanup function.
func startTestNATS(t *testing.T) (*EmbeddedNATS, string, func()) {
	t.Helper()
	port := freePort(t)
	cfg := &config.NATSConfig{EmbeddedPort: port}
	e := NewEmbeddedNATS(cfg)
	if err := e.Start(); err != nil {
		t.Fatalf("startTestNATS: failed to start: %v", err)
	}
	url := e.ClientURL()
	return e, url, func() { e.Stop() }
}

// newTestSwarmConfig returns a SwarmConfig with short timeouts suitable for fast tests.
func newTestSwarmConfig(port int) *config.SwarmConfig {
	return &config.SwarmConfig{
		Enabled:       true,
		Role:          "worker",
		Capabilities:  []string{"general"},
		MaxConcurrent: 2,
		NATS: config.NATSConfig{
			URLs:              []string{fmt.Sprintf("nats://127.0.0.1:%d", port)},
			HeartbeatInterval: "50ms",
			NodeTimeout:       "200ms",
			Embedded:          false,
			EmbeddedPort:      port,
		},
		Temporal: config.TemporalConfig{
			TaskQueue: "test-queue",
		},
	}
}

// newTestConfig returns a full config.Config wrapping a test SwarmConfig.
func newTestConfig(t *testing.T, embeddedPort int) *config.Config {
	t.Helper()
	workspace := t.TempDir()
	return &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         workspace,
				RestrictToWorkspace: true,
				Model:             "test-model",
				MaxTokens:         1024,
				Temperature:       0.0,
				MaxToolIterations: 5,
			},
		},
		Swarm: *newTestSwarmConfig(embeddedPort),
	}
}

// newTestNodeInfo creates a NodeInfo with configurable fields.
func newTestNodeInfo(id string, role NodeRole, capabilities []string, maxTasks int) *NodeInfo {
	return &NodeInfo{
		ID:           id,
		Role:         role,
		Capabilities: capabilities,
		Model:        "test-model",
		Status:       StatusOnline,
		MaxTasks:     maxTasks,
		StartedAt:    time.Now().UnixMilli(),
		Metadata:     make(map[string]string),
	}
}

// connectTestBridge creates a NATSBridge, connects it to the given URL, and returns it.
// The bridge is configured with a SwarmConfig pointing at the given URL.
func connectTestBridge(t *testing.T, url string, nodeInfo *NodeInfo) *NATSBridge {
	t.Helper()
	cfg := &config.SwarmConfig{
		Enabled:       true,
		MaxConcurrent: 2,
		NATS: config.NATSConfig{
			URLs: []string{url},
		},
	}
	msgBus := bus.NewMessageBus()
	bridge := NewNATSBridge(cfg, msgBus, nodeInfo)
	if err := bridge.Connect(context.Background()); err != nil {
		t.Fatalf("connectTestBridge: failed to connect: %v", err)
	}
	return bridge
}

// connectTestNATS connects a raw nats.Conn to the given URL. Returns the connection.
func connectTestNATS(t *testing.T, url string) *nats.Conn {
	t.Helper()
	nc, err := nats.Connect(url)
	if err != nil {
		t.Fatalf("connectTestNATS: %v", err)
	}
	return nc
}

// waitFor polls a condition function at 10ms intervals up to the given timeout.
// Returns true if the condition was met before timeout, false otherwise.
func waitFor(t *testing.T, timeout time.Duration, condition func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// newTestAgentLoop creates an AgentLoop backed by a mock LLM provider.
// The provider returns chatResponse on success, or chatErr if non-nil.
func newTestAgentLoop(t *testing.T, chatResponse string, chatErr error) *agent.AgentLoop {
	t.Helper()
	cfg := newTestConfig(t, 0)
	msgBus := bus.NewMessageBus()
	provider := &mockLLMProvider{
		model: "test-model",
		chatFn: func(ctx context.Context, msgs []providers.Message, tools []providers.ToolDefinition, model string, opts map[string]interface{}) (*providers.LLMResponse, error) {
			if chatErr != nil {
				return nil, chatErr
			}
			return &providers.LLMResponse{
				Content:      chatResponse,
				FinishReason: "stop",
			}, nil
		},
	}
	return agent.NewAgentLoop(cfg, msgBus, provider)
}
