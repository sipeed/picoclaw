// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package memory

import (
	"strings"
	"sync"
)

// ACLType represents the type of ACL entry
type ACLType int

const (
	// ACLAllow is an allow-list entry
	ACLAllow ACLType = iota

	// ACLDeny is a deny-list entry
	ACLDeny
)

// String returns the string representation of the ACL type
func (t ACLType) String() string {
	switch t {
	case ACLAllow:
		return "allow"
	case ACLDeny:
		return "deny"
	default:
		return "unknown"
	}
}

// ACLEntry represents a single ACL rule
type ACLEntry struct {
	// Type is either allow or deny
	Type ACLType `json:"type"`

	// HID is the H-id this entry applies to (empty = all)
	HID string `json:"hid,omitempty"`

	// SID is the S-id this entry applies to (empty = all in H-id)
	SID string `json:"sid,omitempty"`

	// Permission specifies which permission this applies to (empty = all)
	Permission Permission `json:"permission,omitempty"`

	// Reason is an optional reason for this entry
	Reason string `json:"reason,omitempty"`
}

// Matches checks if an ACL entry matches the given requester and permission
func (e *ACLEntry) Matches(hid, sid string, perm Permission) bool {
	// Check permission match
	// PermWildcard matches any permission, otherwise exact match is required
	if e.Permission != PermWildcard && e.Permission != perm {
		return false
	}

	// Check HID match
	if e.HID != "" && e.HID != hid {
		return false
	}

	// Check SID match (only if HID also matches or is empty)
	if e.SID != "" && e.SID != sid {
		// If SID is specified, HID must also match
		if e.HID == "" || e.HID == hid {
			return false
		}
	}

	return true
}

// String returns a string representation of the ACL entry
func (e *ACLEntry) String() string {
	var parts []string

	parts = append(parts, e.Type.String())

	if e.HID == "" {
		parts = append(parts, "*")
	} else {
		if e.SID != "" {
			parts = append(parts, e.HID+"/"+e.SID)
		} else {
			parts = append(parts, e.HID)
			parts = append(parts, "*")
		}
	}

	if e.Permission == PermWildcard {
		parts = append(parts, "*")
	} else {
		parts = append(parts, e.Permission.String())
	}

	return strings.Join(parts, " ")
}

// ACL manages access control lists for memory items
type ACL struct {
	entries []ACLEntry
	mu      sync.RWMutex
}

// NewACL creates a new empty ACL
func NewACL() *ACL {
	return &ACL{
		entries: make([]ACLEntry, 0),
	}
}

// Add adds a new ACL entry
func (a *ACL) Add(entry ACLEntry) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.entries = append(a.entries, entry)
}

// Allow adds an allow entry for the given H-id/S-id and permission
func (a *ACL) Allow(hid, sid string, perm Permission) *ACL {
	a.Add(ACLEntry{
		Type:      ACLAllow,
		HID:       hid,
		SID:       sid,
		Permission: perm,
	})
	return a
}

// Deny adds a deny entry for the given H-id/S-id and permission
func (a *ACL) Deny(hid, sid string, perm Permission) *ACL {
	a.Add(ACLEntry{
		Type:      ACLDeny,
		HID:       hid,
		SID:       sid,
		Permission: perm,
	})
	return a
}

// AllowAll adds an allow-all entry
func (a *ACL) AllowAll() *ACL {
	a.Add(ACLEntry{
		Type:      ACLAllow,
		Permission: PermWildcard,
	})
	return a
}

// DenyAll adds a deny-all entry
func (a *ACL) DenyAll() *ACL {
	a.Add(ACLEntry{
		Type:      ACLDeny,
		Permission: PermWildcard,
	})
	return a
}

// Remove removes ACL entries that match the given criteria
func (a *ACL) Remove(hid, sid string, perm Permission) {
	a.mu.Lock()
	defer a.mu.Unlock()

	newEntries := make([]ACLEntry, 0, len(a.entries))
	for _, e := range a.entries {
		if !e.Matches(hid, sid, perm) {
			newEntries = append(newEntries, e)
		}
	}
	a.entries = newEntries
}

// Clear removes all ACL entries
func (a *ACL) Clear() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.entries = make([]ACLEntry, 0)
}

// Check checks if the given requester and permission is allowed
// Returns: (allowed, explicitMatch)
func (a *ACL) Check(hid, sid string, perm Permission) (bool, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	explicitMatch := false

	for _, e := range a.entries {
		if !e.Matches(hid, sid, perm) {
			continue
		}

		explicitMatch = true

		switch e.Type {
		case ACLAllow:
			return true, true
		case ACLDeny:
			return false, true
		}
	}

	// No explicit match - default deny
	return false, explicitMatch
}

// IsAllowed is a convenience method that returns true if allowed
func (a *ACL) IsAllowed(hid, sid string, perm Permission) bool {
	allowed, _ := a.Check(hid, sid, perm)
	return allowed
}

// Entries returns a copy of all ACL entries
func (a *ACL) Entries() []ACLEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make([]ACLEntry, len(a.entries))
	copy(result, a.entries)
	return result
}

// Len returns the number of ACL entries
func (a *ACL) Len() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.entries)
}

// Merge merges another ACL into this one
func (a *ACL) Merge(other *ACL) {
	if other == nil {
		return
	}

	entries := other.Entries()
	a.mu.Lock()
	defer a.mu.Unlock()
	a.entries = append(a.entries, entries...)
}

// Clone creates a deep copy of the ACL
func (a *ACL) Clone() *ACL {
	a.mu.RLock()
	defer a.mu.RUnlock()

	clone := &ACL{
		entries: make([]ACLEntry, len(a.entries)),
	}
	copy(clone.entries, a.entries)
	return clone
}

// ACLChecker combines permission checking with ACL support
type ACLChecker struct {
	*Checker
	acl *ACL
}

// NewACLChecker creates a new ACL-enabled permission checker
func NewACLChecker() *ACLChecker {
	return &ACLChecker{
		Checker: NewChecker(),
		acl:     NewACL(),
	}
}

// NewACLCheckerWithACL creates a new ACL checker with the given ACL
func NewACLCheckerWithACL(acl *ACL) *ACLChecker {
	return &ACLChecker{
		Checker: NewChecker(),
		acl:     acl,
	}
}

// SetACL sets the ACL for this checker
func (c *ACLChecker) SetACL(acl *ACL) {
	c.acl = acl
}

// GetACL returns the ACL for this checker
func (c *ACLChecker) GetACL() *ACL {
	return c.acl
}

// Check checks permission using both scope-based and ACL-based rules
func (c *ACLChecker) Check(req *AccessRequest) *AccessResult {
	if req == nil || req.Item == nil {
		return &AccessResult{
			Allowed: false,
			Reason:  "invalid request",
		}
	}

	// First check ACL if present
	if c.acl != nil && c.acl.Len() > 0 {
		allowed, explicitMatch := c.acl.Check(req.RequesterHID, req.RequesterSID, req.Permission)

		if explicitMatch {
			if allowed {
				return &AccessResult{
					Allowed: true,
					Reason:  "allowed by ACL",
				}
			}
			return &AccessResult{
				Allowed: false,
				Reason:  "denied by ACL",
			}
		}
	}

	// Fall back to scope-based checking
	return c.Checker.Check(req)
}

// AddACLEntry adds an ACL entry to this checker
func (c *ACLChecker) AddACLEntry(entry ACLEntry) {
	if c.acl == nil {
		c.acl = NewACL()
	}
	c.acl.Add(entry)
}

// Allow adds an allow ACL entry
func (c *ACLChecker) Allow(hid, sid string, perm Permission) {
	if c.acl == nil {
		c.acl = NewACL()
	}
	c.acl.Allow(hid, sid, perm)
}

// Deny adds a deny ACL entry
func (c *ACLChecker) Deny(hid, sid string, perm Permission) {
	if c.acl == nil {
		c.acl = NewACL()
	}
	c.acl.Deny(hid, sid, perm)
}

// MemoryItemWithACL extends MemoryItem with ACL support
type MemoryItemWithACL struct {
	*MemoryItem
	acl *ACL
}

// NewMemoryItemWithACL creates a new memory item with ACL
func NewMemoryItemWithACL(item *MemoryItem) *MemoryItemWithACL {
	return &MemoryItemWithACL{
		MemoryItem: item,
		acl:        NewACL(),
	}
}

// GetACL returns the ACL for this memory item
func (m *MemoryItemWithACL) GetACL() *ACL {
	return m.acl
}

// SetACL sets the ACL for this memory item
func (m *MemoryItemWithACL) SetACL(acl *ACL) {
	m.acl = acl
}

// Allow adds an allow entry to this memory item's ACL
func (m *MemoryItemWithACL) Allow(hid, sid string, perm Permission) {
	if m.acl == nil {
		m.acl = NewACL()
	}
	m.acl.Allow(hid, sid, perm)
}

// Deny adds a deny entry to this memory item's ACL
func (m *MemoryItemWithACL) Deny(hid, sid string, perm Permission) {
	if m.acl == nil {
		m.acl = NewACL()
	}
	m.acl.Deny(hid, sid, perm)
}

// CheckAccess checks access using both scope and ACL
func (m *MemoryItemWithACL) CheckAccess(checker *ACLChecker, requesterHID, requesterSID string, perm Permission) *AccessResult {
	req := &AccessRequest{
		RequesterHID: requesterHID,
		RequesterSID: requesterSID,
		Permission:   perm,
		Item:         m.MemoryItem,
	}

	// Use the item's ACL if checker doesn't have one or has an empty one
	if m.acl != nil && m.acl.Len() > 0 {
		if checker.acl == nil || checker.acl.Len() == 0 {
			checker.SetACL(m.acl)
		}
	}

	return checker.Check(req)
}
