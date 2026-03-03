package telegram

import "testing"

func TestParseChatID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantCID int64
		wantTID int
		wantErr bool
	}{
		{name: "plain private", input: "12345", wantCID: 12345, wantTID: 0},
		{name: "group topic", input: "-100123/45", wantCID: -100123, wantTID: 45},
		{name: "trim spaces", input: "  -100200/7  ", wantCID: -100200, wantTID: 7},
		{name: "topic zero", input: "-100/0", wantCID: -100, wantTID: 0},
		{name: "empty", input: "", wantErr: true},
		{name: "bad chat", input: "abc/def", wantErr: true},
		{name: "missing topic", input: "-100/", wantErr: true},
		{name: "too many parts", input: "-100/1/2", wantErr: true},
		{name: "negative topic", input: "-100/-1", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotCID, gotTID, err := parseChatID(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseChatID(%q) expected error, got nil", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseChatID(%q) unexpected error: %v", tc.input, err)
			}
			if gotCID != tc.wantCID || gotTID != tc.wantTID {
				t.Fatalf("parseChatID(%q) = (%d, %d), want (%d, %d)", tc.input, gotCID, gotTID, tc.wantCID, tc.wantTID)
			}
		})
	}
}

func TestFormatChatID(t *testing.T) {
	if got := formatChatID(-100, 42); got != "-100/42" {
		t.Fatalf("formatChatID(-100, 42) = %q, want %q", got, "-100/42")
	}
	if got := formatChatID(12345, 0); got != "12345" {
		t.Fatalf("formatChatID(12345, 0) = %q, want %q", got, "12345")
	}
}
