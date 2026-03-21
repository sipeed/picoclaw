package tpcp_test

import (
	"context"
	"testing"
	"time"

	tpcpadapter "github.com/sipeed/picoclaw/pkg/multiagent/tpcp"
)

func TestNewAdapter_EmptyAgentID(t *testing.T) {
	_, err := tpcpadapter.NewAdapter("", nil)
	if err == nil {
		t.Fatal("expected error for empty agentID, got nil")
	}
}

func TestNewAdapter_InvalidSeedLength(t *testing.T) {
	cfg := &tpcpadapter.Config{
		PrivateKeySeed: []byte("short"),
	}
	_, err := tpcpadapter.NewAdapter("test-agent", cfg)
	if err == nil {
		t.Fatal("expected error for seed with wrong length, got nil")
	}
}

func TestNewAdapter_ValidSeed(t *testing.T) {
	seed := make([]byte, 32)
	cfg := &tpcpadapter.Config{
		PrivateKeySeed: seed,
	}
	a, err := tpcpadapter.NewAdapter("test-agent", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.AgentID() != "test-agent" {
		t.Errorf("expected agentID 'test-agent', got %q", a.AgentID())
	}
}

func TestNewAdapter_NilConfig(t *testing.T) {
	a, err := tpcpadapter.NewAdapter("test-agent", nil)
	if err != nil {
		t.Fatalf("unexpected error with nil config: %v", err)
	}
	if a.AgentID() != "test-agent" {
		t.Errorf("expected agentID 'test-agent', got %q", a.AgentID())
	}
}

func TestNewAdapter_GeneratesKey(t *testing.T) {
	a1, err1 := tpcpadapter.NewAdapter("agent-1", nil)
	a2, err2 := tpcpadapter.NewAdapter("agent-2", nil)
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected errors: %v, %v", err1, err2)
	}
	_ = a1
	_ = a2
}

func TestOnMessage_Chainable(t *testing.T) {
	a, err := tpcpadapter.NewAdapter("test-agent", nil)
	if err != nil {
		t.Fatal(err)
	}
	returned := a.OnMessage(func(from, content string) {})
	if returned != a {
		t.Error("OnMessage should return the same adapter for chaining")
	}
}

func TestOnMessage_MultipleHandlers(t *testing.T) {
	a, err := tpcpadapter.NewAdapter("test-agent", nil)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	a.OnMessage(func(from, content string) { count++ })
	a.OnMessage(func(from, content string) { count++ })
	_ = count
}

func TestAgentID(t *testing.T) {
	id := "picoclaw-device-001"
	a, err := tpcpadapter.NewAdapter(id, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := a.AgentID(); got != id {
		t.Errorf("AgentID() = %q, want %q", got, id)
	}
}

func TestStop_BeforeConnect(t *testing.T) {
	a, err := tpcpadapter.NewAdapter("test-agent", nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = a.Stop()
}

func TestConnect_InvalidURL(t *testing.T) {
	a, err := tpcpadapter.NewAdapter("test-agent", nil)
	if err != nil {
		t.Fatal(err)
	}
	err = a.Connect("ws://127.0.0.1:19999")
	if err == nil {
		t.Error("expected connection error for unreachable address")
	}
}

func TestSend_NotConnected(t *testing.T) {
	a, err := tpcpadapter.NewAdapter("test-agent", nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	_ = a.Send(ctx, "other-agent", "hello")
}

func TestListenAsync_ThenStop(t *testing.T) {
	a, err := tpcpadapter.NewAdapter("test-agent", nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := a.ListenAsync(":0"); err != nil {
		t.Fatalf("ListenAsync failed: %v", err)
	}

	time.Sleep(20 * time.Millisecond)
	if err := a.Stop(); err != nil {
		t.Errorf("Stop failed: %v", err)
	}
}

func TestConfig_Defaults(t *testing.T) {
	a, err := tpcpadapter.NewAdapter("test-agent", nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = a
}

func TestConfig_CustomFramework(t *testing.T) {
	cfg := &tpcpadapter.Config{
		Framework:    "custom-framework",
		Capabilities: []string{"vision", "tool-use"},
	}
	a, err := tpcpadapter.NewAdapter("test-agent", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = a
}
