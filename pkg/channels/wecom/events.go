package wecom

import (
	"github.com/sipeed/picoclaw/pkg/logger"
)

// EventType 事件类型
type EventType string

const (
	// EventConnected WebSocket 连接成功
	EventConnected EventType = "connected"
	// EventDisconnected WebSocket 断开连接
	EventDisconnected EventType = "disconnected"
	// EventAuthenticated 认证成功
	EventAuthenticated EventType = "authenticated"
	// EventError 发生错误
	EventError EventType = "error"
	// EventReconnecting 正在重连
	EventReconnecting EventType = "reconnecting"
	// EventMessageReceived 收到消息
	EventMessageReceived EventType = "message_received"
	// EventMessageSent 发送消息
	EventMessageSent EventType = "message_sent"
)

// Event 事件
type Event struct {
	Type    EventType
	Payload interface{}
}

// EventHandler 事件处理器
type EventHandler func(event Event)

// EventManager 事件管理器
type EventManager struct {
	handlers map[EventType][]EventHandler
}

// NewEventManager 创建新的事件管理器
func NewEventManager() *EventManager {
	return &EventManager{
		handlers: make(map[EventType][]EventHandler),
	}
}

// On 注册事件处理器
func (em *EventManager) On(eventType EventType, handler EventHandler) {
	em.handlers[eventType] = append(em.handlers[eventType], handler)
}

// Off 移除事件处理器（通过索引）
func (em *EventManager) Off(eventType EventType, index int) {
	handlers := em.handlers[eventType]
	if index >= 0 && index < len(handlers) {
		em.handlers[eventType] = append(handlers[:index], handlers[index+1:]...)
	}
}

// Emit 触发事件
func (em *EventManager) Emit(eventType EventType, payload interface{}) {
	event := Event{
		Type:    eventType,
		Payload: payload,
	}

	logger.DebugCF("wecom_ws", "Event emitted", map[string]any{
		"type": eventType,
	})

	for _, handler := range em.handlers[eventType] {
		go handler(event)
	}
}

// EventPayloadConnected 连接成功事件载荷
type EventPayloadConnected struct {
	URL string
}

// EventPayloadDisconnected 断开连接事件载荷
type EventPayloadDisconnected struct {
	URL   string
	Error error
}

// EventPayloadAuthenticated 认证成功事件载荷
type EventPayloadAuthenticated struct {
	BotID string
}

// EventPayloadError 错误事件载荷
type EventPayloadError struct {
	Error error
}

// EventPayloadMessageReceived 收到消息事件载荷
type EventPayloadMessageReceived struct {
	MsgType string
	ChatID  string
	From    string
}

// EventPayloadMessageSent 发送消息事件载荷
type EventPayloadMessageSent struct {
	MsgType string
	ChatID  string
}

// EventPayloadReconnecting 正在重连事件载荷
type EventPayloadReconnecting struct {
	URL       string
	Attempt   int
	BackoffMs int
}
