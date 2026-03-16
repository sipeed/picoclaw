package onebot

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"

	"jane/pkg/channels"
	"jane/pkg/config"
)

type OneBotChannel struct {
	*channels.BaseChannel
	config        config.OneBotConfig
	conn          *websocket.Conn
	ctx           context.Context
	cancel        context.CancelFunc
	dedup         map[string]struct{}
	dedupRing     []string
	dedupIdx      int
	mu            sync.Mutex
	writeMu       sync.Mutex
	echoCounter   int64
	selfID        int64
	pending       map[string]chan json.RawMessage
	pendingMu     sync.Mutex
	lastMessageID sync.Map
}

type oneBotRawEvent struct {
	PostType      string          `json:"post_type"`
	MessageType   string          `json:"message_type"`
	SubType       string          `json:"sub_type"`
	MessageID     json.RawMessage `json:"message_id"`
	UserID        json.RawMessage `json:"user_id"`
	GroupID       json.RawMessage `json:"group_id"`
	RawMessage    string          `json:"raw_message"`
	Message       json.RawMessage `json:"message"`
	Sender        json.RawMessage `json:"sender"`
	SelfID        json.RawMessage `json:"self_id"`
	Time          json.RawMessage `json:"time"`
	MetaEventType string          `json:"meta_event_type"`
	NoticeType    string          `json:"notice_type"`
	Echo          string          `json:"echo"`
	RetCode       json.RawMessage `json:"retcode"`
	Status        json.RawMessage `json:"status"`
	Data          json.RawMessage `json:"data"`
}

type BotStatus struct {
	Online bool `json:"online"`
	Good   bool `json:"good"`
}

type oneBotSender struct {
	UserID   json.RawMessage `json:"user_id"`
	Nickname string          `json:"nickname"`
	Card     string          `json:"card"`
}

type oneBotAPIRequest struct {
	Action string `json:"action"`
	Params any    `json:"params"`
	Echo   string `json:"echo,omitempty"`
}

type oneBotMessageSegment struct {
	Type string         `json:"type"`
	Data map[string]any `json:"data"`
}

type parseMessageResult struct {
	Text           string
	IsBotMentioned bool
	Media          []string
	ReplyTo        string
}
