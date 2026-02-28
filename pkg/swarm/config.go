// PicoClaw - Ultra-lightweight personal AI agent
// Swarm mode support for multi-agent coordination
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"encoding/json"
	"time"
)

// Config contains all configuration for swarm mode.
type Config struct {
	// Enabled enables swarm mode.
	Enabled bool `json:"enabled" env:"PICOCLAW_SWARM_ENABLED"`

	// NodeID is the unique identifier for this node.
	// If empty, a hostname-based ID will be generated.
	NodeID string `json:"node_id,omitempty" env:"PICOCLAW_SWARM_NODE_ID"`

	// BindAddr is the address to bind for gossip and RPC.
	BindAddr string `json:"bind_addr,omitempty" env:"PICOCLAW_SWARM_BIND_ADDR"`

	// BindPort is the port for gossip protocol.
	BindPort int `json:"bind_port,omitempty" env:"PICOCLAW_SWARM_BIND_PORT"`

	// AdvertiseAddr is the address to advertise to other nodes.
	// If empty, BindAddr will be used.
	AdvertiseAddr string `json:"advertise_addr,omitempty" env:"PICOCLAW_SWARM_ADVERTISE_ADDR"`

	// AdvertisePort is the port to advertise to other nodes.
	// If 0, BindPort will be used.
	AdvertisePort int `json:"advertise_port,omitempty" env:"PICOCLAW_SWARM_ADVERTISE_PORT"`

	// Discovery configuration for node discovery.
	Discovery DiscoveryConfig `json:"discovery"`

	// Handoff configuration for task handoff.
	Handoff HandoffConfig `json:"handoff"`

	// RPC configuration for inter-node communication.
	RPC RPCConfig `json:"rpc"`

	// LoadMonitor configuration for load monitoring.
	LoadMonitor LoadMonitorConfig `json:"load_monitor"`

	// LeaderElection configuration for leader election.
	LeaderElection LeaderElectionConfig `json:"leader_election"`

	// Metrics configuration for observability.
	Metrics MetricsConfig `json:"metrics"`

	// HTTPPort is the HTTP/gateway port where the Pico channel is served.
	// Used for inter-node communication via the Pico WebSocket protocol.
	HTTPPort int `json:"http_port,omitempty"`
}

// DiscoveryConfig contains configuration for node discovery.
type DiscoveryConfig struct {
	// JoinAddrs is a list of existing nodes to join.
	JoinAddrs []string `json:"join_addrs,omitempty"`

	// GossipInterval is the interval between gossip messages.
	GossipInterval Duration `json:"gossip_interval,omitempty"`

	// PushPullInterval is the interval for full state sync.
	PushPullInterval Duration `json:"push_pull_interval,omitempty"`

	// NodeTimeout is the timeout before marking a node as suspect.
	NodeTimeout Duration `json:"node_timeout,omitempty"`

	// DeadNodeTimeout is the timeout before marking a node as dead.
	DeadNodeTimeout Duration `json:"dead_node_timeout,omitempty"`

	// AuthSecret is the shared secret for node authentication.
	// If empty, authentication is disabled (not recommended for production).
	AuthSecret string `json:"auth_secret,omitempty"`

	// RequireAuth requires all nodes to be authenticated.
	RequireAuth bool `json:"require_auth"`

	// EnableMessageSigning enables HMAC signing of all messages.
	EnableMessageSigning bool `json:"enable_message_signing"`
}

// HandoffConfig contains configuration for task handoff.
type HandoffConfig struct {
	// Enabled enables task handoff.
	Enabled bool `json:"enabled"`

	// LoadThreshold is the load score threshold (0-1) above which
	// tasks will be handed off to other nodes.
	LoadThreshold float64 `json:"load_threshold,omitempty"`

	// Timeout is the timeout for a handoff operation.
	Timeout Duration `json:"timeout,omitempty"`

	// MaxRetries is the maximum number of retries for handoff.
	MaxRetries int `json:"max_retries,omitempty"`

	// RetryDelay is the delay between retries.
	RetryDelay Duration `json:"retry_delay,omitempty"`
}

// RPCConfig contains configuration for RPC communication.
type RPCConfig struct {
	// Port is the port for RPC communication.
	Port int `json:"port,omitempty" env:"PICOCLAW_SWARM_RPC_PORT"`

	// Timeout is the default timeout for RPC calls.
	Timeout Duration `json:"timeout,omitempty"`
}

// LoadMonitorConfig contains configuration for load monitoring.
type LoadMonitorConfig struct {
	// Enabled enables load monitoring.
	Enabled bool `json:"enabled"`

	// Interval is the interval between load samples.
	Interval Duration `json:"interval,omitempty"`

	// SampleSize is the number of samples to keep for averaging.
	SampleSize int `json:"sample_size,omitempty"`

	// CPUWeight is the weight for CPU usage in load score (0-1).
	CPUWeight float64 `json:"cpu_weight,omitempty"`

	// MemoryWeight is the weight for memory usage in load score (0-1).
	MemoryWeight float64 `json:"memory_weight,omitempty"`

	// SessionWeight is the weight for active sessions in load score (0-1).
	SessionWeight float64 `json:"session_weight,omitempty"`

	// OffloadThreshold is the load score threshold above which tasks should be offloaded (0-1).
	OffloadThreshold float64 `json:"offload_threshold,omitempty"`

	// MaxMemoryBytes is the maximum memory to use for normalization (default: 1GB).
	MaxMemoryBytes uint64 `json:"max_memory_bytes,omitempty"`

	// MaxGoroutines is the maximum goroutine count for normalization (default: 1000).
	MaxGoroutines int `json:"max_goroutines,omitempty"`

	// MaxSessions is the maximum session count for normalization (default: 100).
	MaxSessions int `json:"max_sessions,omitempty"`
}

// LeaderElectionConfig contains configuration for leader election.
type LeaderElectionConfig struct {
	// Enabled enables leader election.
	Enabled bool `json:"enabled"`

	// ElectionInterval is how often to check leadership.
	ElectionInterval Duration `json:"election_interval,omitempty"`

	// LeaderHeartbeatTimeout is how long before assuming leader is dead.
	LeaderHeartbeatTimeout Duration `json:"leader_heartbeat_timeout,omitempty"`
}

// MetricsConfig contains configuration for metrics collection.
type MetricsConfig struct {
	// Enabled enables metrics collection.
	Enabled bool `json:"enabled"`

	// ExportInterval is how often to export metrics.
	ExportInterval Duration `json:"export_interval,omitempty"`

	// PrometheusEnabled enables Prometheus format export.
	PrometheusEnabled bool `json:"prometheus_enabled"`

	// PrometheusEndpoint is the HTTP endpoint for Prometheus metrics.
	PrometheusEndpoint string `json:"prometheus_endpoint,omitempty"`
}

// Duration is a wrapper around time.Duration for JSON parsing.
type Duration struct {
	time.Duration
}

// UnmarshalJSON parses a duration from JSON.
func (d *Duration) UnmarshalJSON(b []byte) error {
	// Check if it's a string (quoted)
	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := parseJSONString(b, &s); err != nil {
			return err
		}
		var err error
		d.Duration, err = time.ParseDuration(s)
		return err
	}

	// Otherwise it's a number (milliseconds)
	var v float64
	if err := parseJSONNumber(b, &v); err != nil {
		return err
	}
	d.Duration = time.Duration(v)
	return nil
}

// MarshalJSON converts a duration to JSON.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Duration.String())
}

// DefaultConfig returns the default swarm configuration.
func DefaultConfig() *Config {
	return &Config{
		Enabled:  false,
		NodeID:   "",
		BindAddr: "0.0.0.0",
		BindPort: DefaultBindPort,
		Discovery: DiscoveryConfig{
			JoinAddrs:            nil,
			GossipInterval:       Duration{DefaultGossipInterval},
			PushPullInterval:     Duration{DefaultPushPullInterval},
			NodeTimeout:          Duration{DefaultNodeTimeout},
			DeadNodeTimeout:      Duration{DefaultDeadNodeTimeout},
			AuthSecret:           "",
			RequireAuth:          false,
			EnableMessageSigning: false,
		},
		Handoff: HandoffConfig{
			Enabled:       true,
			LoadThreshold: DefaultLoadThreshold,
			Timeout:       Duration{DefaultHandoffTimeout},
			MaxRetries:    DefaultMaxHandoffRetries,
			RetryDelay:    Duration{DefaultHandoffRetryDelay},
		},
		RPC: RPCConfig{
			Port:    DefaultRPCPort,
			Timeout: Duration{10 * time.Second},
		},
		LoadMonitor: LoadMonitorConfig{
			Enabled:          true,
			Interval:         Duration{DefaultLoadSampleInterval},
			SampleSize:       DefaultLoadSampleSize,
			CPUWeight:        DefaultCPUWeight,
			MemoryWeight:     DefaultMemoryWeight,
			SessionWeight:    DefaultSessionWeight,
			OffloadThreshold: DefaultOffloadThreshold,
			MaxMemoryBytes:   DefaultMaxMemoryBytes,
			MaxGoroutines:    DefaultMaxGoroutines,
			MaxSessions:      DefaultMaxSessions,
		},
		LeaderElection: LeaderElectionConfig{
			Enabled:                false,
			ElectionInterval:       Duration{5 * time.Second},
			LeaderHeartbeatTimeout: Duration{10 * time.Second},
		},
		Metrics: MetricsConfig{
			Enabled:            false,
			ExportInterval:     Duration{10 * time.Second},
			PrometheusEnabled:  false,
			PrometheusEndpoint: "/metrics",
		},
	}
}

// parseJSONString parses a JSON string (including quotes).
func parseJSONString(b []byte, s *string) error {
	if len(b) < 2 || b[0] != '"' || b[len(b)-1] != '"' {
		return &json.UnmarshalTypeError{}
	}
	*s = string(b[1 : len(b)-1])
	return nil
}

// parseJSONNumber parses a JSON number.
func parseJSONNumber(b []byte, f *float64) error {
	n, err := json.Number(string(b)).Int64()
	if err == nil {
		*f = float64(n)
		return nil
	}
	*f, err = json.Number(string(b)).Float64()
	return err
}
