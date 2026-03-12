package state

import (
	"os"
	"testing"
)

func TestHeartbeatTargetsPersistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "state-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sm := NewManager(tmpDir)

	if err := sm.SetLastHeartbeatTarget("telegram:-100123"); err != nil {
		t.Fatalf("SetLastHeartbeatTarget failed: %v", err)
	}
	if err := sm.SetHeartbeatTarget("telegram:-100123/42"); err != nil {
		t.Fatalf("SetHeartbeatTarget failed: %v", err)
	}

	if got := sm.GetLastHeartbeatTarget(); got != "telegram:-100123" {
		t.Fatalf("GetLastHeartbeatTarget = %q, want %q", got, "telegram:-100123")
	}
	if got := sm.GetHeartbeatTarget(); got != "telegram:-100123/42" {
		t.Fatalf("GetHeartbeatTarget = %q, want %q", got, "telegram:-100123/42")
	}

	sm2 := NewManager(tmpDir)
	if got := sm2.GetLastHeartbeatTarget(); got != "telegram:-100123" {
		t.Fatalf("persistent GetLastHeartbeatTarget = %q, want %q", got, "telegram:-100123")
	}
	if got := sm2.GetHeartbeatTarget(); got != "telegram:-100123/42" {
		t.Fatalf("persistent GetHeartbeatTarget = %q, want %q", got, "telegram:-100123/42")
	}
}
