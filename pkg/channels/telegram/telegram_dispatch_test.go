package telegram

import (
	"context"
	"testing"

	"github.com/mymmrac/telego"

	"github.com/sipeed/picoclaw/pkg/commands"
)

func TestDispatchCommand_UsesDispatcher(t *testing.T) {
	ch := &TelegramChannel{}
	called := false
	ch.dispatcher = commands.DispatchFunc(func(context.Context, commands.Request) commands.Result {
		called = true
		return commands.Result{Matched: true, Command: "noop"}
	})

	msg := telego.Message{
		Text:      "/help",
		MessageID: 7,
		Chat: telego.Chat{
			ID: 123,
		},
	}

	handled := ch.dispatchCommand(context.Background(), msg)
	if !handled || !called {
		t.Fatalf("handled=%v called=%v", handled, called)
	}
}
