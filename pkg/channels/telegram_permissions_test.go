package channels

import (
	"context"
	"testing"
	"time"

	"github.com/mymmrac/telego"
)

func TestTelegramPermissionManager_HandleCallback(t *testing.T) {
	// Test the callback handling logic without a real bot
	pm := &TelegramPermissionManager{}

	// Simulate a pending permission request
	resultCh := make(chan bool, 1)
	pm.pending.Store("42", resultCh)

	// Simulate allow callback
	handled := pm.HandleCallback(context.Background(), telego.CallbackQuery{
		ID:   "query-1",
		Data: "perm_allow_42",
	})
	if !handled {
		t.Error("expected callback to be handled")
	}

	select {
	case approved := <-resultCh:
		if !approved {
			t.Error("expected approval")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for result")
	}
}

func TestTelegramPermissionManager_HandleCallback_Deny(t *testing.T) {
	pm := &TelegramPermissionManager{}
	resultCh := make(chan bool, 1)
	pm.pending.Store("99", resultCh)

	handled := pm.HandleCallback(context.Background(), telego.CallbackQuery{
		ID:   "query-2",
		Data: "perm_deny_99",
	})
	if !handled {
		t.Error("expected callback to be handled")
	}

	select {
	case approved := <-resultCh:
		if approved {
			t.Error("expected denial")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for result")
	}
}

func TestTelegramPermissionManager_HandleCallback_Unknown(t *testing.T) {
	pm := &TelegramPermissionManager{}

	// Non-permission callback should not be handled
	handled := pm.HandleCallback(context.Background(), telego.CallbackQuery{
		ID:   "query-3",
		Data: "some_other_callback",
	})
	if handled {
		t.Error("expected non-permission callback to not be handled")
	}
}

func TestTelegramPermissionManager_HandleCallback_Expired(t *testing.T) {
	pm := &TelegramPermissionManager{}

	// Permission callback with no pending request (expired)
	handled := pm.HandleCallback(context.Background(), telego.CallbackQuery{
		ID:   "query-4",
		Data: "perm_allow_999",
	})
	if !handled {
		t.Error("expected expired permission callback to still be handled (return true)")
	}
}
