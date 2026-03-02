package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/openclaw/go_node/infra"
)

// Config holds gateway and node settings
type Config struct {
	Gateway   GatewayConfig   `json:"gateway"`
	Node      NodeConfig      `json:"node"`
	Reconnect ReconnectConfig `json:"reconnect"`
	Exec      ExecConfig      `json:"exec"`
	Identity  IdentityConfig  `json:"identity,omitempty"`
}

// ExecConfig holds security settings for system.run and camera mock
type ExecConfig struct {
	WorkDir          string   `json:"workDir"`          // base dir, commands run under this only
	AllowedCommands  []string `json:"allowedCommands"`  // allowed executable names, empty = allow all
	AllowAllCommands bool     `json:"allowAllCommands"` // when true, bypass allowedCommands check and allow any command (including rawCommand)
	CameraMockPath   string   `json:"cameraMockPath"`   // fixed path for camera.snap mock image (default: workDir/camera_mock.jpg)
}

// ReconnectConfig holds auto-reconnect settings
type ReconnectConfig struct {
	MaxRetries      int `json:"maxRetries"`      // 0 = unlimited
	RetryIntervalMs int `json:"retryIntervalMs"` // milliseconds between retries
}

// GatewayConfig holds gateway connection settings
type GatewayConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	TLS      bool   `json:"tls"`
	Token    string `json:"token"`
	Password string `json:"password"`
}

// NodeConfig holds node identity settings
type NodeConfig struct {
	DisplayName string `json:"displayName"`
	NodeID      string `json:"nodeId"`
}

// IdentityConfig holds device identity (Ed25519 keypair) persisted in config.json
type IdentityConfig struct {
	DeviceID      string `json:"deviceId,omitempty"`
	PublicKeyB64  string `json:"publicKeyB64,omitempty"`
	PrivateKeyB64 string `json:"privateKeyB64,omitempty"`
}

// Load loads config from JSON file. Path can be absolute or relative.
// If path is empty, tries: ./config.json, ~/.openclaw/go_node.json
func Load(path string) (*Config, error) {
	if path == "" {
		for _, p := range defaultConfigPaths() {
			if _, err := os.Stat(p); err == nil {
				path = p
				break
			}
		}
	}
	if path == "" {
		return defaultConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	cfg.applyDefaults()
	return &cfg, nil
}

func defaultConfigPaths() []string {
	home, _ := os.UserHomeDir()
	return []string{
		"config.json",
		filepath.Join(home, ".openclaw", "go_node.json"),
	}
}

func defaultConfig() *Config {
	displayName, _ := os.Hostname()
	return &Config{
		Gateway: GatewayConfig{
			Host: "127.0.0.1",
			Port: 18789,
			TLS:  false,
		},
		Node: NodeConfig{
			DisplayName: displayName,
			NodeID:      displayName,
		},
		Reconnect: ReconnectConfig{
			MaxRetries:      0,
			RetryIntervalMs: 5000,
		},
		Exec: ExecConfig{
			WorkDir:         "",
			AllowedCommands: nil,
		},
	}
}

func (c *Config) applyDefaults() {
	g := &c.Gateway
	if g.Host == "" {
		g.Host = "127.0.0.1"
	}
	if g.Port <= 0 {
		g.Port = 18789
	}
	if c.Node.DisplayName == "" {
		c.Node.DisplayName, _ = os.Hostname()
	}
	if c.Node.NodeID == "" {
		c.Node.NodeID = c.Node.DisplayName
	}
	r := &c.Reconnect
	if r.RetryIntervalMs <= 0 {
		r.RetryIntervalMs = 5000
	}
	e := &c.Exec
	if e.WorkDir != "" {
		abs, err := filepath.Abs(e.WorkDir)
		if err == nil {
			e.WorkDir = abs
		}
	}
}

// WebSocketURL returns the gateway WebSocket URL
func (c *Config) WebSocketURL() string {
	scheme := "ws"
	if c.Gateway.TLS {
		scheme = "wss"
	}
	return fmt.Sprintf("%s://%s:%d", scheme, c.Gateway.Host, c.Gateway.Port)
}

// Example writes an example config to path
func Example(path string) error {
	cfg := defaultConfig()
	cfg.Gateway.Token = ""
	if ident, err := infra.GenerateDeviceIdentity(); err == nil {
		cfg.Identity = IdentityConfig{
			DeviceID:      ident.DeviceID,
			PublicKeyB64:  base64.StdEncoding.EncodeToString(ident.PublicKeyRaw),
			PrivateKeyB64: base64.StdEncoding.EncodeToString(ident.PrivateKey),
		}
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}
	return os.WriteFile(path, data, 0600)
}
