// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package identity

import (
	"fmt"
	"strings"
	"sync"
)

// Identity represents a two-level identity model:
// - H-id (Host ID): Represents a tenant/user/group for multi-tenancy isolation
// - S-id (Service ID): Represents a specific instance/node within the H-id
//
// Examples:
//   - H-id: "user-alice", "org-company", "group-team1"
//   - S-id: "node-01", "worker-primary", "coordinator-main"
//
// Full identity: "user-alice/node-01" or "org-company/worker-1"
type Identity struct {
	// HID is the host/tenant identifier (e.g., "user-alice", "org-company")
	HID string `json:"hid"`

	// SID is the service/instance identifier (e.g., "node-01", "worker-primary")
	SID string `json:"sid"`

	// DisplayName is a human-readable name
	DisplayName string `json:"display_name,omitempty"`

	// Metadata contains additional identity information
	Metadata map[string]string `json:"metadata,omitempty"`

	mu sync.RWMutex
}

// NewIdentity creates a new Identity with the given H-id and S-id
func NewIdentity(hid, sid string) *Identity {
	return &Identity{
		HID:      normalizeID(hid),
		SID:      normalizeID(sid),
		Metadata: make(map[string]string),
	}
}

// NewIdentityFromString parses an identity string in format "hid/sid"
func NewIdentityFromString(s string) (*Identity, error) {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid identity format: %q (expected \"hid/sid\")", s)
	}

	hid := strings.TrimSpace(parts[0])
	sid := strings.TrimSpace(parts[1])

	if hid == "" || sid == "" {
		return nil, fmt.Errorf("invalid identity: hid and sid must not be empty")
	}

	return NewIdentity(hid, sid), nil
}

// String returns the identity in "hid/sid" format
func (id *Identity) String() string {
	id.mu.RLock()
	defer id.mu.RUnlock()
	return fmt.Sprintf("%s/%s", id.HID, id.SID)
}

// FullID returns the full identity string
func (id *Identity) FullID() string {
	return id.String()
}

// IsValid returns true if the identity is valid
func (id *Identity) IsValid() bool {
	id.mu.RLock()
	defer id.mu.RUnlock()

	if id.HID == "" || id.SID == "" {
		return false
	}

	// Check for invalid characters
	for _, c := range id.HID + id.SID {
		if !isValidIDChar(c) {
			return false
		}
	}

	return true
}

// Clone returns a deep copy of the identity
func (id *Identity) Clone() *Identity {
	id.mu.RLock()
	defer id.mu.RUnlock()

	clone := &Identity{
		HID:         id.HID,
		SID:         id.SID,
		DisplayName: id.DisplayName,
		Metadata:    make(map[string]string, len(id.Metadata)),
	}

	for k, v := range id.Metadata {
		clone.Metadata[k] = v
	}

	return clone
}

// SetMetadata sets a metadata key-value pair
func (id *Identity) SetMetadata(key, value string) {
	id.mu.Lock()
	defer id.mu.Unlock()
	if id.Metadata == nil {
		id.Metadata = make(map[string]string)
	}
	id.Metadata[key] = value
}

// GetMetadata gets a metadata value by key
func (id *Identity) GetMetadata(key string) (string, bool) {
	id.mu.RLock()
	defer id.mu.RUnlock()
	v, ok := id.Metadata[key]
	return v, ok
}

// IsSameTenant checks if two identities belong to the same tenant (same H-id)
func (id *Identity) IsSameTenant(other *Identity) bool {
	if other == nil {
		return false
	}
	id.mu.RLock()
	defer id.mu.RUnlock()
	other.mu.RLock()
	defer other.mu.RUnlock()
	return id.HID == other.HID
}

// Equals checks if two identities are exactly the same
func (id *Identity) Equals(other *Identity) bool {
	if other == nil {
		return false
	}
	id.mu.RLock()
	defer id.mu.RUnlock()
	other.mu.RLock()
	defer other.mu.RUnlock()
	return id.HID == other.HID && id.SID == other.SID
}

// normalizeID cleans up an ID by trimming whitespace and lowercasing
func normalizeID(id string) string {
	return strings.TrimSpace(strings.ToLower(id))
}

// isValidIDChar checks if a character is valid for an ID
func isValidIDChar(c rune) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '_' || c == '.'
}

// Source defines where an identity was loaded from
type Source int

const (
	// SourceUnknown is when the source is unknown
	SourceUnknown Source = iota
	// SourceCLI is from command-line arguments
	SourceCLI
	// SourceEnv is from environment variables
	SourceEnv
	// SourceConfig is from configuration file
	SourceConfig
	// SourceAuto is auto-generated
	SourceAuto
)

// String returns the string representation of the source
func (s Source) String() string {
	switch s {
	case SourceCLI:
		return "cli"
	case SourceEnv:
		return "env"
	case SourceConfig:
		return "config"
	case SourceAuto:
		return "auto"
	default:
		return "unknown"
	}
}

// LoadedIdentity includes the identity and its source
type LoadedIdentity struct {
	*Identity
	Source Source `json:"source"`
}

// NewLoadedIdentity creates a new LoadedIdentity
func NewLoadedIdentity(id *Identity, source Source) *LoadedIdentity {
	return &LoadedIdentity{
		Identity: id,
		Source:   source,
	}
}
