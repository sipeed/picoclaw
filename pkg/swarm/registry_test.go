// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCapabilityRegistry(t *testing.T) {
	nodeInfo := &NodeInfo{
		ID:   "test-node-1",
		Role: RoleWorker,
	}

	registry := NewCapabilityRegistry(nodeInfo, nil, nil)
	assert.NotNil(t, registry)
	assert.Equal(t, nodeInfo, registry.nodeInfo)
}

func TestCapabilityRegistry_Register(t *testing.T) {
	t.Skip("Requires NATS connection")

	/*
	nodeInfo := &NodeInfo{
		ID:   "test-node-1",
		Role: RoleWorker,
	}

	nc, _ := nats.Connect(nats.DefaultURL)
	defer nc.Close()

	js, _ := nc.JetStream()

	registry := NewCapabilityRegistry(nodeInfo, js, nc)
	err := registry.Initialize(context.Background())
	require.NoError(t, err)

	// Register a capability
	err = registry.Register("test-cap", "A test capability", "1.0.0", nil)
	require.NoError(t, err)

	// Check local storage
	cap, ok := registry.Get("test-cap")
	assert.True(t, ok)
	assert.Equal(t, "test-cap", cap.Name)
	assert.Equal(t, "A test capability", cap.Description)
	*/
}

func TestCapabilityRegistry_List(t *testing.T) {
	nodeInfo := &NodeInfo{
		ID:   "test-node-1",
		Role: RoleWorker,
	}

	registry := NewCapabilityRegistry(nodeInfo, nil, nil)

	// Add capabilities to local store
	registry.caps.Store("cap-1", &Capability{Name: "cap-1"})
	registry.caps.Store("cap-2", &Capability{Name: "cap-2"})

	caps := registry.List()
	assert.Len(t, caps, 2)
}

func TestCapabilityRegistry_Get(t *testing.T) {
	nodeInfo := &NodeInfo{
		ID:   "test-node-1",
		Role: RoleWorker,
	}

	registry := NewCapabilityRegistry(nodeInfo, nil, nil)

	expectedCap := &Capability{
		Name:        "test-cap",
		Description: "Test",
		Version:     "1.0.0",
		NodeID:      "test-node-1",
	}

	registry.caps.Store("test-cap", expectedCap)

	cap, ok := registry.Get("test-cap")
	assert.True(t, ok)
	assert.Equal(t, expectedCap, cap)

	_, ok = registry.Get("non-existent")
	assert.False(t, ok)
}

func TestCapabilitySubject(t *testing.T) {
	assert.Equal(t, "picoclaw.swarm.capability", CapabilitySubject)
}

func TestCapabilityKVBucket(t *testing.T) {
	assert.Equal(t, "PICOCLAW_CAPABILITIES", CapabilityKVBucket)
}
