package wecom

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

// ---- Webhook mode tests ----

func TestNewWeComAIBotChannel_WebhookMode(t *testing.T) {
	t.Run("success with valid config", func(t *testing.T) {
		cfg := config.WeComAIBotConfig{
			Enabled:        true,
			Token:          "test_token",
			EncodingAESKey: "testkey1234567890123456789012345678901234567",
			WebhookPath:    "/webhook/test",
		}

		messageBus := bus.NewMessageBus()
		ch, err := NewWeComAIBotChannel(cfg, messageBus)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if ch == nil {
			t.Fatal("Expected channel to be created")
		}
		if ch.Name() != "wecom_aibot" {
			t.Errorf("Expected name 'wecom_aibot', got '%s'", ch.Name())
		}
		// Webhook mode must implement WebhookHandler.
		if _, ok := ch.(channels.WebhookHandler); !ok {
			t.Error("Webhook mode channel should implement WebhookHandler")
		}
	})

	t.Run("error with missing token", func(t *testing.T) {
		cfg := config.WeComAIBotConfig{
			Enabled:        true,
			EncodingAESKey: "testkey1234567890123456789012345678901234567",
		}
		messageBus := bus.NewMessageBus()
		_, err := NewWeComAIBotChannel(cfg, messageBus)
		if err == nil {
			t.Fatal("Expected error for missing token, got nil")
		}
	})

	t.Run("error with missing encoding key", func(t *testing.T) {
		cfg := config.WeComAIBotConfig{
			Enabled: true,
			Token:   "test_token",
		}
		messageBus := bus.NewMessageBus()
		_, err := NewWeComAIBotChannel(cfg, messageBus)
		if err == nil {
			t.Fatal("Expected error for missing encoding key, got nil")
		}
	})
}

func TestWeComAIBotWebhookChannelStartStop(t *testing.T) {
	cfg := config.WeComAIBotConfig{
		Enabled:        true,
		Token:          "test_token",
		EncodingAESKey: "testkey1234567890123456789012345678901234567",
	}

	messageBus := bus.NewMessageBus()
	ch, err := NewWeComAIBotChannel(cfg, messageBus)
	if err != nil {
		t.Fatalf("Failed to create channel: %v", err)
	}

	ctx := context.Background()

	if err := ch.Start(ctx); err != nil {
		t.Fatalf("Failed to start channel: %v", err)
	}
	if !ch.IsRunning() {
		t.Error("Expected channel to be running after Start")
	}

	if err := ch.Stop(ctx); err != nil {
		t.Fatalf("Failed to stop channel: %v", err)
	}
	if ch.IsRunning() {
		t.Error("Expected channel to be stopped after Stop")
	}
}

func TestWeComAIBotChannelWebhookPath(t *testing.T) {
	t.Run("default path", func(t *testing.T) {
		cfg := config.WeComAIBotConfig{
			Enabled:        true,
			Token:          "test_token",
			EncodingAESKey: "testkey1234567890123456789012345678901234567",
		}
		messageBus := bus.NewMessageBus()
		ch, _ := NewWeComAIBotChannel(cfg, messageBus)

		wh, ok := ch.(channels.WebhookHandler)
		if !ok {
			t.Fatal("Expected channel to implement WebhookHandler")
		}
		expectedPath := "/webhook/wecom-aibot"
		if wh.WebhookPath() != expectedPath {
			t.Errorf("Expected webhook path '%s', got '%s'", expectedPath, wh.WebhookPath())
		}
	})

	t.Run("custom path", func(t *testing.T) {
		customPath := "/custom/webhook"
		cfg := config.WeComAIBotConfig{
			Enabled:        true,
			Token:          "test_token",
			EncodingAESKey: "testkey1234567890123456789012345678901234567",
			WebhookPath:    customPath,
		}
		messageBus := bus.NewMessageBus()
		ch, _ := NewWeComAIBotChannel(cfg, messageBus)

		wh, ok := ch.(channels.WebhookHandler)
		if !ok {
			t.Fatal("Expected channel to implement WebhookHandler")
		}
		if wh.WebhookPath() != customPath {
			t.Errorf("Expected webhook path '%s', got '%s'", customPath, wh.WebhookPath())
		}
	})
}

func TestWeComAIBotChannelGetStreamResponseProcessingMessage(t *testing.T) {
	validAESKey := "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG"

	t.Run("uses default processing message", func(t *testing.T) {
		cfg := config.WeComAIBotConfig{
			Enabled:        true,
			Token:          "test_token",
			EncodingAESKey: validAESKey,
		}

		messageBus := bus.NewMessageBus()
		channel, err := NewWeComAIBotChannel(cfg, messageBus)
		if err != nil {
			t.Fatalf("Failed to create channel: %v", err)
		}
		ch, ok := channel.(*WeComAIBotChannel)
		if !ok {
			t.Fatal("Expected webhook mode channel")
		}

		task := &streamTask{
			StreamID: "stream-default",
			ChatID:   "chat-default",
			Deadline: time.Now().Add(-time.Second),
		}
		ch.streamTasks[task.StreamID] = task
		ch.chatTasks[task.ChatID] = []*streamTask{task}

		resp := decodeStreamResponse(t, ch, ch.getStreamResponse(task, "1234567890", "nonce"))

		if !resp.Stream.Finish {
			t.Fatal("Expected finished stream response after deadline")
		}
		if resp.Stream.Content != config.DefaultWeComAIBotProcessingMessage {
			t.Fatalf("Expected default processing message %q, got %q",
				config.DefaultWeComAIBotProcessingMessage, resp.Stream.Content)
		}
		if !task.StreamClosed {
			t.Fatal("Expected task stream to be marked closed")
		}
		if _, ok := ch.streamTasks[task.StreamID]; ok {
			t.Fatal("Expected closed stream task to be removed from streamTasks")
		}
		if len(ch.chatTasks[task.ChatID]) != 1 {
			t.Fatalf("Expected task to remain queued for response_url delivery, got %d entries",
				len(ch.chatTasks[task.ChatID]))
		}
	})

	t.Run("uses custom processing message", func(t *testing.T) {
		cfg := config.WeComAIBotConfig{
			Enabled:           true,
			Token:             "test_token",
			EncodingAESKey:    validAESKey,
			ProcessingMessage: "Please wait a moment. The result will be delivered in a follow-up message.",
		}

		messageBus := bus.NewMessageBus()
		channel, err := NewWeComAIBotChannel(cfg, messageBus)
		if err != nil {
			t.Fatalf("Failed to create channel: %v", err)
		}
		ch, ok := channel.(*WeComAIBotChannel)
		if !ok {
			t.Fatal("Expected webhook mode channel")
		}

		task := &streamTask{
			StreamID: "stream-custom",
			ChatID:   "chat-custom",
			Deadline: time.Now().Add(-time.Second),
		}

		resp := decodeStreamResponse(t, ch, ch.getStreamResponse(task, "1234567890", "nonce"))

		if resp.Stream.Content != cfg.ProcessingMessage {
			t.Fatalf("Expected custom processing message %q, got %q", cfg.ProcessingMessage, resp.Stream.Content)
		}
	})
}

func TestGenerateStreamID(t *testing.T) {
	cfg := config.WeComAIBotConfig{
		Enabled:        true,
		Token:          "test_token",
		EncodingAESKey: "testkey1234567890123456789012345678901234567",
	}
	messageBus := bus.NewMessageBus()
	ch, _ := NewWeComAIBotChannel(cfg, messageBus)
	webhookCh, ok := ch.(*WeComAIBotChannel)
	if !ok {
		t.Fatal("Expected webhook mode channel")
	}

	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := webhookCh.generateStreamID()
		if len(id) != 10 {
			t.Errorf("Expected stream ID length 10, got %d", len(id))
		}
		if ids[id] {
			t.Errorf("Duplicate stream ID generated: %s", id)
		}
		ids[id] = true
	}
}

func TestEncryptDecrypt(t *testing.T) {
	cfg := config.WeComAIBotConfig{
		Enabled:        true,
		Token:          "test_token",
		EncodingAESKey: "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG", // 43 characters
	}
	messageBus := bus.NewMessageBus()
	ch, _ := NewWeComAIBotChannel(cfg, messageBus)
	webhookCh, ok := ch.(*WeComAIBotChannel)
	if !ok {
		t.Fatal("Expected webhook mode channel")
	}

	plaintext := "Hello, World!"
	receiveid := ""

	encrypted, err := webhookCh.encryptMessage(plaintext, receiveid)
	if err != nil {
		t.Fatalf("Failed to encrypt message: %v", err)
	}
	if encrypted == "" {
		t.Fatal("Encrypted message is empty")
	}

	decrypted, err := decryptMessageWithVerify(encrypted, cfg.EncodingAESKey, receiveid)
	if err != nil {
		t.Fatalf("Failed to decrypt message: %v", err)
	}
	if decrypted != plaintext {
		t.Errorf("Expected decrypted message '%s', got '%s'", plaintext, decrypted)
	}
}

func TestGenerateSignature(t *testing.T) {
	token := "test_token"
	timestamp := "1234567890"
	nonce := "test_nonce"
	encrypt := "encrypted_msg"

	signature := computeSignature(token, timestamp, nonce, encrypt)
	if signature == "" {
		t.Error("Generated signature is empty")
	}
	if !verifySignature(token, signature, timestamp, nonce, encrypt) {
		t.Error("Generated signature does not verify correctly")
	}
}

func decodeStreamResponse(t *testing.T, ch *WeComAIBotChannel, encryptedResponse string) WeComAIBotStreamResponse {
	t.Helper()

	var wrapped WeComAIBotEncryptedResponse
	if err := json.Unmarshal([]byte(encryptedResponse), &wrapped); err != nil {
		t.Fatalf("Failed to unmarshal encrypted response: %v", err)
	}

	plaintext, err := decryptMessageWithVerify(wrapped.Encrypt, ch.config.EncodingAESKey, "")
	if err != nil {
		t.Fatalf("Failed to decrypt response: %v", err)
	}

	var resp WeComAIBotStreamResponse
	if err := json.Unmarshal([]byte(plaintext), &resp); err != nil {
		t.Fatalf("Failed to unmarshal decrypted response: %v", err)
	}

	return resp
}

// ---- WebSocket long-connection mode tests ----

func TestNewWeComAIBotChannel_WSMode(t *testing.T) {
	t.Run("success with bot_id and secret", func(t *testing.T) {
		cfg := config.WeComAIBotConfig{
			Enabled: true,
			BotID:   "test_bot_id",
			Secret:  "test_secret",
		}
		messageBus := bus.NewMessageBus()
		ch, err := NewWeComAIBotChannel(cfg, messageBus)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if ch == nil {
			t.Fatal("Expected channel to be created")
		}
		if ch.Name() != "wecom_aibot" {
			t.Errorf("Expected name 'wecom_aibot', got '%s'", ch.Name())
		}
		// WebSocket mode must NOT implement WebhookHandler.
		if _, ok := ch.(channels.WebhookHandler); ok {
			t.Error("WebSocket mode channel should NOT implement WebhookHandler")
		}
	})

	t.Run("ws mode takes priority over webhook fields", func(t *testing.T) {
		cfg := config.WeComAIBotConfig{
			Enabled:        true,
			BotID:          "test_bot_id",
			Secret:         "test_secret",
			Token:          "also_set",
			EncodingAESKey: "testkey1234567890123456789012345678901234567",
		}
		messageBus := bus.NewMessageBus()
		ch, err := NewWeComAIBotChannel(cfg, messageBus)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if _, ok := ch.(*WeComAIBotWSChannel); !ok {
			t.Error("Expected WebSocket mode channel when both BotID+Secret and Token+Key are set")
		}
	})

	t.Run("error with missing bot_id", func(t *testing.T) {
		cfg := config.WeComAIBotConfig{
			Enabled: true,
			Secret:  "test_secret",
		}
		messageBus := bus.NewMessageBus()
		_, err := NewWeComAIBotChannel(cfg, messageBus)
		// Missing bot_id alone means neither WS mode nor webhook mode is fully configured.
		if err == nil {
			t.Fatal("Expected error for missing bot_id, got nil")
		}
	})

	t.Run("error with missing secret", func(t *testing.T) {
		cfg := config.WeComAIBotConfig{
			Enabled: true,
			BotID:   "test_bot_id",
		}
		messageBus := bus.NewMessageBus()
		_, err := NewWeComAIBotChannel(cfg, messageBus)
		if err == nil {
			t.Fatal("Expected error for missing secret, got nil")
		}
	})
}

func TestWeComAIBotWSChannelStartStop(t *testing.T) {
	cfg := config.WeComAIBotConfig{
		Enabled: true,
		BotID:   "test_bot_id",
		Secret:  "test_secret",
	}
	messageBus := bus.NewMessageBus()
	ch, err := NewWeComAIBotChannel(cfg, messageBus)
	if err != nil {
		t.Fatalf("Failed to create channel: %v", err)
	}

	ctx := context.Background()

	// Start launches a background goroutine; it should not block or return an error.
	if err := ch.Start(ctx); err != nil {
		t.Fatalf("Failed to start channel: %v", err)
	}
	if !ch.IsRunning() {
		t.Error("Expected channel to be running after Start")
	}

	// Stop should work regardless of whether the WebSocket actually connected.
	if err := ch.Stop(ctx); err != nil {
		t.Fatalf("Failed to stop channel: %v", err)
	}
	if ch.IsRunning() {
		t.Error("Expected channel to be stopped after Stop")
	}
}

func TestGenerateRandomID(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 200; i++ {
		id := generateRandomID(10)
		if len(id) != 10 {
			t.Errorf("Expected ID length 10, got %d", len(id))
		}
		if ids[id] {
			t.Errorf("Duplicate ID generated: %s", id)
		}
		ids[id] = true
	}
}

func TestWSGenerateID(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 200; i++ {
		id := wsGenerateID()
		if len(id) != 10 {
			t.Errorf("Expected ID length 10, got %d", len(id))
		}
		if ids[id] {
			t.Errorf("Duplicate wsGenerateID result: %s", id)
		}
		ids[id] = true
	}
}

// ---- Webhook streaming fallback tests ----

// makeWebhookChannel creates a started WeComAIBotChannel for testing.
func makeWebhookChannel(t *testing.T) *WeComAIBotChannel {
	t.Helper()
	cfg := config.WeComAIBotConfig{
		Enabled:        true,
		Token:          "test_token",
		EncodingAESKey: "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG",
	}
	ch, err := NewWeComAIBotChannel(cfg, bus.NewMessageBus())
	if err != nil {
		t.Fatalf("create channel: %v", err)
	}
	wc := ch.(*WeComAIBotChannel)
	wc.ctx, wc.cancel = context.WithCancel(context.Background())
	return wc
}

// makeStreamTask creates and registers a streamTask for testing.
func makeStreamTask(t *testing.T, ch *WeComAIBotChannel, streamID, chatID string, deadline time.Time) *streamTask {
	t.Helper()
	task := &streamTask{
		StreamID: streamID,
		ChatID:   chatID,
		Deadline: deadline,
		answerCh: make(chan string, 1),
	}
	task.ctx, task.cancel = context.WithCancel(ch.ctx)
	ch.taskMu.Lock()
	ch.streamTasks[streamID] = task
	ch.chatTasks[chatID] = append(ch.chatTasks[chatID], task)
	ch.taskMu.Unlock()
	return task
}

// TestGetStreamResponse_ImmediateAnswer verifies that when the agent has already
// placed its answer in answerCh, getStreamResponse returns a finish=true response
// and fully removes the task.
func TestGetStreamResponse_ImmediateAnswer(t *testing.T) {
	ch := makeWebhookChannel(t)
	defer ch.cancel()

	task := makeStreamTask(t, ch, "stream-1", "chat-1", time.Now().Add(30*time.Second))
	task.answerCh <- "hello from agent"

	result := ch.getStreamResponse(task, "ts123", "nonce123")
	if result == "" {
		t.Fatal("expected non-empty encrypted response")
	}

	ch.taskMu.RLock()
	_, exists := ch.streamTasks["stream-1"]
	ch.taskMu.RUnlock()
	if exists {
		t.Error("task should have been removed from streamTasks after normal finish")
	}
	if !task.Finished {
		t.Error("task.Finished should be true after normal finish")
	}
}

// TestGetStreamResponse_DeadlinePassed verifies that when the stream deadline has
// elapsed (no agent reply yet), getStreamResponse closes the stream but keeps the
// task alive so the response_url fallback can still deliver the answer.
func TestGetStreamResponse_DeadlinePassed(t *testing.T) {
	ch := makeWebhookChannel(t)
	defer ch.cancel()

	task := makeStreamTask(t, ch, "stream-2", "chat-2", time.Now().Add(-time.Millisecond))

	result := ch.getStreamResponse(task, "ts456", "nonce456")
	if result == "" {
		t.Fatal("expected non-empty encrypted response")
	}

	ch.taskMu.RLock()
	_, stillStreaming := ch.streamTasks["stream-2"]
	ch.taskMu.RUnlock()
	if stillStreaming {
		t.Error("task should have been removed from streamTasks after deadline")
	}
	if !task.StreamClosed {
		t.Error("task.StreamClosed should be true after deadline")
	}
	if task.Finished {
		t.Error("task.Finished must remain false: agent reply still expected via response_url")
	}
}

// TestGetStreamResponse_StillPending verifies that when neither the agent has
// replied nor the deadline has passed, getStreamResponse returns without altering
// task state (client should poll again).
func TestGetStreamResponse_StillPending(t *testing.T) {
	ch := makeWebhookChannel(t)
	defer ch.cancel()

	task := makeStreamTask(t, ch, "stream-3", "chat-3", time.Now().Add(30*time.Second))

	result := ch.getStreamResponse(task, "ts789", "nonce789")
	if result == "" {
		t.Fatal("expected non-empty encrypted response")
	}

	ch.taskMu.RLock()
	_, exists := ch.streamTasks["stream-3"]
	ch.taskMu.RUnlock()
	if !exists {
		t.Error("pending task should still be in streamTasks")
	}
	if task.Finished || task.StreamClosed {
		t.Error("pending task should not be finished or stream-closed")
	}
	// Cleanup.
	ch.removeTask(task)
}
