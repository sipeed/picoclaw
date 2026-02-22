// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package memory

import (
	"testing"
	"time"
)

func TestMemoryScope_String(t *testing.T) {
	tests := []struct {
		scope MemoryScope
		want  string
	}{
		{ScopePrivate, "private"},
		{ScopeShared, "shared"},
		{ScopePublic, "public"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.scope.String(); got != tt.want {
				t.Errorf("MemoryScope.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseMemoryScope(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		want    MemoryScope
		wantErr bool
	}{
		{"private", "private", ScopePrivate, false},
		{"shared", "shared", ScopeShared, false},
		{"public", "public", ScopePublic, false},
		{"invalid", "invalid", ScopePrivate, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMemoryScope(tt.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMemoryScope() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseMemoryScope() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMemoryType_String(t *testing.T) {
	tests := []struct {
		memType MemoryType
		want    string
	}{
		{TypeConversation, "conversation"},
		{TypeKnowledge, "knowledge"},
		{TypeContext, "context"},
		{TypeToolResult, "tool_result"},
		{TypeUserPreference, "user_preference"},
		{TypeSystem, "system"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.memType.String(); got != tt.want {
				t.Errorf("MemoryType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMemoryScope_Comparison(t *testing.T) {
	tests := []struct {
		name       string
		scope      MemoryScope
		other      MemoryScope
		more       bool
		less       bool
		level      int
	}{
		{"private vs shared", ScopePrivate, ScopeShared, false, true, 0},
		{"shared vs public", ScopeShared, ScopePublic, false, true, 1},
		{"public vs private", ScopePublic, ScopePrivate, true, false, 2},
		{"same", ScopeShared, ScopeShared, false, false, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.scope.IsMorePermissive(tt.other); got != tt.more {
				t.Errorf("IsMorePermissive() = %v, want %v", got, tt.more)
			}
			if got := tt.scope.IsLessPermissive(tt.other); got != tt.less {
				t.Errorf("IsLessPermissive() = %v, want %v", got, tt.less)
			}
			if got := tt.scope.ScopeLevel(); got != tt.level {
				t.Errorf("ScopeLevel() = %v, want %v", got, tt.level)
			}
		})
	}
}

func TestNewMemoryItem(t *testing.T) {
	hid := "user-alice"
	sid := "node-01"
	content := "test content"

	item := NewMemoryItem(hid, sid, ScopePrivate, TypeContext, content)

	if item.OwnerHID != hid {
		t.Errorf("OwnerHID = %v, want %v", item.OwnerHID, hid)
	}
	if item.OwnerSID != sid {
		t.Errorf("OwnerSID = %v, want %v", item.OwnerSID, sid)
	}
	if item.Scope != ScopePrivate {
		t.Errorf("Scope = %v, want %v", item.Scope, ScopePrivate)
	}
	if item.Type != TypeContext {
		t.Errorf("Type = %v, want %v", item.Type, TypeContext)
	}
	if item.Content != content {
		t.Errorf("Content = %v, want %v", item.Content, content)
	}
	if item.ID == "" {
		t.Errorf("ID should not be empty")
	}
	if item.CreatedAt == 0 {
		t.Errorf("CreatedAt should be set")
	}
	if !item.IsValid() {
		t.Errorf("Item should be valid")
	}
}

func TestMemoryItem_Expiration(t *testing.T) {
	item := NewMemoryItem("user-alice", "node-01", ScopePrivate, TypeContext, "test")

	// No expiration by default
	if item.IsExpired() {
		t.Errorf("New item should not be expired")
	}

	// Set expiration
	item.SetExpiration(1 * time.Hour)
	if item.IsExpired() {
		t.Errorf("Item with future expiration should not be expired")
	}

	// Set past expiration
	item.SetExpirationAt(time.Now().UnixMilli() - 1000)
	if !item.IsExpired() {
		t.Errorf("Item with past expiration should be expired")
	}

	// Clear expiration
	item.ClearExpiration()
	if item.IsExpired() {
		t.Errorf("Item with cleared expiration should not be expired")
	}
}

func TestMemoryItem_Metadata(t *testing.T) {
	item := NewMemoryItem("user-alice", "node-01", ScopePrivate, TypeContext, "test")

	item.SetMetadata("key1", "value1")
	item.SetMetadata("key2", "value2")

	v, ok := item.GetMetadata("key1")
	if !ok || v != "value1" {
		t.Errorf("Metadata not set correctly")
	}

	_, ok = item.GetMetadata("missing")
	if ok {
		t.Errorf("Expected false for missing key")
	}
}

func TestMemoryItem_Tags(t *testing.T) {
	item := NewMemoryItem("user-alice", "node-01", ScopePrivate, TypeContext, "test")

	item.AddTag("tag1")
	item.AddTag("tag2")
	item.AddTag("tag1") // Duplicate

	if !item.HasTag("tag1") {
		t.Errorf("Expected tag1")
	}
	if !item.HasTag("tag2") {
		t.Errorf("Expected tag2")
	}
	if len(item.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(item.Tags))
	}

	item.RemoveTag("tag1")
	if item.HasTag("tag1") {
		t.Errorf("tag1 should be removed")
	}
	if len(item.Tags) != 1 {
		t.Errorf("Expected 1 tag, got %d", len(item.Tags))
	}
}

func TestMemoryItem_Touch(t *testing.T) {
	item := NewMemoryItem("user-alice", "node-01", ScopePrivate, TypeContext, "test")

	firstAccess := item.AccessedAt
	firstCount := item.AccessCount

	time.Sleep(10 * time.Millisecond)
	item.Touch()

	if item.AccessedAt <= firstAccess {
		t.Errorf("AccessedAt should be updated")
	}
	if item.AccessCount != firstCount+1 {
		t.Errorf("AccessCount should be incremented")
	}
}

func TestMemoryItem_Clone(t *testing.T) {
	original := NewMemoryItem("user-alice", "node-01", ScopePrivate, TypeContext, "test")
	original.SetMetadata("key1", "value1")
	original.AddTag("tag1")
	original.SetKey("test-key")

	clone := original.Clone()

	// Verify all fields match
	if clone.ID != original.ID {
		t.Errorf("Clone.ID mismatch")
	}
	if clone.OwnerHID != original.OwnerHID {
		t.Errorf("Clone.OwnerHID mismatch")
	}
	if clone.Content != original.Content {
		t.Errorf("Clone.Content mismatch")
	}
	if !clone.HasTag("tag1") {
		t.Errorf("Clone should have tag1")
	}
	if clone.Key != original.Key {
		t.Errorf("Clone.Key mismatch")
	}

	// Modify clone and ensure original is unchanged
	clone.SetMetadata("key2", "value2")
	_, ok := original.GetMetadata("key2")
	if ok {
		t.Errorf("Modifying clone affected original")
	}
}

func TestMemoryFilter_Matches(t *testing.T) {
	item := NewMemoryItem("user-alice", "node-01", ScopeShared, TypeContext, "test")
	item.SetKey("test-key")
	item.AddTag("tag1")

	tests := []struct {
		name   string
		filter *MemoryFilter
		want   bool
	}{
		{
			name:   "no filter",
			filter: &MemoryFilter{},
			want:   true,
		},
		{
			name:   "matching HID",
			filter: &MemoryFilter{OwnerHID: "user-alice"},
			want:   true,
		},
		{
			name:   "non-matching HID",
			filter: &MemoryFilter{OwnerHID: "user-bob"},
			want:   false,
		},
		{
			name:   "matching SID",
			filter: &MemoryFilter{OwnerSID: "node-01"},
			want:   true,
		},
		{
			name:   "matching Scope",
			filter: &MemoryFilter{Scope: ScopeShared, ScopeSet: true},
			want:   true,
		},
		{
			name:   "non-matching Scope",
			filter: &MemoryFilter{Scope: ScopePrivate, ScopeSet: true},
			want:   false,
		},
		{
			name:   "matching Type",
			filter: &MemoryFilter{Type: TypeContext, TypeSet: true},
			want:   true,
		},
		{
			name:   "matching Key",
			filter: &MemoryFilter{Key: "test-key"},
			want:   true,
		},
		{
			name:   "matching Tag",
			filter: &MemoryFilter{Tags: []string{"tag1"}},
			want:   true,
		},
		{
			name:   "non-matching Tag",
			filter: &MemoryFilter{Tags: []string{"tag2"}},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.filter.Matches(item); got != tt.want {
				t.Errorf("MemoryFilter.Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChecker_PrivateAccess(t *testing.T) {
	checker := NewChecker()
	item := NewPrivateMemory("user-alice", "node-01", TypeContext, "test")

	tests := []struct {
		name         string
		requesterHID string
		requesterSID string
		allowed      bool
	}{
		{"owner", "user-alice", "node-01", true},
		{"same tenant different node", "user-alice", "node-02", false},
		{"different tenant", "user-bob", "node-01", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &AccessRequest{
				RequesterHID: tt.requesterHID,
				RequesterSID: tt.requesterSID,
				Permission:   PermRead,
				Item:         item,
			}
			result := checker.Check(req)
			if result.Allowed != tt.allowed {
				t.Errorf("Checker.Check() = %v, want %v (reason: %s)", result.Allowed, tt.allowed, result.Reason)
			}
		})
	}
}

func TestChecker_SharedAccess(t *testing.T) {
	checker := NewChecker()
	item := NewSharedMemory("user-alice", "node-01", TypeContext, "test")

	tests := []struct {
		name         string
		requesterHID string
		requesterSID string
		perm         Permission
		allowed      bool
	}{
		{"owner read", "user-alice", "node-01", PermRead, true},
		{"owner write", "user-alice", "node-01", PermWrite, true},
		{"same tenant read", "user-alice", "node-02", PermRead, true},
		{"same tenant write", "user-alice", "node-02", PermWrite, false},
		{"different tenant", "user-bob", "node-01", PermRead, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &AccessRequest{
				RequesterHID: tt.requesterHID,
				RequesterSID: tt.requesterSID,
				Permission:   tt.perm,
				Item:         item,
			}
			result := checker.Check(req)
			if result.Allowed != tt.allowed {
				t.Errorf("Checker.Check() = %v, want %v", result.Allowed, tt.allowed)
			}
		})
	}
}

func TestChecker_PublicAccess(t *testing.T) {
	checker := NewChecker()
	item := NewPublicMemory("user-alice", "node-01", TypeKnowledge, "test")

	tests := []struct {
		name         string
		requesterHID string
		requesterSID string
		perm         Permission
		allowed      bool
	}{
		{"anyone read", "user-bob", "node-99", PermRead, true},
		{"owner write", "user-alice", "node-01", PermWrite, true},
		{"other write", "user-bob", "node-99", PermWrite, false},
		{"owner delete", "user-alice", "node-01", PermDelete, true},
		{"other delete", "user-bob", "node-99", PermDelete, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &AccessRequest{
				RequesterHID: tt.requesterHID,
				RequesterSID: tt.requesterSID,
				Permission:   tt.perm,
				Item:         item,
			}
			result := checker.Check(req)
			if result.Allowed != tt.allowed {
				t.Errorf("Checker.Check() = %v, want %v", result.Allowed, tt.allowed)
			}
		})
	}
}

func TestChecker_ConvenienceMethods(t *testing.T) {
	checker := NewChecker()
	item := NewSharedMemory("user-alice", "node-01", TypeContext, "test")

	if !checker.CanRead(item, "user-alice", "node-01") {
		t.Errorf("CanRead should return true for owner")
	}
	if !checker.CanWrite(item, "user-alice", "node-01") {
		t.Errorf("CanWrite should return true for owner")
	}
	if !checker.CanDelete(item, "user-alice", "node-01") {
		t.Errorf("CanDelete should return true for owner")
	}
	if !checker.CanShare(item, "user-alice", "node-01") {
		t.Errorf("CanShare should return true for owner")
	}
}

func TestChecker_FilterByPermission(t *testing.T) {
	checker := NewChecker()

	items := []*MemoryItem{
		NewPrivateMemory("user-alice", "node-01", TypeContext, "private"),
		NewSharedMemory("user-alice", "node-01", TypeContext, "shared"),
		NewPublicMemory("user-alice", "node-01", TypeKnowledge, "public"),
		NewPrivateMemory("user-bob", "node-01", TypeContext, "bob-private"),
	}

	// Filter for user-alice/node-01 (can access all of alice's items)
	filtered := checker.FilterByPermission(items, "user-alice", "node-01", PermRead)
	if len(filtered) != 3 {
		t.Errorf("Expected 3 items for owner, got %d", len(filtered))
	}

	// Filter for user-alice/node-02 (can access shared and public)
	filtered = checker.FilterByPermission(items, "user-alice", "node-02", PermRead)
	if len(filtered) != 2 {
		t.Errorf("Expected 2 items for same tenant, got %d", len(filtered))
	}

	// Filter for user-bob/node-01 (can access own private and alice's public)
	filtered = checker.FilterByPermission(items, "user-bob", "node-01", PermRead)
	if len(filtered) != 2 {
		t.Errorf("Expected 2 items for different tenant, got %d", len(filtered))
	}
}

func TestACL_AddCheck(t *testing.T) {
	acl := NewACL()

	// Add allow rule
	acl.Allow("user-bob", "", PermRead)

	// Check access
	allowed, _ := acl.Check("user-bob", "node-01", PermRead)
	if !allowed {
		t.Errorf("Expected allow for user-bob")
	}

	allowed, _ = acl.Check("user-bob", "node-01", PermWrite)
	if allowed {
		t.Errorf("Expected deny for write permission")
	}

	// Add deny rule
	acl.Deny("user-charlie", "", PermRead)

	allowed, _ = acl.Check("user-charlie", "node-01", PermRead)
	if allowed {
		t.Errorf("Expected deny for user-charlie")
	}
}

func TestACL_EntryMatching(t *testing.T) {
	entry := ACLEntry{
		Type:  ACLAllow,
		HID:   "user-alice",
		SID:   "node-01",
		Permission: PermRead,
	}

	tests := []struct {
		name string
		hid  string
		sid  string
		perm Permission
		want bool
	}{
		{"exact match", "user-alice", "node-01", PermRead, true},
		{"different sid", "user-alice", "node-02", PermRead, false},
		{"different hid", "user-bob", "node-01", PermRead, false},
		{"different perm", "user-alice", "node-01", PermWrite, false},
		{"wildcard permission", "user-alice", "node-01", PermRead, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := entry.Matches(tt.hid, tt.sid, tt.perm); got != tt.want {
				t.Errorf("ACLEntry.Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestACL_AllowDenyAll(t *testing.T) {
	acl := NewACL()

	// Default deny
	allowed, _ := acl.Check("anyone", "anywhere", PermRead)
	if allowed {
		t.Errorf("Expected default deny")
	}

	// Allow all
	acl.AllowAll()
	allowed, _ = acl.Check("anyone", "anywhere", PermRead)
	if !allowed {
		t.Errorf("Expected allow after AllowAll")
	}

	// Deny all
	acl.Clear()
	acl.DenyAll()
	allowed, _ = acl.Check("anyone", "anywhere", PermRead)
	if allowed {
		t.Errorf("Expected deny after DenyAll")
	}
}

func TestACL_Remove(t *testing.T) {
	acl := NewACL()
	acl.Allow("user-alice", "", PermRead)
	acl.Allow("user-bob", "", PermRead)

	if len(acl.Entries()) != 2 {
		t.Errorf("Expected 2 entries")
	}

	// Remove user-bob
	acl.Remove("user-bob", "", PermRead)

	if len(acl.Entries()) != 1 {
		t.Errorf("Expected 1 entry after removal")
	}

	allowed, _ := acl.Check("user-bob", "node-01", PermRead)
	if allowed {
		t.Errorf("Expected deny after removal")
	}
}

func TestACLChecker_Integration(t *testing.T) {
	checker := NewACLChecker()
	item := NewPrivateMemory("user-alice", "node-01", TypeContext, "test")

	// Without ACL, owner has access
	req := &AccessRequest{
		RequesterHID: "user-alice",
		RequesterSID: "node-01",
		Permission:   PermRead,
		Item:         item,
	}
	result := checker.Check(req)
	if !result.Allowed {
		t.Errorf("Owner should have access by default")
	}

	// Add ACL denying user-alice
	checker.Deny("user-alice", "node-01", PermRead)

	result = checker.Check(req)
	if result.Allowed {
		t.Errorf("ACL deny should prevent access")
	}

	// Add ACL allowing user-bob
	checker.Allow("user-bob", "node-01", PermRead)

	req.RequesterHID = "user-bob"
	result = checker.Check(req)
	if !result.Allowed {
		t.Errorf("ACL allow should grant access")
	}
}

func TestACLEntry_String(t *testing.T) {
	tests := []struct {
		entry ACLEntry
		want  string
	}{
		{
			ACLEntry{Type: ACLAllow, Permission: PermWildcard},
			"allow * *",
		},
		{
			ACLEntry{Type: ACLAllow, HID: "user-alice", Permission: PermRead},
			"allow user-alice * read",
		},
		{
			ACLEntry{Type: ACLDeny, HID: "user-alice", SID: "node-01", Permission: PermWrite},
			"deny user-alice/node-01 write",
		},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.entry.String(); got != tt.want {
				t.Errorf("ACLEntry.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultScopeForType(t *testing.T) {
	tests := []struct {
		memType MemoryType
		want    MemoryScope
	}{
		{TypeConversation, ScopeShared},
		{TypeKnowledge, ScopeShared},
		{TypeContext, ScopePrivate},
		{TypeToolResult, ScopePrivate},
		{TypeUserPreference, ScopeShared},
		{TypeSystem, ScopeShared},
	}

	for _, tt := range tests {
		t.Run(tt.memType.String(), func(t *testing.T) {
			if got := DefaultScopeForType(tt.memType); got != tt.want {
				t.Errorf("DefaultScopeForType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScopeIsAtLeast(t *testing.T) {
	if !ScopeIsAtLeast(ScopePublic, ScopePrivate) {
		t.Errorf("Public should be at least Private")
	}
	if !ScopeIsAtLeast(ScopeShared, ScopeShared) {
		t.Errorf("Shared should be at least Shared")
	}
	if ScopeIsAtLeast(ScopePrivate, ScopeShared) {
		t.Errorf("Private should not be at least Shared")
	}
}

func TestNewMemoryQuery(t *testing.T) {
	query := NewMemoryQuery()

	if query.SortBy != "created_at" {
		t.Errorf("Default SortBy should be created_at")
	}
	if query.SortOrder != "desc" {
		t.Errorf("Default SortOrder should be desc")
	}
	if query.Limit != 100 {
		t.Errorf("Default Limit should be 100")
	}
}

func TestMemoryItemWithACL(t *testing.T) {
	item := NewMemoryItem("user-alice", "node-01", ScopePrivate, TypeContext, "test")
	itemWithACL := NewMemoryItemWithACL(item)

	// Add ACL to allow user-bob
	itemWithACL.Allow("user-bob", "", PermRead)

	checker := NewACLChecker()

	// Check that user-bob can read
	result := itemWithACL.CheckAccess(checker, "user-bob", "node-99", PermRead)
	if !result.Allowed {
		t.Errorf("user-bob should be allowed by ACL")
	}

	// Check that user-bob cannot write
	result = itemWithACL.CheckAccess(checker, "user-bob", "node-99", PermWrite)
	if result.Allowed {
		t.Errorf("user-bob should not have write permission")
	}
}
