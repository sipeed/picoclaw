package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
)

func TestScreenshotTool_Execute_URL_Send(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewScreenshotTool(tmpDir, true, 1024*1024)
	tool.SetContext("telegram", "999")

	tool.lookPath = func(s string) (string, error) { return "/usr/bin/" + s, nil }
	tool.runCommand = func(ctx context.Context, command string, args ...string) error {
		var output string
		for _, arg := range args {
			if len(arg) > len("--screenshot=") && arg[:13] == "--screenshot=" {
				output = arg[13:]
			}
		}
		if output == "" {
			t.Fatalf("missing screenshot output arg")
		}
		return os.WriteFile(output, []byte("fake-image"), 0644)
	}

	var sent bus.OutboundMessage
	tool.SetSendCallback(func(msg bus.OutboundMessage) error {
		sent = msg
		return nil
	})

	result := tool.Execute(context.Background(), map[string]interface{}{
		"url":     "https://example.com",
		"caption": "cap",
	})
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if !result.Silent {
		t.Fatalf("expected silent result")
	}
	if len(sent.Attachments) != 1 {
		t.Fatalf("expected one attachment, got %d", len(sent.Attachments))
	}
	if sent.Attachments[0].Type != "image" {
		t.Fatalf("expected image attachment")
	}
}

func TestScreenshotTool_Execute_NoSendReturnsPath(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewScreenshotTool(tmpDir, true, 1024*1024)

	tool.lookPath = func(s string) (string, error) { return "/usr/bin/" + s, nil }
	tool.runCommand = func(ctx context.Context, command string, args ...string) error {
		output := filepath.Join(tmpDir, "tmp", "screenshots", "out.png")
		for _, arg := range args {
			if len(arg) > len("--screenshot=") && arg[:13] == "--screenshot=" {
				output = arg[13:]
			}
		}
		return os.WriteFile(output, []byte("fake-image"), 0644)
	}
	tool.now = func() time.Time { return time.Date(2026, 2, 14, 10, 0, 0, 0, time.UTC) }

	result := tool.Execute(context.Background(), map[string]interface{}{
		"url":  "https://example.com",
		"send": false,
	})
	if result.IsError {
		t.Fatalf("expected success, got %s", result.ForLLM)
	}
	if result.ForUser == "" {
		t.Fatalf("expected user-visible path")
	}
}

func TestScreenshotTool_Execute_NoTargetWhenSend(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewScreenshotTool(tmpDir, true, 1024*1024)
	tool.lookPath = func(s string) (string, error) { return "/usr/bin/" + s, nil }
	tool.runCommand = func(ctx context.Context, command string, args ...string) error {
		for _, arg := range args {
			if len(arg) > len("--screenshot=") && arg[:13] == "--screenshot=" {
				return os.WriteFile(arg[13:], []byte("fake-image"), 0644)
			}
		}
		return nil
	}
	tool.SetSendCallback(func(msg bus.OutboundMessage) error { return nil })

	result := tool.Execute(context.Background(), map[string]interface{}{
		"url": "https://example.com",
	})
	if !result.IsError {
		t.Fatalf("expected error without target context")
	}
}
