# Cross-Channel Command Registry Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a single-source command system that auto-syncs command handling and Telegram menu registration at startup, while keeping registration optional per channel (e.g., WhatsApp parsing only).

**Architecture:** Introduce a shared `pkg/commands` domain for command definitions + dispatcher, then adapt channels via optional capabilities (`CommandRegistrarCapable`) without changing the core `Channel` interface. Telegram registers menu commands asynchronously with retry and never blocks startup; WhatsApp reuses dispatcher for text command parsing.

**Tech Stack:** Go, `testing` package, existing channel architecture (`pkg/channels/*`), Telegram telego SDK.

---

**Required skills during execution:** @test-driven-development, @verification-before-completion, @requesting-code-review

### Task 1: Create Command Domain (Definition + Registry)

**Files:**
- Create: `pkg/commands/definition.go`
- Create: `pkg/commands/registry.go`
- Test: `pkg/commands/registry_test.go`

**Step 1: Write the failing test**

```go
package commands

import "testing"

func TestRegistry_FilterByChannel(t *testing.T) {
	defs := []Definition{
		{Name: "help", Description: "Show help"},
		{Name: "admin", Description: "Admin only", Channels: []string{"telegram"}},
	}
	r := NewRegistry(defs)

	gotTG := r.ForChannel("telegram")
	if len(gotTG) != 2 {
		t.Fatalf("telegram defs = %d, want 2", len(gotTG))
	}

	gotWA := r.ForChannel("whatsapp")
	if len(gotWA) != 1 || gotWA[0].Name != "help" {
		t.Fatalf("whatsapp defs = %+v, want only help", gotWA)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/commands -run TestRegistry_FilterByChannel -v`  
Expected: FAIL with missing package/files.

**Step 3: Write minimal implementation**

```go
package commands

type Definition struct {
	Name        string
	Description string
	Usage       string
	Aliases     []string
	Channels    []string
}

type Registry struct {
	defs []Definition
}

func NewRegistry(defs []Definition) *Registry { return &Registry{defs: defs} }

func (r *Registry) ForChannel(channel string) []Definition {
	out := make([]Definition, 0, len(r.defs))
	for _, d := range r.defs {
		if len(d.Channels) == 0 {
			out = append(out, d)
			continue
		}
		for _, ch := range d.Channels {
			if ch == channel {
				out = append(out, d)
				break
			}
		}
	}
	return out
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/commands -run TestRegistry_FilterByChannel -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/commands/definition.go pkg/commands/registry.go pkg/commands/registry_test.go
git commit -m "feat(commands): add command definitions and channel-aware registry"
```

### Task 2: Add Dispatcher and Parse Result Contract

**Files:**
- Create: `pkg/commands/dispatcher.go`
- Test: `pkg/commands/dispatcher_test.go`

**Step 1: Write the failing test**

```go
package commands

import (
	"context"
	"testing"
)

func TestDispatcher_MatchSlashCommand(t *testing.T) {
	called := false
	defs := []Definition{
		{
			Name: "help",
			Handler: func(context.Context, Request) error {
				called = true
				return nil
			},
		},
	}
	d := NewDispatcher(NewRegistry(defs))

	res := d.Dispatch(context.Background(), Request{
		Channel: "telegram",
		Text:    "/help",
	})
	if !res.Matched || !called || res.Err != nil {
		t.Fatalf("dispatch result = %+v, called=%v", res, called)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/commands -run TestDispatcher_MatchSlashCommand -v`  
Expected: FAIL with undefined types (`Handler`, `Request`, `Dispatcher`).

**Step 3: Write minimal implementation**

```go
type Handler func(ctx context.Context, req Request) error

type Request struct {
	Channel   string
	ChatID    string
	SenderID  string
	Text      string
	MessageID string
}

type Result struct {
	Matched bool
	Command string
	Err     error
}

type Dispatcher struct {
	reg *Registry
}

func NewDispatcher(reg *Registry) *Dispatcher { return &Dispatcher{reg: reg} }

func (d *Dispatcher) Dispatch(ctx context.Context, req Request) Result {
	cmd := strings.TrimSpace(strings.TrimPrefix(strings.SplitN(req.Text, " ", 2)[0], "/"))
	for _, def := range d.reg.ForChannel(req.Channel) {
		if def.Name == cmd {
			err := def.Handler(ctx, req)
			return Result{Matched: true, Command: def.Name, Err: err}
		}
	}
	return Result{Matched: false}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/commands -run TestDispatcher_MatchSlashCommand -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/commands/dispatcher.go pkg/commands/dispatcher_test.go pkg/commands/definition.go
git commit -m "feat(commands): add dispatcher and request/result contracts"
```

### Task 3: Move Telegram Built-in Commands to Single Source

**Files:**
- Create: `pkg/commands/builtin.go`
- Modify: `pkg/channels/telegram/telegram_commands.go`
- Test: `pkg/commands/builtin_test.go`

**Step 1: Write the failing test**

```go
package commands

import "testing"

func TestBuiltinDefinitions_ContainsTelegramDefaults(t *testing.T) {
	defs := BuiltinDefinitions(nil, nil)
	names := map[string]bool{}
	for _, d := range defs {
		names[d.Name] = true
	}
	for _, want := range []string{"help", "start", "show", "list"} {
		if !names[want] {
			t.Fatalf("missing command %q", want)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/commands -run TestBuiltinDefinitions_ContainsTelegramDefaults -v`  
Expected: FAIL with missing `BuiltinDefinitions`.

**Step 3: Write minimal implementation**

```go
func BuiltinDefinitions(bot TelegramReplyer, cfg *config.Config) []Definition {
	return []Definition{
		{Name: "start", Description: "Start the bot", Usage: "/start", Handler: NewStartHandler(bot)},
		{Name: "help", Description: "Show this help message", Usage: "/help", Handler: NewHelpHandler(bot, nil)},
		{Name: "show", Description: "Show current configuration", Usage: "/show [model|channel]", Handler: NewShowHandler(bot, cfg)},
		{Name: "list", Description: "List available options", Usage: "/list [models|channels]", Handler: NewListHandler(bot, cfg)},
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/commands -run TestBuiltinDefinitions_ContainsTelegramDefaults -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/commands/builtin.go pkg/commands/builtin_test.go pkg/channels/telegram/telegram_commands.go
git commit -m "refactor(commands): centralize built-in command definitions"
```

### Task 4: Add Channel Capability Interface for Platform Registration

**Files:**
- Modify: `pkg/channels/interfaces.go`
- Create: `pkg/channels/interfaces_command_test.go`

**Step 1: Write the failing test**

```go
package channels

import (
	"context"
	"testing"

	"github.com/sipeed/picoclaw/pkg/commands"
)

type mockRegistrar struct{}

func (mockRegistrar) RegisterCommands(context.Context, []commands.Definition) error { return nil }

func TestCommandRegistrarCapable_Compiles(t *testing.T) {
	var _ CommandRegistrarCapable = mockRegistrar{}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/channels -run TestCommandRegistrarCapable_Compiles -v`  
Expected: FAIL with undefined `CommandRegistrarCapable`.

**Step 3: Write minimal implementation**

```go
type CommandRegistrarCapable interface {
	RegisterCommands(ctx context.Context, defs []commands.Definition) error
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/channels -run TestCommandRegistrarCapable_Compiles -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/channels/interfaces.go pkg/channels/interfaces_command_test.go
git commit -m "feat(channels): add optional command registrar capability interface"
```

### Task 5: Integrate Telegram Startup Registration (Non-Blocking + Retry)

**Files:**
- Modify: `pkg/channels/telegram/telegram.go`
- Create: `pkg/channels/telegram/command_registration.go`
- Test: `pkg/channels/telegram/command_registration_test.go`

**Step 1: Write the failing test**

```go
package telegram

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/commands"
)

func TestRegisterCommandsAsync_DoesNotBlockStart(t *testing.T) {
	ch := &TelegramChannel{}
	started := make(chan struct{})

	ch.registerFunc = func(context.Context, []commands.Definition) error {
		close(started)
		return errors.New("temp fail")
	}

	ch.startCommandRegistration(context.Background(), []commands.Definition{{Name: "help"}})

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("registration did not start asynchronously")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/channels/telegram -run TestRegisterCommandsAsync_DoesNotBlockStart -v`  
Expected: FAIL with undefined `registerFunc` / `startCommandRegistration`.

**Step 3: Write minimal implementation**

```go
func (c *TelegramChannel) RegisterCommands(ctx context.Context, defs []commands.Definition) error {
	cmds := make([]telego.BotCommand, 0, len(defs))
	for _, d := range defs {
		cmds = append(cmds, telego.BotCommand{Command: d.Name, Description: d.Description})
	}
	return c.bot.SetMyCommands(ctx, &telego.SetMyCommandsParams{Commands: cmds})
}

func (c *TelegramChannel) startCommandRegistration(ctx context.Context, defs []commands.Definition) {
	go retryWithBackoff(ctx, func(ctx context.Context) error {
		return c.RegisterCommands(ctx, defs)
	})
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/channels/telegram -run TestRegisterCommandsAsync_DoesNotBlockStart -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/channels/telegram/telegram.go pkg/channels/telegram/command_registration.go pkg/channels/telegram/command_registration_test.go
git commit -m "feat(telegram): register bot commands asynchronously with retry on startup"
```

### Task 6: Route Telegram Incoming Messages Through Shared Dispatcher

**Files:**
- Modify: `pkg/channels/telegram/telegram.go`
- Modify: `pkg/channels/telegram/telegram_commands.go`
- Test: `pkg/channels/telegram/telegram_dispatch_test.go`

**Step 1: Write the failing test**

```go
package telegram

import (
	"context"
	"testing"

	"github.com/sipeed/picoclaw/pkg/commands"
)

func TestHandleCommand_UsesDispatcher(t *testing.T) {
	ch := &TelegramChannel{}
	called := false
	ch.dispatcher = &commands.DispatcherStub{
		OnDispatch: func(context.Context, commands.Request) commands.Result {
			called = true
			return commands.Result{Matched: true}
		},
	}

	handled := ch.dispatchCommand(context.Background(), "/help", "1", "2", "3")
	if !handled || !called {
		t.Fatalf("handled=%v called=%v", handled, called)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/channels/telegram -run TestHandleCommand_UsesDispatcher -v`  
Expected: FAIL with missing `dispatcher` / `dispatchCommand`.

**Step 3: Write minimal implementation**

```go
func (c *TelegramChannel) dispatchCommand(
	ctx context.Context,
	text, chatID, senderID, messageID string,
) bool {
	res := c.dispatcher.Dispatch(ctx, commands.Request{
		Channel: "telegram", Text: text, ChatID: chatID, SenderID: senderID, MessageID: messageID,
	})
	return res.Matched
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/channels/telegram -run TestHandleCommand_UsesDispatcher -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/channels/telegram/telegram.go pkg/channels/telegram/telegram_commands.go pkg/channels/telegram/telegram_dispatch_test.go
git commit -m "refactor(telegram): use shared command dispatcher for incoming commands"
```

### Task 7: Add WhatsApp Command Parsing via Shared Dispatcher

**Files:**
- Modify: `pkg/channels/whatsapp/whatsapp.go`
- Modify: `pkg/channels/whatsapp_native/whatsapp_native.go`
- Test: `pkg/channels/whatsapp/whatsapp_command_test.go`
- Test: `pkg/channels/whatsapp_native/whatsapp_command_test.go`

**Step 1: Write the failing test**

```go
package whatsapp

import (
	"context"
	"testing"

	"github.com/sipeed/picoclaw/pkg/commands"
)

func TestIncomingMessage_CommandHandledByDispatcher(t *testing.T) {
	ch := &WhatsAppChannel{}
	called := false
	ch.dispatcher = commands.DispatchFunc(func(context.Context, commands.Request) commands.Result {
		called = true
		return commands.Result{Matched: true}
	})

	handled := ch.tryHandleCommand(context.Background(), "/help", "chat1", "user1", "mid1")
	if !handled || !called {
		t.Fatalf("handled=%v called=%v", handled, called)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/channels/whatsapp -run TestIncomingMessage_CommandHandledByDispatcher -v`  
Expected: FAIL with undefined dispatcher/handler method.

**Step 3: Write minimal implementation**

```go
func (c *WhatsAppChannel) tryHandleCommand(ctx context.Context, text, chatID, senderID, messageID string) bool {
	if c.dispatcher == nil {
		return false
	}
	res := c.dispatcher.Dispatch(ctx, commands.Request{
		Channel: "whatsapp", Text: text, ChatID: chatID, SenderID: senderID, MessageID: messageID,
	})
	return res.Matched
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/channels/whatsapp ./pkg/channels/whatsapp_native -run CommandHandledByDispatcher -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/channels/whatsapp/whatsapp.go pkg/channels/whatsapp_native/whatsapp_native.go pkg/channels/whatsapp/whatsapp_command_test.go pkg/channels/whatsapp_native/whatsapp_command_test.go
git commit -m "feat(whatsapp): reuse shared dispatcher for text commands"
```

### Task 8: End-to-End Verification and Docs Update

**Files:**
- Modify: `README.md`
- Modify: `README.zh.md`
- Modify: `config/config.example.json` (if command-related docs are added)

**Step 1: Write the failing test (integration smoke)**

```go
// pkg/channels/telegram/command_integration_test.go
func TestStart_RegistrationFailureDoesNotStopChannel(t *testing.T) {
	// Build channel with mocked register func returning error,
	// assert Start() returns nil and channel remains running.
}
```

**Step 2: Run tests to verify current gaps**

Run: `go test ./pkg/channels/telegram -run TestStart_RegistrationFailureDoesNotStopChannel -v`  
Expected: FAIL before implementation is complete.

**Step 3: Finish minimal missing implementation + docs**

```markdown
- Add a section: "Command Registry"
- Explain: one-source definitions, Telegram auto registration, WhatsApp parsing-only behavior.
```

**Step 4: Run full verification**

Run: `go test ./pkg/commands ./pkg/channels/... ./pkg/config/... -count=1`  
Expected: PASS.

Run: `go test ./... -count=1`  
Expected: PASS (or document known unrelated failures with logs).

**Step 5: Commit**

```bash
git add README.md README.zh.md config/config.example.json pkg/commands pkg/channels
git commit -m "docs(channels): document single-source command registry behavior"
```

