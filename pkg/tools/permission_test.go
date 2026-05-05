package tools

import (
	"testing"
)

func TestPermissionCache_Check_NoPermission(t *testing.T) {
	pc := NewPermissionCache()
	result := pc.Check("/desktop")
	if result != "" {
		t.Errorf("Expected empty string, got %s", result)
	}
}
