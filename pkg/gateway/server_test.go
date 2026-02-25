package gateway

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// stubLLMProvider implements LLMProvider for gateway tests (no real LLM).
type stubLLMProvider struct{}

func (stubLLMProvider) Chat(
	_ context.Context,
	_ []providers.Message,
	_ []providers.ToolDefinition,
	_ string,
	_ map[string]any,
) (*providers.LLMResponse, error) {
	return &providers.LLMResponse{Content: "ok", ToolCalls: nil}, nil
}

func (stubLLMProvider) GetDefaultModel() string { return "test" }

// TestGatewayVerification runs the full Gateway protocol verification (plan-based).
// See docs/gateway-verification.md for the checklist.
func TestGatewayVerification(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg := config.DefaultConfig()
	cfg.Gateway.Host = "127.0.0.1"
	cfg.Gateway.Port = 18790 // unused when using Serve(listener)
	cfg.Gateway.Password = "test"
	cfg.Agents.Defaults.Workspace = t.TempDir()

	msgBus := bus.NewMessageBus()
	registry := agent.NewAgentRegistry(cfg, stubLLMProvider{})
	if registry.GetDefaultAgent() == nil {
		t.Fatal("registry has no default agent")
	}

	srv := NewServer(&cfg.Gateway, registry, msgBus)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()
	addr := listener.Addr().String()

	go func() {
		_ = srv.Serve(ctx, listener)
	}()

	// Fake "agent": consume inbound, write to session (so chat.history returns messages), then publish outbound
	go func() {
		for {
			msg, ok := msgBus.ConsumeInbound(ctx)
			if !ok {
				return
			}
			ag := registry.GetDefaultAgent()
			if ag != nil && msg.SessionKey != "" {
				ag.Sessions.AddMessage(msg.SessionKey, "user", msg.Content)
				ag.Sessions.AddMessage(msg.SessionKey, "assistant", "echo:"+msg.Content)
				_ = ag.Sessions.Save(msg.SessionKey)
			}
			msgBus.PublishOutbound(bus.OutboundMessage{
				Channel: "web",
				ChatID:  msg.ChatID,
				Content: "echo:" + msg.Content,
				State:   "final",
			})
		}
	}()

	// 1. HTTP /health
	resp, err := http.Get("http://" + addr + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /health status: %d", resp.StatusCode)
	}

	// 2. HTTP /ready
	resp2, err := http.Get("http://" + addr + "/ready")
	if err != nil {
		t.Fatalf("GET /ready: %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("GET /ready status: %d", resp2.StatusCode)
	}

	// 3–11. WebSocket RPC + event
	wsURL := "ws://" + addr + "/"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial: %v", err)
	}
	defer conn.Close()

	readRes := func(id string) (ok bool, payload interface{}, errMsg string) {
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		var frame GatewayFrame
		if err := json.Unmarshal(data, &frame); err != nil {
			t.Fatalf("unmarshal res: %v", err)
		}
		if frame.Type != "res" || frame.ID != id {
			t.Fatalf("unexpected frame: type=%s id=%s", frame.Type, frame.ID)
		}
		if !frame.Ok && frame.Error != nil {
			return false, nil, frame.Error.Message
		}
		return frame.Ok, frame.Payload, ""
	}

	sendReq := func(method string, params interface{}) string {
		id := "req-" + method + "-1"
		body := GatewayFrame{Type: "req", ID: id, Method: method, Params: mustMarshal(params)}
		if err := conn.WriteJSON(body); err != nil {
			t.Fatalf("write %s: %v", method, err)
		}
		return id
	}

	// 3. connect
	connectID := sendReq("connect", map[string]any{"auth": map[string]string{"password": "test"}})
	ok, payload, errStr := readRes(connectID)
	if !ok || errStr != "" {
		t.Fatalf("connect: ok=%v err=%s", ok, errStr)
	}
	pm, _ := payload.(map[string]interface{})
	if pm["protocol"] == nil || pm["server"] == nil {
		t.Errorf("connect payload: %v", payload)
	}

	// 4. sessions.list
	id := sendReq("sessions.list", map[string]any{"limit": 10})
	ok, payload, errStr = readRes(id)
	if !ok || errStr != "" {
		t.Fatalf("sessions.list: ok=%v err=%s", ok, errStr)
	}
	if _, ok := payload.(map[string]interface{}); !ok {
		t.Errorf("sessions.list payload type: %T", payload)
	}

	// 5. sessions.patch (create session)
	sessionKey := "verify-session-" + time.Now().Format("20060102150405")
	id = sendReq("sessions.patch", map[string]any{"key": sessionKey, "label": "Verify"})
	ok, payload, errStr = readRes(id)
	if !ok || errStr != "" {
		t.Fatalf("sessions.patch: ok=%v err=%s", ok, errStr)
	}

	// 6. sessions.resolve
	id = sendReq("sessions.resolve", map[string]any{"key": sessionKey})
	ok, payload, errStr = readRes(id)
	if !ok || errStr != "" {
		t.Fatalf("sessions.resolve: ok=%v err=%s", ok, errStr)
	}
	resolved, _ := payload.(map[string]interface{})
	resolvedKey, _ := resolved["key"].(string)
	if resolvedKey == "" {
		t.Errorf("sessions.resolve key empty: %v", payload)
	}

	// 8. chat.subscribe (before send so we receive the event)
	id = sendReq("chat.subscribe", map[string]any{"sessionKey": resolvedKey})
	ok, _, errStr = readRes(id)
	if !ok || errStr != "" {
		t.Fatalf("chat.subscribe: ok=%v err=%s", ok, errStr)
	}

	// 9. chat.send
	userMsg := "hello-verification"
	id = sendReq("chat.send", map[string]any{
		"sessionKey": resolvedKey,
		"message":    userMsg,
		"deliver":    true,
	})
	ok, payload, errStr = readRes(id)
	if !ok || errStr != "" {
		t.Fatalf("chat.send: ok=%v err=%s", ok, errStr)
	}

	// 10. wait for event (chat, state final)
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var eventFrame GatewayFrame
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read event: %v", err)
		}
		if err := json.Unmarshal(data, &eventFrame); err != nil {
			t.Fatalf("unmarshal event: %v", err)
		}
		if eventFrame.Type == "event" && eventFrame.Event == "chat" {
			break
		}
	}
	pl, _ := eventFrame.Payload.(map[string]interface{})
	state, _ := pl["state"].(string)
	if state != "final" {
		t.Errorf("event state: want final, got %s", state)
	}
	msgObj, _ := pl["message"].(map[string]interface{})
	contentArr, _ := msgObj["content"].([]interface{})
	if len(contentArr) == 0 {
		t.Error("event message.content empty")
	} else {
		first, _ := contentArr[0].(map[string]interface{})
		text, _ := first["text"].(string)
		if !strings.Contains(text, "echo:"+userMsg) {
			t.Errorf("event content: want echo:%s, got %s", userMsg, text)
		}
	}

	// 11. chat.history
	id = sendReq("chat.history", map[string]any{"sessionKey": resolvedKey, "limit": 20})
	ok, payload, errStr = readRes(id)
	if !ok || errStr != "" {
		t.Fatalf("chat.history: ok=%v err=%s", ok, errStr)
	}
	hist, _ := payload.(map[string]interface{})
	msgs, _ := hist["messages"].([]interface{})
	if len(msgs) < 2 {
		t.Errorf("chat.history messages: want at least 2, got %d", len(msgs))
	}

	// 7. sessions.delete (run after history so list/patch/resolve/history are covered)
	id = sendReq("sessions.delete", map[string]any{"key": sessionKey})
	ok, _, errStr = readRes(id)
	if !ok || errStr != "" {
		t.Fatalf("sessions.delete: ok=%v err=%s", ok, errStr)
	}
}

func mustMarshal(v interface{}) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// TestGatewayConnectAuth checks connect with token/password and invalid auth.
func TestGatewayConnectAuth(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg := config.DefaultConfig()
	cfg.Gateway.Host = "127.0.0.1"
	cfg.Gateway.Port = 18790
	cfg.Gateway.Password = "secret"
	cfg.Agents.Defaults.Workspace = t.TempDir()

	msgBus := bus.NewMessageBus()
	registry := agent.NewAgentRegistry(cfg, stubLLMProvider{})
	srv := NewServer(&cfg.Gateway, registry, msgBus)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	go func() { _ = srv.Serve(ctx, listener) }()
	go func() {
		for {
			_, ok := msgBus.ConsumeInbound(ctx)
			if !ok {
				return
			}
		}
	}()

	u, _ := url.Parse("ws://" + listener.Addr().String() + "/")
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	sendAndRead := func(method string, params map[string]any) (ok bool, errMsg string) {
		id := "auth-" + method + "-1"
		_ = conn.WriteJSON(GatewayFrame{Type: "req", ID: id, Method: method, Params: mustMarshal(params)})
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		var res GatewayFrame
		_ = json.Unmarshal(data, &res)
		if res.Error != nil {
			errMsg = res.Error.Message
		}
		return res.Ok, errMsg
	}

	ok, _ := sendAndRead("connect", map[string]any{"auth": map[string]string{"password": "wrong"}})
	if ok {
		t.Error("connect with wrong password should fail")
	}
	ok, errStr := sendAndRead("connect", map[string]any{"auth": map[string]string{"password": "secret"}})
	if !ok || errStr != "" {
		t.Errorf("connect with correct password: ok=%v err=%s", ok, errStr)
	}
}

// TestGatewaySessionsDeleteRejectsMain ensures sessions.delete returns error for main session.
func TestGatewaySessionsDeleteRejectsMain(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg := config.DefaultConfig()
	cfg.Gateway.Host = "127.0.0.1"
	cfg.Gateway.Port = 0
	cfg.Gateway.Password = "secret"
	cfg.Agents.Defaults.Workspace = t.TempDir()

	msgBus := bus.NewMessageBus()
	registry := agent.NewAgentRegistry(cfg, stubLLMProvider{})
	srv := NewServer(&cfg.Gateway, registry, msgBus)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	go func() { _ = srv.Serve(ctx, listener) }()
	go func() {
		for {
			_, ok := msgBus.ConsumeInbound(ctx)
			if !ok {
				return
			}
		}
	}()

	u, _ := url.Parse("ws://" + listener.Addr().String() + "/")
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	sendReq := func(method string, params map[string]any) string {
		id := "del-main-1"
		_ = conn.WriteJSON(GatewayFrame{Type: "req", ID: id, Method: method, Params: mustMarshal(params)})
		return id
	}
	readRes := func(id string) (ok bool, errMsg string) {
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		var res GatewayFrame
		_ = json.Unmarshal(data, &res)
		if res.Error != nil {
			errMsg = res.Error.Message
		}
		return res.Ok, errMsg
	}

	// connect
	sendReq("connect", map[string]any{"auth": map[string]string{"password": "secret"}})
	ok, _ := readRes("del-main-1")
	if !ok {
		t.Fatal("connect failed")
	}

	// sessions.delete with key "main" must fail
	sendReq("sessions.delete", map[string]any{"key": "main"})
	ok, errStr := readRes("del-main-1")
	if ok || !strings.Contains(errStr, "Cannot delete the main session") {
		t.Errorf("sessions.delete main: want error containing 'Cannot delete the main session', got ok=%v err=%q", ok, errStr)
	}
}
