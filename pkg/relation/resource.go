// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package relation

import (
	"fmt"
	"strings"
)

// ResourceType defines the type of resource
type ResourceType string

const (
	// ResourceMemory represents a memory item
	ResourceMemory ResourceType = "memory"

	// ResourceNode represents a swarm node
	ResourceNode ResourceType = "node"

	// ResourceTask represents a swarm task
	ResourceTask ResourceType = "task"

	// ResourceChannel represents a communication channel
	ResourceChannel ResourceType = "channel"

	// ResourceConfig represents a configuration item
	ResourceConfig ResourceType = "config"

	// ResourceSkill represents a skill/ability
	ResourceSkill ResourceType = "skill"

	// ResourceWorkflow represents a workflow definition
	ResourceWorkflow ResourceType = "workflow"

	// ResourceAny is a wildcard for any resource type
	ResourceAny ResourceType = "*"
)

// ResourceID is a unique identifier for a resource
type ResourceID struct {
	// Type is the type of resource
	Type ResourceType `json:"type"`

	// HID is the tenant/owner H-id
	HID string `json:"hid,omitempty"`

	// ID is the specific resource identifier
	ID string `json:"id"`

	// Namespace is an optional namespace for the resource
	Namespace string `json:"namespace,omitempty"`
}

// NewResourceID creates a new resource ID
func NewResourceID(typ ResourceType, id string) *ResourceID {
	return &ResourceID{
		Type: typ,
		ID:   id,
	}
}

// NewResourceIDWithHID creates a new resource ID with H-id
func NewResourceIDWithHID(typ ResourceType, hid, id string) *ResourceID {
	return &ResourceID{
		Type: typ,
		HID:  hid,
		ID:   id,
	}
}

// String returns the string representation of the resource ID
func (r *ResourceID) String() string {
	parts := []string{}
	if r.Type != "" {
		parts = append(parts, string(r.Type))
	}
	if r.HID != "" {
		parts = append(parts, r.HID)
	}
	if r.Namespace != "" {
		parts = append(parts, r.Namespace)
	}
	if r.ID != "" {
		parts = append(parts, r.ID)
	}
	return strings.Join(parts, ":")
}

// ParseResourceID parses a resource ID from a string
// Format: type:hid:namespace:id or type:hid:id or type:id
func ParseResourceID(s string) (*ResourceID, error) {
	parts := strings.Split(s, ":")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid resource ID format: %s", s)
	}

	r := &ResourceID{}

	switch len(parts) {
	case 2:
		r.Type = ResourceType(parts[0])
		r.ID = parts[1]
	case 3:
		r.Type = ResourceType(parts[0])
		// Could be hid:id or namespace:id
		// Assume it's hid:id for backward compatibility
		r.HID = parts[1]
		r.ID = parts[2]
	case 4:
		r.Type = ResourceType(parts[0])
		r.HID = parts[1]
		r.Namespace = parts[2]
		r.ID = parts[3]
	default:
		return nil, fmt.Errorf("invalid resource ID format: %s", s)
	}

	return r, nil
}

// IsWildcard returns true if this resource ID is a wildcard
func (r *ResourceID) IsWildcard() bool {
	return r.Type == ResourceAny || r.ID == "*"
}

// Matches returns true if this resource ID matches the target
func (r *ResourceID) Matches(target *ResourceID) bool {
	if r == nil || target == nil {
		return false
	}

	// Type match (with wildcard support)
	if r.Type != ResourceAny && target.Type != ResourceAny && r.Type != target.Type {
		return false
	}

	// HID match (with wildcard support)
	if r.HID != "" && r.HID != "*" && target.HID != "" && r.HID != target.HID {
		return false
	}

	// Namespace match
	if r.Namespace != "" && target.Namespace != "" && r.Namespace != target.Namespace {
		return false
	}

	// ID match (with wildcard support)
	if r.ID != "" && r.ID != "*" && target.ID != "" && r.ID != target.ID {
		return false
	}

	return true
}

// Clone returns a copy of the resource ID
func (r *ResourceID) Clone() *ResourceID {
	if r == nil {
		return nil
	}
	return &ResourceID{
		Type:      r.Type,
		HID:       r.HID,
		ID:        r.ID,
		Namespace: r.Namespace,
	}
}

// Resource represents a securable resource in the system
type Resource struct {
	// ID is the unique identifier for this resource
	ID *ResourceID `json:"id"`

	// OwnerHID is the H-id that owns this resource
	OwnerHID string `json:"owner_hid"`

	// OwnerSID is the S-id that created this resource
	OwnerSID string `json:"owner_sid,omitempty"`

	// Attributes contains additional resource attributes
	Attributes map[string]string `json:"attributes,omitempty"`

	// Tags for resource categorization
	Tags []string `json:"tags,omitempty"`
}

// NewResource creates a new resource
func NewResource(typ ResourceType, hid, id string) *Resource {
	return &Resource{
		ID:       NewResourceIDWithHID(typ, hid, id),
		OwnerHID: hid,
		Attributes: make(map[string]string),
		Tags: make([]string, 0),
	}
}

// GetAttribute returns an attribute value
func (r *Resource) GetAttribute(key string) (string, bool) {
	if r.Attributes == nil {
		return "", false
	}
	v, ok := r.Attributes[key]
	return v, ok
}

// SetAttribute sets an attribute value
func (r *Resource) SetAttribute(key, value string) {
	if r.Attributes == nil {
		r.Attributes = make(map[string]string)
	}
	r.Attributes[key] = value
}

// HasTag checks if the resource has a specific tag
func (r *Resource) HasTag(tag string) bool {
	for _, t := range r.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// AddTag adds a tag to the resource
func (r *Resource) AddTag(tag string) {
	for _, t := range r.Tags {
		if t == tag {
			return
		}
	}
	r.Tags = append(r.Tags, tag)
}

// RemoveTag removes a tag from the resource
func (r *Resource) RemoveTag(tag string) {
	for i, t := range r.Tags {
		if t == tag {
			r.Tags = append(r.Tags[:i], r.Tags[i+1:]...)
			return
		}
	}
}

// Matches checks if this resource matches the given filter
type ResourceFilter struct {
	Type      ResourceType
	HID       string
	Namespace string
	ID        string
	Tag       string
}

// Matches checks if a resource matches the filter
func (f *ResourceFilter) Matches(r *Resource) bool {
	if f.Type != "" && f.Type != ResourceAny && r.ID.Type != f.Type {
		return false
	}
	if f.HID != "" && r.OwnerHID != f.HID {
		return false
	}
	if f.Namespace != "" && r.ID.Namespace != f.Namespace {
		return false
	}
	if f.ID != "" && r.ID.ID != f.ID {
		return false
	}
	if f.Tag != "" && !r.HasTag(f.Tag) {
		return false
	}
	return true
}
