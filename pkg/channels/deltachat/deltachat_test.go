package deltachat

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/media"
)

func TestNewDeltaChatChannel(t *testing.T) {
	msgBus := bus.NewMessageBus()

	// A fake rpc server so resolveServerPath succeeds regardless of host setup.
	fakeServer := filepath.Join(t.TempDir(), "deltachat-rpc-server")
	if err := os.WriteFile(fakeServer, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Run("missing email", func(t *testing.T) {
		bc := &config.Channel{Type: config.ChannelDeltaChat, Enabled: true}
		cfg := &config.DeltaChatSettings{RPCServerPath: fakeServer}
		_, err := NewDeltaChatChannel(bc, cfg, msgBus)
		if err == nil {
			t.Fatal("expected error for missing email")
		}
		if !strings.Contains(err.Error(), "@nine.testrun.org") {
			t.Fatalf("error = %v, want bootstrap server guidance", err)
		}
		if !strings.Contains(err.Error(), "Next step:") || !strings.Contains(err.Error(), "picoclaw g") {
			t.Fatalf("error = %v, want next-step guidance", err)
		}
	})

	t.Run("bootstrap server marker", func(t *testing.T) {
		bc := &config.Channel{Type: config.ChannelDeltaChat, Enabled: true}
		cfg := &config.DeltaChatSettings{Email: "@mehl.cloud", RPCServerPath: fakeServer}
		if _, err := NewDeltaChatChannel(bc, cfg, msgBus); err != nil {
			t.Fatalf("unexpected error for bootstrap marker: %v", err)
		}
	})

	t.Run("missing rpc server", func(t *testing.T) {
		bc := &config.Channel{Type: config.ChannelDeltaChat, Enabled: true}
		cfg := &config.DeltaChatSettings{
			Email:         "bot@example.org",
			RPCServerPath: filepath.Join(t.TempDir(), "does-not-exist"),
		}
		if _, err := NewDeltaChatChannel(bc, cfg, msgBus); err == nil {
			t.Error("expected error for missing rpc server path")
		}
	})

	t.Run("valid config", func(t *testing.T) {
		bc := &config.Channel{Type: config.ChannelDeltaChat, Enabled: true}
		cfg := &config.DeltaChatSettings{
			Email:         "bot@example.org",
			RPCServerPath: fakeServer,
			DataDir:       t.TempDir(),
		}
		ch, err := NewDeltaChatChannel(bc, cfg, msgBus)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ch.Name() != config.ChannelDeltaChat {
			t.Errorf("Name() = %q, want %q", ch.Name(), config.ChannelDeltaChat)
		}
		if ch.IsRunning() {
			t.Error("new channel should not be running")
		}
	})
}

func TestResolveServerPathUsesPATH(t *testing.T) {
	dir := t.TempDir()
	fakeServer := filepath.Join(dir, "deltachat-rpc-server")
	if err := os.WriteFile(fakeServer, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	got, err := resolveServerPath("")
	if err != nil {
		t.Fatalf("resolveServerPath: %v", err)
	}
	if got != fakeServer {
		t.Fatalf("resolveServerPath() = %q, want %q", got, fakeServer)
	}
}

func TestMentionsBot(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		displayName string
		email       string
		want        bool
	}{
		{"display name", "hey PicoBot can you help", "PicoBot", "bot@example.org", true},
		{"case insensitive name", "hey picobot", "PicoBot", "bot@example.org", true},
		{"short display name exact", "hey bot can you help", "bot", "bot@example.org", true},
		{"short display name with punctuation", "AI, summarize this", "ai", "bot@example.org", true},
		{"multi word display name", "hey PicoClaw Bot, can you help", "PicoClaw Bot", "bot@example.org", true},
		{"email local part", "@bot please summarize", "", "bot@example.org", true},
		{"email local part with punctuation", "please summarize, @bot.", "", "bot@example.org", true},
		{"no mention", "just chatting here", "PicoBot", "bot@example.org", false},
		{"local part without @", "the robot is cool", "", "bot@example.org", false},
		{"short display name inside word", "the robot is cool", "bot", "bot@example.org", false},
		{"short display name inside mail", "please email me later", "ai", "bot@example.org", false},
		{"display name with prefix word", "hey SuperPicoClaw Bot", "PicoClaw Bot", "bot@example.org", false},
		{"email local part inside handle", "hello @botanic", "", "bot@example.org", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mentionsBot(tt.content, tt.displayName, tt.email); got != tt.want {
				t.Errorf("mentionsBot(%q, %q, %q) = %v, want %v", tt.content, tt.displayName, tt.email, got, tt.want)
			}
		})
	}
}

func TestHandleMessageMarksSeenOnlyAfterDispatch(t *testing.T) {
	tests := []struct {
		name           string
		chatType       string
		contactRequest bool
		mentionOnly    bool
		closeBus       bool
		acceptFails    bool
		wantCalls      []string
	}{
		{name: "successful dispatch", chatType: "Single", wantCalls: []string{"markseen_msgs"}},
		{
			name:           "contact request accepted before seen",
			chatType:       "Single",
			contactRequest: true,
			wantCalls:      []string{"accept_chat", "markseen_msgs"},
		},
		{
			name:           "failed contact request acceptance stays unseen",
			chatType:       "Single",
			contactRequest: true,
			acceptFails:    true,
			wantCalls:      []string{"accept_chat"},
		},
		{name: "ignored group trigger", chatType: "Group", mentionOnly: true},
		{name: "failed local publish", chatType: "Single", closeBus: true},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messageID := int64(42 + i)
			chat := dcChat{ID: 99, Name: "chat", ChatType: tt.chatType, IsContactRequest: tt.contactRequest}
			msgBus := bus.NewMessageBus()
			if tt.closeBus {
				msgBus.Close()
			} else {
				defer msgBus.Close()
			}

			ch := newTestChannelWithBus(t, msgBus, func(bc *config.Channel) {
				bc.GroupTrigger.MentionOnly = tt.mentionOnly
			})
			ch.ctx = context.Background()
			ch.accountID = 7
			ch.selfAddr = "bot@example.org"

			calls := make(chan rpcRequest, 4)
			rpc, cleanup := newMockRPC(t, func(req rpcRequest) string {
				switch req.Method {
				case "get_message":
					return rpcResult(req, dcMessage{
						ID:     messageID,
						ChatID: chat.ID,
						Text:   "hello",
						Sender: &dcContact{Address: "alice@example.org", DisplayName: "Alice"},
					})
				case "get_full_chat_by_id":
					return rpcResult(req, chat)
				case "accept_chat":
					calls <- req
					if tt.acceptFails {
						return rpcUnexpectedMethod(req)
					}
					return rpcResult(req, nil)
				case "markseen_msgs":
					calls <- req
					return rpcResult(req, nil)
				default:
					return rpcUnexpectedMethod(req)
				}
			})
			defer cleanup()
			ch.rpc = rpc

			ch.handleMessage(messageID)

			var gotCalls []rpcRequest
			for {
				select {
				case call := <-calls:
					gotCalls = append(gotCalls, call)
				default:
					goto drained
				}
			}

		drained:
			if len(gotCalls) != len(tt.wantCalls) {
				t.Fatalf("calls = %v, want %v", rpcMethods(gotCalls), tt.wantCalls)
			}
			for i, want := range tt.wantCalls {
				if gotCalls[i].Method != want {
					t.Fatalf("calls = %v, want %v", rpcMethods(gotCalls), tt.wantCalls)
				}
			}
			if tt.contactRequest && len(gotCalls) > 0 {
				if params := gotCalls[0].Params; len(params) != 2 ||
					params[0] != float64(7) || params[1] != float64(chat.ID) {
					t.Fatalf("accept_chat params = %#v, want account 7 chat %d", params, chat.ID)
				}
			}
		})
	}
}

func TestEnsureAccountRejectsUnconfiguredAccount(t *testing.T) {
	ch := newTestChannel(t)

	rpc, cleanup := newMockRPC(t, func(req rpcRequest) string {
		switch req.Method {
		case "get_all_accounts":
			return rpcResult(req, []dcAccount{{ID: 7, Kind: "Configured", Addr: "bot@example.org"}})
		case "is_configured":
			return rpcResult(req, false)
		default:
			return rpcUnexpectedMethod(req)
		}
	})
	defer cleanup()
	ch.rpc = rpc

	err := ch.ensureAccount(context.Background())
	if err == nil {
		t.Fatal("expected not-configured error")
	}
	if !strings.Contains(err.Error(), "is not configured") {
		t.Fatalf("error = %v, want not-configured error", err)
	}
}

func TestEnsureAccountCreatesBootstrapAccountAndStops(t *testing.T) {
	ch := newTestChannel(t)
	ch.config.Email = "@mehl.cloud"

	rpc, cleanup := newMockRPC(t, func(req rpcRequest) string {
		switch req.Method {
		case "add_account":
			return rpcResult(req, int64(9))
		case "add_transport_from_qr":
			if req.Params[0] != float64(9) {
				t.Fatalf("account id = %v, want 9", req.Params[0])
			}
			if req.Params[1] != "DCACCOUNT:https://mehl.cloud/new" {
				t.Fatalf("qr = %v", req.Params[1])
			}
			return rpcResult(req, nil)
		case "get_config":
			if req.Params[1] != "addr" {
				t.Fatalf("get_config key = %v, want addr", req.Params[1])
			}
			return rpcResult(req, "bot123@mehl.cloud")
		default:
			return rpcUnexpectedMethod(req)
		}
	})
	defer cleanup()
	ch.rpc = rpc

	err := ch.ensureAccount(context.Background())
	if err == nil {
		t.Fatal("expected created-account instruction error")
	}
	if !strings.Contains(err.Error(), "bot123@mehl.cloud") || !strings.Contains(err.Error(), "run PicoClaw again") {
		t.Fatalf("error = %v, want generated email and rerun instruction", err)
	}
}

func TestJoinInviteLinkUsesConfiguredJoinInviteLink(t *testing.T) {
	ch := newTestChannel(t)
	ch.accountID = 7
	ch.config.JoinInviteLink = "DCACCOUNT:https://example.org/new"

	rpc, cleanup := newMockRPC(t, func(req rpcRequest) string {
		switch req.Method {
		case "secure_join":
			if req.Params[0] != float64(7) {
				t.Fatalf("account id = %v, want 7", req.Params[0])
			}
			if req.Params[1] != "DCACCOUNT:https://example.org/new" {
				t.Fatalf("invite link = %v", req.Params[1])
			}
			return rpcResult(req, int64(42))
		case "accept_chat":
			if req.Params[0] != float64(7) || req.Params[1] != float64(42) {
				t.Fatalf("accept_chat params = %#v, want account 7 chat 42", req.Params)
			}
			return rpcResult(req, nil)
		default:
			return rpcUnexpectedMethod(req)
		}
	})
	defer cleanup()
	ch.rpc = rpc

	if err := ch.joinInviteLink(context.Background()); err != nil {
		t.Fatalf("joinInviteLink: %v", err)
	}
}

func TestEnsureAccountUsesConfiguredAccount(t *testing.T) {
	ch := newTestChannel(t)
	ch.config.DisplayName = "Local Bot"
	avatar := filepath.Join(t.TempDir(), "avatar.png")
	if err := os.WriteFile(avatar, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	ch.config.AvatarImage = avatar

	profileConfigCalls := 0
	mdnsConfigured := false
	rpc, cleanup := newMockRPC(t, func(req rpcRequest) string {
		switch req.Method {
		case "get_all_accounts":
			return rpcResult(req, []dcAccount{{ID: 7, Kind: "Configured", Addr: "bot@example.org"}})
		case "is_configured":
			return rpcResult(req, true)
		case "batch_set_config":
			cfg, _ := req.Params[1].(map[string]any)
			if _, ok := cfg["bot"]; ok {
				if cfg["bot"] != "1" || cfg["mdns_enabled"] != "1" {
					t.Fatalf("bot config = %#v, want bot and mdns_enabled set to 1", cfg)
				}
				mdnsConfigured = true
				return rpcResult(req, nil)
			}
			profileConfigCalls++
			if cfg["displayname"] != "Local Bot" {
				t.Fatalf("displayname = %v, want Local Bot", cfg["displayname"])
			}
			if cfg["selfavatar"] != avatar {
				t.Fatalf("selfavatar = %v, want %s", cfg["selfavatar"], avatar)
			}
			return rpcResult(req, nil)
		case "select_account":
			return rpcResult(req, nil)
		case "start_io":
			if !mdnsConfigured {
				return rpcUnexpectedMethod(req)
			}
			return rpcResult(req, nil)
		default:
			return rpcUnexpectedMethod(req)
		}
	})
	defer cleanup()
	ch.rpc = rpc

	if err := ch.ensureAccount(context.Background()); err != nil {
		t.Fatalf("ensureAccount: %v", err)
	}
	if profileConfigCalls != 1 {
		t.Fatalf("profile config calls = %d, want 1", profileConfigCalls)
	}
	if ch.accountID != 7 {
		t.Fatalf("accountID = %d, want 7", ch.accountID)
	}
}

func TestEnsureAccountSkipsMissingAvatarImage(t *testing.T) {
	ch := newTestChannel(t)
	ch.config.AvatarImage = filepath.Join(t.TempDir(), "missing.png")

	profileConfigCalls := 0
	rpc, cleanup := newMockRPC(t, func(req rpcRequest) string {
		switch req.Method {
		case "get_all_accounts":
			return rpcResult(req, []dcAccount{{ID: 7, Kind: "Configured", Addr: "bot@example.org"}})
		case "is_configured":
			return rpcResult(req, true)
		case "batch_set_config":
			cfg, _ := req.Params[1].(map[string]any)
			if _, ok := cfg["bot"]; !ok {
				profileConfigCalls++
			}
			return rpcResult(req, nil)
		case "select_account", "start_io":
			return rpcResult(req, nil)
		default:
			return rpcUnexpectedMethod(req)
		}
	})
	defer cleanup()
	ch.rpc = rpc

	if err := ch.ensureAccount(context.Background()); err != nil {
		t.Fatalf("ensureAccount: %v", err)
	}
	if profileConfigCalls != 0 {
		t.Fatalf("profile config calls = %d, want 0", profileConfigCalls)
	}
}

func TestEnsureAccountRequiresConfiguredAccountWhenAccountMissing(t *testing.T) {
	ch := newTestChannel(t)

	rpc, cleanup := newMockRPC(t, func(req rpcRequest) string {
		switch req.Method {
		case "get_all_accounts":
			return rpcResult(req, []dcAccount{})
		default:
			return rpcUnexpectedMethod(req)
		}
	})
	defer cleanup()
	ch.rpc = rpc

	err := ch.ensureAccount(context.Background())
	if err == nil {
		t.Fatal("expected not-configured error")
	}
	if !strings.Contains(err.Error(), "is not configured") {
		t.Fatalf("error = %v, want not-configured error", err)
	}
}

// TestRPCClientRoundTrip drives the JSON-RPC client against an in-process mock
// server over pipes, verifying id correlation and error propagation.
func TestRPCClientRoundTrip(t *testing.T) {
	reqR, reqW := io.Pipe()   // client -> server
	respR, respW := io.Pipe() // server -> client

	c := &rpcClient{
		stdin:   reqW,
		stdout:  respR,
		pending: make(map[uint64]chan rpcResponse),
	}
	go c.readLoop()

	// Mock server: echo method "ping" -> "pong", anything else -> error.
	go func() {
		scanner := bufio.NewScanner(reqR)
		for scanner.Scan() {
			var req rpcRequest
			if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
				continue
			}
			var resp string
			if req.Method == "ping" {
				resp = `{"jsonrpc":"2.0","id":` + itoa(req.ID) + `,"result":"pong"}`
			} else {
				resp = `{"jsonrpc":"2.0","id":` + itoa(req.ID) + `,"error":{"code":-1,"message":"boom"}}`
			}
			_, _ = respW.Write([]byte(resp + "\n"))
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	raw, err := c.call(ctx, "ping")
	if err != nil {
		t.Fatalf("ping call: %v", err)
	}
	var result string
	if err := json.Unmarshal(raw, &result); err != nil || result != "pong" {
		t.Fatalf("ping result = %q (err %v), want pong", result, err)
	}

	if _, err := c.call(ctx, "explode"); err == nil {
		t.Fatal("expected error from explode call")
	}
}

func itoa(n uint64) string {
	b, _ := json.Marshal(n)
	return string(b)
}

// newTestChannel builds a DeltaChatChannel with a valid config (backed by a fake
// rpc-server binary) but without starting any IO, for unit-testing methods in
// isolation.
func newTestChannel(t *testing.T) *DeltaChatChannel {
	return newTestChannelWithBus(t, bus.NewMessageBus(), nil)
}

func newTestChannelWithBus(t *testing.T, msgBus *bus.MessageBus, configure func(*config.Channel)) *DeltaChatChannel {
	t.Helper()
	fakeServer := filepath.Join(t.TempDir(), "deltachat-rpc-server")
	if err := os.WriteFile(fakeServer, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	bc := &config.Channel{Type: config.ChannelDeltaChat, Enabled: true}
	if configure != nil {
		configure(bc)
	}
	cfg := &config.DeltaChatSettings{
		Email:         "bot@example.org",
		RPCServerPath: fakeServer,
		DataDir:       t.TempDir(),
	}
	ch, err := NewDeltaChatChannel(bc, cfg, msgBus)
	if err != nil {
		t.Fatalf("new channel: %v", err)
	}
	return ch
}

// newMockRPC wires an rpcClient to an in-process server that replies to every
// request with handler(req), so methods that call the rpc can be tested without
// a real deltachat-rpc-server.
func newMockRPC(t *testing.T, handler func(req rpcRequest) string) (*rpcClient, func()) {
	t.Helper()
	reqR, reqW := io.Pipe()
	respR, respW := io.Pipe()
	c := &rpcClient{
		stdin:   reqW,
		stdout:  respR,
		pending: make(map[uint64]chan rpcResponse),
	}
	go c.readLoop()
	go func() {
		scanner := bufio.NewScanner(reqR)
		for scanner.Scan() {
			var req rpcRequest
			if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
				continue
			}
			_, _ = respW.Write([]byte(handler(req) + "\n"))
		}
	}()
	return c, func() { _ = reqW.Close(); _ = respW.Close() }
}

func rpcResult(req rpcRequest, result any) string {
	raw, _ := json.Marshal(result)
	return `{"jsonrpc":"2.0","id":` + itoa(req.ID) + `,"result":` + string(raw) + `}`
}

func rpcUnexpectedMethod(req rpcRequest) string {
	return `{"jsonrpc":"2.0","id":` + itoa(req.ID) + `,"error":{"code":-32601,"message":"unexpected method"}}`
}

// TestMessageDataJSON pins the camelCase keys and omitempty behavior expected by
// Delta Chat's send_msg MessageData parameter.
func TestMessageDataJSON(t *testing.T) {
	raw, err := json.Marshal(dcMessageData{File: "/tmp/x.png"})
	if err != nil {
		t.Fatal(err)
	}
	if got := string(raw); got != `{"file":"/tmp/x.png"}` {
		t.Errorf("json = %s, want only the file field", got)
	}

	raw, _ = json.Marshal(dcMessageData{Text: "hi", File: "/f", Filename: "f.bin"})
	if got := string(raw); got != `{"text":"hi","file":"/f","filename":"f.bin"}` {
		t.Errorf("json = %s, want text/file/filename in camelCase", got)
	}
}

// TestRegisterInboundFile checks that an inbound attachment is copied out of
// Delta Chat's account directory into the tool-readable media temp dir and
// registered with delete-on-cleanup, and that the absence of a store yields an
// empty ref for the annotation fallback.
func TestRegisterInboundFile(t *testing.T) {
	ch := newTestChannel(t)

	tmp := filepath.Join(t.TempDir(), "doc.pdf")
	if err := os.WriteFile(tmp, []byte("%PDF-1.4"), 0o644); err != nil {
		t.Fatal(err)
	}
	msg := &dcMessage{File: tmp, FileName: "doc.pdf", FileMime: "application/pdf"}

	if ref := ch.registerInboundFile("scope", msg); ref != "" {
		t.Errorf("ref without media store = %q, want empty", ref)
	}

	store := media.NewFileMediaStore()
	ch.SetMediaStore(store)
	ref := ch.registerInboundFile("scope", msg)
	if !strings.HasPrefix(ref, "media://") {
		t.Fatalf("ref = %q, want a media:// ref", ref)
	}
	path, meta, err := store.ResolveWithMeta(ref)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })

	// The registered path must be a copy in the media temp dir (tool-readable),
	// not the original blob path, and must have the same contents.
	if path == tmp {
		t.Errorf("path = %q, want a copy in the media temp dir, not the blob path", path)
	}
	if !strings.HasPrefix(filepath.Clean(path), filepath.Clean(media.TempDir())) {
		t.Errorf("path = %q, want it under media temp dir %q", path, media.TempDir())
	}
	if data, rerr := os.ReadFile(path); rerr != nil || string(data) != "%PDF-1.4" {
		t.Errorf("copied file contents = %q (err %v), want %q", string(data), rerr, "%PDF-1.4")
	}
	if meta.ContentType != "application/pdf" {
		t.Errorf("content type = %q, want application/pdf", meta.ContentType)
	}
	if meta.CleanupPolicy != media.CleanupPolicyDeleteOnCleanup {
		t.Errorf("cleanup policy = %q, want delete_on_cleanup", meta.CleanupPolicy)
	}
}

func TestSend_CurrentNumericChatIDAllowedWithoutRecipientResolution(t *testing.T) {
	ch := newTestChannel(t)
	ch.rpc, _ = newMockRPC(t, func(req rpcRequest) string {
		if req.Method != "misc_send_msg" {
			return rpcUnexpectedMethod(req)
		}
		if got, _ := req.Params[1].(float64); got != 42 {
			t.Fatalf("chat id = %v, want 42", req.Params[1])
		}
		return rpcResult(req, []any{int64(1001), map[string]any{}})
	})
	ch.accountID = 7
	ch.SetRunning(true)

	ids, err := ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "42",
		Content: "hello",
		Context: bus.InboundContext{ChatID: "42", SenderID: "friend@example.org"},
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if len(ids) != 1 || ids[0] != "1001" {
		t.Fatalf("ids = %v, want [1001]", ids)
	}
}

func TestSend_CrossChatNumericDeniedByDefault(t *testing.T) {
	ch := newTestChannel(t)
	ch.rpc, _ = newMockRPC(t, func(req rpcRequest) string {
		t.Fatalf("unexpected rpc call: %s", req.Method)
		return rpcUnexpectedMethod(req)
	})
	ch.accountID = 7
	ch.SetRunning(true)

	_, err := ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "99",
		Content: "hello",
		Context: bus.InboundContext{ChatID: "42", SenderID: "admin@example.org"},
	})
	if err == nil || !strings.Contains(err.Error(), "allow_crosspost") {
		t.Fatalf("Send error = %v, want crosspost recipient gate", err)
	}
}

func TestSend_EmailRecipientDeniedByDefault(t *testing.T) {
	ch := newTestChannel(t)
	ch.rpc, _ = newMockRPC(t, func(req rpcRequest) string {
		t.Fatalf("unexpected rpc call: %s", req.Method)
		return rpcUnexpectedMethod(req)
	})
	ch.accountID = 7
	ch.SetRunning(true)

	_, err := ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "friend@example.org",
		Content: "hello",
		Context: bus.InboundContext{ChatID: "42", SenderID: "admin@example.org"},
	})
	if err == nil || !strings.Contains(err.Error(), "allow_crosspost") {
		t.Fatalf("Send error = %v, want crosspost recipient gate", err)
	}
}

func TestSend_EmailRecipientRequiresAllowFrom(t *testing.T) {
	ch := newTestChannelWithBus(t, bus.NewMessageBus(), func(bc *config.Channel) {
		bc.AllowFrom = config.FlexibleStringSlice{"admin@example.org"}
	})
	ch.config.AllowCrosspost = true
	ch.rpc, _ = newMockRPC(t, func(req rpcRequest) string {
		t.Fatalf("unexpected rpc call: %s", req.Method)
		return rpcUnexpectedMethod(req)
	})
	ch.accountID = 7
	ch.SetRunning(true)

	_, err := ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "friend@example.org",
		Content: "hello",
		Context: bus.InboundContext{ChatID: "42", SenderID: "other@example.org"},
	})
	if err == nil || !strings.Contains(err.Error(), "allow_from") {
		t.Fatalf("Send error = %v, want allow_from gate", err)
	}
}

func TestSend_EmailRecipientUsesSessionScopeSenderForAdmin(t *testing.T) {
	ch := newTestChannelWithBus(t, bus.NewMessageBus(), func(bc *config.Channel) {
		bc.AllowFrom = config.FlexibleStringSlice{"admin@example.org"}
	})
	ch.config.AllowCrosspost = true

	var sentChatID float64
	ch.rpc, _ = newMockRPC(t, func(req rpcRequest) string {
		switch req.Method {
		case "lookup_contact_id_by_addr":
			return rpcResult(req, int64(11))
		case "get_chat_id_by_contact_id":
			return rpcResult(req, int64(99))
		case "misc_send_msg":
			sentChatID, _ = req.Params[1].(float64)
			return rpcResult(req, []any{int64(1233), map[string]any{}})
		default:
			return rpcUnexpectedMethod(req)
		}
	})
	ch.accountID = 7
	ch.SetRunning(true)

	_, err := ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "friend@example.org",
		Content: "hello",
		Context: bus.InboundContext{ChatID: "friend@example.org"},
		Scope: &bus.OutboundScope{
			Channel: config.ChannelDeltaChat,
			Values: map[string]string{
				"chat":   "direct:42",
				"sender": "admin@example.org",
			},
		},
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if sentChatID != 99 {
		t.Fatalf("sent chat id = %v, want 99", sentChatID)
	}
}

func TestSend_EmailRecipientResolvesForAllowFromWildcard(t *testing.T) {
	ch := newTestChannelWithBus(t, bus.NewMessageBus(), func(bc *config.Channel) {
		bc.AllowFrom = config.FlexibleStringSlice{"*"}
	})
	ch.config.AllowCrosspost = true

	var sentChatID float64
	ch.rpc, _ = newMockRPC(t, func(req rpcRequest) string {
		switch req.Method {
		case "lookup_contact_id_by_addr":
			return rpcResult(req, int64(11))
		case "get_chat_id_by_contact_id":
			return rpcResult(req, int64(99))
		case "misc_send_msg":
			sentChatID, _ = req.Params[1].(float64)
			return rpcResult(req, []any{int64(1236), map[string]any{}})
		default:
			return rpcUnexpectedMethod(req)
		}
	})
	ch.accountID = 7
	ch.SetRunning(true)

	_, err := ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "friend@example.org",
		Content: "hello",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if sentChatID != 99 {
		t.Fatalf("sent chat id = %v, want 99", sentChatID)
	}
}

func TestSend_CrossChatNumericUsesSessionScopeChatForGate(t *testing.T) {
	ch := newTestChannel(t)
	ch.rpc, _ = newMockRPC(t, func(req rpcRequest) string {
		t.Fatalf("unexpected rpc call: %s", req.Method)
		return rpcUnexpectedMethod(req)
	})
	ch.accountID = 7
	ch.SetRunning(true)

	_, err := ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "99",
		Content: "hello",
		Context: bus.InboundContext{ChatID: "99", SenderID: "admin@example.org"},
		Scope: &bus.OutboundScope{
			Channel: config.ChannelDeltaChat,
			Values:  map[string]string{"chat": "direct:42"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "allow_crosspost") {
		t.Fatalf("Send error = %v, want crosspost recipient gate", err)
	}
}

func TestSend_EmailRecipientResolvesForAdmin(t *testing.T) {
	ch := newTestChannelWithBus(t, bus.NewMessageBus(), func(bc *config.Channel) {
		bc.AllowFrom = config.FlexibleStringSlice{"admin@example.org"}
	})
	ch.config.AllowCrosspost = true

	var sentChatID float64
	ch.rpc, _ = newMockRPC(t, func(req rpcRequest) string {
		switch req.Method {
		case "lookup_contact_id_by_addr":
			if req.Params[1] != "friend@example.org" {
				t.Fatalf("lookup addr = %v, want friend@example.org", req.Params[1])
			}
			return rpcResult(req, int64(11))
		case "get_chat_id_by_contact_id":
			return rpcResult(req, nil)
		case "create_chat_by_contact_id":
			return rpcResult(req, int64(99))
		case "misc_send_msg":
			sentChatID, _ = req.Params[1].(float64)
			return rpcResult(req, []any{int64(1234), map[string]any{}})
		default:
			return rpcUnexpectedMethod(req)
		}
	})
	ch.accountID = 7
	ch.SetRunning(true)

	ids, err := ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "Friend <friend@example.org>",
		Content: "hello",
		Context: bus.InboundContext{ChatID: "42", SenderID: "admin@example.org"},
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if sentChatID != 99 {
		t.Fatalf("sent chat id = %v, want 99", sentChatID)
	}
	if len(ids) != 1 || ids[0] != "1234" {
		t.Fatalf("ids = %v, want [1234]", ids)
	}
}

func TestSend_AliasRecipientResolvesForAdmin(t *testing.T) {
	ch := newTestChannelWithBus(t, bus.NewMessageBus(), func(bc *config.Channel) {
		bc.AllowFrom = config.FlexibleStringSlice{"admin@example.org"}
	})
	ch.config.AllowCrosspost = true

	var sentChatID float64
	ch.rpc, _ = newMockRPC(t, func(req rpcRequest) string {
		switch req.Method {
		case "get_contacts":
			if req.Params[2] != "Alice" {
				t.Fatalf("contact query = %v, want Alice", req.Params[2])
			}
			return rpcResult(req, []dcContact{{ID: 12, Address: "alice@example.org", DisplayName: "Alice"}})
		case "get_chat_id_by_contact_id":
			return rpcResult(req, int64(88))
		case "misc_send_msg":
			sentChatID, _ = req.Params[1].(float64)
			return rpcResult(req, []any{int64(1235), map[string]any{}})
		default:
			return rpcUnexpectedMethod(req)
		}
	})
	ch.accountID = 7
	ch.SetRunning(true)

	_, err := ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "Alice",
		Content: "hello",
		Context: bus.InboundContext{ChatID: "42", SenderID: "admin@example.org"},
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if sentChatID != 88 {
		t.Fatalf("sent chat id = %v, want 88", sentChatID)
	}
}

// TestSendMedia verifies SendMedia resolves a media ref to a local path and
// drives send_msg with the expected MessageData, returning the new message id.
func TestSendMedia(t *testing.T) {
	ch := newTestChannel(t)

	tmp := filepath.Join(t.TempDir(), "photo.png")
	if err := os.WriteFile(tmp, []byte("\x89PNGfake"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := media.NewFileMediaStore()
	ch.SetMediaStore(store)
	ref, err := store.Store(tmp, media.MediaMeta{Filename: "photo.png"}, "scope")
	if err != nil {
		t.Fatal(err)
	}

	captured := make(chan rpcRequest, 1)
	rpc, cleanup := newMockRPC(t, func(req rpcRequest) string {
		captured <- req
		return `{"jsonrpc":"2.0","id":` + itoa(req.ID) + `,"result":4242}`
	})
	defer cleanup()
	ch.rpc = rpc
	ch.accountID = 7
	ch.SetRunning(true)

	ids, err := ch.SendMedia(context.Background(), bus.OutboundMediaMessage{
		ChatID: "99",
		Parts: []bus.MediaPart{{
			Type:     "image",
			Ref:      ref,
			Caption:  "hello",
			Filename: "photo.png",
		}},
	})
	if err != nil {
		t.Fatalf("SendMedia: %v", err)
	}
	if len(ids) != 1 || ids[0] != "4242" {
		t.Fatalf("ids = %v, want [4242]", ids)
	}

	select {
	case req := <-captured:
		if req.Method != "send_msg" {
			t.Errorf("method = %q, want send_msg", req.Method)
		}
		if len(req.Params) != 3 {
			t.Fatalf("params = %v, want [accountID, chatID, data]", req.Params)
		}
		if got, _ := req.Params[0].(float64); got != 7 {
			t.Errorf("account id = %v, want 7", req.Params[0])
		}
		if got, _ := req.Params[1].(float64); got != 99 {
			t.Errorf("chat id = %v, want 99", req.Params[1])
		}
		data, ok := req.Params[2].(map[string]any)
		if !ok {
			t.Fatalf("data param = %T, want object", req.Params[2])
		}
		if data["file"] != tmp {
			t.Errorf("file = %v, want %s", data["file"], tmp)
		}
		if data["text"] != "hello" {
			t.Errorf("text = %v, want hello", data["text"])
		}
		if data["filename"] != "photo.png" {
			t.Errorf("filename = %v, want photo.png", data["filename"])
		}
		if _, present := data["viewtype"]; present {
			t.Errorf("viewtype should be omitted (Delta Chat infers it), got %v", data["viewtype"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("mock server never received the request")
	}
}

// TestSendMediaNoStore ensures SendMedia fails cleanly without a media store.
func TestSendMediaNoStore(t *testing.T) {
	ch := newTestChannel(t)
	ch.SetRunning(true)
	if _, err := ch.SendMedia(context.Background(), bus.OutboundMediaMessage{ChatID: "1"}); err == nil {
		t.Error("expected error when no media store is configured")
	}
}

// TestSendMediaVoice verifies that a send_tts-sourced audio part is delivered
// with viewtype "Voice" so Delta Chat renders it as a voice bubble.
func TestSendMediaVoice(t *testing.T) {
	ch := newTestChannel(t)

	tmp := filepath.Join(t.TempDir(), "tts-123.ogg")
	if err := os.WriteFile(tmp, []byte("OggSfake"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := media.NewFileMediaStore()
	ch.SetMediaStore(store)
	ref, err := store.Store(tmp, media.MediaMeta{
		Filename:    "tts-123.ogg",
		ContentType: "audio/ogg",
		Source:      "tool:send_tts",
	}, "scope")
	if err != nil {
		t.Fatal(err)
	}

	captured := make(chan rpcRequest, 1)
	rpc, cleanup := newMockRPC(t, func(req rpcRequest) string {
		captured <- req
		return `{"jsonrpc":"2.0","id":` + itoa(req.ID) + `,"result":7}`
	})
	defer cleanup()
	ch.rpc = rpc
	ch.accountID = 1
	ch.SetRunning(true)

	if _, err := ch.SendMedia(context.Background(), bus.OutboundMediaMessage{
		ChatID: "5",
		Parts:  []bus.MediaPart{{Type: "audio", Ref: ref, ContentType: "audio/ogg"}},
	}); err != nil {
		t.Fatalf("SendMedia: %v", err)
	}

	select {
	case req := <-captured:
		data, ok := req.Params[2].(map[string]any)
		if !ok {
			t.Fatalf("data param = %T, want object", req.Params[2])
		}
		if data["viewtype"] != "Voice" {
			t.Errorf("viewtype = %v, want Voice", data["viewtype"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("mock server never received the request")
	}
}

// TestDeltaChatViewtype pins the rule that only voice audio is forced to a view
// type; everything else is left to Delta Chat's own detection.
func TestDeltaChatViewtype(t *testing.T) {
	tests := []struct {
		name string
		part bus.MediaPart
		meta media.MediaMeta
		want string
	}{
		{
			"tts audio",
			bus.MediaPart{Type: "audio"},
			media.MediaMeta{Source: "tool:send_tts", ContentType: "audio/ogg"},
			"Voice",
		},
		{"voice filename", bus.MediaPart{Type: "audio", Filename: "my-voice.mp3"}, media.MediaMeta{}, "Voice"},
		{
			"plain audio",
			bus.MediaPart{Type: "audio", Filename: "song.mp3"},
			media.MediaMeta{ContentType: "audio/mpeg"},
			"",
		},
		{"image", bus.MediaPart{Type: "image", Filename: "photo.png"}, media.MediaMeta{ContentType: "image/png"}, ""},
		{"file", bus.MediaPart{Type: "file", Filename: "doc.pdf"}, media.MediaMeta{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deltaChatViewtype(tt.part, tt.meta); got != tt.want {
				t.Errorf("deltaChatViewtype() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestVoiceCapabilities checks that Delta Chat advertises ASR and TTS so the
// gateway's startup capability log is accurate.
func TestVoiceCapabilities(t *testing.T) {
	ch := newTestChannel(t)
	caps := ch.VoiceCapabilities()
	if !caps.ASR || !caps.TTS {
		t.Errorf("VoiceCapabilities() = %+v, want both ASR and TTS true", caps)
	}
}

func rpcMethods(requests []rpcRequest) []string {
	methods := make([]string, len(requests))
	for i, req := range requests {
		methods[i] = req.Method
	}
	return methods
}
