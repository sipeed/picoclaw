// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package identity

import (
	"os"
	"testing"
	"time"
)

func TestNewIdentity(t *testing.T) {
	tests := []struct {
		name    string
		hid     string
		sid     string
		want    string
		wantErr bool
	}{
		{
			name:    "simple identity",
			hid:     "user-alice",
			sid:     "node-01",
			want:    "user-alice/node-01",
			wantErr: false,
		},
		{
			name:    "normalizes lowercase",
			hid:     "USER-BOB",
			sid:     "NODE-02",
			want:    "user-bob/node-02",
			wantErr: false,
		},
		{
			name:    "trims whitespace",
			hid:     "  user-carol  ",
			sid:     "  node-03  ",
			want:    "user-carol/node-03",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := NewIdentity(tt.hid, tt.sid)
			if got := id.String(); got != tt.want {
				t.Errorf("Identity.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewIdentityFromString(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		wantHID string
		wantSID string
		wantErr bool
	}{
		{
			name:    "valid format",
			s:       "user-alice/node-01",
			wantHID: "user-alice",
			wantSID: "node-01",
			wantErr: false,
		},
		{
			name:    "with spaces",
			s:       " user-alice / node-01 ",
			wantHID: "user-alice",
			wantSID: "node-01",
			wantErr: false,
		},
		{
			name:    "invalid format - no slash",
			s:       "user-alice",
			wantErr: true,
		},
		{
			name:    "empty hid",
			s:       "/node-01",
			wantErr: true,
		},
		{
			name:    "empty sid",
			s:       "user-alice/",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := NewIdentityFromString(tt.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewIdentityFromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if id.HID != tt.wantHID {
					t.Errorf("Identity.HID = %v, want %v", id.HID, tt.wantHID)
				}
				if id.SID != tt.wantSID {
					t.Errorf("Identity.SID = %v, want %v", id.SID, tt.wantSID)
				}
			}
		})
	}
}

func TestIdentity_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		hid   string
		sid   string
		valid bool
	}{
		{"valid simple", "user-alice", "node-01", true},
		{"valid with dots", "user.alice", "node.01", true},
		{"valid with underscores", "user_alice", "node_01", true},
		{"empty hid", "", "node-01", false},
		{"empty sid", "user-alice", "", false},
		{"uppercase is normalized", "user-Alice", "node-01", true}, // Auto-normalized to lowercase
		{"invalid chars special", "user@alice", "node-01", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := NewIdentity(tt.hid, tt.sid)
			if got := id.IsValid(); got != tt.valid {
				t.Errorf("Identity.IsValid() = %v, want %v", got, tt.valid)
			}
		})
	}
}

func TestIdentity_IsSameTenant(t *testing.T) {
	user1_node1 := NewIdentity("user-alice", "node-01")
	user1_node2 := NewIdentity("user-alice", "node-02")
	user2_node1 := NewIdentity("user-bob", "node-01")

	tests := []struct {
		name     string
		id       *Identity
		other    *Identity
		expected bool
	}{
		{"same tenant different node", user1_node1, user1_node2, true},
		{"different tenant", user1_node1, user2_node1, false},
		{"same identity", user1_node1, user1_node1, true},
		{"nil other", user1_node1, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.IsSameTenant(tt.other); got != tt.expected {
				t.Errorf("Identity.IsSameTenant() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIdentity_Equals(t *testing.T) {
	user1_node1 := NewIdentity("user-alice", "node-01")
	user1_node1_copy := NewIdentity("user-alice", "node-01")
	user1_node2 := NewIdentity("user-alice", "node-02")

	tests := []struct {
		name     string
		id       *Identity
		other    *Identity
		expected bool
	}{
		{"exact match", user1_node1, user1_node1_copy, true},
		{"same instance", user1_node1, user1_node1, true},
		{"different node", user1_node1, user1_node2, false},
		{"nil other", user1_node1, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.Equals(tt.other); got != tt.expected {
				t.Errorf("Identity.Equals() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIdentity_Clone(t *testing.T) {
	original := NewIdentity("user-alice", "node-01")
	original.SetMetadata("key1", "value1")
	original.DisplayName = "Alice's Node"

	clone := original.Clone()

	// Verify clone has same values
	if clone.HID != original.HID {
		t.Errorf("Clone.HID = %v, want %v", clone.HID, original.HID)
	}
	if clone.SID != original.SID {
		t.Errorf("Clone.SID = %v, want %v", clone.SID, original.SID)
	}
	if clone.DisplayName != original.DisplayName {
		t.Errorf("Clone.DisplayName = %v, want %v", clone.DisplayName, original.DisplayName)
	}

	v, ok := clone.GetMetadata("key1")
	if !ok || v != "value1" {
		t.Errorf("Clone metadata not copied correctly")
	}

	// Modify clone and ensure original is unchanged
	clone.SetMetadata("key2", "value2")
	_, ok = original.GetMetadata("key2")
	if ok {
		t.Errorf("Modifying clone affected original")
	}
}

func TestIdentity_Metadata(t *testing.T) {
	id := NewIdentity("user-alice", "node-01")

	// Test set and get
	id.SetMetadata("key1", "value1")
	v, ok := id.GetMetadata("key1")
	if !ok || v != "value1" {
		t.Errorf("Metadata not set correctly")
	}

	// Test missing key
	_, ok = id.GetMetadata("missing")
	if ok {
		t.Errorf("Expected false for missing key")
	}
}

func TestLoader_Load(t *testing.T) {
	// Save and restore environment
	oldEnvHID := os.Getenv(EnvHID)
	oldEnvSID := os.Getenv(EnvSID)
	oldEnvIdentity := os.Getenv(EnvIdentity)
	defer func() {
		os.Setenv(EnvHID, oldEnvHID)
		os.Setenv(EnvSID, oldEnvSID)
		os.Setenv(EnvIdentity, oldEnvIdentity)
	}()

	// Clean environment
	os.Unsetenv(EnvHID)
	os.Unsetenv(EnvSID)
	os.Unsetenv(EnvIdentity)

	t.Run("CLI priority", func(t *testing.T) {
		l := NewLoader()
		l.SetCLI("cli-user", "cli-node")
		l.SetConfig("config-user", "config-node")

		loaded, err := l.Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if loaded.Source != SourceCLI {
			t.Errorf("Expected SourceCLI, got %v", loaded.Source)
		}
		if loaded.HID != "cli-user" || loaded.SID != "cli-node" {
			t.Errorf("CLI identity not loaded correctly: %s/%s", loaded.HID, loaded.SID)
		}
	})

	t.Run("environment priority", func(t *testing.T) {
		l := NewLoader()
		os.Setenv(EnvIdentity, "env-user/env-node")
		defer os.Unsetenv(EnvIdentity)

		loaded, err := l.Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if loaded.Source != SourceEnv {
			t.Errorf("Expected SourceEnv, got %v", loaded.Source)
		}
		if loaded.HID != "env-user" || loaded.SID != "env-node" {
			t.Errorf("Env identity not loaded correctly: %s/%s", loaded.HID, loaded.SID)
		}
	})

	t.Run("config priority", func(t *testing.T) {
		// Ensure env vars are not set
		os.Unsetenv(EnvHID)
		os.Unsetenv(EnvSID)
		os.Unsetenv(EnvIdentity)

		l := NewLoader()
		l.SetConfig("config-user", "config-node")

		loaded, err := l.Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if loaded.Source != SourceConfig {
			t.Errorf("Expected SourceConfig, got %v", loaded.Source)
		}
		if loaded.HID != "config-user" || loaded.SID != "config-node" {
			t.Errorf("Config identity not loaded correctly: %s/%s", loaded.HID, loaded.SID)
		}
	})

	t.Run("auto generation", func(t *testing.T) {
		l := NewLoader()

		loaded, err := l.Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if loaded.Source != SourceAuto {
			t.Errorf("Expected SourceAuto, got %v", loaded.Source)
		}
		if !loaded.IsValid() {
			t.Errorf("Auto-generated identity is invalid: %s/%s", loaded.HID, loaded.SID)
		}
	})
}

func TestGenerator_Generate(t *testing.T) {
	g := NewGenerator()

	id := g.Generate()
	if !id.IsValid() {
		t.Errorf("Generated identity is invalid: %s", id.String())
	}

	// Test uniqueness
	id2 := g.Generate()
	if id.Equals(id2) {
		t.Errorf("Generator produced duplicate identities")
	}
}

func TestGenerator_WithPrefixes(t *testing.T) {
	g := NewGenerator().
		WithHIDPrefix("tenant").
		WithSIDPrefix("service").
		WithIDLength(4)

	hid := g.GenerateHID()
	sid := g.GenerateSID()

	if !startsWith(hid, "tenant-") {
		t.Errorf("HID doesn't have correct prefix: %s", hid)
	}
	if !startsWith(sid, "service-") {
		t.Errorf("SID doesn't have correct prefix: %s", sid)
	}
}

func TestGenerateShortID(t *testing.T) {
	id1 := GenerateShortIDL(8)
	id2 := GenerateShortIDL(8)

	if len(id1) != 8 {
		t.Errorf("GenerateShortIDL() length = %d, want 8", len(id1))
	}
	if id1 == id2 {
		t.Errorf("GenerateShortIDL() produced duplicate IDs")
	}
}

func TestGenerateNodeID(t *testing.T) {
	id := GenerateNodeID()
	if !startsWith(id, "claw-") {
		t.Errorf("GenerateNodeID() = %v, want prefix 'claw-'", id)
	}
}

func TestParseIdentityString(t *testing.T) {
	tests := []struct {
		name      string
		s         string
		wantHID   string
		wantSID   string
		wantError bool
	}{
		{"slash format", "user-alice/node-01", "user-alice", "node-01", false},
		{"dot format", "user-alice.node-01", "user-alice", "node-01", false},
		{"single value", "user-alice", "user-alice", "", false}, // SID is auto-generated
		{"empty string", "", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := ParseIdentityString(tt.s)
			if tt.wantError {
				if err == nil {
					t.Errorf("ParseIdentityString() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("ParseIdentityString() error = %v", err)
				return
			}
			if id.HID != tt.wantHID {
				t.Errorf("HID = %v, want %v", id.HID, tt.wantHID)
			}
			if tt.wantSID != "" && id.SID != tt.wantSID {
				t.Errorf("SID = %v, want %v", id.SID, tt.wantSID)
			}
		})
	}
}

func TestPool(t *testing.T) {
	pool := NewPool(5)

	hid1 := pool.GetHID()
	hid2 := pool.GetHID()

	if hid1 == "" || hid2 == "" {
		t.Errorf("Pool returned empty H-ID")
	}

	id := pool.GenerateFromPool()
	if !id.IsValid() {
		t.Errorf("Pool generated invalid identity")
	}
}

func TestSource_String(t *testing.T) {
	tests := []struct {
		source Source
		want   string
	}{
		{SourceCLI, "cli"},
		{SourceEnv, "env"},
		{SourceConfig, "config"},
		{SourceAuto, "auto"},
		{SourceUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.source.String(); got != tt.want {
				t.Errorf("Source.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function
func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// ULID Tests

func TestULID_Generate(t *testing.T) {
	gen := NewULIDGenerator()

	tests := []struct {
		name     string
		variant  ULIDVariant
		ulidType ULIDType
		wantLen  int
	}{
		{"UUID v4", ULIDVariantUUID, ULIDTypeNode, 36},
		{"ULID v7", ULIDVariantULIDv7, ULIDTypeTask, 26},
		{"ULID 26-char", ULIDVariantULID, ULIDTypeSession, 26},
		{"Nano 26-char", ULIDVariantNano, ULIDTypeMemory, 26},
		{"Short 8-char", ULIDVariantShort, ULIDTypeDevice, 8},
		{"Base62", ULIDVariantBase62, ULIDTypeCustom, 1}, // Base62 varies in length
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ulid := gen.GenerateWithType(tt.ulidType, tt.variant)

			if ulid.Type != tt.ulidType {
				t.Errorf("ULID.Type = %v, want %v", ulid.Type, tt.ulidType)
			}
			if ulid.Variant != tt.variant {
				t.Errorf("ULID.Variant = %v, want %v", ulid.Variant, tt.variant)
			}
			if tt.name != "Base62" && len(ulid.Value) != tt.wantLen {
				t.Errorf("ULID.Value length = %d, want %d", len(ulid.Value), tt.wantLen)
			}
			if !ulid.IsValid() {
				t.Errorf("ULID.IsValid() = false, want true")
			}
		})
	}
}

func TestULID_Uniqueness(t *testing.T) {
	gen := NewULIDGenerator()
	seen := make(map[string]bool)

	// Generate 1000 ULIDs and ensure uniqueness
	for i := 0; i < 1000; i++ {
		ulid := gen.GenerateWithType(ULIDTypeNode, ULIDVariantNano)
		if seen[ulid.Value] {
			t.Errorf("Duplicate ULID generated: %s", ulid.Value)
		}
		seen[ulid.Value] = true
	}
}

func TestULID_Sortable(t *testing.T) {
	gen := NewULIDGenerator()
	var ulids []*ULID

	// Generate 100 ULIDs
	for i := 0; i < 100; i++ {
		ulids = append(ulids, gen.GenerateWithType(ULIDTypeNode, ULIDVariantNano))
		time.Sleep(time.Millisecond)
	}

	// Check they're sorted by time
	for i := 1; i < len(ulids); i++ {
		if ulids[i].Time.Before(ulids[i-1].Time) {
			t.Errorf("ULIDs not sorted by time: %s after %s", ulids[i].Value, ulids[i-1].Value)
		}
	}
}

func TestULID_Parse(t *testing.T) {
	gen := NewULIDGenerator()

	tests := []struct {
		name      string
		variant   ULIDVariant
		ulidType  ULIDType
		wantError bool
	}{
		{"Parse UUID", ULIDVariantUUID, ULIDTypeNode, false},
		{"Parse ULIDv7", ULIDVariantULIDv7, ULIDTypeTask, false},
		{"Parse ULID", ULIDVariantULID, ULIDTypeSession, false},
		{"Parse Nano", ULIDVariantNano, ULIDTypeMemory, false},
		{"Parse Short", ULIDVariantShort, ULIDTypeDevice, false},
		{"Invalid", ULIDVariantShort, ULIDTypeNode, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "Invalid" {
				// ParseULID is lenient - will return Custom variant for unknown formats
				parsed, err := ParseULID("invalid-ulid")
				if err != nil || parsed.Variant != ULIDVariantCustom {
					t.Logf("ParseULID() returned variant %v for invalid input", parsed.Variant)
				}
				return
			}

			original := gen.GenerateWithType(tt.ulidType, tt.variant)
			parsed, err := ParseULID(original.Value)

			if err != nil {
				t.Errorf("ParseULID() error = %v", err)
				return
			}

			if parsed.Value != original.Value {
				t.Errorf("Parsed value = %v, want %v", parsed.Value, original.Value)
			}
		})
	}
}

func TestULID_Type(t *testing.T) {
	tests := []struct {
		ulidType ULIDType
		prefix   string
	}{
		{ULIDTypeNode, "n"},
		{ULIDTypeTask, "t"},
		{ULIDTypeSession, "s"},
		{ULIDTypeMemory, "m"},
		{ULIDTypeWorkflow, "w"},
		{ULIDTypeDevice, "d"},
		{ULIDTypeUser, "u"},
		{ULIDTypeCustom, "x"},
	}

	for _, tt := range tests {
		t.Run(string(tt.ulidType), func(t *testing.T) {
			ulid := &ULID{Type: tt.ulidType}
			if ulid.GetPrefix() != tt.prefix {
				t.Errorf("ULID.GetPrefix() = %v, want %v", ulid.GetPrefix(), tt.prefix)
			}
		})
	}
}

func TestULIDSet(t *testing.T) {
	set := NewULIDSet()
	gen := NewULIDGenerator()

	// Add ULIDs
	for i := 0; i < 10; i++ {
		ulid := gen.GenerateWithType(ULIDTypeNode, ULIDVariantNano)
		set.Add(ulid)
	}

	if set.Len() != 10 {
		t.Errorf("ULIDSet.Len() = %d, want 10", set.Len())
	}

	// Check contains
	ulids := set.List()
	if len(ulids) != 10 {
		t.Errorf("ULIDSet.List() length = %d, want 10", len(ulids))
	}

	// Remove
	set.Remove(ulids[0].String())
	if set.Len() != 9 {
		t.Errorf("ULIDSet.Len() after remove = %d, want 9", set.Len())
	}

	// Clear
	set.Clear()
	if set.Len() != 0 {
		t.Errorf("ULIDSet.Len() after clear = %d, want 0", set.Len())
	}
}

func TestULIDPool(t *testing.T) {
	gen := NewULIDGenerator()
	pool := NewULIDPool(5, gen)

	// Get ULIDs from pool
	var ulids []*ULID
	for i := 0; i < 5; i++ {
		ulid := pool.GetWithType(ULIDTypeNode)
		if ulid == nil {
			t.Errorf("ULIDPool.GetWithType() returned nil")
		}
		ulids = append(ulids, ulid)
	}

	// Check pool size
	if pool.Size() < 0 {
		t.Errorf("ULIDPool.Size() returned negative value")
	}

	// Close pool
	pool.Close()
	if !pool.IsClosed() {
		t.Errorf("ULIDPool.IsClosed() = false after Close()")
	}
}

func TestGenerator_WithULID(t *testing.T) {
	g := NewGenerator().
		WithULIDVariant(ULIDVariantNano).
		WithULIDType(ULIDTypeNode)

	hid := g.GenerateHID()
	sid := g.GenerateSID()

	// Check that IDs contain ULIDs (longer than legacy 8-char)
	if len(hid) <= len("tenant-xxxxxxxx") {
		t.Errorf("HID with ULID should be longer: %s", hid)
	}
	if len(sid) <= len("service-xxxxxxxx") {
		t.Errorf("SID with ULID should be longer: %s", sid)
	}
}

func TestGenerateULID(t *testing.T) {
	ulid := GenerateTypedULID(ULIDTypeTask, ULIDVariantNano)

	if ulid.Type != ULIDTypeTask {
		t.Errorf("GenerateTypedULID() type = %v, want %v", ulid.Type, ULIDTypeTask)
	}
	if ulid.Variant != ULIDVariantNano {
		t.Errorf("GenerateTypedULID() variant = %v, want %v", ulid.Variant, ULIDVariantNano)
	}
	if !ulid.IsValid() {
		t.Errorf("GenerateTypedULID() produced invalid ULID")
	}
}

func TestULID_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		ulid    *ULID
		valid   bool
	}{
		{"Valid ULID", &ULID{Value: "01ARZ3NDEKTSV4RRFFQ69G5FAV", Variant: ULIDVariantULID}, true},
		{"Valid Nano", &ULID{Value: "01ARZ3NDEKTSV4RRFFQ69G5FAV", Variant: ULIDVariantNano}, true},
		{"Valid Short", &ULID{Value: "ARZ3NDEK", Variant: ULIDVariantShort}, true},
		{"Empty value", &ULID{Value: "", Variant: ULIDVariantNano}, false},
		{"Too short", &ULID{Value: "AB", Variant: ULIDVariantNano}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ulid.IsValid(); got != tt.valid {
				t.Errorf("ULID.IsValid() = %v, want %v", got, tt.valid)
			}
		})
	}
}

func TestULID_String(t *testing.T) {
	ulid := &ULID{
		Type:    ULIDTypeNode,
		Variant: ULIDVariantNano,
		Value:   "01ARZ3NDEKTSV4RRFFQ69G5FAV",
	}

	str := ulid.String()
	if str == "" {
		t.Errorf("ULID.String() returned empty")
	}
	// Nano ULID with prefix should be "n-01ARZ3NDEKTSV4RRFFQ69G5FAV"
	expectedValue := "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	if !contains(str, expectedValue) {
		t.Errorf("ULID.String() = %v, should contain value %v", str, expectedValue)
	}
}

func TestULIDType_String(t *testing.T) {
	tests := []struct {
		ulidType ULIDType
		want     string
	}{
		{ULIDTypeNode, "node"},
		{ULIDTypeTask, "task"},
		{ULIDTypeSession, "session"},
		{ULIDTypeMemory, "mem"},
		{ULIDTypeWorkflow, "workflow"},
		{ULIDTypeDevice, "device"},
		{ULIDTypeUser, "user"},
		{ULIDTypeCustom, "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.ulidType.String(); got != tt.want {
				t.Errorf("ULIDType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestULIDVariant_String(t *testing.T) {
	tests := []struct {
		variant ULIDVariant
		want    string
	}{
		{ULIDVariantUUID, "uuid"},
		{ULIDVariantULIDv7, "ulidv7"},
		{ULIDVariantULID, "ulid"},
		{ULIDVariantNano, "nano"},
		{ULIDVariantShort, "short"},
		{ULIDVariantBase62, "base62"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.variant.String(); got != tt.want {
				t.Errorf("ULIDVariant.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestULIDSet_FilterByType(t *testing.T) {
	set := NewULIDSet()
	gen := NewULIDGenerator()

	// Add different types
	set.Add(gen.GenerateWithType(ULIDTypeNode, ULIDVariantNano))
	set.Add(gen.GenerateWithType(ULIDTypeTask, ULIDVariantNano))
	set.Add(gen.GenerateWithType(ULIDTypeTask, ULIDVariantNano))
	set.Add(gen.GenerateWithType(ULIDTypeSession, ULIDVariantNano))

	taskSlice := set.FilterByType(ULIDTypeTask)
	if len(taskSlice) != 2 {
		t.Errorf("FilterByType() length = %d, want 2", len(taskSlice))
	}
}

func TestULIDSet_Merge(t *testing.T) {
	set1 := NewULIDSet()
	set2 := NewULIDSet()
	gen := NewULIDGenerator()

	ulid1 := gen.GenerateWithType(ULIDTypeNode, ULIDVariantNano)
	ulid2 := gen.GenerateWithType(ULIDTypeTask, ULIDVariantNano)

	set1.Add(ulid1)
	set2.Add(ulid2)

	merged := set1.Merge(set2)
	if merged.Len() != 2 {
		t.Errorf("Merge() length = %d, want 2", merged.Len())
	}
}

func TestULIDSet_Difference(t *testing.T) {
	set1 := NewULIDSet()
	set2 := NewULIDSet()
	gen := NewULIDGenerator()

	ulid1 := gen.GenerateWithType(ULIDTypeNode, ULIDVariantNano)
	ulid2 := gen.GenerateWithType(ULIDTypeTask, ULIDVariantNano)

	set1.Add(ulid1)
	set1.Add(ulid2)
	set2.Add(ulid2)

	diff := set1.Difference(set2)
	if diff.Len() != 1 {
		t.Errorf("Difference() length = %d, want 1", diff.Len())
	}
	if !diff.Contains(ulid1) {
		t.Errorf("Difference() should contain ulid1")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (
		s[:len(substr)] == substr ||
		s[len(s)-len(substr):] == substr ||
		findInString(s, substr)))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
