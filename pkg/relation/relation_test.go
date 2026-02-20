// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package relation

import (
	"testing"
)

func TestResourceID_String(t *testing.T) {
	tests := []struct {
		name   string
		id     *ResourceID
		expect string
	}{
		{
			name: "minimal",
			id:   NewResourceID(ResourceMemory, "mem-123"),
			expect: "memory:mem-123",
		},
		{
			name: "with HID",
			id:   NewResourceIDWithHID(ResourceMemory, "user-alice", "mem-123"),
			expect: "memory:user-alice:mem-123",
		},
		{
			name: "with namespace",
			id: &ResourceID{Type: ResourceMemory, HID: "user-alice", ID: "mem-123", Namespace: "default"},
			expect: "memory:user-alice:default:mem-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.String(); got != tt.expect {
				t.Errorf("ResourceID.String() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestParseResourceID(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		wantType string
		wantHID  string
		wantID   string
		wantErr  bool
	}{
		{
			name:     "minimal",
			s:        "memory:mem-123",
			wantType: "memory",
			wantHID:  "",
			wantID:   "mem-123",
			wantErr:  false,
		},
		{
			name:     "with HID",
			s:        "memory:user-alice:mem-123",
			wantType: "memory",
			wantHID:  "user-alice",
			wantID:   "mem-123",
			wantErr:  false,
		},
		{
			name:     "invalid",
			s:        "memory",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseResourceID(tt.s)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseResourceID() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("ParseResourceID() error = %v", err)
				return
			}
			if got.Type != ResourceType(tt.wantType) {
				t.Errorf("Type = %v, want %v", got.Type, tt.wantType)
			}
			if got.HID != tt.wantHID {
				t.Errorf("HID = %v, want %v", got.HID, tt.wantHID)
			}
			if got.ID != tt.wantID {
				t.Errorf("ID = %v, want %v", got.ID, tt.wantID)
			}
		})
	}
}

func TestResourceID_Matches(t *testing.T) {
	wildcard := NewResourceID(ResourceAny, "*")
	specific := NewResourceIDWithHID(ResourceMemory, "user-alice", "mem-123")

	tests := []struct {
		name    string
		a        *ResourceID
		b        *ResourceID
		expected bool
	}{
		{"exact match", specific, specific, true},
		{"wildcard matches specific", wildcard, specific, true},
		{"specific matches wildcard", specific, wildcard, false},
		{"different type", specific, NewResourceID(ResourceNode, "node-01"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.Matches(tt.b); got != tt.expected {
				t.Errorf("ResourceID.Matches() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRelation_String(t *testing.T) {
	rel := NewRelation(
		"user-alice",
		NewResourceIDWithHID(ResourceMemory, "user-alice", "mem-123"),
		RelationReader,
	)

	want := "user-alice reader memory:user-alice:mem-123"
	if got := rel.String(); got != want {
		t.Errorf("Relation.String() = %v, want %v", got, want)
	}
}

func TestRelation_Clone(t *testing.T) {
	original := NewRelation(
		"user-alice",
		NewResourceIDWithHID(ResourceMemory, "user-alice", "mem-123"),
		RelationReader,
	)
	original.SetAttribute("key1", "value1")

	clone := original.Clone()

	if clone.SubjectHID != original.SubjectHID {
		t.Errorf("Clone.SubjectHID mismatch")
	}
	if clone.RelationType != original.RelationType {
		t.Errorf("Clone.RelationType mismatch")
	}

	// Verify attribute was copied
	v, ok := clone.GetAttribute("key1")
	if !ok || v != "value1" {
		t.Errorf("Clone attributes not copied")
	}

	// Modify clone and ensure original is unchanged
	clone.SetAttribute("key2", "value2")
	_, ok = original.GetAttribute("key2")
	if ok {
		t.Errorf("Modifying clone affected original")
	}
}

func TestRelationTypeHierarchy(t *testing.T) {
	tests := []struct {
		has      RelationType
		requires RelationType
		expected  bool
	}{
		{RelationOwner, RelationReader, true},
		{RelationOwner, RelationWriter, true},
		{RelationOwner, RelationAdmin, true},
		{RelationAdmin, RelationReader, true},
		// Note: Admin has higher hierarchy level than Writer, but doesn't have write permission
		// This is checked via ActionAllowedByRelation, not HasPrivilege
		{RelationReader, RelationWriter, false},
		{RelationWriter, RelationAdmin, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.has)+"_has_"+string(tt.requires), func(t *testing.T) {
			if got := HasPrivilege(tt.has, tt.requires); got != tt.expected {
				t.Errorf("HasPrivilege(%s, %s) = %v, want %v", tt.has, tt.requires, got, tt.expected)
			}
		})
	}
}

func TestNewRelationSet(t *testing.T) {
	rs := NewRelationSet()

	if rs.Len() != 0 {
		t.Errorf("NewRelationSet should be empty")
	}

	rel := NewRelation("user-alice", NewResourceID(ResourceMemory, "mem-123"), RelationReader)
	rs.Add(rel)

	if rs.Len() != 1 {
		t.Errorf("Expected 1 relation, got %d", rs.Len())
	}
}

func TestRelationSet_FindBySubject(t *testing.T) {
	rs := NewRelationSet()

	rel1 := NewRelation("user-alice", NewResourceID(ResourceMemory, "mem-1"), RelationReader)
	rel2 := NewRelation("user-alice", NewResourceID(ResourceMemory, "mem-2"), RelationWriter)
	rel3 := NewRelation("user-bob", NewResourceID(ResourceMemory, "mem-3"), RelationReader)

	rs.Add(rel1)
	rs.Add(rel2)
	rs.Add(rel3)

	// Find user-alice's relations
	relations := rs.FindBySubject("user-alice", "")
	if len(relations) != 2 {
		t.Errorf("Expected 2 relations for user-alice, got %d", len(relations))
	}

	// Find specific S-id relations
	relations = rs.FindBySubject("user-alice", "node-01")
	if len(relations) != 0 {
		t.Errorf("Expected 0 relations for user-alice/node-01, got %d", len(relations))
	}
}

func TestRelationSet_Remove(t *testing.T) {
	rs := NewRelationSet()

	rel := NewRelation("user-alice", NewResourceID(ResourceMemory, "mem-123"), RelationReader)
	rs.Add(rel)

	if rs.Len() != 1 {
		t.Errorf("Expected 1 relation, got %d", rs.Len())
	}

	// Remove the relation
	removed := rs.Remove("user-alice", "", rel.Resource, RelationReader)
	if removed != 1 {
		t.Errorf("Expected to remove 1 relation, got %d", removed)
	}

	if rs.Len() != 0 {
		t.Errorf("Expected 0 relations after removal, got %d", rs.Len())
	}
}

func TestRelationSet_Merge(t *testing.T) {
	rs1 := NewRelationSet()
	rs2 := NewRelationSet()

	rel1 := NewRelation("user-alice", NewResourceID(ResourceMemory, "mem-1"), RelationReader)
	rel2 := NewRelation("user-bob", NewResourceID(ResourceMemory, "mem-2"), RelationWriter)

	rs1.Add(rel1)
	rs2.Add(rel2)

	rs1.Merge(rs2)

	if rs1.Len() != 2 {
		t.Errorf("Expected 2 relations after merge, got %d", rs1.Len())
	}
}

func TestRegistry_Add(t *testing.T) {
	registry := NewRegistry()

	rel := NewRelation("user-alice", NewResourceIDWithHID(ResourceMemory, "user-alice", "mem-123"), RelationReader)

	err := registry.Add(rel)
	if err != nil {
		t.Errorf("Add() error = %v", err)
	}

	if registry.Count() != 1 {
		t.Errorf("Expected 1 relation, got %d", registry.Count())
	}

	// Duplicate should fail
	err = registry.Add(rel)
	if err == nil {
		t.Errorf("Expected error for duplicate relation")
	}
}

func TestRegistry_Update(t *testing.T) {
	registry := NewRegistry()

	rel := NewRelation("user-alice", NewResourceIDWithHID(ResourceMemory, "user-alice", "mem-123"), RelationReader)
	registry.Add(rel)

	// Update to writer
	rel.RelationType = RelationWriter
	err := registry.Update(rel)
	if err != nil {
		t.Errorf("Update() error = %v", err)
	}

	// Check updated relation
	privilege := registry.GetPrivilege("user-alice", "", rel.Resource)
	if privilege != RelationWriter {
		t.Errorf("Expected RelationWriter after update, got %s", privilege)
	}
}

func TestRegistry_GetPrivilege(t *testing.T) {
	registry := NewRegistry()

	rel1 := NewRelation("user-alice", NewResourceIDWithHID(ResourceMemory, "user-alice", "mem-1"), RelationReader)
	rel2 := NewRelation("user-alice", NewResourceIDWithHID(ResourceMemory, "user-alice", "mem-1"), RelationWriter)

	registry.Add(rel1)
	registry.Add(rel2)

	// Get highest privilege
	privilege := registry.GetPrivilege("user-alice", "", rel1.Resource)
	if privilege != RelationWriter {
		t.Errorf("Expected RelationWriter, got %s", privilege)
	}
}

func TestRegistry_HasPrivilege(t *testing.T) {
	registry := NewRegistry()

	rel := NewRelation("user-alice", NewResourceIDWithHID(ResourceMemory, "user-alice", "mem-1"), RelationWriter)
	registry.Add(rel)

	// Check has privilege
	if !registry.HasPrivilege("user-alice", "", rel.Resource, RelationReader) {
		t.Errorf("Expected to have reader privilege")
	}

	if !registry.HasPrivilege("user-alice", "", rel.Resource, RelationWriter) {
		t.Errorf("Expected to have writer privilege")
	}

	// Should not have admin privilege
	if registry.HasPrivilege("user-alice", "", rel.Resource, RelationAdmin) {
		t.Errorf("Expected to not have admin privilege")
	}
}

func TestRegistry_Clear(t *testing.T) {
	registry := NewRegistry()

	rel := NewRelation("user-alice", NewResourceIDWithHID(ResourceMemory, "user-alice", "mem-1"), RelationReader)
	registry.Add(rel)

	if registry.Count() != 1 {
		t.Errorf("Expected 1 relation, got %d", registry.Count())
	}

	registry.Clear()

	if registry.Count() != 0 {
		t.Errorf("Expected 0 relations after clear, got %d", registry.Count())
	}
}

func TestStatement_Matches(t *testing.T) {
	resource := NewResourceIDWithHID(ResourceMemory, "user-alice", "mem-123")

	tests := []struct {
		name   string
		stmt   *Statement
		action Action
		resource *ResourceID
		expected bool
	}{
		{
			name:     "exact match",
			stmt:     NewAllowStatement(ActionRead, resource),
			action:   ActionRead,
			resource: resource,
			expected: true,
		},
		{
			name:     "action mismatch",
			stmt:     NewAllowStatement(ActionRead, resource),
			action:   ActionWrite,
			resource: resource,
			expected: false,
		},
		{
			name:     "wildcard action",
			stmt:     &Statement{Effect: EffectAllow, Action: ActionAny, Resource: resource},
			action:   ActionRead,
			resource: resource,
			expected: true,
		},
		{
			name:     "wildcard resource",
			stmt:     &Statement{Effect: EffectAllow, Action: ActionRead, Resource: NewResourceID(ResourceAny, "*")},
			action:   ActionRead,
			resource: resource,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.stmt.Matches(tt.action, tt.resource); got != tt.expected {
				t.Errorf("Statement.Matches() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPolicy_Evaluate(t *testing.T) {
	policy := NewPolicy("test-policy")

	resource := NewResourceIDWithHID(ResourceMemory, "user-alice", "mem-123")

	// Add allow statement
	policy.AddAllow(ActionRead, resource)

	effect := policy.Evaluate(ActionRead, resource)
	if effect != EffectAllow {
		t.Errorf("Expected allow for read, got %s", effect)
	}

	effect = policy.Evaluate(ActionWrite, resource)
	if effect != EffectDeny {
		t.Errorf("Expected deny for write, got %s", effect)
	}

	// Add deny statement
	policy.AddDeny(ActionWrite, resource)

	effect = policy.Evaluate(ActionWrite, resource)
	if effect != EffectDeny {
		t.Errorf("Expected deny (explicit), got %s", effect)
	}

	// Deny should take precedence over allow
	policy.AddAllow(ActionWrite, resource)
	effect = policy.Evaluate(ActionWrite, resource)
	if effect != EffectDeny {
		t.Errorf("Expected deny (deny takes precedence), got %s", effect)
	}
}

func TestPolicy_IsInScope(t *testing.T) {
	policy := NewPolicy("test-policy")
	policy.Scope = []string{"user-alice", "user-bob"}

	if !policy.IsInScope("user-alice") {
		t.Errorf("Expected user-alice in scope")
	}

	if policy.IsInScope("user-charlie") {
		t.Errorf("Expected user-charlie not in scope")
	}
}

func TestPolicySet_Evaluate(t *testing.T) {
	ps := NewPolicySet()

	policy1 := NewPolicy("policy-1")
	policy1.AddAllow(ActionRead, NewResourceID(ResourceMemory, "mem-1"))

	policy2 := NewPolicy("policy-2")
	policy2.AddDeny(ActionWrite, NewResourceID(ResourceMemory, "mem-1"))

	ps.Add(policy1)
	ps.Add(policy2)

	resource := NewResourceID(ResourceMemory, "mem-1")

	effect := ps.Evaluate("user-alice", ActionRead, resource)
	if effect != EffectAllow {
		t.Errorf("Expected allow for read, got %s", effect)
	}

	effect = ps.Evaluate("user-alice", ActionWrite, resource)
	if effect != EffectDeny {
		t.Errorf("Expected deny for write, got %s", effect)
	}
}

func TestAuthorizer_Authorize(t *testing.T) {
	authz := NewAuthorizer()

	resource := NewResourceIDWithHID(ResourceMemory, "user-alice", "mem-123")

	// Owner should be allowed
	req := &AuthzRequest{
		SubjectHID: "user-alice",
		Action:     ActionRead,
		Resource:   resource,
	}

	result := authz.Authorize(req)
	if !result.Allowed {
		t.Errorf("Owner should be allowed, reason: %s", result.Reason)
	}

	// Non-owner should be denied by default
	req.SubjectHID = "user-bob"
	result = authz.Authorize(req)
	if result.Allowed {
		t.Errorf("Non-owner should be denied, reason: %s", result.Reason)
	}

	// Grant relation to user-bob
	err := authz.Grant("user-bob", "", resource, RelationReader)
	if err != nil {
		t.Errorf("Grant() error = %v", err)
	}

	// Now user-bob should be allowed for read
	req.SubjectHID = "user-bob"
	req.Action = ActionRead
	result = authz.Authorize(req)
	if !result.Allowed {
		t.Errorf("user-bob should be allowed as reader, reason: %s", result.Reason)
	}

	// But not for write
	req.Action = ActionWrite
	result = authz.Authorize(req)
	if result.Allowed {
		t.Errorf("user-bob should not be allowed as writer, reason: %s", result.Reason)
	}
}

func TestAuthorizer_ConvenienceMethods(t *testing.T) {
	authz := NewAuthorizer()

	resource := NewResourceIDWithHID(ResourceMemory, "user-alice", "mem-123")

	// Grant reader relation
	authz.Grant("user-bob", "", resource, RelationReader)

	tests := []struct {
		name   string
		method  string
		subject string
		allowed bool
	}{
		{"owner read", "CanRead", "user-alice", true},
		{"owner write", "CanWrite", "user-alice", true},
		{"owner delete", "CanDelete", "user-alice", true},
		{"owner execute", "CanExecute", "user-alice", true},
		{"owner admin", "CanAdmin", "user-alice", true},
		{"owner share", "CanShare", "user-alice", true},
		{"reader read", "CanRead", "user-bob", true},
		{"reader write", "CanWrite", "user-bob", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var allowed bool
			switch tt.method {
			case "CanRead":
				allowed = authz.CanRead(tt.subject, "", resource)
			case "CanWrite":
				allowed = authz.CanWrite(tt.subject, "", resource)
			case "CanDelete":
				allowed = authz.CanDelete(tt.subject, "", resource)
			case "CanExecute":
				allowed = authz.CanExecute(tt.subject, "", resource)
			case "CanAdmin":
				allowed = authz.CanAdmin(tt.subject, "", resource)
			case "CanShare":
				allowed = authz.CanShare(tt.subject, "", resource)
			}

			if allowed != tt.allowed {
				t.Errorf("%s() = %v, want %v", tt.method, allowed, tt.allowed)
			}
		})
	}
}

func TestAuthorizer_FilterAuthorized(t *testing.T) {
	authz := NewAuthorizer()

	resource1 := NewResourceIDWithHID(ResourceMemory, "user-alice", "mem-1")
	resource2 := NewResourceIDWithHID(ResourceMemory, "user-alice", "mem-2") // Same owner

	// Grant read access to user-bob for resource1
	authz.Grant("user-bob", "", resource1, RelationReader)

	// Grant read access to user-bob for resource2
	authz.Grant("user-bob", "", resource2, RelationReader)

	resources := []*ResourceID{resource1, resource2}
	authorized := authz.FilterAuthorized("user-bob", "", resources, ActionRead)

	if len(authorized) != 2 {
		t.Errorf("Expected 2 authorized resources, got %d", len(authorized))
	}

	// Filter by write should return none (user-bob only has Reader, not owner)
	authorized = authz.FilterAuthorized("user-bob", "", resources, ActionWrite)
	if len(authorized) != 0 {
		t.Errorf("Expected 0 authorized resources for write, got %d", len(authorized))
	}
}

func TestAuthorizer_Stats(t *testing.T) {
	authz := NewAuthorizer()

	stats := authz.Stats()

	if stats["relation_count"] != 0 {
		t.Errorf("Expected 0 relations, got %v", stats["relation_count"])
	}

	resource := NewResourceIDWithHID(ResourceMemory, "user-alice", "mem-1")
	authz.Grant("user-bob", "", resource, RelationReader)

	stats = authz.Stats()

	if stats["relation_count"] != 1 {
		t.Errorf("Expected 1 relation, got %v", stats["relation_count"])
	}
}

func TestActionAllowedByRelation(t *testing.T) {
	tests := []struct {
		action Action
		relType RelationType
		expected bool
	}{
		{ActionRead, RelationReader, true},
		{ActionRead, RelationWriter, true},
		{ActionRead, RelationAdmin, true},
		{ActionWrite, RelationReader, false},
		{ActionWrite, RelationWriter, true},
		{ActionDelete, RelationAdmin, true},
		{ActionDelete, RelationWriter, false},
		{ActionExecute, RelationExecutor, true},
		{ActionExecute, RelationWriter, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.action)+"_"+string(tt.relType), func(t *testing.T) {
			if got := ActionAllowedByRelation(tt.action, tt.relType); got != tt.expected {
				t.Errorf("ActionAllowedByRelation(%s, %s) = %v, want %v", tt.action, tt.relType, got, tt.expected)
			}
		})
	}
}

func TestParseEffect(t *testing.T) {
	tests := []struct {
		input string
		want  Effect
	}{
		{"allow", EffectAllow},
		{"ALLOW", EffectAllow},
		{"deny", EffectDeny},
		{"DENY", EffectDeny},
		{"invalid", EffectDeny}, // Returns deny on invalid
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseEffect(tt.input)
			if tt.input == "invalid" && err == nil {
				t.Errorf("Expected error for invalid input")
			}
			if tt.input != "invalid" && got != tt.want {
				t.Errorf("ParseEffect(%s) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNewRelationWithSID(t *testing.T) {
	rel := NewRelationWithSID(
		"user-alice",
		"node-01",
		NewResourceIDWithHID(ResourceMemory, "user-alice", "mem-123"),
		RelationReader,
	)

	if rel.SubjectHID != "user-alice" {
		t.Errorf("SubjectHID = %v, want user-alice", rel.SubjectHID)
	}
	if rel.SubjectSID != "node-01" {
		t.Errorf("SubjectSID = %v, want node-01", rel.SubjectSID)
	}

	if !rel.IsDirect() {
		t.Errorf("Expected IsDirect to be true")
	}
	if rel.IsTenant() {
		t.Errorf("Expected IsTenant to be false")
	}
}

func TestNewRelation(t *testing.T) {
	rel := NewRelation(
		"user-alice",
		NewResourceIDWithHID(ResourceMemory, "user-alice", "mem-123"),
		RelationReader,
	)

	if rel.SubjectSID != "" {
		t.Errorf("Expected empty SubjectSID")
	}

	if rel.IsDirect() {
		t.Errorf("Expected IsDirect to be false")
	}
	if !rel.IsTenant() {
		t.Errorf("Expected IsTenant to be true")
	}
}

func TestResource_SetGetAttribute(t *testing.T) {
	resource := NewResource(ResourceMemory, "user-alice", "mem-123")

	resource.SetAttribute("key1", "value1")
	resource.SetAttribute("key2", "value2")

	v, ok := resource.GetAttribute("key1")
	if !ok || v != "value1" {
		t.Errorf("GetAttribute failed")
	}

	_, ok = resource.GetAttribute("missing")
	if ok {
		t.Errorf("Expected false for missing key")
	}
}

func TestResource_Tags(t *testing.T) {
	resource := NewResource(ResourceMemory, "user-alice", "mem-123")

	resource.AddTag("tag1")
	resource.AddTag("tag2")
	resource.AddTag("tag1") // Duplicate

	if !resource.HasTag("tag1") {
		t.Errorf("Expected tag1")
	}
	if !resource.HasTag("tag2") {
		t.Errorf("Expected tag2")
	}
	if len(resource.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(resource.Tags))
	}

	resource.RemoveTag("tag1")
	if resource.HasTag("tag1") {
		t.Errorf("tag1 should be removed")
	}
}
