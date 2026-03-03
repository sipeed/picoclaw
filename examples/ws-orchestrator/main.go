// ws-orchestrator is a minimal WebSocket server that mimics an orchestration
// service for smoke-testing the picoclaw WebSocket channel.
//
// Usage:
//
//	go run ./examples/ws-orchestrator -addr :9090 -token secret
//
// Then configure picoclaw with:
//
//	"websocket": {
//	    "enabled": true,
//	    "ws_url": "ws://localhost:9090/ws",
//	    "access_token": "secret",
//	    "agent_id": "researcher",
//	    "reconnect_interval": 5
//	}
//
// The orchestrator will:
//  1. Validate the auth handshake (agent_id + token)
//  2. Send a test message after successful auth
//  3. Print any messages received from the agent
//  4. Read lines from stdin and send them as inbound messages
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/websocket"
)

type WSEnvelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type WSAuthData struct {
	AgentID string `json:"agent_id"`
	Token   string `json:"token"`
}

type WSInboundMessage struct {
	SenderID   string            `json:"sender_id"`
	ChatID     string            `json:"chat_id"`
	Content    string            `json:"content"`
	SessionKey string            `json:"session_key,omitempty"`
	MessageID  string            `json:"message_id,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type WSOutboundMessage struct {
	Channel     string `json:"channel"`
	ChatID      string `json:"chat_id"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
}

var (
	addr  = flag.String("addr", ":9090", "listen address")
	token = flag.String("token", "secret", "expected access token")
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

// connMu guards activeConn so stdin sender and HTTP handler don't race.
var (
	connMu     sync.Mutex
	activeConn *websocket.Conn
)

func main() {
	flag.Parse()

	http.HandleFunc("/ws", handleWS)

	// Read stdin in background for interactive message sending.
	go stdinLoop()

	log.Printf("ws-orchestrator listening on %s/ws (token=%q)", *addr, *token)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade error: %v", err)
		return
	}
	defer func() {
		connMu.Lock()
		if activeConn == conn {
			activeConn = nil
		}
		connMu.Unlock()
		conn.Close()
	}()

	log.Println("agent connected")

	// 1. Read auth envelope
	_, data, err := conn.ReadMessage()
	if err != nil {
		log.Printf("read auth: %v", err)
		return
	}

	var env WSEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		log.Printf("bad envelope: %v", err)
		return
	}
	if env.Type != "auth" {
		log.Printf("expected auth, got %q", env.Type)
		conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "expected auth"))
		return
	}

	var auth WSAuthData
	if err := json.Unmarshal(env.Data, &auth); err != nil {
		log.Printf("bad auth data: %v", err)
		return
	}
	if auth.Token != *token {
		log.Printf("bad token from %s: %q", auth.AgentID, auth.Token)
		conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "bad token"))
		return
	}

	log.Printf("auth OK: agent_id=%s", auth.AgentID)

	connMu.Lock()
	activeConn = conn
	connMu.Unlock()

	// 2. Send a test message
	sendInbound(conn, WSInboundMessage{
		SenderID:   "orchestrator",
		ChatID:     "smoke-test",
		Content:    fmt.Sprintf("Hello %s, this is a smoke test from the orchestrator.", auth.AgentID),
		SessionKey: "smoke-001",
		MessageID:  "msg-001",
	})

	// 3. Read loop — print messages from agent
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("agent disconnected: %v", err)
			return
		}
		var env WSEnvelope
		if err := json.Unmarshal(data, &env); err != nil {
			log.Printf("bad envelope from agent: %v", err)
			continue
		}
		switch env.Type {
		case "message":
			var out WSOutboundMessage
			if err := json.Unmarshal(env.Data, &out); err != nil {
				log.Printf("bad outbound message: %v", err)
				continue
			}
			log.Printf("[%s] chat=%s type=%s content=%s",
				out.Channel, out.ChatID, out.ContentType, out.Content)
		default:
			log.Printf("unknown envelope type %q: %s", env.Type, string(env.Data))
		}
	}
}

func sendInbound(conn *websocket.Conn, msg WSInboundMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("marshal inbound: %v", err)
		return
	}
	env := WSEnvelope{Type: "message", Data: json.RawMessage(data)}
	envData, err := json.Marshal(env)
	if err != nil {
		log.Printf("marshal envelope: %v", err)
		return
	}
	if err := conn.WriteMessage(websocket.TextMessage, envData); err != nil {
		log.Printf("write inbound: %v", err)
		return
	}
	log.Printf("→ sent: chat=%s content=%s", msg.ChatID, msg.Content)
}

// stdinLoop reads lines from stdin and sends them as inbound messages.
func stdinLoop() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Type a message and press Enter to send to the agent (chat_id=interactive):")
	msgID := 0
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		connMu.Lock()
		conn := activeConn
		connMu.Unlock()
		if conn == nil {
			fmt.Println("(no agent connected)")
			continue
		}
		msgID++
		sendInbound(conn, WSInboundMessage{
			SenderID:   "orchestrator",
			ChatID:     "interactive",
			Content:    line,
			SessionKey: "interactive",
			MessageID:  fmt.Sprintf("stdin-%d", msgID),
		})
	}
}
