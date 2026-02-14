package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
)

func TestSendFileTool_Execute_Success(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.png")
	if err := os.WriteFile(filePath, []byte("png-data"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	tool := NewSendFileTool(tmpDir, true, 1024*1024)
	tool.SetContext("telegram", "123")

	var sent bus.OutboundMessage
	tool.SetSendCallback(func(msg bus.OutboundMessage) error {
		sent = msg
		return nil
	})

	result := tool.Execute(context.Background(), map[string]interface{}{
		"path":    "test.png",
		"caption": "hello",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if !result.Silent {
		t.Fatalf("expected silent result")
	}
	if sent.Channel != "telegram" || sent.ChatID != "123" {
		t.Fatalf("unexpected target: %s:%s", sent.Channel, sent.ChatID)
	}
	if len(sent.Attachments) != 1 {
		t.Fatalf("expected one attachment, got %d", len(sent.Attachments))
	}
	if sent.Attachments[0].Type != "image" {
		t.Fatalf("expected image attachment, got %s", sent.Attachments[0].Type)
	}
}

func TestSendFileTool_Execute_NoTarget(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	tool := NewSendFileTool(tmpDir, true, 1024*1024)
	tool.SetSendCallback(func(msg bus.OutboundMessage) error { return nil })

	result := tool.Execute(context.Background(), map[string]interface{}{
		"path": filePath,
	})

	if !result.IsError {
		t.Fatalf("expected error when no channel/chat context")
	}
}

func TestSendFileTool_Execute_TooLarge(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "big.bin")
	if err := os.WriteFile(filePath, make([]byte, 64), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	tool := NewSendFileTool(tmpDir, true, 32)
	tool.SetContext("telegram", "123")
	tool.SetSendCallback(func(msg bus.OutboundMessage) error { return nil })

	result := tool.Execute(context.Background(), map[string]interface{}{
		"path": filePath,
	})
	if !result.IsError {
		t.Fatalf("expected too large error")
	}
}
