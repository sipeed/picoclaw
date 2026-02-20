// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package memory

import (
	"time"

	"github.com/google/uuid"
)

// MemoryItem represents a single memory entry in the system
type MemoryItem struct {
	// ID is a unique identifier for this memory item
	ID string `json:"id"`

	// OwnerHID is the H-id (tenant) that owns this memory
	OwnerHID string `json:"owner_hid"`

	// OwnerSID is the S-id (instance) that created this memory
	OwnerSID string `json:"owner_sid"`

	// Scope defines the visibility scope of this memory
	Scope MemoryScope `json:"scope"`

	// Type defines the type of memory content
	Type MemoryType `json:"type"`

	// Key is an optional user-defined key for this memory
	Key string `json:"key,omitempty"`

	// Content is the actual memory content
	Content string `json:"content"`

	// Embedding is an optional vector embedding for semantic search
	Embedding []float32 `json:"embedding,omitempty"`

	// Metadata contains additional information about this memory
	Metadata map[string]string `json:"metadata,omitempty"`

	// Tags are optional tags for categorization and filtering
	Tags []string `json:"tags,omitempty"`

	// ExpiresAt is an optional expiration time (0 = never expires)
	ExpiresAt int64 `json:"expires_at,omitempty"`

	// CreatedAt is the timestamp when this memory was created
	CreatedAt int64 `json:"created_at"`

	// UpdatedAt is the timestamp when this memory was last updated
	UpdatedAt int64 `json:"updated_at"`

	// AccessedAt is the timestamp when this memory was last accessed
	AccessedAt int64 `json:"accessed_at"`

	// AccessCount tracks how many times this memory has been accessed
	AccessCount int64 `json:"access_count"`

	// Size is the approximate size in bytes
	Size int64 `json:"size"`
}

// NewMemoryItem creates a new memory item with the given parameters
func NewMemoryItem(ownerHID, ownerSID string, scope MemoryScope, memType MemoryType, content string) *MemoryItem {
	now := time.Now().UnixMilli()
	return &MemoryItem{
		ID:        generateMemoryID(),
		OwnerHID:  ownerHID,
		OwnerSID:  ownerSID,
		Scope:     scope,
		Type:      memType,
		Content:   content,
		Metadata:  make(map[string]string),
		Tags:      make([]string, 0),
		CreatedAt: now,
		UpdatedAt: now,
		AccessedAt: now,
		Size:      int64(len(content)),
	}
}

// NewPrivateMemory creates a new private memory item
func NewPrivateMemory(ownerHID, ownerSID string, memType MemoryType, content string) *MemoryItem {
	return NewMemoryItem(ownerHID, ownerSID, ScopePrivate, memType, content)
}

// NewSharedMemory creates a new shared memory item (same H-id)
func NewSharedMemory(ownerHID, ownerSID string, memType MemoryType, content string) *MemoryItem {
	return NewMemoryItem(ownerHID, ownerSID, ScopeShared, memType, content)
}

// NewPublicMemory creates a new public memory item (any H-id)
func NewPublicMemory(ownerHID, ownerSID string, memType MemoryType, content string) *MemoryItem {
	return NewMemoryItem(ownerHID, ownerSID, ScopePublic, memType, content)
}

// IsExpired returns true if the memory item has expired
func (m *MemoryItem) IsExpired() bool {
	if m.ExpiresAt == 0 {
		return false
	}
	return time.Now().UnixMilli() > m.ExpiresAt
}

// Touch updates the accessed timestamp and increments access count
func (m *MemoryItem) Touch() {
	m.AccessedAt = time.Now().UnixMilli()
	m.AccessCount++
}

// UpdateContent updates the content and recalculates size
func (m *MemoryItem) UpdateContent(content string) {
	m.Content = content
	m.UpdatedAt = time.Now().UnixMilli()
	m.Size = int64(len(content))
}

// SetKey sets the key for this memory item
func (m *MemoryItem) SetKey(key string) {
	m.Key = key
	m.UpdatedAt = time.Now().UnixMilli()
}

// SetMetadata sets a metadata key-value pair
func (m *MemoryItem) SetMetadata(key, value string) {
	if m.Metadata == nil {
		m.Metadata = make(map[string]string)
	}
	m.Metadata[key] = value
	m.UpdatedAt = time.Now().UnixMilli()
}

// GetMetadata gets a metadata value by key
func (m *MemoryItem) GetMetadata(key string) (string, bool) {
	if m.Metadata == nil {
		return "", false
	}
	v, ok := m.Metadata[key]
	return v, ok
}

// AddTag adds a tag to the memory item
func (m *MemoryItem) AddTag(tag string) {
	for _, t := range m.Tags {
		if t == tag {
			return // Already exists
		}
	}
	m.Tags = append(m.Tags, tag)
	m.UpdatedAt = time.Now().UnixMilli()
}

// RemoveTag removes a tag from the memory item
func (m *MemoryItem) RemoveTag(tag string) {
	for i, t := range m.Tags {
		if t == tag {
			m.Tags = append(m.Tags[:i], m.Tags[i+1:]...)
			m.UpdatedAt = time.Now().UnixMilli()
			return
		}
	}
}

// HasTag checks if the memory item has a specific tag
func (m *MemoryItem) HasTag(tag string) bool {
	for _, t := range m.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// SetExpiration sets the expiration time for this memory item
func (m *MemoryItem) SetExpiration(duration time.Duration) {
	m.ExpiresAt = time.Now().Add(duration).UnixMilli()
}

// SetExpirationAt sets a specific expiration timestamp
func (m *MemoryItem) SetExpirationAt(timestamp int64) {
	m.ExpiresAt = timestamp
}

// ClearExpiration clears the expiration time (makes it never expire)
func (m *MemoryItem) ClearExpiration() {
	m.ExpiresAt = 0
}

// GetFullID returns the full ID with H-id and S-id prefix
func (m *MemoryItem) GetFullID() string {
	return m.OwnerHID + "/" + m.OwnerSID + "/" + m.ID
}

// Clone creates a deep copy of the memory item
func (m *MemoryItem) Clone() *MemoryItem {
	clone := &MemoryItem{
		ID:         m.ID,
		OwnerHID:   m.OwnerHID,
		OwnerSID:   m.OwnerSID,
		Scope:      m.Scope,
		Type:       m.Type,
		Key:        m.Key,
		Content:    m.Content,
		ExpiresAt:  m.ExpiresAt,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
		AccessedAt: m.AccessedAt,
		AccessCount: m.AccessCount,
		Size:       m.Size,
		Metadata:   make(map[string]string, len(m.Metadata)),
		Tags:       make([]string, len(m.Tags)),
	}

	// Copy metadata
	for k, v := range m.Metadata {
		clone.Metadata[k] = v
	}

	// Copy tags
	copy(clone.Tags, m.Tags)

	// Copy embedding if present
	if m.Embedding != nil {
		clone.Embedding = make([]float32, len(m.Embedding))
		copy(clone.Embedding, m.Embedding)
	}

	return clone
}

// IsValid returns true if the memory item is valid
func (m *MemoryItem) IsValid() bool {
	return m.ID != "" && m.OwnerHID != "" && m.OwnerSID != "" && m.Content != ""
}

// MemoryFilter is used to filter memory items when querying
type MemoryFilter struct {
	// OwnerHID filters by owner H-id (empty = all)
	OwnerHID string
	// OwnerSID filters by owner S-id (empty = all)
	OwnerSID string
	// Scope filters by scope (zero = wildcard unless ScopeSet is true)
	Scope      MemoryScope
	ScopeSet   bool // true if Scope was explicitly set
	// Type filters by type (zero = wildcard unless TypeSet is true)
	Type    MemoryType
	TypeSet bool // true if Type was explicitly set
	// Tags filters by tags (any match)
	Tags []string
	// Key filters by key (empty = all)
	Key string
	// MinCreatedAt filters by minimum creation time
	MinCreatedAt int64
	// MaxCreatedAt filters by maximum creation time
	MaxCreatedAt int64
	// IncludeExpired includes expired items if true
	IncludeExpired bool
	// Limit limits the number of results
	Limit int
	// Offset skips the first N results
	Offset int
}

// Matches checks if a memory item matches the filter
func (f *MemoryFilter) Matches(item *MemoryItem) bool {
	if f.OwnerHID != "" && item.OwnerHID != f.OwnerHID {
		return false
	}
	if f.OwnerSID != "" && item.OwnerSID != f.OwnerSID {
		return false
	}
	// Scope check: if ScopeSet is true, do exact match; otherwise wildcard
	if f.ScopeSet && item.Scope != f.Scope {
		return false
	}
	// Type check: if TypeSet is true, do exact match; otherwise wildcard
	if f.TypeSet && item.Type != f.Type {
		return false
	}
	if f.Key != "" && item.Key != f.Key {
		return false
	}
	if !f.IncludeExpired && item.IsExpired() {
		return false
	}
	if f.MinCreatedAt > 0 && item.CreatedAt < f.MinCreatedAt {
		return false
	}
	if f.MaxCreatedAt > 0 && item.CreatedAt > f.MaxCreatedAt {
		return false
	}
	if len(f.Tags) > 0 {
		hasAnyTag := false
		for _, tag := range f.Tags {
			if item.HasTag(tag) {
				hasAnyTag = true
				break
			}
		}
		if !hasAnyTag {
			return false
		}
	}
	return true
}

// generateMemoryID generates a unique memory ID
func generateMemoryID() string {
	return "mem-" + uuid.New().String()[:8]
}

// MemoryQuery is used for complex queries with sorting and pagination
type MemoryQuery struct {
	*MemoryFilter
	SortBy     string // "created_at", "updated_at", "accessed_at", "size"
	SortOrder  string // "asc", "desc"
	Limit      int
	Offset     int
}

// NewMemoryQuery creates a new memory query
func NewMemoryQuery() *MemoryQuery {
	return &MemoryQuery{
		MemoryFilter: &MemoryFilter{},
		SortBy:       "created_at",
		SortOrder:    "desc",
		Limit:        100,
		Offset:       0,
	}
}
