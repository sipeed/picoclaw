// pico-echo-server is a minimal Pico Protocol WebSocket server for testing
// the pico_client channel. It accepts connections, prints received messages
// to stdout, and forwards stdin lines as message.create to all connected clients.
//
// Usage:
//
//	go run ./examples/pico-echo-server -addr :9090 -token secret
//
// Then configure pico_client with url=ws://localhost:9090/ws&token=secret.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type picoMessage struct {
	Type      string         `json:"type"`
	ID        string         `json:"id,omitempty"`
	SessionID string         `json:"session_id,omitempty"`
	Timestamp int64          `json:"timestamp,omitempty"`
	Payload   map[string]any `json:"payload,omitempty"`
}

var upgrader = websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}

type server struct {
	token string
	mu    sync.Mutex
	conns map[*websocket.Conn]string // conn → sessionID
}

func (s *server) handleWS(w http.ResponseWriter, r *http.Request) {
	if s.token != "" {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer "+s.token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade: %v", err)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		sessionID = fmt.Sprintf("sess-%d", time.Now().UnixMilli())
	}

	s.mu.Lock()
	s.conns[conn] = sessionID
	s.mu.Unlock()

	log.Printf("[+] client connected (session=%s)", sessionID)

	defer func() {
		s.mu.Lock()
		delete(s.conns, conn)
		s.mu.Unlock()
		conn.Close()
		log.Printf("[-] client disconnected (session=%s)", sessionID)
	}()

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("read error: %v", err)
			}
			return
		}

		var msg picoMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			log.Printf("bad json: %v", err)
			continue
		}

		switch msg.Type {
		case "ping":
			pong := picoMessage{Type: "pong", ID: msg.ID, Timestamp: time.Now().UnixMilli()}
			conn.WriteJSON(pong)

		case "message.send":
			content, _ := msg.Payload["content"].(string)
			fmt.Printf("[%s] %s\n", sessionID, content)

		case "typing.start":
			log.Printf("[%s] typing...", sessionID)

		case "typing.stop":
			log.Printf("[%s] stopped typing", sessionID)

		default:
			log.Printf("[%s] unknown type: %s", sessionID, msg.Type)
		}
	}
}

func (s *server) broadcast(content string) {
	msg := picoMessage{
		Type:      "message.create",
		Timestamp: time.Now().UnixMilli(),
		Payload:   map[string]any{"content": content},
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for conn, sid := range s.conns {
		msg.SessionID = sid
		if err := conn.WriteJSON(msg); err != nil {
			log.Printf("write to %s failed: %v", sid, err)
		}
	}
}

func main() {
	addr := flag.String("addr", ":9090", "listen address")
	token := flag.String("token", "", "auth token (empty = no auth)")
	flag.Parse()

	s := &server{
		token: *token,
		conns: make(map[*websocket.Conn]string),
	}

	http.HandleFunc("/ws", s.handleWS)

	log.Printf("listening on %s", *addr)
	log.Printf("connect with: ws://localhost%s/ws", *addr)
	fmt.Println("Type messages to send to connected clients (Ctrl+C to quit):")

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			s.broadcast(line)
			log.Printf("[server] sent: %s", line)
		}
	}()

	log.Fatal(http.ListenAndServe(*addr, nil))
}
