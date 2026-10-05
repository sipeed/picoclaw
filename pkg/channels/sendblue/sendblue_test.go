package sendblue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

const (
	line   = "+15550000001"
	sender = "+15550000002"
)

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	cfg := config.DefaultConfig()
	err := json.Unmarshal(
		[]byte(
			`{"sendblue":{"enabled":true,"type":"sendblue","allow_from":["+15550000002"],"settings":{"api_key":"test-key","api_secret":"test-secret","from_number":"+15550000001","webhook_secret":"test-webhook-secret"}}}`,
		),
		&cfg.Channels,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := config.InitChannelList(cfg.Channels); err != nil {
		t.Fatal(err)
	}
	return cfg
}

func testChannel(t *testing.T) (*SendblueChannel, *bus.MessageBus) {
	t.Helper()
	cfg := testConfig(t)
	b := bus.NewMessageBus()
	manager, err := channels.NewManager(cfg, b, nil)
	if err != nil {
		t.Fatal(err)
	}
	ch, ok := manager.GetChannel("sendblue")
	if !ok {
		t.Fatal("Sendblue factory was not discovered")
	}
	c := ch.(*SendblueChannel)
	if err := c.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Stop(context.Background()); b.Close() })
	return c, b
}

func payload(changes map[string]any) string {
	event := map[string]any{
		"message_handle": "in-1",
		"is_outbound":    false,
		"from_number":    sender,
		"to_number":      line,
		"content":        "Hello agent",
		"status":         "RECEIVED",
	}
	for k, v := range changes {
		event[k] = v
	}
	data, _ := json.Marshal(event)
	return string(data)
}

func receive(c *SendblueChannel, body, secret string) int {
	r := httptest.NewRequest(http.MethodPost, c.WebhookPath(), strings.NewReader(body))
	r.Header.Set("Sb-Signing-Secret", secret)
	w := httptest.NewRecorder()
	c.ServeHTTP(w, r)
	return w.Code
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestWebhookToBusAndReply(t *testing.T) {
	c, b := testChannel(t)
	server := httptest.NewServer(c)
	defer server.Close()
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			req, _ := http.NewRequest(http.MethodPost, server.URL+c.WebhookPath(), strings.NewReader(payload(nil)))
			req.Header.Set("Sb-Signing-Secret", "test-webhook-secret")
			resp, err := server.Client().Do(req)
			if err != nil {
				t.Error(err)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != 204 {
				t.Errorf("webhook status = %d", resp.StatusCode)
			}
		})
	}
	wg.Wait()
	var msg bus.InboundMessage
	select {
	case msg = <-b.InboundChan():
	case <-time.After(time.Second):
		t.Fatal("no inbound message")
	}
	if msg.Channel != "sendblue" || msg.ChatID != sender || msg.SenderID != sender || msg.Content != "Hello agent" ||
		msg.Context.ChatType != "direct" {
		t.Fatalf("wrong inbound message: %+v", msg)
	}
	select {
	case <-b.InboundChan():
		t.Fatal("duplicate was published")
	default:
	}
	var calls atomic.Int32
	api := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != "POST" || r.URL.Path != "/api/send-message" {
			t.Errorf("wrong endpoint: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Sb-Api-Key-Id") != "test-key" || r.Header.Get("Sb-Api-Secret-Key") != "test-secret" {
			t.Error("missing auth headers")
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body["from_number"] != line || body["number"] != sender || body["content"] != "Hello human" {
			t.Errorf("wrong send body: %v", body)
		}
		fmt.Fprint(w, `{"message_handle":"out-1","status":"QUEUED","error_code":0}`)
	}))
	defer api.Close()
	target, _ := url.Parse(api.URL)
	c.client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "api.sendblue.com" {
			t.Error("unexpected API host")
		}
		clone := r.Clone(r.Context())
		clone.URL.Scheme = target.Scheme
		clone.URL.Host = target.Host
		return api.Client().Transport.RoundTrip(clone)
	})
	ids, err := c.Send(context.Background(), bus.OutboundMessage{ChatID: msg.ChatID, Content: "Hello human"})
	if err != nil || len(ids) != 1 || ids[0] != "out-1" || calls.Load() != 1 {
		t.Fatalf("send = %v, %v, calls=%d", ids, err, calls.Load())
	}
}

func TestWebhookBoundaries(t *testing.T) {
	cases := []struct {
		name, secret string
		changes      map[string]any
		status       int
	}{
		{"bad secret", "wrong", nil, 401},
		{"outbound", "test-webhook-secret", map[string]any{"is_outbound": true}, 204},
		{"missing direction", "test-webhook-secret", map[string]any{"is_outbound": nil}, 400},
		{"invalid direction", "test-webhook-secret", map[string]any{"is_outbound": "false"}, 400},
		{"other line", "test-webhook-secret", map[string]any{"to_number": sender}, 204},
		{"other sender", "test-webhook-secret", map[string]any{"from_number": "+15550000003"}, 204},
		{"group", "test-webhook-secret", map[string]any{"group_id": "group"}, 204},
		{"delivery update", "test-webhook-secret", map[string]any{"status": "DELIVERED"}, 204},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, b := testChannel(t)
			if code := receive(c, payload(tc.changes), tc.secret); code != tc.status {
				t.Fatalf("status %d, want %d", code, tc.status)
			}
			select {
			case <-b.InboundChan():
				t.Fatal("unexpected inbound")
			default:
			}
		})
	}
	c, _ := testChannel(t)
	if code := receive(c, "{", "test-webhook-secret"); code != 400 {
		t.Fatalf("malformed: %d", code)
	}
	if code := receive(c, strings.Repeat("x", 65537), "test-webhook-secret"); code != 413 {
		t.Fatalf("oversize: %d", code)
	}
}

func TestSendFailuresArePermanent(t *testing.T) {
	for _, tc := range []struct {
		status int
		body   string
	}{
		{401, `{}`},
		{429, `{}`},
		{500, `{}`},
		{302, `{}`},
		{200, `{}`},
		{200, `{"message_handle":"out-1","status":"ERROR"}`},
		{200, `{"message_handle":"out-1","status":"DECLINED","error_code":4001}`},
	} {
		t.Run(fmt.Sprintf("%d/%s", tc.status, tc.body), func(t *testing.T) {
			c, _ := testChannel(t)
			calls := 0
			c.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
				calls++
				return &http.Response{
					StatusCode: tc.status,
					Body:       io.NopCloser(strings.NewReader(tc.body)),
					Header:     make(http.Header),
				}, nil
			})
			_, err := c.Send(context.Background(), bus.OutboundMessage{ChatID: sender, Content: "Hello"})
			if !errors.Is(err, channels.ErrSendFailed) || calls != 1 {
				t.Fatalf("err=%v calls=%d", err, calls)
			}
		})
	}
	c, _ := testChannel(t)
	c.client.Transport = roundTripFunc(
		func(*http.Request) (*http.Response, error) { return nil, context.DeadlineExceeded },
	)
	_, err := c.Send(context.Background(), bus.OutboundMessage{ChatID: sender, Content: "Hello"})
	if !errors.Is(err, channels.ErrSendFailed) {
		t.Fatal(err)
	}
}

func TestQueueFailureIsNotAcknowledged(t *testing.T) {
	c, b := testChannel(t)
	b.Close()
	if code := receive(c, payload(nil), "test-webhook-secret"); code != 503 {
		t.Fatalf("status = %d", code)
	}
	if len(c.seen) != 0 {
		t.Fatal("failed publication remembered as accepted")
	}
}

func TestConfigFailsClosed(t *testing.T) {
	cfg := testConfig(t)
	decoded, err := cfg.Channels["sendblue"].GetDecoded()
	if err != nil {
		t.Fatal(err)
	}
	settings := decoded.(*config.SendblueSettings)
	cfg.Channels["sendblue"].AllowFrom = nil
	if _, err := NewSendblueChannel(cfg.Channels["sendblue"], settings, bus.NewMessageBus()); err == nil {
		t.Fatal("empty allowlist accepted")
	}
	cfg.Channels["sendblue"].AllowFrom = []string{sender}
	settings.WebhookSecret.Set("")
	if _, err := NewSendblueChannel(cfg.Channels["sendblue"], settings, bus.NewMessageBus()); err == nil {
		t.Fatal("empty secret accepted")
	}
}

func TestConfigurationSaveReloadAndEnvironment(t *testing.T) {
	cfg := testConfig(t)
	path := filepath.Join(t.TempDir(), "config.json")
	if err := config.SaveConfig(path, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := config.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := loaded.Channels["sendblue"].GetDecoded()
	if err != nil {
		t.Fatal(err)
	}
	settings := raw.(*config.SendblueSettings)
	if settings.APIKey.String() != "test-key" || settings.APISecret.String() != "test-secret" ||
		settings.WebhookSecret.String() != "test-webhook-secret" ||
		settings.FromNumber != line {
		t.Fatal("config save/reload lost channel settings")
	}
	t.Setenv("PICOCLAW_CHANNELS_SENDBLUE_API_KEY", "env-key")
	t.Setenv("PICOCLAW_CHANNELS_SENDBLUE_API_SECRET", "env-secret")
	t.Setenv("PICOCLAW_CHANNELS_SENDBLUE_FROM_NUMBER", line)
	t.Setenv("PICOCLAW_CHANNELS_SENDBLUE_WEBHOOK_SECRET", "env-webhook")
	cfg = testConfig(t)
	raw, err = cfg.Channels["sendblue"].GetDecoded()
	if err != nil {
		t.Fatal(err)
	}
	settings = raw.(*config.SendblueSettings)
	if settings.APIKey.String() != "env-key" || settings.APISecret.String() != "env-secret" ||
		settings.WebhookSecret.String() != "env-webhook" {
		t.Fatal("environment overrides not loaded")
	}
}

type conversationProvider struct {
	messages chan []providers.Message
	count    int
}

func (p *conversationProvider) GetDefaultModel() string { return "fixture-model" }

func (p *conversationProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	options map[string]any,
) (*providers.LLMResponse, error) {
	p.count++
	p.messages <- append([]providers.Message(nil), messages...)
	return &providers.LLMResponse{Content: fmt.Sprintf("Reply %d", p.count), FinishReason: "stop"}, nil
}

func TestRealAgentLoopRoundTripAndFollowup(t *testing.T) {
	cfg := testConfig(t)
	cfg.Agents.Defaults.Workspace = t.TempDir()
	cfg.Agents.Defaults.ModelName = "fixture-model"
	cfg.Tools.MCP.Enabled = false
	b := bus.NewMessageBus()
	manager, err := channels.NewManager(cfg, b, nil)
	if err != nil {
		t.Fatal(err)
	}
	ch, ok := manager.GetChannel("sendblue")
	if !ok {
		t.Fatal("channel missing")
	}
	c := ch.(*SendblueChannel)
	replies := make(chan map[string]string, 8)
	c.client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			return nil, err
		}
		replies <- body
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(
				strings.NewReader(`{"message_handle":"out-fixture","status":"QUEUED","error_code":0}`),
			),
			Header: make(http.Header),
		}, nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	if err := manager.StartAll(ctx); err != nil {
		cancel()
		t.Fatal(err)
	}
	provider := &conversationProvider{messages: make(chan []providers.Message, 4)}
	loop := agent.NewAgentLoop(cfg, b, provider)
	loop.SetChannelManager(manager)
	done := make(chan error, 1)
	go func() { done <- loop.Run(ctx) }()
	defer func() {
		cancel()
		loop.Stop()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("agent did not stop")
		}
		loop.Close()
		_ = manager.StopAll(context.Background())
		b.Close()
	}()
	var history []providers.Message
	for index := 1; index <= 2; index++ {
		if code := receive(
			c,
			payload(
				map[string]any{
					"message_handle": fmt.Sprintf("in-%d", index),
					"content":        fmt.Sprintf("Hello agent %d", index),
				},
			),
			"test-webhook-secret",
		); code != 204 {
			t.Fatal(code)
		}
		select {
		case reply := <-replies:
			if reply["content"] != fmt.Sprintf("Reply %d", index) || reply["number"] != sender ||
				reply["from_number"] != line {
				t.Fatalf("wrong reply: %v", reply)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("agent reply timed out")
		}
		select {
		case history = <-provider.messages:
		case <-time.After(time.Second):
			t.Fatal("no model call")
		}
	}
	text := fmt.Sprint(history)
	for _, want := range []string{"Hello agent 1", "Reply 1", "Hello agent 2"} {
		if !strings.Contains(text, want) {
			t.Fatalf("follow-up lost %q", want)
		}
	}
}
