// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"fmt"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// EmbeddedNATS wraps an embedded NATS server for development mode
type EmbeddedNATS struct {
	server *server.Server
	cfg    *config.NATSConfig
}

// NewEmbeddedNATS creates a new embedded NATS server
func NewEmbeddedNATS(cfg *config.NATSConfig) *EmbeddedNATS {
	return &EmbeddedNATS{cfg: cfg}
}

// Start starts the embedded NATS server
func (e *EmbeddedNATS) Start() error {
	port := e.cfg.EmbeddedPort
	if port == 0 {
		port = 4222
	}

	host := e.cfg.EmbeddedHost
	if host == "" {
		host = "0.0.0.0" // Default to listening on all interfaces for external access
	}

	opts := &server.Options{
		Host:           host,
		Port:           port,
		NoLog:          false,
		NoSigs:         true,
		MaxControlLine: 2048,
		MaxPayload:     4 * 1024 * 1024, // 4MB
		JetStream:      true,             // Enable JetStream
		// Use memory storage for JetStream (no persistence)
		StoreDir: "memory://",
	}

	ns, err := server.NewServer(opts)
	if err != nil {
		return fmt.Errorf("failed to create embedded NATS server: %w", err)
	}

	go ns.Start()

	// Wait for server to be ready
	if !ns.ReadyForConnections(10 * time.Second) {
		return fmt.Errorf("embedded NATS server failed to start within timeout")
	}

	e.server = ns
	logger.InfoCF("swarm", "Embedded NATS server started", map[string]interface{}{
		"host": host,
		"port": port,
	})

	return nil
}

// Stop stops the embedded NATS server
func (e *EmbeddedNATS) Stop() {
	if e.server != nil {
		e.server.Shutdown()
		logger.InfoC("swarm", "Embedded NATS server stopped")
	}
}

// ClientURL returns the URL for local clients to connect
func (e *EmbeddedNATS) ClientURL() string {
	port := e.cfg.EmbeddedPort
	if port == 0 {
		port = 4222
	}
	return fmt.Sprintf("nats://localhost:%d", port)
}

// ExternalURL returns the URL for external clients to connect (uses the actual hostname)
func (e *EmbeddedNATS) ExternalURL(hostname string) string {
	port := e.cfg.EmbeddedPort
	if port == 0 {
		port = 4222
	}
	if hostname == "" {
		hostname = "localhost"
	}
	return fmt.Sprintf("nats://%s:%d", hostname, port)
}

// IsRunning returns true if the embedded server is running
func (e *EmbeddedNATS) IsRunning() bool {
	return e.server != nil && e.server.Running()
}
