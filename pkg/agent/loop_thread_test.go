package agent

import "testing"

func TestWithTelegramThread(t *testing.T) {
	al := &AgentLoop{}

	tests := []struct {
		name     string
		channel  string
		chatID   string
		threadID int
		want     string
	}{
		{name: "non telegram unchanged", channel: "discord", chatID: "123", threadID: 7, want: "123"},
		{name: "zero thread unchanged", channel: "telegram", chatID: "123", threadID: 0, want: "123"},
		{name: "negative thread unchanged", channel: "telegram", chatID: "123", threadID: -1, want: "123"},
		{name: "append thread", channel: "telegram", chatID: "123", threadID: 7, want: "123/7"},
		{name: "replace thread", channel: "telegram", chatID: "123/5", threadID: 7, want: "123/7"},
		{name: "group id", channel: "telegram", chatID: "-100123", threadID: 42, want: "-100123/42"},
		{name: "empty chat unchanged", channel: "telegram", chatID: "", threadID: 9, want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := al.withTelegramThread(tc.channel, tc.chatID, tc.threadID)
			if got != tc.want {
				t.Fatalf(
					"withTelegramThread(%q, %q, %d) = %q, want %q",
					tc.channel,
					tc.chatID,
					tc.threadID,
					got,
					tc.want,
				)
			}
		})
	}
}
