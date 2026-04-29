// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package protocoltypes

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestMessage_NameJSONRoundtrip verifies that the Name field — added for
// multi-user sender attribution (issue #2702) — round-trips through
// json.Marshal/Unmarshal. The session persistence layer (JSONL append +
// in-memory snapshot via json.MarshalIndent) relies on this transparency
// to carry sender attribution into stored history without bespoke code.
func TestMessage_NameJSONRoundtrip(t *testing.T) {
	in := Message{Role: "user", Content: "hi", Name: "U_alice"}
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if !strings.Contains(string(data), `"name":"U_alice"`) {
		t.Errorf("expected name in marshaled JSON, got: %s", string(data))
	}
	var out Message
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if out.Name != "U_alice" {
		t.Errorf("Name after roundtrip = %q, want U_alice", out.Name)
	}
}

// TestMessage_NameOmittedWhenEmpty preserves wire/persistence backward
// compatibility for direct-channel turns and pre-2702 session files: an
// empty Name must not appear as `"name":""` in either marshaled output.
func TestMessage_NameOmittedWhenEmpty(t *testing.T) {
	in := Message{Role: "user", Content: "hi"}
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if strings.Contains(string(data), `"name"`) {
		t.Errorf("empty Name should be omitted; got: %s", string(data))
	}
}

// TestMessage_LegacyHistoryUnmarshalsToEmptyName verifies that historical
// session files written before this field existed still load cleanly: the
// Name field is simply zero-valued.
func TestMessage_LegacyHistoryUnmarshalsToEmptyName(t *testing.T) {
	legacy := []byte(`{"role":"user","content":"old message"}`)
	var msg Message
	if err := json.Unmarshal(legacy, &msg); err != nil {
		t.Fatalf("json.Unmarshal legacy: %v", err)
	}
	if msg.Name != "" {
		t.Errorf("legacy Name = %q, want empty", msg.Name)
	}
	if msg.Content != "old message" {
		t.Errorf("legacy Content = %q, want preserved", msg.Content)
	}
}
