package qq

import (
	"context"
	"testing"
	"time"

	"github.com/tencent-connect/botgo/dto"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

func newTestQQChannel() (*QQChannel, *bus.MessageBus) {
	messageBus := bus.NewMessageBus()
	ch := &QQChannel{
		BaseChannel: channels.NewBaseChannel("qq", config.QQConfig{}, messageBus, nil),
		ctx:         context.Background(),
		dedup:       make(map[string]time.Time),
		done:        make(chan struct{}),
	}
	return ch, messageBus
}

func TestHandleC2CMessage_PublishesAccountIDMetadata(t *testing.T) {
	ch, messageBus := newTestQQChannel()

	handler := ch.handleC2CMessage()
	err := handler(nil, &dto.WSC2CMessageData{
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
		t.Fatal("expected inbound message to be published")
	}
	if inbound.Metadata["account_id"] != "7750283E123456" {
		t.Fatalf("account_id = %q, want %q", inbound.Metadata["account_id"], "7750283E123456")
	}
}
