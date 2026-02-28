package whatsapp

import (
	"context"
	"testing"

	"github.com/sipeed/picoclaw/pkg/commands"
)

func TestTryHandleCommand_UsesDispatcher(t *testing.T) {
	ch := &WhatsAppChannel{}
	called := false
	ch.dispatcher = commands.DispatchFunc(func(context.Context, commands.Request) commands.Result {
		called = true
		return commands.Result{Matched: true, Handled: true}
	})

	handled := ch.tryHandleCommand(context.Background(), "/help", "chat1", "user1", "mid1")
	if !handled || !called {
		t.Fatalf("handled=%v called=%v", handled, called)
	}
}

func TestTryHandleCommand_MatchedWithoutHandler_DoesNotFallThrough(t *testing.T) {
	ch := &WhatsAppChannel{}
	ch.dispatcher = commands.DispatchFunc(func(context.Context, commands.Request) commands.Result {
		return commands.Result{Matched: true, Handled: false, Command: "unknown"}
	})

	handled := ch.tryHandleCommand(context.Background(), "/unknown", "chat1", "user1", "mid1")
	if !handled {
		t.Fatal("expected matched command to be treated as handled")
	}
}
