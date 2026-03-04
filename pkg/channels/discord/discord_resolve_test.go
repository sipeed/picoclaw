package discord

import (
	"testing"
)

func TestChannelRefRegex(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		wantID string
		wantOK bool
	}{
		{"basic channel ref", "<#123456789>", "123456789", true},
		{"long id", "<#9876543210123456>", "9876543210123456", true},
		{"no match plain text", "hello world", "", false},
		{"no match partial", "<#>", "", false},
		{"no match letters", "<#abc>", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := channelRefRe.FindStringSubmatch(tt.input)
			if tt.wantOK {
				if len(matches) < 2 || matches[1] != tt.wantID {
					t.Errorf("channelRefRe(%q) = %v, want ID %q", tt.input, matches, tt.wantID)
				}
			} else {
				if len(matches) >= 2 {
					t.Errorf("channelRefRe(%q) should not match, got %v", tt.input, matches)
				}
			}
		})
	}
}

func TestMsgLinkRegex(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantGuild string
		wantChan  string
		wantMsg   string
		wantOK    bool
	}{
		{
			"discord.com link",
			"https://discord.com/channels/111/222/333",
			"111", "222", "333", true,
		},
		{
			"discordapp.com link",
			"https://discordapp.com/channels/111/222/333",
			"111", "222", "333", true,
		},
		{
			"real world ids",
			"check this https://discord.com/channels/9000000000000001/9000000000000002/9000000000000003 please",
			"9000000000000001", "9000000000000002", "9000000000000003", true,
		},
		{"no match http", "http://discord.com/channels/1/2/3", "", "", "", false},
		{"no match missing segment", "https://discord.com/channels/1/2", "", "", "", false},
		{"no match plain text", "hello world", "", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := msgLinkRe.FindStringSubmatch(tt.input)
			if tt.wantOK {
				if len(matches) < 4 {
					t.Fatalf("msgLinkRe(%q) didn't match, want guild=%s chan=%s msg=%s",
						tt.input, tt.wantGuild, tt.wantChan, tt.wantMsg)
				}
				if matches[1] != tt.wantGuild || matches[2] != tt.wantChan || matches[3] != tt.wantMsg {
					t.Errorf("msgLinkRe(%q) = guild=%s chan=%s msg=%s, want %s/%s/%s",
						tt.input, matches[1], matches[2], matches[3],
						tt.wantGuild, tt.wantChan, tt.wantMsg)
				}
			} else {
				if len(matches) >= 4 {
					t.Errorf("msgLinkRe(%q) should not match, got %v", tt.input, matches)
				}
			}
		})
	}
}

func TestMsgLinkRegex_MultipleMatches(t *testing.T) {
	input := "see https://discord.com/channels/1/2/3 and https://discord.com/channels/4/5/6 and https://discord.com/channels/7/8/9 and https://discord.com/channels/10/11/12"
	matches := msgLinkRe.FindAllStringSubmatch(input, 3)
	if len(matches) != 3 {
		t.Fatalf("expected 3 matches (capped), got %d", len(matches))
	}
	// Verify the 3rd match is 7/8/9 (not 10/11/12)
	if matches[2][1] != "7" || matches[2][2] != "8" || matches[2][3] != "9" {
		t.Errorf("3rd match = %v, want guild=7 chan=8 msg=9", matches[2])
	}
}
