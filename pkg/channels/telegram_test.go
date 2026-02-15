package channels

import (
	"testing"
)

func TestParseTelegramChatID(t *testing.T) {
	tests := []struct {
		name         string
		chatIDStr    string
		wantChatID   int64
		wantThreadID int
		wantErr      bool
	}{
		{
			name:         "chat only",
			chatIDStr:    "123456789",
			wantChatID:   123456789,
			wantThreadID: 0,
			wantErr:      false,
		},
		{
			name:         "negative chat ID (private chat)",
			chatIDStr:    "-987654321",
			wantChatID:   -987654321,
			wantThreadID: 0,
			wantErr:      false,
		},
		{
			name:         "chat with thread",
			chatIDStr:    "123456789:42",
			wantChatID:   123456789,
			wantThreadID: 42,
			wantErr:      false,
		},
		{
			name:         "negative chat with thread",
			chatIDStr:    "-987654321:100",
			wantChatID:   -987654321,
			wantThreadID: 100,
			wantErr:      false,
		},
		{
			name:         "invalid chat ID",
			chatIDStr:    "invalid",
			wantChatID:   0,
			wantThreadID: 0,
			wantErr:      true,
		},
		{
			name:         "invalid thread ID",
			chatIDStr:    "123456789:invalid",
			wantChatID:   123456789,
			wantThreadID: 0,
			wantErr:      true,
		},
		{
			name:         "empty string",
			chatIDStr:    "",
			wantChatID:   0,
			wantThreadID: 0,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chatID, threadID, err := parseTelegramChatID(tt.chatIDStr)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTelegramChatID(%q) error = %v, wantErr %v", tt.chatIDStr, err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				if chatID != tt.wantChatID {
					t.Errorf("parseTelegramChatID(%q) chatID = %d, want %d", tt.chatIDStr, chatID, tt.wantChatID)
				}
				if threadID != tt.wantThreadID {
					t.Errorf("parseTelegramChatID(%q) threadID = %d, want %d", tt.chatIDStr, threadID, tt.wantThreadID)
				}
			}
		})
	}
}

func TestParseChatID(t *testing.T) {
	tests := []struct {
		name      string
		chatIDStr string
		want      int64
		wantErr   bool
	}{
		{
			name:      "positive chat ID",
			chatIDStr: "123456789",
			want:      123456789,
			wantErr:   false,
		},
		{
			name:      "negative chat ID",
			chatIDStr: "-987654321",
			want:      -987654321,
			wantErr:   false,
		},
		{
			name:      "invalid chat ID",
			chatIDStr: "invalid",
			want:      0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseChatID(tt.chatIDStr)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("parseChatID(%q) error = %v, wantErr %v", tt.chatIDStr, err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseChatID(%q) = %d, want %d", tt.chatIDStr, got, tt.want)
			}
		})
	}
}
