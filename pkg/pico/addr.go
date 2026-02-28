// Package pico provides a reusable WebSocket client for the Pico Protocol.
//
// In addition to the low-level Client, this package defines shared types and
// helpers used by both the Pico channel (server) and the swarm subsystem
// (client) for inter-node communication.
//
// The Client type encapsulates the connect → send → receive → close lifecycle
// for a single request-reply exchange with a Pico WebSocket endpoint. It is
// intentionally stateless: each call to SendRequest opens a new connection,
// performs the exchange, and closes the connection.
//
// This package depends only on pkg/pico/protocol and gorilla/websocket;
// it has no knowledge of swarm, channels, or any other higher-level construct.
package pico

import "fmt"

// BuildNodeAddr constructs a host:port address string for inter-node
// communication. It replaces the scattered fmt.Sprintf("%s:%d", ...) calls.
func BuildNodeAddr(addr string, port int) string {
	return fmt.Sprintf("%s:%d", addr, port)
}
