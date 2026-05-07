package agent

import (
	"strings"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/routing"
	"github.com/sipeed/picoclaw/pkg/session"
)

// DispatchRequest is the normalized runtime input passed into the agent loop
// after routing and session allocation have completed.
type DispatchRequest struct {
	SessionKey     string
	SessionAliases []string
	InboundContext *bus.InboundContext
	RouteResult    *routing.ResolvedRoute
	SessionScope   *session.SessionScope
	UserMessage    string
	Media          []string
}

func (r DispatchRequest) Channel() string {
	if r.InboundContext == nil {
		return ""
	}
	return r.InboundContext.Channel
}

func (r DispatchRequest) ChatID() string {
	if r.InboundContext == nil {
		return ""
	}
	return r.InboundContext.ChatID
}

func (r DispatchRequest) MessageID() string {
	if r.InboundContext == nil {
		return ""
	}
	return r.InboundContext.MessageID
}

func (r DispatchRequest) ReplyToMessageID() string {
	if r.InboundContext == nil {
		return ""
	}
	return r.InboundContext.ReplyToMessageID
}

func (r DispatchRequest) SenderID() string {
	if r.InboundContext == nil {
		return ""
	}
	return r.InboundContext.SenderID
}

func normalizeProcessOptionsInPlace(opts *processOptions) {
	if opts == nil {
		return
	}
	*opts = normalizeProcessOptions(*opts)
}

func normalizeProcessOptions(opts processOptions) processOptions {
	if opts.Dispatch.SessionKey == "" {
		opts.Dispatch.SessionKey = strings.TrimSpace(opts.SessionKey)
	}
	if len(opts.Dispatch.SessionAliases) == 0 && len(opts.SessionAliases) > 0 {
		opts.Dispatch.SessionAliases = append([]string(nil), opts.SessionAliases...)
	}
	if opts.Dispatch.UserMessage == "" {
		opts.Dispatch.UserMessage = opts.UserMessage
	}
	if len(opts.Dispatch.Media) == 0 && len(opts.Media) > 0 {
		opts.Dispatch.Media = append([]string(nil), opts.Media...)
	}
	if opts.Dispatch.RouteResult == nil {
		opts.Dispatch.RouteResult = cloneResolvedRoute(opts.RouteResult)
	}
	if opts.Dispatch.SessionScope == nil {
		opts.Dispatch.SessionScope = session.CloneScope(opts.SessionScope)
	}
	if opts.Dispatch.InboundContext == nil {
		if opts.InboundContext != nil {
			opts.Dispatch.InboundContext = cloneInboundContext(opts.InboundContext)
		} else if opts.Channel != "" || opts.ChatID != "" || opts.SenderID != "" ||
			opts.MessageID != "" || opts.ReplyToMessageID != "" {
			inbound := bus.InboundContext{
				Channel:          strings.TrimSpace(opts.Channel),
				ChatID:           strings.TrimSpace(opts.ChatID),
				SenderID:         strings.TrimSpace(opts.SenderID),
				MessageID:        strings.TrimSpace(opts.MessageID),
				ReplyToMessageID: strings.TrimSpace(opts.ReplyToMessageID),
			}
			inbound.ChatType = inferChatTypeFromSessionScope(opts.Dispatch.SessionScope)
			if inbound.Channel != "" || inbound.ChatID != "" || inbound.SenderID != "" ||
				inbound.MessageID != "" || inbound.ReplyToMessageID != "" {
				inbound = bus.NormalizeInboundMessage(bus.InboundMessage{Context: inbound}).Context
				opts.Dispatch.InboundContext = &inbound
			}
		}
	}

	// Keep legacy mirrors populated while the rest of the runtime migrates.
	opts.SessionKey = opts.Dispatch.SessionKey
	opts.SessionAliases = append([]string(nil), opts.Dispatch.SessionAliases...)
	opts.UserMessage = opts.Dispatch.UserMessage
	opts.Media = append([]string(nil), opts.Dispatch.Media...)
	opts.InboundContext = cloneInboundContext(opts.Dispatch.InboundContext)
	opts.RouteResult = cloneResolvedRoute(opts.Dispatch.RouteResult)
	opts.SessionScope = session.CloneScope(opts.Dispatch.SessionScope)
	if opts.InboundContext != nil {
		if opts.Channel == "" {
			opts.Channel = opts.InboundContext.Channel
		}
		if opts.ChatID == "" {
			opts.ChatID = opts.InboundContext.ChatID
		}
		if opts.MessageID == "" {
			opts.MessageID = opts.InboundContext.MessageID
		}
		if opts.ReplyToMessageID == "" {
			opts.ReplyToMessageID = opts.InboundContext.ReplyToMessageID
		}
		if opts.SenderID == "" {
			opts.SenderID = opts.InboundContext.SenderID
		}
	}

	return opts
}

func inferChatTypeFromSessionScope(scope *session.SessionScope) string {
	if scope == nil || len(scope.Values) == 0 {
		return ""
	}
	chatValue := strings.TrimSpace(scope.Values["chat"])
	if chatValue == "" {
		return ""
	}
	chatType, _, ok := strings.Cut(chatValue, ":")
	if !ok {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(chatType))
}
