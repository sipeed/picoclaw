package mattermost

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestNewMattermostChannel(t *testing.T) {
	msgBus := bus.NewMessageBus()

	t.Run("missing url", func(t *testing.T) {
		cfg := config.MattermostConfig{
			URL:      "",
			BotToken: *config.NewSecureString("token"),
		}
		_, err := NewMattermostChannel(cfg, msgBus)
		if err == nil {
			t.Fatal("expected error for missing url, got nil")
		}
	})

	t.Run("missing bot token", func(t *testing.T) {
		cfg := config.MattermostConfig{
			URL: "https://mattermost.example.com",
		}
		_, err := NewMattermostChannel(cfg, msgBus)
		if err == nil {
			t.Fatal("expected error for missing bot token, got nil")
		}
	})

	t.Run("valid config", func(t *testing.T) {
		cfg := config.MattermostConfig{
			URL:      "https://mattermost.example.com",
			BotToken: *config.NewSecureString("token"),
		}
		ch, err := NewMattermostChannel(cfg, msgBus)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ch.Name() != "mattermost" {
			t.Fatalf("Name() = %q, want %q", ch.Name(), "mattermost")
		}
		if ch.IsRunning() {
			t.Fatal("new channel should not be running")
		}
		if ch.typingStop == nil {
			t.Fatal("typingStop map should be initialized")
		}
	})
}

func TestBuildWSURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "https without trailing slash",
			in:   "https://mm.example.com",
			want: "wss://mm.example.com/api/v4/websocket",
		},
		{
			name: "https with trailing slash",
			in:   "https://mm.example.com/",
			want: "wss://mm.example.com/api/v4/websocket",
		},
		{
			name: "http",
			in:   "http://localhost:8065",
			want: "ws://localhost:8065/api/v4/websocket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildWSURL(tt.in)
			if got != tt.want {
				t.Fatalf("buildWSURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestStripBotMention(t *testing.T) {
	ch := &MattermostChannel{
		botUsername: "PicoclawBot",
	}

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "mention at start",
			in:   "@PicoclawBot hello",
			want: "hello",
		},
		{
			name: "case insensitive mention",
			in:   "@picoclawbot hello",
			want: "hello",
		},
		{
			name: "multiple mentions",
			in:   "hi @PicoclawBot and @PICOCLAWBOT",
			want: "hi  and",
		},
		{
			name: "no mention",
			in:   "hello world",
			want: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ch.stripBotMention(tt.in)
			if got != tt.want {
				t.Fatalf("stripBotMention(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestCleanupTypingLoop(t *testing.T) {
	ch := &MattermostChannel{
		typingStop: make(map[string]chan struct{}),
	}

	stop := make(chan struct{})
	ch.typingStop["chat-1"] = stop
	ch.cleanupTypingLoop("chat-1", stop)
	if _, ok := ch.typingStop["chat-1"]; ok {
		t.Fatal("expected cleanupTypingLoop to remove current typing entry")
	}

	oldStop := make(chan struct{})
	newStop := make(chan struct{})
	ch.typingStop["chat-1"] = newStop
	ch.cleanupTypingLoop("chat-1", oldStop)
	if got := ch.typingStop["chat-1"]; got != newStop {
		t.Fatal("cleanupTypingLoop should not remove newer typing entry")
	}
}

func TestUploadFile(t *testing.T) {
	const (
		wantToken   = "test-token"
		wantChatID  = "channel-1"
		wantFileID  = "file-123"
		wantContent = "mattermost upload stream test"
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want %s", r.Method, http.MethodPost)
		}
		if r.URL.Path != "/api/v4/files" {
			t.Fatalf("path = %s, want /api/v4/files", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+wantToken {
			t.Fatalf("Authorization = %q, want %q", got, "Bearer "+wantToken)
		}
		if got := r.Header.Get("Content-Type"); !strings.HasPrefix(got, "multipart/form-data; boundary=") {
			t.Fatalf("Content-Type = %q, want multipart/form-data", got)
		}

		mr, err := r.MultipartReader()
		if err != nil {
			t.Fatalf("MultipartReader() error: %v", err)
		}

		var gotChatID string
		var gotFilename string
		var gotContent string
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("NextPart() error: %v", err)
			}
			data, err := io.ReadAll(part)
			if err != nil {
				t.Fatalf("ReadAll(part) error: %v", err)
			}

			switch part.FormName() {
			case "channel_id":
				gotChatID = string(data)
			case "files":
				gotFilename = part.FileName()
				gotContent = string(data)
			}
		}

		if gotChatID != wantChatID {
			t.Fatalf("channel_id = %q, want %q", gotChatID, wantChatID)
		}
		if gotFilename != "upload.txt" {
			t.Fatalf("filename = %q, want %q", gotFilename, "upload.txt")
		}
		if gotContent != wantContent {
			t.Fatalf("file content = %q, want %q", gotContent, wantContent)
		}

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"file_infos":[{"id":"` + wantFileID + `"}]}`))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "upload.txt")
	if err := os.WriteFile(filePath, []byte(wantContent), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	cfg := config.MattermostConfig{
		URL:      server.URL,
		BotToken: *config.NewSecureString(wantToken),
	}
	ch, err := NewMattermostChannel(cfg, bus.NewMessageBus())
	if err != nil {
		t.Fatalf("NewMattermostChannel() error: %v", err)
	}
	ch.ctx = context.Background()

	gotFileID, err := ch.uploadFile(wantChatID, filePath, "")
	if err != nil {
		t.Fatalf("uploadFile() error: %v", err)
	}
	if gotFileID != wantFileID {
		t.Fatalf("uploadFile() fileID = %q, want %q", gotFileID, wantFileID)
	}
}
