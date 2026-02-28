//go:build whatsapp_native

package whatsapp

import (
	"context"
	"testing"

	"github.com/sipeed/picoclaw/pkg/commands"
)

func TestTryHandleCommand_UsesDispatcher(t *testing.T) {
	ch := &WhatsAppNativeChannel{}
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
