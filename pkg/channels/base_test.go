package channels

import (
	"context"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
)

func TestBaseChannelIsAllowed(t *testing.T) {
	tests := []struct {
		name      string
		allowList []string
		senderID  string
		want      bool
	}{
		{
			name:      "empty allowlist allows all",
			allowList: nil,
			senderID:  "anyone",
			want:      true,
		},
		{
			name:      "compound sender matches numeric allowlist",
			allowList: []string{"123456"},
			senderID:  "123456|alice",
			want:      true,
		},
		{
			name:      "compound sender matches username allowlist",
			allowList: []string{"@alice"},
			senderID:  "123456|alice",
			want:      true,
		},
		{
			name:      "numeric sender matches legacy compound allowlist",
			allowList: []string{"123456|alice"},
			senderID:  "123456",
			want:      true,
		},
		{
			name:      "non matching sender is denied",
			allowList: []string{"123456"},
			senderID:  "654321|bob",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := NewBaseChannel("test", nil, nil, tt.allowList)
			if got := ch.IsAllowed(tt.senderID); got != tt.want {
				t.Fatalf("IsAllowed(%q) = %v, want %v", tt.senderID, got, tt.want)
			}
		})
	}
}

func TestBaseChannelHandleMessageAllowList(t *testing.T) {
	msgBus := bus.NewMessageBus()
	ch := NewBaseChannel("test", nil, msgBus, []string{"allowed"})

	ch.HandleMessage("blocked", "chat-1", "denied", nil, nil)

	deniedCtx, deniedCancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer deniedCancel()
	if msg, ok := msgBus.ConsumeInbound(deniedCtx); ok {
		t.Fatalf("expected denied sender to be dropped, got message: %+v", msg)
	}

	ch.HandleMessage("allowed", "chat-1", "accepted", []string{"m1"}, map[string]string{"k": "v"})

	allowedCtx, allowedCancel := context.WithTimeout(context.Background(), time.Second)
	defer allowedCancel()
	msg, ok := msgBus.ConsumeInbound(allowedCtx)
	if !ok {
		t.Fatal("expected allowed sender message to be published")
	}
	if msg.Channel != "test" || msg.SenderID != "allowed" || msg.ChatID != "chat-1" || msg.Content != "accepted" {
		t.Fatalf("unexpected inbound message: %+v", msg)
	}
	if len(msg.Media) != 1 || msg.Media[0] != "m1" {
		t.Fatalf("unexpected media payload: %+v", msg.Media)
	}
	if msg.Metadata["k"] != "v" {
		t.Fatalf("unexpected metadata: %+v", msg.Metadata)
	}
}
