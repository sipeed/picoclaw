package signal

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestMarkdownToSignal(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantText   string
		wantStyles []string
	}{
		{
			name:       "plain text unchanged",
			input:      "Hello world",
			wantText:   "Hello world",
			wantStyles: nil,
		},
		{
			name:       "empty string",
			input:      "",
			wantText:   "",
			wantStyles: nil,
		},
		{
			name:       "bold",
			input:      "**hello** world",
			wantText:   "hello world",
			wantStyles: []string{"0:5:BOLD"},
		},
		{
			name:       "multiple bold",
			input:      "**hello** and **world**",
			wantText:   "hello and world",
			wantStyles: []string{"0:5:BOLD", "10:5:BOLD"},
		},
		{
			name:       "italic",
			input:      "It is *really* good",
			wantText:   "It is really good",
			wantStyles: []string{"6:6:ITALIC"},
		},
		{
			name:       "bold and italic",
			input:      "**Alice** is *great*",
			wantText:   "Alice is great",
			wantStyles: []string{"0:5:BOLD", "9:5:ITALIC"},
		},
		{
			name:       "strikethrough",
			input:      "~~not available~~ found it",
			wantText:   "not available found it",
			wantStyles: []string{"0:13:STRIKETHROUGH"},
		},
		{
			name:       "unmatched bold markers left as-is",
			input:      "**unclosed bold",
			wantText:   "**unclosed bold",
			wantStyles: nil,
		},
		{
			name:       "unmatched italic marker left as-is",
			input:      "*unclosed italic",
			wantText:   "*unclosed italic",
			wantStyles: nil,
		},
		{
			name:       "heading stripped",
			input:      "## Tasks\nHere are some tasks",
			wantText:   "Tasks\nHere are some tasks",
			wantStyles: nil,
		},
		{
			name:       "list markers converted (dash)",
			input:      "- Alice\n- Bob\n- Carol",
			wantText:   "• Alice\n• Bob\n• Carol",
			wantStyles: nil,
		},
		{
			name:       "list markers converted (asterisk)",
			input:      "* Alice\n* Bob\n* Carol",
			wantText:   "• Alice\n• Bob\n• Carol",
			wantStyles: nil,
		},
		{
			name:       "asterisk list with bold content",
			input:      "* **First item**: description one\n* **Second item**: description two",
			wantText:   "• First item: description one\n• Second item: description two",
			wantStyles: []string{"2:10:BOLD", "32:11:BOLD"},
		},
		{
			name:       "blockquote stripped",
			input:      "> Some quote here",
			wantText:   "Some quote here",
			wantStyles: nil,
		},
		{
			name:       "bold with non-ASCII characters",
			input:      "**Blåbær** er på bordet",
			wantText:   "Blåbær er på bordet",
			wantStyles: []string{"0:6:BOLD"},
		},
		{
			name:       "mixed formatting and line-level",
			input:      "## Results\n- **Alice** is in group A\n- **Bob** is in group B",
			wantText:   "Results\n• Alice is in group A\n• Bob is in group B",
			wantStyles: []string{"10:5:BOLD", "32:3:BOLD"},
		},
		{
			name:       "inline code",
			input:      "Run `kubectl get pods` to check",
			wantText:   "Run kubectl get pods to check",
			wantStyles: []string{"4:16:MONOSPACE"},
		},
		{
			name:       "code block",
			input:      "Example:\n```bash\necho hello\n```\nDone",
			wantText:   "Example:\necho hello\nDone",
			wantStyles: []string{"9:10:MONOSPACE"},
		},
		{
			name:       "code block with language tag",
			input:      "```python\nprint(\"hi\")\n```",
			wantText:   "print(\"hi\")",
			wantStyles: []string{"0:11:MONOSPACE"},
		},
		{
			name:       "code preserves markdown inside",
			input:      "Use `**not bold**` in code",
			wantText:   "Use **not bold** in code",
			wantStyles: []string{"4:12:MONOSPACE"},
		},
		{
			name:       "inline code and bold mixed",
			input:      "**Important**: use `cmd` here",
			wantText:   "Important: use cmd here",
			wantStyles: []string{"0:9:BOLD", "15:3:MONOSPACE"},
		},
		{
			name:       "markdown link converted",
			input:      "See [Google](https://google.com) for more",
			wantText:   "See Google (https://google.com) for more",
			wantStyles: nil,
		},
		{
			name:       "link with bold text",
			input:      "**Check** [docs](https://example.com)",
			wantText:   "Check docs (https://example.com)",
			wantStyles: []string{"0:5:BOLD"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotText, gotStyles := markdownToSignal(tt.input)
			if gotText != tt.wantText {
				t.Errorf("text = %q, want %q", gotText, tt.wantText)
			}
			if !reflect.DeepEqual(gotStyles, tt.wantStyles) {
				t.Errorf("styles = %v, want %v", gotStyles, tt.wantStyles)
			}
		})
	}
}

func TestExtensionFromMIME(t *testing.T) {
	tests := []struct {
		mime string
		want string
	}{
		{"image/jpeg", ".jpg"},
		{"image/jpeg; charset=utf-8", ".jpg"},
		{"image/png", ".png"},
		{"image/gif", ".gif"},
		{"image/webp", ".webp"},
		{"audio/mpeg", ".mp3"},
		{"audio/mp3", ".mp3"},
		{"audio/ogg", ".ogg"},
		{"audio/mp4", ".m4a"},
		{"audio/aac", ".m4a"},
		{"video/mp4", ".mp4"},
		{"application/pdf", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.mime, func(t *testing.T) {
			if got := extensionFromMIME(tt.mime); got != tt.want {
				t.Errorf("extensionFromMIME(%q) = %q, want %q", tt.mime, got, tt.want)
			}
		})
	}
}

func TestSignalEventDeserialization(t *testing.T) {
	tests := []struct {
		name       string
		json       string
		wantSource string
		wantName   string
		wantMsg    string
		wantGroup  bool
	}{
		{
			name: "direct message",
			json: `{
				"envelope": {
					"source": "+4512345678",
					"sourceNumber": "+4512345678",
					"sourceUuid": "abc-def-123",
					"sourceName": "John Doe",
					"sourceDevice": 1,
					"timestamp": 1700000000000,
					"dataMessage": {
						"timestamp": 1700000000000,
						"message": "Hello bot",
						"expiresInSeconds": 0,
						"viewOnce": false
					}
				},
				"account": "+4587654321"
			}`,
			wantSource: "+4512345678",
			wantName:   "John Doe",
			wantMsg:    "Hello bot",
			wantGroup:  false,
		},
		{
			name: "group message",
			json: `{
				"envelope": {
					"source": "+4512345678",
					"sourceNumber": "+4512345678",
					"sourceUuid": "abc-def-123",
					"sourceName": "Jane",
					"sourceDevice": 2,
					"timestamp": 1700000000000,
					"dataMessage": {
						"timestamp": 1700000000000,
						"message": "Hi group",
						"groupInfo": {
							"groupId": "R3JvdXBJZEhlcmU=",
							"type": "DELIVER"
						}
					}
				},
				"account": "+4587654321"
			}`,
			wantSource: "+4512345678",
			wantName:   "Jane",
			wantMsg:    "Hi group",
			wantGroup:  true,
		},
		{
			name: "message with attachment",
			json: `{
				"envelope": {
					"source": "+4512345678",
					"sourceNumber": "+4512345678",
					"sourceUuid": "abc-def-123",
					"sourceName": "John",
					"sourceDevice": 1,
					"timestamp": 1700000000000,
					"dataMessage": {
						"timestamp": 1700000000000,
						"message": "",
						"attachments": [{
							"contentType": "image/jpeg",
							"filename": "photo.jpg",
							"id": "att-123",
							"size": 54321
						}]
					}
				},
				"account": "+4587654321"
			}`,
			wantSource: "+4512345678",
			wantName:   "John",
			wantMsg:    "",
			wantGroup:  false,
		},
		{
			name: "no data message (e.g. receipt)",
			json: `{
				"envelope": {
					"source": "+4512345678",
					"sourceNumber": "+4512345678",
					"sourceUuid": "abc-def-123",
					"sourceName": "John",
					"sourceDevice": 1,
					"timestamp": 1700000000000
				},
				"account": "+4587654321"
			}`,
			wantSource: "+4512345678",
			wantName:   "John",
			wantMsg:    "",
			wantGroup:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var event signalEvent
			if err := json.Unmarshal([]byte(tt.json), &event); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if event.Envelope.SourceNumber != tt.wantSource {
				t.Errorf("sourceNumber = %q, want %q", event.Envelope.SourceNumber, tt.wantSource)
			}
			if event.Envelope.SourceName != tt.wantName {
				t.Errorf("sourceName = %q, want %q", event.Envelope.SourceName, tt.wantName)
			}

			if event.Envelope.DataMessage != nil {
				if event.Envelope.DataMessage.Message != tt.wantMsg {
					t.Errorf("message = %q, want %q", event.Envelope.DataMessage.Message, tt.wantMsg)
				}
				gotGroup := event.Envelope.DataMessage.GroupInfo != nil
				if gotGroup != tt.wantGroup {
					t.Errorf("isGroup = %v, want %v", gotGroup, tt.wantGroup)
				}
			} else if tt.wantMsg != "" {
				t.Errorf("dataMessage is nil, want message %q", tt.wantMsg)
			}
		})
	}
}

func TestIsGroupChat(t *testing.T) {
	tests := []struct {
		name   string
		chatID string
		want   bool
	}{
		{
			name:   "phone number is not group",
			chatID: "+4571376774",
			want:   false,
		},
		{
			name:   "base64 group ID is group",
			chatID: "abc123def456==",
			want:   true,
		},
		{
			name:   "empty string is not group",
			chatID: "",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isGroupChat(tt.chatID); got != tt.want {
				t.Errorf("isGroupChat(%q) = %v, want %v", tt.chatID, got, tt.want)
			}
		})
	}
}

func TestParseMessageID(t *testing.T) {
	tests := []struct {
		name      string
		messageID string
		wantTS    int64
		wantPhone string
		wantOK    bool
	}{
		{
			name:      "valid direct message ID",
			messageID: "1700000000000:+4512345678",
			wantTS:    1700000000000,
			wantPhone: "+4512345678",
			wantOK:    true,
		},
		{
			name:      "empty string",
			messageID: "",
			wantTS:    0,
			wantPhone: "",
			wantOK:    false,
		},
		{
			name:      "no colon",
			messageID: "1700000000000",
			wantTS:    0,
			wantPhone: "",
			wantOK:    false,
		},
		{
			name:      "invalid timestamp",
			messageID: "notanumber:+4512345678",
			wantTS:    0,
			wantPhone: "",
			wantOK:    false,
		},
		{
			name:      "colon at end",
			messageID: "1700000000000:",
			wantTS:    0,
			wantPhone: "",
			wantOK:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, phone, ok := parseMessageID(tt.messageID)
			if ok != tt.wantOK {
				t.Errorf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ts != tt.wantTS {
				t.Errorf("timestamp = %d, want %d", ts, tt.wantTS)
			}
			if phone != tt.wantPhone {
				t.Errorf("phone = %q, want %q", phone, tt.wantPhone)
			}
		})
	}
}
