package agent

import (
	"context"
	"os"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

type streamingTestProvider struct {
	chatCalled       bool
	chatStreamCalled bool
	chunks           []string
	response         *providers.LLMResponse
}

func (p *streamingTestProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	options map[string]any,
) (*providers.LLMResponse, error) {
	p.chatCalled = true
	if p.response != nil {
		return p.response, nil
	}
	return &providers.LLMResponse{Content: "fallback"}, nil
}

func (p *streamingTestProvider) ChatStream(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	options map[string]any,
	onChunk func(accumulated string),
) (*providers.LLMResponse, error) {
	p.chatStreamCalled = true
	for _, chunk := range p.chunks {
		onChunk(chunk)
	}
	if p.response != nil {
		return p.response, nil
	}
	return &providers.LLMResponse{Content: "streamed"}, nil
}

func (p *streamingTestProvider) GetDefaultModel() string {
	return "test-model"
}

type recordingStreamer struct {
	updates   []string
	finalized []string
	canceled  bool
}

func (s *recordingStreamer) Update(ctx context.Context, content string) error {
	s.updates = append(s.updates, content)
	return nil
}

func (s *recordingStreamer) Finalize(ctx context.Context, content string) error {
	s.finalized = append(s.finalized, content)
	return nil
}

func (s *recordingStreamer) Cancel(ctx context.Context) {
	s.canceled = true
}

type recordingStreamDelegate struct {
	streamer *recordingStreamer
	ok       bool
}

func (d *recordingStreamDelegate) GetStreamer(
	ctx context.Context,
	channel, chatID string,
) (bus.Streamer, bool) {
	if !d.ok {
		return nil, false
	}
	return d.streamer, true
}

func newStreamingTestLoop(
	t *testing.T,
	streamingEnabled bool,
	provider *streamingTestProvider,
	streamer *recordingStreamer,
) (*AgentLoop, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "agent-streaming-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp() error = %v", err)
	}

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
				StreamingEnabled:  streamingEnabled,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	msgBus.SetStreamDelegate(&recordingStreamDelegate{
		streamer: streamer,
		ok:       true,
	})

	return NewAgentLoop(cfg, msgBus, provider), func() {
		os.RemoveAll(tmpDir)
	}
}

func TestProcessMessage_UsesStreamingProviderWhenEnabled(t *testing.T) {
	provider := &streamingTestProvider{
		chunks: []string{"Hel", "Hello"},
		response: &providers.LLMResponse{
			Content: "Hello",
		},
	}
	streamer := &recordingStreamer{}
	al, cleanup := newStreamingTestLoop(t, true, provider, streamer)
	defer cleanup()

	response, err := al.processMessage(context.Background(), testInboundMessage(bus.InboundMessage{
		Channel:  "pico",
		ChatID:   "pico:test-session",
		SenderID: "user-1",
		Content:  "hello",
	}))
	if err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}

	if response != "Hello" {
		t.Fatalf("response = %q, want %q", response, "Hello")
	}
	if !provider.chatStreamCalled {
		t.Fatal("expected ChatStream to be used")
	}
	if provider.chatCalled {
		t.Fatal("expected Chat() not to be used when streaming is enabled")
	}
	if len(streamer.updates) != 2 {
		t.Fatalf("stream updates = %#v, want 2 accumulated updates", streamer.updates)
	}
	if len(streamer.finalized) != 1 || streamer.finalized[0] != "Hello" {
		t.Fatalf("finalized = %#v, want final Hello", streamer.finalized)
	}
	if streamer.canceled {
		t.Fatal("streamer should not be canceled on direct response")
	}
}

func TestProcessMessage_SkipsStreamingProviderWhenDisabled(t *testing.T) {
	provider := &streamingTestProvider{
		response: &providers.LLMResponse{
			Content: "No stream",
		},
	}
	streamer := &recordingStreamer{}
	al, cleanup := newStreamingTestLoop(t, false, provider, streamer)
	defer cleanup()

	response, err := al.processMessage(context.Background(), testInboundMessage(bus.InboundMessage{
		Channel:  "pico",
		ChatID:   "pico:test-session",
		SenderID: "user-1",
		Content:  "hello",
	}))
	if err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}

	if response != "No stream" {
		t.Fatalf("response = %q, want %q", response, "No stream")
	}
	if provider.chatStreamCalled {
		t.Fatal("expected ChatStream not to be used when streaming is disabled")
	}
	if !provider.chatCalled {
		t.Fatal("expected Chat() to be used when streaming is disabled")
	}
	if len(streamer.updates) != 0 || len(streamer.finalized) != 0 || streamer.canceled {
		t.Fatalf(
			"streamer should stay unused, got updates=%#v finalized=%#v canceled=%v",
			streamer.updates,
			streamer.finalized,
			streamer.canceled,
		)
	}
}

func TestProcessMessage_CancelsStreamingWhenToolCallsFollow(t *testing.T) {
	provider := &streamingTestProvider{
		chunks: []string{"Thinking"},
		response: &providers.LLMResponse{
			Content: "Thinking",
			ToolCalls: []providers.ToolCall{
				{
					ID:   "call-1",
					Name: "missing_tool",
					Arguments: map[string]any{
						"q": "stream",
					},
				},
			},
		},
	}
	streamer := &recordingStreamer{}
	al, cleanup := newStreamingTestLoop(t, true, provider, streamer)
	defer cleanup()

	response, err := al.processMessage(context.Background(), testInboundMessage(bus.InboundMessage{
		Channel:  "pico",
		ChatID:   "pico:test-session",
		SenderID: "user-1",
		Content:  "hello",
	}))
	if err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}

	if response == "" {
		t.Fatal("expected follow-up tool result content after tool-call path")
	}
	if !provider.chatStreamCalled {
		t.Fatal("expected ChatStream to be used")
	}
	if !streamer.canceled {
		t.Fatal("expected streamer to be canceled when tool calls follow streamed text")
	}
	if len(streamer.finalized) != 0 {
		t.Fatalf("finalized = %#v, want no finalize after tool calls", streamer.finalized)
	}
}
