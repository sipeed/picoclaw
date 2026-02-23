// PicoClaw - Ultra-lightweight personal AI agent
// Swarm mode support for multi-agent coordination
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import "time"

const (
	// Default values for configurable parameters

	// DefaultBindPort is the default port for gossip protocol.
	DefaultBindPort = 7946

	// DefaultRPCPort is the default port for RPC communication.
	DefaultRPCPort = 7947

	// DefaultNodeTimeout is the default timeout before marking a node as suspect.
	DefaultNodeTimeout = 5 * time.Second

	// DefaultDeadNodeTimeout is the default timeout before removing a dead node.
	DefaultDeadNodeTimeout = 30 * time.Second

	// DefaultGossipInterval is the default interval between gossip messages.
	DefaultGossipInterval = 1 * time.Second

	// DefaultPushPullInterval is the default interval for full state sync.
	DefaultPushPullInterval = 30 * time.Second

	// DefaultHandoffTimeout is the default timeout for a handoff operation.
	DefaultHandoffTimeout = 30 * time.Second

	// DefaultHandoffRetryDelay is the default delay between handoff retries.
	DefaultHandoffRetryDelay = 5 * time.Second

	// DefaultMaxHandoffRetries is the default maximum number of handoff retries.
	DefaultMaxHandoffRetries = 3

	// DefaultLoadSampleInterval is the default interval between load samples.
	DefaultLoadSampleInterval = 5 * time.Second

	// DefaultLoadSampleSize is the default number of load samples to keep.
	DefaultLoadSampleSize = 60

	// Thresholds and limits

	// DefaultLoadThreshold is the default load score threshold for handoff.
	DefaultLoadThreshold = 0.8

	// DefaultAvailableLoadThreshold is the threshold below which a node is considered available (0-1).
	DefaultAvailableLoadThreshold = 0.9

	// DefaultOffloadThreshold is the default threshold above which tasks should be offloaded (0-1).
	DefaultOffloadThreshold = 0.8

	// DefaultMaxMemoryBytes is the default max memory for normalization (1GB).
	DefaultMaxMemoryBytes = 1024 * 1024 * 1024

	// DefaultMaxGoroutines is the default max goroutine count for normalization.
	DefaultMaxGoroutines = 1000

	// DefaultMaxSessions is the default max session count for normalization.
	DefaultMaxSessions = 100

	// DefaultCPUWeight is the default weight for CPU in load score calculation.
	DefaultCPUWeight = 0.3

	// DefaultMemoryWeight is the default weight for memory in load score calculation.
	DefaultMemoryWeight = 0.3

	// DefaultSessionWeight is the default weight for sessions in load score calculation.
	DefaultSessionWeight = 0.4

	// Buffer sizes

	// MaxGossipMessageSize is the maximum size of a gossip message (64KB).
	MaxGossipMessageSize = 64 * 1024

	// MaxSessionMessageSize is the maximum size of a session transfer message (128KB).
	MaxSessionMessageSize = 128 * 1024

	// UDP write deadline

	// DefaultUDPWriteDeadline is the default write deadline for UDP operations.
	DefaultUDPWriteDeadline = 5 * time.Second

	// Poll intervals

	// HandoffResponsePollInterval is the interval for polling handoff responses.
	HandoffResponsePollInterval = 100 * time.Millisecond

	// Trend analysis

	// TrendIncreasingThreshold is the slope threshold for detecting increasing trend.
	TrendIncreasingThreshold = 0.01

	// TrendDecreasingThreshold is the slope threshold for detecting decreasing trend.
	TrendDecreasingThreshold = -0.01
)
