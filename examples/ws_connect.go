package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run examples/ws_connect.go <ws_url> <token>")
		fmt.Println("Example: go run examples/ws_connect.go ws://localhost:18800/pico/ws my-secret-token")
		return
	}

	wsURL := os.Args[1]
	token := os.Args[2]

	// The token is passed via the Sec-Websocket-Protocol header as "token.<value>".
	// This is the standard way to pass an auth token for WebSockets in a browser-compatible way.
	header := http.Header{}
	header.Set("Sec-Websocket-Protocol", "token."+token)

	fmt.Printf("Connecting to %s with token...\n", wsURL)
	dialer := websocket.DefaultDialer
	conn, resp, err := dialer.Dial(wsURL, header)
	if err != nil {
		if resp != nil {
			log.Fatalf("Handshake failed with status %d: %v", resp.StatusCode, err)
		}
		log.Fatalf("Connection failed: %v", err)
	}
	defer conn.Close()

	// On success, the server echoes the subprotocol back.
	fmt.Printf("Connected successfully! Server subprotocol: %s\n", conn.Subprotocol())

	// Send a simple ping message using the Pico protocol format
	fmt.Println("Sending ping message...")
	pingMsg := map[string]any{
		"type": "ping",
		"id":   "msg-1",
	}
	if err := conn.WriteJSON(pingMsg); err != nil {
		log.Fatalf("Failed to send ping: %v", err)
	}

	// Read response
	fmt.Println("Waiting for pong...")
	var pongMsg map[string]any
	if err := conn.ReadJSON(&pongMsg); err != nil {
		log.Fatalf("Failed to read pong: %v", err)
	}

	fmt.Printf("Received response: %+v\n", pongMsg)
	fmt.Println("Test completed successfully.")
}
