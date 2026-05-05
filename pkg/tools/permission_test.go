package tools

import (
	"context"
	"testing"
)

func TestPermissionCache_Check_NoPermission(t *testing.T) {
	pc := NewPermissionCache()
	result := pc.Check("/desktop")
	if result != "" {
		t.Errorf("Expected empty string, got %s", result)
	}
}

func TestRequestPermissionTool_Name(t *testing.T) {
	pc := NewPermissionCache()
	tool := NewRequestPermissionTool(pc)

	if tool.Name() != "request_permission" {
		t.Errorf("Expected 'request_permission', got %s", tool.Name())
	}
}

func TestRequestPermissionTool_Execute(t *testing.T) {
	pc := NewPermissionCache()
	tool := NewRequestPermissionTool(pc)

	result := tool.Execute(context.Background(), map[string]any{
		"path":    "/desktop",
		"command": "ls /desktop",
	})

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.ForUser == "" {
		t.Error("Expected non-empty ForUser message")
	}
	if result.ForLLM == "" {
		t.Error("Expected non-empty ForLLM message")
	}
}

func TestPermissionCache_GrantAndCheck(t *testing.T) {
	pc := NewPermissionCache()
	pc.Grant("/desktop", "session")

	result := pc.Check("/desktop")
	if result != "session" {
		t.Errorf("Expected 'session', got %s", result)
	}
}

func TestPermissionCache_ParentPathMatch(t *testing.T) {
	pc := NewPermissionCache()
	pc.Grant("/desktop", "session")

	// Child path should match parent's "session" permission
	result := pc.Check("/desktop/folder")
	if result != "session" {
		t.Errorf("Expected 'session' for child path, got %s", result)
	}
}

func TestPermissionCache_Revoke(t *testing.T) {
	pc := NewPermissionCache()
	pc.Grant("/desktop", "session")
	pc.Revoke("/desktop")

	result := pc.Check("/desktop")
	if result != "" {
		t.Errorf("Expected empty after revoke, got %s", result)
	}
}

func TestPermissionCache_Denied(t *testing.T) {
	pc := NewPermissionCache()
	pc.Grant("/desktop", "denied")

	result := pc.Check("/desktop")
	if result != "denied" {
		t.Errorf("Expected 'denied', got %s", result)
	}
}

func TestPermissionCache_Empty(t *testing.T) {
	pc := NewPermissionCache()

	result := pc.Check("/anything")
	if result != "" {
		t.Errorf("Expected empty string, got %s", result)
	}
}
