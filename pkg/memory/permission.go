// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package memory

import (
	"fmt"
)

// Permission represents the type of permission being checked
type Permission int

const (
	// PermWildcard matches any permission (used for ACL entries)
	PermWildcard Permission = -1

	// PermRead allows reading memory content
	PermRead Permission = iota

	// PermWrite allows writing/updating memory content
	PermWrite

	// PermDelete allows deleting memory
	PermDelete

	// PermShare allows changing memory scope
	PermShare
)

// String returns the string representation of the permission
func (p Permission) String() string {
	switch p {
	case PermWildcard:
		return "*"
	case PermRead:
		return "read"
	case PermWrite:
		return "write"
	case PermDelete:
		return "delete"
	case PermShare:
		return "share"
	default:
		return "unknown"
	}
}

// AccessRequest represents a request to access a memory item
type AccessRequest struct {
	// RequesterHID is the H-id of the requester
	RequesterHID string

	// RequesterSID is the S-id of the requester
	RequesterSID string

	// Permission is the type of permission being requested
	Permission Permission

	// Item is the memory item being accessed
	Item *MemoryItem
}

// AccessResult represents the result of a permission check
type AccessResult struct {
	// Allowed is true if access is granted
	Allowed bool

	// Reason is a human-readable explanation of the decision
	Reason string
}

// Checker handles permission checking for memory access
type Checker struct {
	// DefaultDeny causes all checks to deny by default unless explicitly allowed
	DefaultDeny bool
}

// NewChecker creates a new permission checker
func NewChecker() *Checker {
	return &Checker{
		DefaultDeny: false,
	}
}

// Check checks if the given access request should be allowed
func (c *Checker) Check(req *AccessRequest) *AccessResult {
	if req == nil || req.Item == nil {
		return &AccessResult{
			Allowed: false,
			Reason:  "invalid request",
		}
	}

	item := req.Item

	// Check expiration first
	if item.IsExpired() {
		return &AccessResult{
			Allowed: false,
			Reason:  "memory item has expired",
		}
	}

	// Get the effective scope for the requester
	effectiveScope := c.getEffectiveScope(item, req.RequesterHID, req.RequesterSID)

	// Check permission based on scope
	switch effectiveScope {
	case ScopePrivate:
		return c.checkPrivateAccess(req)
	case ScopeShared:
		return c.checkSharedAccess(req)
	case ScopePublic:
		return c.checkPublicAccess(req)
	default:
		return &AccessResult{
			Allowed: false,
			Reason:  "invalid scope",
		}
	}
}

// getEffectiveScope determines the effective scope for a requester
func (c *Checker) getEffectiveScope(item *MemoryItem, requesterHID, requesterSID string) MemoryScope {
	// Owner always has the item's actual scope
	if item.OwnerHID == requesterHID && item.OwnerSID == requesterSID {
		return item.Scope
	}

	// Same H-id but different S-id: treat as shared access
	if item.OwnerHID == requesterHID {
		if item.Scope == ScopePrivate {
			return ScopePrivate // No access
		}
		return ScopeShared
	}

	// Different H-id
	return item.Scope
}

// checkPrivateAccess checks access for private-scoped memory
func (c *Checker) checkPrivateAccess(req *AccessRequest) *AccessResult {
	item := req.Item

	// Only the owner S-id can access private memory
	if item.OwnerHID == req.RequesterHID && item.OwnerSID == req.RequesterSID {
		return c.checkOwnerPermissions(req)
	}

	return &AccessResult{
		Allowed: false,
		Reason:  fmt.Sprintf("private memory owned by %s/%s", item.OwnerHID, item.OwnerSID),
	}
}

// checkSharedAccess checks access for shared-scoped memory
func (c *Checker) checkSharedAccess(req *AccessRequest) *AccessResult {
	item := req.Item

	// Same H-id: full access
	if item.OwnerHID == req.RequesterHID {
		if item.OwnerSID == req.RequesterSID {
			return c.checkOwnerPermissions(req)
		}
		return c.checkTenantPermissions(req)
	}

	// Different H-id: no access for shared memory
	return &AccessResult{
		Allowed: false,
		Reason:  fmt.Sprintf("shared memory restricted to H-id %s", item.OwnerHID),
	}
}

// checkPublicAccess checks access for public-scoped memory
func (c *Checker) checkPublicAccess(req *AccessRequest) *AccessResult {
	item := req.Item

	// All H-ids can read public memory
	if req.Permission == PermRead {
		return &AccessResult{
			Allowed: true,
			Reason:  "public memory is readable",
		}
	}

	// Write/Delete/Share requires ownership
	if item.OwnerHID == req.RequesterHID && item.OwnerSID == req.RequesterSID {
		return c.checkOwnerPermissions(req)
	}

	return &AccessResult{
		Allowed: false,
		Reason:  "only owner can modify public memory",
	}
}

// checkOwnerPermissions checks permissions for the owner
func (c *Checker) checkOwnerPermissions(req *AccessRequest) *AccessResult {
	// Owners have full permissions
	return &AccessResult{
		Allowed: true,
		Reason:  "owner has full permissions",
	}
}

// checkTenantPermissions checks permissions for same-tenant non-owners
func (c *Checker) checkTenantPermissions(req *AccessRequest) *AccessResult {
	// Same-tenant S-ids can read shared memory
	if req.Permission == PermRead {
		return &AccessResult{
			Allowed: true,
			Reason:  "shared memory is readable within tenant",
		}
	}

	return &AccessResult{
		Allowed: false,
		Reason:  "only owner can modify shared memory",
	}
}

// CanRead is a convenience method to check read permission
func (c *Checker) CanRead(item *MemoryItem, requesterHID, requesterSID string) bool {
	req := &AccessRequest{
		RequesterHID:  requesterHID,
		RequesterSID:  requesterSID,
		Permission:    PermRead,
		Item:          item,
	}
	return c.Check(req).Allowed
}

// CanWrite is a convenience method to check write permission
func (c *Checker) CanWrite(item *MemoryItem, requesterHID, requesterSID string) bool {
	req := &AccessRequest{
		RequesterHID:  requesterHID,
		RequesterSID:  requesterSID,
		Permission:    PermWrite,
		Item:          item,
	}
	return c.Check(req).Allowed
}

// CanDelete is a convenience method to check delete permission
func (c *Checker) CanDelete(item *MemoryItem, requesterHID, requesterSID string) bool {
	req := &AccessRequest{
		RequesterHID:  requesterHID,
		RequesterSID:  requesterSID,
		Permission:    PermDelete,
		Item:          item,
	}
	return c.Check(req).Allowed
}

// CanShare is a convenience method to check share permission
func (c *Checker) CanShare(item *MemoryItem, requesterHID, requesterSID string) bool {
	req := &AccessRequest{
		RequesterHID:  requesterHID,
		RequesterSID:  requesterSID,
		Permission:    PermShare,
		Item:          item,
	}
	return c.Check(req).Allowed
}

// FilterByPermission filters a list of memory items to those accessible by the requester
func (c *Checker) FilterByPermission(items []*MemoryItem, requesterHID, requesterSID string, perm Permission) []*MemoryItem {
	result := make([]*MemoryItem, 0, len(items))
	req := &AccessRequest{
		RequesterHID: requesterHID,
		RequesterSID: requesterSID,
		Permission:   perm,
	}

	for _, item := range items {
		req.Item = item
		if c.Check(req).Allowed {
			result = append(result, item)
		}
	}

	return result
}

// ScopeHierarchy defines the hierarchy of scopes for permission inheritance
var ScopeHierarchy = []MemoryScope{
	ScopePrivate,
	ScopeShared,
	ScopePublic,
}

// ScopeIsAtLeast returns true if the given scope is at least as permissive as the minimum
func ScopeIsAtLeast(scope, minimum MemoryScope) bool {
	return scope.ScopeLevel() >= minimum.ScopeLevel()
}

// ScopeIsAtMost returns true if the given scope is at most as permissive as the maximum
func ScopeIsAtMost(scope, maximum MemoryScope) bool {
	return scope.ScopeLevel() <= maximum.ScopeLevel()
}

// ScopeIsBetween returns true if the given scope is within the inclusive range
func ScopeIsBetween(scope, min, max MemoryScope) bool {
	level := scope.ScopeLevel()
	return level >= min.ScopeLevel() && level <= max.ScopeLevel()
}

// DefaultScopeForType returns the recommended default scope for a given memory type
func DefaultScopeForType(memType MemoryType) MemoryScope {
	switch memType {
	case TypeConversation:
		return ScopeShared
	case TypeKnowledge:
		return ScopeShared
	case TypeContext:
		return ScopePrivate
	case TypeToolResult:
		return ScopePrivate
	case TypeUserPreference:
		return ScopeShared
	case TypeSystem:
		return ScopeShared
	default:
		return ScopePrivate
	}
}

// NewMemoryWithDefaultScope creates a memory item with the default scope for its type
func NewMemoryWithDefaultScope(ownerHID, ownerSID string, memType MemoryType, content string) *MemoryItem {
	scope := DefaultScopeForType(memType)
	return NewMemoryItem(ownerHID, ownerSID, scope, memType, content)
}
