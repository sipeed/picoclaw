package qq

import (
	"context"
	"testing"
	"time"

	"github.com/tencent-connect/botgo/dto"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
)

func TestHandleC2CMessage_IncludesAccountIDMetadata(t *testing.T) {
	messageBus := bus.NewMessageBus()
	ch := &QQChannel{
		BaseChannel: channels.NewBaseChannel("qq", nil, messageBus, nil),
		dedup:       make(map[string]time.Time),
		done:        make(chan struct{}),
		ctx:         context.Background(),
	}

	err := ch.handleC2CMessage()(nil, &dto.WSC2CMessageData{
		ID:      "msg-1",
		Content: "hello",
		Author: &dto.User{
			ID: "7750283E123456",
		},
	})
	if err != nil {
		t.Fatalf("handleC2CMessage() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	inbound, ok := messageBus.ConsumeInbound(ctx)
	if !ok {
		t.Fatal("expected inbound message")
	}
	if inbound.Metadata["account_id"] != "7750283E123456" {
		t.Fatalf("account_id metadata = %q, want %q", inbound.Metadata["account_id"], "7750283E123456")
	}
}
