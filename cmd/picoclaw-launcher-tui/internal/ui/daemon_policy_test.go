package ui

import "testing"

func TestShouldBlockEnableDaemonWhenRestartRequired(t *testing.T) {
	if !shouldBlockEnableDaemon(true) {
		t.Fatalf("expected enable to be blocked when restart is required")
	}
}

func TestShouldBlockEnableDaemonWhenNoRestartRequired(t *testing.T) {
	if shouldBlockEnableDaemon(false) {
		t.Fatalf("expected enable not to be blocked when restart is not required")
	}
}
