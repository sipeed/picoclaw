// Package tpcp provides TPCP (Telepathy Communication Protocol) integration
// for PicoClaw multi-agent coordination across multiple devices.
//
// TPCP solves the network transport problem that the in-process blackboard
// cannot: agents running on different machines (Raspberry Pi, cloud VMs,
// laptops) can send signed messages, synchronize shared state via CRDT, and
// guarantee message delivery via a dead-letter queue.
//
// Usage — two PicoClaw instances communicating:
//
//	// Device A (192.168.1.10)
//	nodeA, _ := tpcp.NewAdapter("picoclaw-agent-a", nil)
//	nodeA.OnMessage(func(from, content string) {
//	    fmt.Printf("Received from %s: %s\n", from, content)
//	})
//	nodeA.ListenAsync(":8765")
//
//	// Device B (192.168.1.20) — connects to Device A
//	nodeB, _ := tpcp.NewAdapter("picoclaw-agent-b", nil)
//	nodeB.Connect("ws://192.168.1.10:8765")
//	nodeB.Send(context.Background(), "picoclaw-agent-a", "hello from device B")
//
// For relay-based routing (internet/NAT traversal), pass a relay URL as a peer:
//
//	node.Connect("wss://relay.agent-telepathy.io")
//	node.Send(ctx, "picoclaw-remote-agent", "hello over relay")
package tpcp

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"sync"

	tpcpgo "github.com/Etriti00/agent-telepathy/tpcp-go/tpcp"
)

// Config holds optional configuration for an Adapter. All fields are optional;
// zero values select safe defaults.
type Config struct {
	// PrivateKeySeed is a 32-byte Ed25519 seed. If nil, a new key is generated.
	PrivateKeySeed []byte

	// Framework identifies the agent runtime in identity announcements.
	// Defaults to "picoclaw".
	Framework string

	// Capabilities advertises what this agent can do.
	// Defaults to ["chat", "task"].
	Capabilities []string

	// AllowedOrigins restricts WebSocket browser origins for the listener.
	// If empty, all non-browser connections are accepted.
	AllowedOrigins []string
}

// MessageHandler is called for every inbound TPCP message.
type MessageHandler func(from, content string)

// Adapter wraps a TPCP node with a PicoClaw-friendly interface.
// It is safe for concurrent use.
type Adapter struct {
	node     *tpcpgo.TPCPNode
	agentID  string
	handlers []MessageHandler
	mu       sync.RWMutex
}

// NewAdapter creates a TPCP adapter for a PicoClaw agent.
//
// agentID should be the same stable identifier used by the PicoClaw instance
// (e.g. its configured agent name or a UUID). cfg may be nil to use defaults.
func NewAdapter(agentID string, cfg *Config) (*Adapter, error) {
	if agentID == "" {
		return nil, fmt.Errorf("tpcp: agentID must not be empty")
	}

	c := resolveConfig(cfg)

	// Build Ed25519 keypair from seed or generate a fresh one.
	var privKey ed25519.PrivateKey
	var pubKey ed25519.PublicKey

	if len(c.PrivateKeySeed) > 0 {
		if len(c.PrivateKeySeed) != 32 {
			return nil, fmt.Errorf("tpcp: PrivateKeySeed must be exactly 32 bytes (got %d)", len(c.PrivateKeySeed))
		}
		privKey = ed25519.NewKeyFromSeed(c.PrivateKeySeed)
		pubKey = privKey.Public().(ed25519.PublicKey)
	} else {
		var err error
		pubKey, privKey, err = ed25519.GenerateKey(nil)
		if err != nil {
			return nil, fmt.Errorf("tpcp: key generation failed: %w", err)
		}
	}

	identity := &tpcpgo.AgentIdentity{
		AgentID:      agentID,
		Framework:    c.Framework,
		PublicKey:    base64.StdEncoding.EncodeToString(pubKey),
		Capabilities: c.Capabilities,
		Modality:     []string{"text"},
	}

	node := tpcpgo.NewTPCPNode(identity, privKey)
	node.AllowedOrigins = c.AllowedOrigins

	a := &Adapter{node: node, agentID: agentID}

	// Route all inbound BROADCAST and TASK_REQUEST messages to registered handlers.
	for _, intent := range []tpcpgo.Intent{tpcpgo.IntentBroadcast, tpcpgo.IntentTaskRequest, tpcpgo.IntentCritique} {
		intent := intent // capture
		node.RegisterHandler(intent, func(env *tpcpgo.TPCPEnvelope) {
			content := extractContent(env)
			a.mu.RLock()
			handlers := a.handlers
			a.mu.RUnlock()
			for _, h := range handlers {
				h(env.Header.SenderID, content)
			}
		})
	}

	return a, nil
}

// OnMessage registers a handler for inbound messages. Multiple handlers can be
// registered; they are called in registration order for each message.
func (a *Adapter) OnMessage(h MessageHandler) *Adapter {
	a.mu.Lock()
	a.handlers = append(a.handlers, h)
	a.mu.Unlock()
	return a
}

// ListenAsync starts the TPCP WebSocket server in a background goroutine and
// returns immediately once the port is bound. addr follows net.Listen syntax
// (e.g. ":8765" or "0.0.0.0:8765").
//
// Errors after startup (e.g. accept errors) are silently logged by the
// underlying TPCP node. Use Listen() if you need the goroutine under your
// own supervision.
func (a *Adapter) ListenAsync(addr string) error {
	ready := make(chan error, 1)
	go func() {
		if err := a.node.Listen(addr); err != nil {
			ready <- err
		}
	}()

	// Wait for the node's Ready channel (closed once the server is bound).
	select {
	case err := <-ready:
		return fmt.Errorf("tpcp: listen on %s failed: %w", addr, err)
	case <-a.node.Ready:
		return nil
	}
}

// Connect opens a WebSocket connection to a peer or relay at url.
// url should be a WebSocket URL, e.g.:
//   - "ws://192.168.1.10:8765"   — direct P2P to another PicoClaw device
//   - "wss://relay.example.com"  — relay for NAT traversal
//
// Connect performs a TPCP HANDSHAKE so the remote node registers this agent.
// It returns once the connection is established; message exchange happens
// asynchronously.
func (a *Adapter) Connect(url string) error {
	if err := a.node.Connect(url); err != nil {
		return fmt.Errorf("tpcp: connect to %s failed: %w", url, err)
	}
	return nil
}

// Send sends a text message to the agent identified by targetID.
// The message is signed with this node's Ed25519 key before transmission.
//
// If the target is not currently connected, the message is queued in the
// dead-letter queue (DLQ) and delivered when connectivity is restored.
func (a *Adapter) Send(_ context.Context, targetID, message string) error {
	payload := &tpcpgo.TextPayload{
		PayloadType: "text",
		Content:     message,
		Language:    "en",
	}
	if err := a.node.SendMessage(a.agentID, targetID, tpcpgo.IntentBroadcast, payload); err != nil {
		return fmt.Errorf("tpcp: send to %s failed: %w", targetID, err)
	}
	return nil
}

// Broadcast sends a text message to all connected peers.
func (a *Adapter) Broadcast(_ context.Context, message string) error {
	payload := &tpcpgo.TextPayload{
		PayloadType: "text",
		Content:     message,
		Language:    "en",
	}
	const broadcastID = "00000000-0000-0000-0000-000000000000"
	if err := a.node.SendMessage(a.agentID, broadcastID, tpcpgo.IntentBroadcast, payload); err != nil {
		return fmt.Errorf("tpcp: broadcast failed: %w", err)
	}
	return nil
}

// AgentID returns the stable agent identifier for this node.
func (a *Adapter) AgentID() string { return a.agentID }

// Stop shuts down the TPCP node, closing all connections.
func (a *Adapter) Stop() error {
	return a.node.Stop()
}

// ── helpers ───────────────────────────────────────────────────────────────────

func resolveConfig(cfg *Config) Config {
	c := Config{
		Framework:    "picoclaw",
		Capabilities: []string{"chat", "task"},
	}
	if cfg == nil {
		return c
	}
	if cfg.Framework != "" {
		c.Framework = cfg.Framework
	}
	if len(cfg.Capabilities) > 0 {
		c.Capabilities = cfg.Capabilities
	}
	if len(cfg.PrivateKeySeed) > 0 {
		c.PrivateKeySeed = cfg.PrivateKeySeed
	}
	if len(cfg.AllowedOrigins) > 0 {
		c.AllowedOrigins = cfg.AllowedOrigins
	}
	return c
}

func extractContent(env *tpcpgo.TPCPEnvelope) string {
	if env.Payload == nil {
		return ""
	}
	// Payload is interface{}; try common shapes.
	switch p := env.Payload.(type) {
	case map[string]interface{}:
		if c, ok := p["content"].(string); ok {
			return c
		}
		// Fallback: JSON-encode the whole payload
		return fmt.Sprintf("%v", p)
	case string:
		return p
	default:
		return fmt.Sprintf("%v", p)
	}
}
