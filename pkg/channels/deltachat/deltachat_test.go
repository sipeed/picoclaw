package deltachat

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestNewDeltaChatChannel(t *testing.T) {
	msgBus := bus.NewMessageBus()

	// A fake rpc server so resolveServerPath succeeds regardless of host setup.
	fakeServer := filepath.Join(t.TempDir(), "deltachat-rpc-server")
	if err := os.WriteFile(fakeServer, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Run("missing email", func(t *testing.T) {
		bc := &config.Channel{Type: config.ChannelDeltaChat, Enabled: true}
		cfg := &config.DeltaChatSettings{Password: *config.NewSecureString("pw"), RPCServerPath: fakeServer}
		if _, err := NewDeltaChatChannel(bc, cfg, msgBus); err == nil {
			t.Error("expected error for missing email")
		}
	})

	t.Run("missing password", func(t *testing.T) {
		bc := &config.Channel{Type: config.ChannelDeltaChat, Enabled: true}
		cfg := &config.DeltaChatSettings{Email: "bot@example.org", RPCServerPath: fakeServer}
		if _, err := NewDeltaChatChannel(bc, cfg, msgBus); err == nil {
			t.Error("expected error for missing password")
		}
	})

	t.Run("missing rpc server", func(t *testing.T) {
		bc := &config.Channel{Type: config.ChannelDeltaChat, Enabled: true}
		cfg := &config.DeltaChatSettings{
			Email:         "bot@example.org",
			Password:      *config.NewSecureString("pw"),
			RPCServerPath: filepath.Join(t.TempDir(), "does-not-exist"),
		}
		if _, err := NewDeltaChatChannel(bc, cfg, msgBus); err == nil {
			t.Error("expected error for missing rpc server path")
		}
	})

	t.Run("valid config", func(t *testing.T) {
		bc := &config.Channel{Type: config.ChannelDeltaChat, Enabled: true}
		cfg := &config.DeltaChatSettings{
			Email:         "bot@example.org",
			Password:      *config.NewSecureString("pw"),
			RPCServerPath: fakeServer,
			DataDir:       t.TempDir(),
		}
		ch, err := NewDeltaChatChannel(bc, cfg, msgBus)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ch.Name() != config.ChannelDeltaChat {
			t.Errorf("Name() = %q, want %q", ch.Name(), config.ChannelDeltaChat)
		}
		if ch.IsRunning() {
			t.Error("new channel should not be running")
		}
	})
}

func TestMentionsBot(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		displayName string
		email       string
		want        bool
	}{
		{"display name", "hey PicoBot can you help", "PicoBot", "bot@example.org", true},
		{"case insensitive name", "hey picobot", "PicoBot", "bot@example.org", true},
		{"email local part", "@bot please summarize", "", "bot@example.org", true},
		{"no mention", "just chatting here", "PicoBot", "bot@example.org", false},
		{"local part without @", "the robot is cool", "", "bot@example.org", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mentionsBot(tt.content, tt.displayName, tt.email); got != tt.want {
				t.Errorf("mentionsBot(%q, %q, %q) = %v, want %v", tt.content, tt.displayName, tt.email, got, tt.want)
			}
		})
	}
}

func TestExpandHome(t *testing.T) {
	home, _ := os.UserHomeDir()
	tests := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"/abs/path", "/abs/path"},
		{"~", home},
		{"~/sub", filepath.Join(home, "sub")},
		{"relative", "relative"},
	}
	for _, tt := range tests {
		if got := expandHome(tt.in); got != tt.want {
			t.Errorf("expandHome(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestResolveDataDir(t *testing.T) {
	if got := resolveDataDir("/explicit/dir", "x"); got != "/explicit/dir" {
		t.Errorf("explicit data dir = %q, want /explicit/dir", got)
	}
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".picoclaw", "deltachat", "mychan")
	if got := resolveDataDir("", "mychan"); got != want {
		t.Errorf("default data dir = %q, want %q", got, want)
	}
}

func TestDeltaChatSettingsDecode(t *testing.T) {
	raw := []byte(`{
		"enabled": true,
		"type": "deltachat",
		"allow_from": ["alice@example.org"],
		"settings": {
			"email": "bot@example.org",
			"display_name": "PicoBot",
			"imap_port": 993
		}
	}`)
	var bc config.Channel
	if err := json.Unmarshal(raw, &bc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	bc.Type = config.ChannelDeltaChat
	decoded, err := bc.GetDecoded()
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	cfg, ok := decoded.(*config.DeltaChatSettings)
	if !ok {
		t.Fatalf("decoded type = %T, want *config.DeltaChatSettings", decoded)
	}
	if cfg.Email != "bot@example.org" {
		t.Errorf("email = %q, want bot@example.org", cfg.Email)
	}
	if cfg.DisplayName != "PicoBot" {
		t.Errorf("display_name = %q, want PicoBot", cfg.DisplayName)
	}
	if cfg.IMAPPort != 993 {
		t.Errorf("imap_port = %d, want 993", cfg.IMAPPort)
	}
}

// TestRPCClientRoundTrip drives the JSON-RPC client against an in-process mock
// server over pipes, verifying id correlation and error propagation.
func TestRPCClientRoundTrip(t *testing.T) {
	reqR, reqW := io.Pipe()   // client -> server
	respR, respW := io.Pipe() // server -> client

	c := &rpcClient{
		stdin:   reqW,
		stdout:  respR,
		pending: make(map[uint64]chan rpcResponse),
	}
	go c.readLoop()

	// Mock server: echo method "ping" -> "pong", anything else -> error.
	go func() {
		scanner := bufio.NewScanner(reqR)
		for scanner.Scan() {
			var req rpcRequest
			if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
				continue
			}
			var resp string
			if req.Method == "ping" {
				resp = `{"jsonrpc":"2.0","id":` + itoa(req.ID) + `,"result":"pong"}`
			} else {
				resp = `{"jsonrpc":"2.0","id":` + itoa(req.ID) + `,"error":{"code":-1,"message":"boom"}}`
			}
			_, _ = respW.Write([]byte(resp + "\n"))
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	raw, err := c.call(ctx, "ping")
	if err != nil {
		t.Fatalf("ping call: %v", err)
	}
	var result string
	if err := json.Unmarshal(raw, &result); err != nil || result != "pong" {
		t.Fatalf("ping result = %q (err %v), want pong", result, err)
	}

	if _, err := c.call(ctx, "explode"); err == nil {
		t.Fatal("expected error from explode call")
	}
}

func itoa(n uint64) string {
	b, _ := json.Marshal(n)
	return string(b)
}
