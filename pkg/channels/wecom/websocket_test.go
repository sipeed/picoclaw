package wecom

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWeComWSChannel(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.WeComWSConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "success with valid config",
			cfg: config.WeComWSConfig{
				Enabled: true,
				BotID:   "test_bot_id",
				Secret:  "test_secret",
			},
			wantErr: false,
		},
		{
			name: "error with missing bot_id",
			cfg: config.WeComWSConfig{
				Enabled: true,
				Secret:  "test_secret",
			},
			wantErr: true,
			errMsg:  "bot_id and secret are required",
		},
		{
			name: "error with missing secret",
			cfg: config.WeComWSConfig{
				Enabled: true,
				BotID:   "test_bot_id",
			},
			wantErr: true,
			errMsg:  "bot_id and secret are required",
		},
		{
			name: "success with default values",
			cfg: config.WeComWSConfig{
				Enabled: true,
				BotID:   "test_bot_id",
				Secret:  "test_secret",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messageBus := bus.NewMessageBus()
			ch, err := NewWeComWSChannel(tt.cfg, messageBus)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, ch)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, ch)
				assert.Equal(t, "wecom_ws", ch.Name())
				assert.Equal(t, tt.cfg.BotID, ch.config.BotID)
				assert.Equal(t, tt.cfg.Secret, ch.config.Secret)
				// 验证默认值被正确设置
				if tt.cfg.WSURL == "" {
					assert.Equal(t, defaultWSURL, ch.config.WSURL)
				}
			}
		})
	}
}

func TestWeComWSChannelStartStop(t *testing.T) {
	cfg := config.WeComWSConfig{
		Enabled: true,
		BotID:   "test_bot_id",
		Secret:  "test_secret",
	}
	messageBus := bus.NewMessageBus()
	ch, err := NewWeComWSChannel(cfg, messageBus)
	require.NoError(t, err)

	ctx := context.Background()

	// 测试 Start
	err = ch.Start(ctx)
	require.NoError(t, err)
	assert.True(t, ch.IsRunning())

	// 等待一段时间让 goroutine 启动
	time.Sleep(100 * time.Millisecond)

	// 测试 Stop
	err = ch.Stop(ctx)
	require.NoError(t, err)
	assert.False(t, ch.IsRunning())
}

func TestWeComWSChannelName(t *testing.T) {
	cfg := config.WeComWSConfig{
		Enabled: true,
		BotID:   "test_bot_id",
		Secret:  "test_secret",
	}
	messageBus := bus.NewMessageBus()
	ch, err := NewWeComWSChannel(cfg, messageBus)
	require.NoError(t, err)

	assert.Equal(t, "wecom_ws", ch.Name())
}

func TestWeComWSChannelIsAllowed(t *testing.T) {
	tests := []struct {
		name      string
		allowFrom []string
		senderID  string
		want      bool
	}{
		{
			name:      "empty allowlist allows all",
			allowFrom: []string{},
			senderID:  "any_user",
			want:      true,
		},
		{
			name:      "allowlist restricts users",
			allowFrom: []string{"allowed_user"},
			senderID:  "allowed_user",
			want:      true,
		},
		{
			name:      "not in allowlist",
			allowFrom: []string{"allowed_user"},
			senderID:  "other_user",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.WeComWSConfig{
				Enabled:   true,
				BotID:     "test_bot_id",
				Secret:    "test_secret",
				AllowFrom: tt.allowFrom,
			}
			messageBus := bus.NewMessageBus()
			ch, err := NewWeComWSChannel(cfg, messageBus)
			require.NoError(t, err)

			got := ch.IsAllowed(tt.senderID)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWeComWSChannelReasoningChannelID(t *testing.T) {
	cfg := config.WeComWSConfig{
		Enabled:            true,
		BotID:              "test_bot_id",
		Secret:             "test_secret",
		ReasoningChannelID: "reasoning_channel_123",
	}
	messageBus := bus.NewMessageBus()
	ch, err := NewWeComWSChannel(cfg, messageBus)
	require.NoError(t, err)

	assert.Equal(t, "reasoning_channel_123", ch.ReasoningChannelID())
}

func TestShouldRespondInGroup(t *testing.T) {
	tests := []struct {
		name         string
		groupTrigger config.GroupTriggerConfig
		isMentioned  bool
		content      string
		wantRespond  bool
		wantContent  string
	}{
		{
			name:        "mentioned always responds",
			isMentioned: true,
			content:     "@bot hello",
			wantRespond: true,
			wantContent: "@bot hello", // BaseChannel.ShouldRespondInGroup 不会去除 mention
		},
		{
			name: "mention only without mention",
			groupTrigger: config.GroupTriggerConfig{
				MentionOnly: true,
			},
			isMentioned: false,
			content:     "hello",
			wantRespond: false,
			wantContent: "hello", // 当不响应时，返回原始内容
		},
		{
			name: "prefix match",
			groupTrigger: config.GroupTriggerConfig{
				Prefixes: []string{"/bot", "@bot"},
			},
			isMentioned: false,
			content:     "/bot hello",
			wantRespond: true,
			wantContent: "hello",
		},
		{
			name: "prefix no match",
			groupTrigger: config.GroupTriggerConfig{
				Prefixes: []string{"/bot"},
			},
			isMentioned: false,
			content:     "hello",
			wantRespond: false,
			wantContent: "hello", // 当不响应时，返回原始内容
		},
		{
			name:        "no group trigger config",
			isMentioned: false,
			content:     "hello",
			wantRespond: true,
			wantContent: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.WeComWSConfig{
				Enabled:      true,
				BotID:        "test_bot_id",
				Secret:       "test_secret",
				GroupTrigger: tt.groupTrigger,
			}
			messageBus := bus.NewMessageBus()
			ch, err := NewWeComWSChannel(cfg, messageBus)
			require.NoError(t, err)

			respond, content := ch.ShouldRespondInGroup(tt.isMentioned, tt.content)
			assert.Equal(t, tt.wantRespond, respond)
			assert.Equal(t, tt.wantContent, content)
		})
	}
}

func TestWeComWSMessageStructure(t *testing.T) {
	tests := []struct {
		name string
		msg  WeComWSMessage
	}{
		{
			name: "subscribe message",
			msg: WeComWSMessage{
				Cmd: string(CmdSubscribe),
				Headers: MessageHeaders{
					ReqID: "test_req_id",
				},
				Body: mustMarshal(t, SubscribeBody{
					Secret: "test_secret",
					BotID:  "test_bot_id",
				}),
			},
		},
		{
			name: "ping message",
			msg: WeComWSMessage{
				Cmd: string(CmdPing),
				Headers: MessageHeaders{
					ReqID: "test_req_id",
				},
				Body: json.RawMessage("{}"),
			},
		},
		{
			name: "response message",
			msg: WeComWSMessage{
				Cmd: string(CmdAIBotResponse),
				Headers: MessageHeaders{
					ReqID: "test_req_id",
				},
				Body: mustMarshal(t, ResponseMessage{
					MsgType: "stream",
					Stream: &StreamContent{
						ID:      "test_stream_id",
						Finish:  true,
						Content: "Hello",
					},
				}),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 验证可以正确序列化和反序列化
			data, err := json.Marshal(tt.msg)
			require.NoError(t, err)

			var decoded WeComWSMessage
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			assert.Equal(t, tt.msg.Cmd, decoded.Cmd)
			assert.Equal(t, tt.msg.Headers.ReqID, decoded.Headers.ReqID)
		})
	}
}

func TestCallbackMessageStructure(t *testing.T) {
	tests := []struct {
		name string
		msg  CallbackMessage
	}{
		{
			name: "text message",
			msg: CallbackMessage{
				MsgID:       "msg_123",
				AIBotID:     "bot_456",
				ChatID:      "chat_789",
				ChatType:    "single",
				From:        From{UserID: "user_001"},
				ResponseURL: "https://example.com/response",
				MsgType:     "text",
				Text:        &Text{Content: "Hello"},
			},
		},
		{
			name: "image message",
			msg: CallbackMessage{
				MsgID:       "msg_123",
				AIBotID:     "bot_456",
				ChatID:      "chat_789",
				ChatType:    "group",
				From:        From{UserID: "user_001"},
				ResponseURL: "https://example.com/response",
				MsgType:     "image",
				Image: &Image{
					URL: "https://example.com/image.jpg",
					MD5: "abc123",
				},
			},
		},
		{
			name: "mixed message",
			msg: CallbackMessage{
				MsgID:       "msg_123",
				AIBotID:     "bot_456",
				ChatID:      "chat_789",
				ChatType:    "group",
				From:        From{UserID: "user_001"},
				ResponseURL: "https://example.com/response",
				MsgType:     "mixed",
				Mixed: &Mixed{
					MsgItem: []MixedItem{
						{MsgType: "text", Text: &Text{Content: "Hello"}},
						{MsgType: "image", Image: &Image{URL: "https://example.com/image.jpg"}},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 验证可以正确序列化和反序列化
			data, err := json.Marshal(tt.msg)
			require.NoError(t, err)

			var decoded CallbackMessage
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			assert.Equal(t, tt.msg.MsgID, decoded.MsgID)
			assert.Equal(t, tt.msg.MsgType, decoded.MsgType)
		})
	}
}

func TestWeComWSConfigDefaults(t *testing.T) {
	cfg := config.WeComWSConfig{
		Enabled: true,
		BotID:   "test_bot_id",
		Secret:  "test_secret",
		// 其他字段使用零值
	}
	messageBus := bus.NewMessageBus()
	ch, err := NewWeComWSChannel(cfg, messageBus)
	require.NoError(t, err)

	// 验证默认值
	assert.Equal(t, defaultWSURL, ch.config.WSURL)
	assert.Equal(t, int(defaultReconnectInterval.Seconds()), ch.config.ReconnectInterval)
	assert.Equal(t, int(defaultHeartbeatInterval.Seconds()), ch.config.HeartbeatInterval)
	assert.Equal(t, int(defaultReplyTimeout.Seconds()), ch.config.ReplyTimeout)
	assert.Equal(t, defaultMaxReconnectAttempts, ch.config.MaxReconnectAttempts)
}

// Helper function
func mustMarshal(t *testing.T, v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	require.NoError(t, err)
	return data
}
