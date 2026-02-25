package channels

import (
	"testing"

	"github.com/tencent-connect/botgo/dto"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestQQChannel_RecordInboundAndBuildMessage(t *testing.T) {
	channel, err := NewQQChannel(config.QQConfig{}, nil)
	if err != nil {
		t.Fatalf("failed to create qq channel: %v", err)
	}

	channel.recordInboundMessage("group-1", "msg-a", qqChatKindGroup)

	if got := channel.resolveChatKind("group-1"); got != qqChatKindGroup {
		t.Fatalf("expected chat kind %q, got %q", qqChatKindGroup, got)
	}

	first := channel.buildC2CMessage("group-1", "hello")
	if first.MsgType != int(dto.TextMsg) {
		t.Fatalf("expected msg_type %d, got %d", int(dto.TextMsg), first.MsgType)
	}
	if first.MsgID != "msg-a" {
		t.Fatalf("expected msg_id msg-a, got %q", first.MsgID)
	}
	if first.MsgSeq != 1 {
		t.Fatalf("expected first msg_seq 1, got %d", first.MsgSeq)
	}

	second := channel.buildC2CMessage("group-1", "hello again")
	if second.MsgSeq != 2 {
		t.Fatalf("expected second msg_seq 2, got %d", second.MsgSeq)
	}

	channel.recordInboundMessage("group-1", "msg-b", qqChatKindGroup)
	third := channel.buildC2CMessage("group-1", "after new inbound")
	if third.MsgID != "msg-b" {
		t.Fatalf("expected latest msg_id msg-b, got %q", third.MsgID)
	}
	if third.MsgSeq != 1 {
		t.Fatalf("expected msg_seq reset to 1 after new inbound, got %d", third.MsgSeq)
	}
}

func TestQQChannel_DefaultChatKindAndMessageWithoutContext(t *testing.T) {
	channel, err := NewQQChannel(config.QQConfig{}, nil)
	if err != nil {
		t.Fatalf("failed to create qq channel: %v", err)
	}

	if got := channel.resolveChatKind("unknown-chat"); got != qqChatKindDirect {
		t.Fatalf("expected default chat kind %q, got %q", qqChatKindDirect, got)
	}

	msg := channel.buildC2CMessage("unknown-chat", "plain")
	if msg.MsgType != int(dto.TextMsg) {
		t.Fatalf("expected msg_type %d, got %d", int(dto.TextMsg), msg.MsgType)
	}
	if msg.MsgID != "" {
		t.Fatalf("expected empty msg_id for chat without context, got %q", msg.MsgID)
	}
	if msg.MsgSeq != 0 {
		t.Fatalf("expected msg_seq 0 for chat without context, got %d", msg.MsgSeq)
	}
}
