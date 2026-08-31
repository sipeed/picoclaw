package irc

import (
	"reflect"
	"testing"

	"github.com/ergochat/irc-go/ircevent"
	"github.com/ergochat/irc-go/ircmsg"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestNewIRCChannel(t *testing.T) {
	msgBus := bus.NewMessageBus()

	t.Run("missing server", func(t *testing.T) {
		bc := &config.Channel{Type: config.ChannelIRC, Enabled: true}
		cfg := &config.IRCSettings{Nick: "bot"}
		_, err := NewIRCChannel(bc, cfg, msgBus)
		if err == nil {
			t.Error("expected error for missing server, got nil")
		}
	})

	t.Run("missing nick", func(t *testing.T) {
		bc := &config.Channel{Type: config.ChannelIRC, Enabled: true}
		cfg := &config.IRCSettings{Server: "irc.example.com:6667"}
		_, err := NewIRCChannel(bc, cfg, msgBus)
		if err == nil {
			t.Error("expected error for missing nick, got nil")
		}
	})

	t.Run("valid config", func(t *testing.T) {
		bc := &config.Channel{Type: config.ChannelIRC, Enabled: true}
		cfg := &config.IRCSettings{
			Server:   "irc.example.com:6667",
			Nick:     "testbot",
			Channels: []string{"#test"},
		}
		ch, err := NewIRCChannel(bc, cfg, msgBus)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ch.Name() != "irc" {
			t.Errorf("Name() = %q, want %q", ch.Name(), "irc")
		}
		if ch.IsRunning() {
			t.Error("new channel should not be running")
		}
	})
}

func TestExtractHost(t *testing.T) {
	tests := []struct {
		server string
		want   string
	}{
		{"irc.libera.chat:6697", "irc.libera.chat"},
		{"localhost:6667", "localhost"},
		{"irc.example.com", "irc.example.com"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.server, func(t *testing.T) {
			got := extractHost(tt.server)
			if got != tt.want {
				t.Errorf("extractHost(%q) = %q, want %q", tt.server, got, tt.want)
			}
		})
	}
}

func TestRequestedCapabilities(t *testing.T) {
	configured := []string{"echo-message", "batch", "batch"}
	got := requestedCapabilities(configured)
	want := []string{"echo-message", "batch"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("requestedCapabilities() = %v, want %v", got, want)
	}
	if !reflect.DeepEqual(configured, []string{"echo-message", "batch", "batch"}) {
		t.Fatalf("requestedCapabilities mutated its input: %v", configured)
	}

	defaults := requestedCapabilities(nil)
	wantDefaults := []string{"server-time", "message-tags", "batch", multilineBatchType}
	if !reflect.DeepEqual(defaults, wantDefaults) {
		t.Fatalf("requestedCapabilities(nil) = %v, want %v", defaults, wantDefaults)
	}

	explicitMultiline := requestedCapabilities([]string{multilineBatchType})
	wantExplicit := []string{multilineBatchType, "message-tags", "batch"}
	if !reflect.DeepEqual(explicitMultiline, wantExplicit) {
		t.Fatalf("requestedCapabilities(multiline) = %v, want %v", explicitMultiline, wantExplicit)
	}
}

func multilineTestLine(content string, concat bool) *ircevent.Batch {
	msg := ircmsg.Message{
		Source:  "nick!user@host",
		Command: "PRIVMSG",
		Params:  []string{"#channel", content},
	}
	msg.SetTag("batch", "123")
	if concat {
		msg.SetTag(multilineConcatTag, "")
	}
	return &ircevent.Batch{Message: msg}
}

func multilineTestBatch(items ...*ircevent.Batch) *ircevent.Batch {
	msg := ircmsg.Message{
		Source:  "nick!user@host",
		Command: "BATCH",
		Params:  []string{"+123", multilineBatchType, "#channel"},
	}
	msg.SetTag("account", "alice")
	msg.SetTag("msgid", "message-123")
	return &ircevent.Batch{Message: msg, Items: items}
}

func TestAssembleMultilineBatch(t *testing.T) {
	batch := multilineTestBatch(
		multilineTestLine("hello", false),
		multilineTestLine("", false),
		multilineTestLine("how is ", false),
		multilineTestLine("everyone?", true),
	)
	batch.Items[0].SetTag("time", "2026-08-31T00:00:00Z")

	got, ok := assembleMultilineBatch(batch)
	if !ok {
		t.Fatal("assembleMultilineBatch rejected a valid multiline batch")
	}
	if got.Params[1] != "hello\n\nhow is everyone?" {
		t.Fatalf("assembled content = %q", got.Params[1])
	}
	if got.Source != batch.Source {
		t.Fatalf("assembled source = %q, want batch source %q", got.Source, batch.Source)
	}
	for tag, want := range map[string]string{
		"account": "alice",
		"msgid":   "message-123",
		"time":    "2026-08-31T00:00:00Z",
	} {
		if present, gotValue := got.GetTag(tag); !present || gotValue != want {
			t.Fatalf("assembled %s tag = (%v, %q), want (true, %q)", tag, present, gotValue, want)
		}
	}
	if got.HasTag("batch") || got.HasTag(multilineConcatTag) {
		t.Fatal("assembled message retained transport-only multiline tags")
	}
}

func TestNestedMultilineBatchIsDeliveredOnce(t *testing.T) {
	multiline := multilineTestBatch(
		multilineTestLine("hello", false),
		multilineTestLine("world", false),
	)
	outer := &ircevent.Batch{
		Message: ircmsg.Message{
			Command: "BATCH",
			Params:  []string{"+history", "chathistory", "#channel"},
		},
		Items: []*ircevent.Batch{
			{Message: ircmsg.Message{
				Source:  "other!user@host",
				Command: "PRIVMSG",
				Params:  []string{"#channel", "before"},
			}},
			multiline,
		},
	}

	conn := &ircevent.Connection{}
	var received []string
	conn.AddCallback("PRIVMSG", func(msg ircmsg.Message) {
		received = append(received, msg.Params[1])
	})
	channel := &IRCChannel{}
	conn.AddBatchCallback(func(batch *ircevent.Batch) bool {
		return channel.onMultilineBatch(conn, batch)
	})

	conn.HandleBatch(outer)

	want := []string{"before", "hello\nworld"}
	if !reflect.DeepEqual(received, want) {
		t.Fatalf("nested batch deliveries = %q, want %q", received, want)
	}
}

func TestDeeplyNestedMultilineBatchDoesNotRecurse(t *testing.T) {
	batch := multilineTestBatch(multilineTestLine("hello", false))
	for i := 0; i < 10_000; i++ {
		batch = &ircevent.Batch{
			Message: ircmsg.Message{
				Command: "BATCH",
				Params:  []string{"+outer", "example/outer"},
			},
			Items: []*ircevent.Batch{batch},
		}
	}

	conn := &ircevent.Connection{}
	var received []string
	conn.AddCallback("PRIVMSG", func(msg ircmsg.Message) {
		received = append(received, msg.Params[1])
	})
	channel := &IRCChannel{}
	conn.AddBatchCallback(func(batch *ircevent.Batch) bool {
		return channel.onMultilineBatch(conn, batch)
	})

	conn.HandleBatch(batch)

	if !reflect.DeepEqual(received, []string{"hello"}) {
		t.Fatalf("deeply nested batch deliveries = %q, want [hello]", received)
	}
}

func TestAssembleMultilineBatchRejectsMalformedBatches(t *testing.T) {
	validBatch := func() *ircevent.Batch {
		return multilineTestBatch(multilineTestLine("hello", false))
	}

	tests := map[string]func(*ircevent.Batch){
		"unknown batch type": func(batch *ircevent.Batch) {
			batch.Params[1] = "chathistory"
		},
		"empty batch": func(batch *ircevent.Batch) {
			batch.Items = nil
		},
		"mismatched target": func(batch *ircevent.Batch) {
			batch.Items[0].Params[0] = "#other"
		},
		"mixed command": func(batch *ircevent.Batch) {
			batch.Items[0].Command = "NOTICE"
		},
		"nested batch": func(batch *ircevent.Batch) {
			batch.Items[0].Command = "BATCH"
		},
		"mismatched source": func(batch *ircevent.Batch) {
			batch.Items[0].Source = "mallory!user@host"
		},
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			batch := validBatch()
			mutate(batch)
			if _, ok := assembleMultilineBatch(batch); ok {
				t.Fatal("assembleMultilineBatch accepted a malformed batch")
			}
		})
	}
}

func TestNickMentionedAt(t *testing.T) {
	tests := []struct {
		name    string
		content string
		nick    string
		want    int
	}{
		{"colon prefix", "bot: hello", "bot", 0},
		{"comma prefix", "bot, hello", "bot", 0},
		{"case insensitive", "BOT: hello", "bot", 0},
		{"word boundary mid", "hey bot what's up", "bot", 4},
		{"no mention", "hello world", "bot", -1},
		{"substring mismatch", "robotics are cool", "bot", -1},
		{"nick at end", "hello bot", "bot", 6},
		{"empty content", "", "bot", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nickMentionedAt(tt.content, tt.nick)
			if got != tt.want {
				t.Errorf("nickMentionedAt(%q, %q) = %d, want %d", tt.content, tt.nick, got, tt.want)
			}
		})
	}
}

func TestIsBotMentioned(t *testing.T) {
	tests := []struct {
		name    string
		content string
		nick    string
		want    bool
	}{
		{"colon prefix", "bot: hello", "bot", true},
		{"comma prefix", "bot, hello", "bot", true},
		{"case insensitive", "BOT: hello", "bot", true},
		{"word boundary mid", "hey bot what's up", "bot", true},
		{"no mention", "hello world", "bot", false},
		{"substring mismatch", "robotics are cool", "bot", false},
		{"nick at end", "hello bot", "bot", true},
		{"empty content", "", "bot", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isBotMentioned(tt.content, tt.nick)
			if got != tt.want {
				t.Errorf("isBotMentioned(%q, %q) = %v, want %v", tt.content, tt.nick, got, tt.want)
			}
		})
	}
}

func TestStripBotMention(t *testing.T) {
	tests := []struct {
		name    string
		content string
		nick    string
		want    string
	}{
		{"colon prefix", "bot: hello there", "bot", "hello there"},
		{"comma prefix", "bot, help me", "bot", "help me"},
		{"case insensitive", "BOT: hello", "bot", "hello"},
		{"no prefix match", "hello bot", "bot", "hello bot"},
		{"only prefix", "bot:", "bot", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripBotMention(tt.content, tt.nick)
			if got != tt.want {
				t.Errorf("stripBotMention(%q, %q) = %q, want %q", tt.content, tt.nick, got, tt.want)
			}
		})
	}
}
