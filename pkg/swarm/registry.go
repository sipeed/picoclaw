// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/sipeed/picoclaw/pkg/logger"
)

const (
	// CapabilitySubject is the base subject for capability discovery
	CapabilitySubject = "picoclaw.swarm.capability"
	// CapabilityKVBucket is the KV bucket for storing capabilities
	CapabilityKVBucket = "PICOCLAW_CAPABILITIES"
)

// CapabilityRegistry manages dynamic capability registration and discovery
type CapabilityRegistry struct {
	caps     sync.Map // map[string]*Capability
	nodeInfo *NodeInfo
	js       nats.JetStreamContext
	nc       *nats.Conn
	mu       sync.RWMutex
}

// NewCapabilityRegistry creates a new capability registry
func NewCapabilityRegistry(nodeInfo *NodeInfo, js nats.JetStreamContext, nc *nats.Conn) *CapabilityRegistry {
	return &CapabilityRegistry{
		nodeInfo: nodeInfo,
		js:       js,
		nc:       nc,
	}
}

// Initialize sets up the capability registry
func (r *CapabilityRegistry) Initialize(ctx context.Context) error {
	// Create KV bucket for capabilities
	_, err := r.js.KeyValue(CapabilityKVBucket)
	if err != nil {
		// Bucket doesn't exist, create it
		_, err = r.js.CreateKeyValue(&nats.KeyValueConfig{
			Bucket:      CapabilityKVBucket,
			Description: "PicoClaw swarm capability registry",
			MaxBytes:    1024 * 1024 * 10, // 10MB
			Storage:     nats.FileStorage,
			Replicas:    1,
		})
		if err != nil {
			return fmt.Errorf("failed to create capabilities bucket: %w", err)
		}
		logger.InfoC("swarm", fmt.Sprintf("Created capability bucket: %s", CapabilityKVBucket))
	}

	// Subscribe to capability announcements for discovery
	_, err = r.nc.Subscribe(CapabilitySubject+".announce", func(msg *nats.Msg) {
		r.handleCapabilityAnnouncement(msg)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to capability announcements: %w", err)
	}

	// Subscribe to capability queries
	_, err = r.nc.Subscribe(CapabilitySubject+".query", func(msg *nats.Msg) {
		r.handleCapabilityQuery(msg)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to capability queries: %w", err)
	}

	logger.InfoC("swarm", "Capability registry initialized")

	return nil
}

// Register registers a capability for this node
func (r *CapabilityRegistry) Register(name, description, version string, metadata map[string]interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cap := &Capability{
		Name:        name,
		Description: description,
		Version:     version,
		Metadata:    metadata,
		NodeID:      r.nodeInfo.ID,
		RegisteredAt: time.Now().UnixMilli(),
	}

	// Store locally
	r.caps.Store(name, cap)

	// Store in KV for cross-node discovery
	kv, err := r.js.KeyValue(CapabilityKVBucket)
	if err != nil {
		return fmt.Errorf("failed to get capabilities bucket: %w", err)
	}

	key := fmt.Sprintf("%s:%s", r.nodeInfo.ID, name)
	data, err := json.Marshal(cap)
	if err != nil {
		return fmt.Errorf("failed to marshal capability: %w", err)
	}

	_, err = kv.Put(key, data)
	if err != nil {
		return fmt.Errorf("failed to store capability: %w", err)
	}

	// Announce to swarm
	announcement := map[string]interface{}{
		"type":        "register",
		"capability":  cap,
		"timestamp":   time.Now().UnixMilli(),
	}
	announcementData, _ := json.Marshal(announcement)

	err = r.nc.Publish(CapabilitySubject+".announce", announcementData)
	if err != nil {
		logger.WarnCF("swarm", "Failed to announce capability", map[string]interface{}{
			"name":  name,
			"error": err.Error(),
		})
	}

	logger.InfoCF("swarm", "Registered capability", map[string]interface{}{
		"name":    name,
		"version": version,
	})

	return nil
}

// Unregister removes a capability
func (r *CapabilityRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Remove from local store
	r.caps.Delete(name)

	// Remove from KV
	kv, err := r.js.KeyValue(CapabilityKVBucket)
	if err != nil {
		return fmt.Errorf("failed to get capabilities bucket: %w", err)
	}

	key := fmt.Sprintf("%s:%s", r.nodeInfo.ID, name)
	err = kv.Delete(key)
	if err != nil && err != nats.ErrKeyNotFound {
		return fmt.Errorf("failed to delete capability: %w", err)
	}

	// Announce removal
	announcement := map[string]interface{}{
		"type":      "unregister",
		"node_id":   r.nodeInfo.ID,
		"name":      name,
		"timestamp": time.Now().UnixMilli(),
	}
	announcementData, err := json.Marshal(announcement)
	if err != nil {
		return fmt.Errorf("failed to marshal announcement: %w", err)
	}

	if err := r.nc.Publish(CapabilitySubject+".announce", announcementData); err != nil {
		logger.WarnCF("swarm", "Failed to publish unregister announcement", map[string]interface{}{
			"error": err.Error(),
		})
	}

	logger.InfoCF("swarm", "Unregistered capability", map[string]interface{}{
		"name": name,
	})

	return nil
}

// Get retrieves a local capability by name
func (r *CapabilityRegistry) Get(name string) (*Capability, bool) {
	val, ok := r.caps.Load(name)
	if !ok {
		return nil, false
	}
	cap, ok := val.(*Capability)
	return cap, ok
}

// List returns all local capabilities
func (r *CapabilityRegistry) List() []Capability {
	caps := make([]Capability, 0)
	r.caps.Range(func(key, value interface{}) bool {
		if cap, ok := value.(*Capability); ok {
			caps = append(caps, *cap)
		}
		return true
	})
	return caps
}

// Discover finds capabilities across the swarm
func (r *CapabilityRegistry) Discover(ctx context.Context, name, version string) ([]Capability, error) {
	// Query KV store for capabilities
	kv, err := r.js.KeyValue(CapabilityKVBucket)
	if err != nil {
		return nil, fmt.Errorf("failed to get capabilities bucket: %w", err)
	}

	// List all keys
	watcher, err := kv.WatchAll(nats.Context(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to watch capabilities: %w", err)
	}
	defer watcher.Stop()

	capabilities := make([]Capability, 0)
	timeout := time.After(5 * time.Second)

collect:
	for {
		select {
		case entry := <-watcher.Updates():
			if entry == nil {
				break collect
			}

			var cap Capability
			if err := json.Unmarshal(entry.Value(), &cap); err != nil {
				continue
			}

			// Filter by name if specified
			if name != "" && cap.Name != name {
				continue
			}

			// Filter by version if specified
			if version != "" && cap.Version != version {
				continue
			}

			capabilities = append(capabilities, cap)

		case <-timeout:
			break collect
		}
	}

	return capabilities, nil
}

// DiscoverByNode finds all capabilities provided by a specific node
func (r *CapabilityRegistry) DiscoverByNode(ctx context.Context, nodeID string) ([]Capability, error) {
	return r.Discover(ctx, "", "")
}

// handleCapabilityAnnouncement processes capability announcements from other nodes
func (r *CapabilityRegistry) handleCapabilityAnnouncement(msg *nats.Msg) {
	var announcement struct {
		Type       string     `json:"type"`
		Capability *Capability `json:"capability,omitempty"`
		NodeID     string     `json:"node_id,omitempty"`
		Name       string     `json:"name,omitempty"`
	}

	if err := json.Unmarshal(msg.Data, &announcement); err != nil {
		return
	}

	// Ignore our own announcements
	switch announcement.Type {
	case "register":
		if announcement.Capability != nil && announcement.Capability.NodeID != r.nodeInfo.ID {
			logger.DebugCF("swarm", "Capability announced by peer", map[string]interface{}{
				"node_id": announcement.Capability.NodeID,
				"name":    announcement.Capability.Name,
			})
			// Store peer capability
			r.caps.Store(announcement.Capability.NodeID+":"+announcement.Capability.Name, announcement.Capability)
		}
	case "unregister":
		if announcement.NodeID != r.nodeInfo.ID {
			logger.DebugCF("swarm", "Capability unregistered by peer", map[string]interface{}{
				"node_id": announcement.NodeID,
				"name":    announcement.Name,
			})
			r.caps.Delete(announcement.NodeID + ":" + announcement.Name)
		}
	}
}

// handleCapabilityQuery responds to capability discovery queries
func (r *CapabilityRegistry) handleCapabilityQuery(msg *nats.Msg) {
	var query CapabilityRequest
	if err := json.Unmarshal(msg.Data, &query); err != nil {
		return
	}

	// Don't respond to our own queries
	if query.RequesterID == r.nodeInfo.ID {
		return
	}

	// Get our capabilities
	caps := r.List()

	// Filter if requested
	if query.Capability != "" {
		filtered := make([]Capability, 0)
		for _, cap := range caps {
			if cap.Name == query.Capability {
				if query.Version == "" || cap.Version == query.Version {
					filtered = append(filtered, cap)
				}
			}
		}
		caps = filtered
	}

	// Send response
	response := CapabilityResponse{
		Capabilities: caps,
		RequestID:    query.RequesterID,
		Timestamp:    time.Now().UnixMilli(),
	}

	data, _ := json.Marshal(response)
	_ = msg.Respond(data)
}

// QueryCapabilities sends a query to discover capabilities from other nodes
func (r *CapabilityRegistry) QueryCapabilities(ctx context.Context, capability, version string) ([]Capability, error) {
	// Send query
	query := CapabilityRequest{
		RequesterID: r.nodeInfo.ID,
		Capability:  capability,
		Version:     version,
	}

	queryData, _ := json.Marshal(query)

	// Create inbox for responses
	inbox := nats.NewInbox()
	sub, err := r.nc.SubscribeSync(inbox)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}
	defer sub.Unsubscribe()

	// Publish query
	err = r.nc.PublishRequest(CapabilitySubject+".query", inbox, queryData)
	if err != nil {
		return nil, fmt.Errorf("failed to publish query: %w", err)
	}

	// Collect responses with timeout
	capabilities := make([]Capability, 0)
	timeout := time.After(3 * time.Second)

	for {
		select {
		case <-timeout:
			return capabilities, nil
		default:
			msg, err := sub.NextMsg(500 * time.Millisecond)
			if err != nil {
				if err == nats.ErrTimeout {
					continue
				}
				return capabilities, nil
			}

			var response CapabilityResponse
			if err := json.Unmarshal(msg.Data, &response); err != nil {
				continue
			}

			capabilities = append(capabilities, response.Capabilities...)
		}
	}
}
