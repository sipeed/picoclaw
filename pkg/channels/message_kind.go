package channels

import (
	"strings"

	"github.com/sipeed/picoclaw/pkg/bus"
)

func outboundMessageKind(msg bus.OutboundMessage) string {
	if len(msg.Context.Raw) == 0 {
		return ""
	}
	return strings.TrimSpace(msg.Context.Raw["message_kind"])
}

// OutboundMessageFinalizesTrackedToolFeedback reports whether a normal
// user-visible outbound message may safely reuse the tracked tool-feedback
// carrier by editing it in-place.
//
// Terminal replies that must preserve chronology, such as final replies after
// steering/follow-up input, are expected to bypass tracked tool-feedback
// finalization and be sent as new messages instead.
func OutboundMessageFinalizesTrackedToolFeedback(msg bus.OutboundMessage) bool {
	kind := strings.ToLower(outboundMessageKind(msg))
	switch kind {
	case "", "tool_feedback":
		return kind == ""
	case "thought", "tool_calls", "final_reply":
		return false
	default:
		return true
	}
}

// OutboundMessageDismissesTrackedToolFeedback reports whether the outgoing
// message is terminal user-facing content that should clear any previously
// tracked tool-feedback carrier after a fresh send.
func OutboundMessageDismissesTrackedToolFeedback(msg bus.OutboundMessage) bool {
	kind := strings.ToLower(outboundMessageKind(msg))
	switch kind {
	case "tool_feedback", "thought", "tool_calls":
		return false
	default:
		return true
	}
}
