// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"fmt"
	"strings"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestEmbeddedNATS(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "start and stop",
			fn: func(t *testing.T) {
				port := freePort(t)
				cfg := &config.NATSConfig{EmbeddedPort: port}
				e := NewEmbeddedNATS(cfg)

				if e.IsRunning() {
					t.Error("IsRunning() = true before Start, want false")
				}

				if err := e.Start(); err != nil {
					t.Fatalf("Start() error: %v", err)
				}

				if !e.IsRunning() {
					t.Error("IsRunning() = false after Start, want true")
				}

				e.Stop()

				if e.IsRunning() {
					t.Error("IsRunning() = true after Stop, want false")
				}
			},
		},
		{
			name: "client URL format",
			fn: func(t *testing.T) {
				port := freePort(t)
				cfg := &config.NATSConfig{EmbeddedPort: port}
				e := NewEmbeddedNATS(cfg)
				if err := e.Start(); err != nil {
					t.Fatalf("Start() error: %v", err)
				}
				defer e.Stop()

				url := e.ClientURL()
				want := fmt.Sprintf("nats://127.0.0.1:%d", port)
				if url != want {
					t.Errorf("ClientURL() = %q, want %q", url, want)
				}
			},
		},
		{
			name: "connect client",
			fn: func(t *testing.T) {
				port := freePort(t)
				cfg := &config.NATSConfig{EmbeddedPort: port}
				e := NewEmbeddedNATS(cfg)
				if err := e.Start(); err != nil {
					t.Fatalf("Start() error: %v", err)
				}
				defer e.Stop()

				nc, err := nats.Connect(e.ClientURL())
				if err != nil {
					t.Fatalf("nats.Connect() error: %v", err)
				}
				defer nc.Close()

				if !nc.IsConnected() {
					t.Error("IsConnected() = false, want true")
				}
			},
		},
		{
			name: "multiple start stop cycles",
			fn: func(t *testing.T) {
				port := freePort(t)
				cfg := &config.NATSConfig{EmbeddedPort: port}

				for i := 0; i < 3; i++ {
					e := NewEmbeddedNATS(cfg)
					if err := e.Start(); err != nil {
						t.Fatalf("cycle %d: Start() error: %v", i, err)
					}
					if !e.IsRunning() {
						t.Errorf("cycle %d: IsRunning() = false after Start", i)
					}
					e.Stop()
					if e.IsRunning() {
						t.Errorf("cycle %d: IsRunning() = true after Stop", i)
					}
				}
			},
		},
		{
			name: "custom port",
			fn: func(t *testing.T) {
				port := freePort(t)
				cfg := &config.NATSConfig{EmbeddedPort: port}
				e := NewEmbeddedNATS(cfg)
				if err := e.Start(); err != nil {
					t.Fatalf("Start() error: %v", err)
				}
				defer e.Stop()

				url := e.ClientURL()
				if !strings.Contains(url, fmt.Sprintf(":%d", port)) {
					t.Errorf("ClientURL() = %q, does not contain port %d", url, port)
				}

				// Verify we can actually connect on this port
				nc, err := nats.Connect(url)
				if err != nil {
					t.Fatalf("Connect to custom port %d error: %v", port, err)
				}
				nc.Close()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.fn)
	}
}
