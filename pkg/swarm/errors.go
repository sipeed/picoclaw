// PicoClaw - Ultra-lightweight personal AI agent
// Swarm mode support for multi-agent coordination
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import "errors"

var (
	// ErrNodeNotFound is returned when a node is not found in the cluster.
	ErrNodeNotFound = errors.New("node not found")

	// ErrNodeNotAvailable is returned when a node is not available for handoff.
	ErrNodeNotAvailable = errors.New("node not available")

	// ErrNoHealthyNodes is returned when no healthy nodes are available.
	ErrNoHealthyNodes = errors.New("no healthy nodes available")

	// ErrHandoffTimeout is returned when a handoff operation times out.
	ErrHandoffTimeout = errors.New("handoff timeout")

	// ErrHandoffRejected is returned when a handoff is rejected by the target node.
	ErrHandoffRejected = errors.New("handoff rejected")

	// ErrHandoffInProgress is returned when a handoff is already in progress.
	ErrHandoffInProgress = errors.New("handoff already in progress")

	// ErrInvalidNodeInfo is returned when node information is invalid.
	ErrInvalidNodeInfo = errors.New("invalid node information")

	// ErrDiscoveryDisabled is returned when discovery is disabled.
	ErrDiscoveryDisabled = errors.New("discovery disabled")

	// ErrTransportClosed is returned when the transport is closed.
	ErrTransportClosed = errors.New("transport closed")

	// ErrSessionNotFound is returned when a session is not found.
	ErrSessionNotFound = errors.New("session not found")

	// ErrCapabilityNotSupported is returned when a required capability is not supported.
	ErrCapabilityNotSupported = errors.New("capability not supported")

	// ErrAuthenticationFailed is returned when authentication fails.
	ErrAuthenticationFailed = errors.New("authentication failed")

	// ErrInvalidSignature is returned when a signature verification fails.
	ErrInvalidSignature = errors.New("invalid signature")
)
