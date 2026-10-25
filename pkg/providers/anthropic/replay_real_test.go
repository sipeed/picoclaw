//go:build replay

package anthropicprovider

// Replay real session-store files through buildParams and inspect the exact
// request JSON. Detects the poison classes Anthropic rejects:
//   - assistant messages without a usable content field
//   - tool_use blocks missing id or name
//   - tool_result blocks whose tool_use_id was never emitted upstream
//
// Run: go test -tags replay,goolm,stdjson,whatsapp_native -run TestReplayRealSessions -v ./pkg/providers/anthropic

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

func TestReplayRealSessions(t *testing.T) {
	files := []string{
		"/home/sebg/mb-kit-work/sess6054.jsonl",
		"/home/sebg/mb-kit-work/sess0bf2.jsonl",
	}
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if os.IsNotExist(err) {
			t.Skipf("local session file %s not present", f)
		} else if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		var msgs []protocoltypes.Message
		for _, line := range strings.Split(string(raw), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var m protocoltypes.Message
			if err := json.Unmarshal([]byte(line), &m); err != nil {
				t.Fatalf("unmarshal %s: %v", f, err)
			}
			msgs = append(msgs, m)
		}
		params, err := buildParams(msgs, nil, "claude-haiku-5-5", map[string]any{"max_tokens": 1024})
		if err != nil {
			t.Fatalf("buildParams %s: %v", f, err)
		}
		j, err := json.Marshal(params)
		if err != nil {
			t.Fatalf("marshal %s: %v", f, err)
		}
		// Unmarshal request back and validate invariants like Anthropic would.
		var req struct {
			Messages []struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
			} `json:"messages"`
		}
		if err := json.Unmarshal(j, &req); err != nil {
			t.Fatalf("re-unmarshal %s: %v", f, err)
		}
		emitted := map[string]bool{}
		problems := 0
		for i, m := range req.Messages {
			// content field present and non-null?
			if len(m.Content) == 0 || string(m.Content) == "null" {
				t.Errorf("%s: messages[%d] role=%s content MISSING/null -> Anthropic 'Field required'", f, i, m.Role)
				problems++
				continue
			}
			var blocks []struct {
				Type string `json:"type"`
				ID   string `json:"id"`
				Name string `json:"name"`
			}
			if err := json.Unmarshal(m.Content, &blocks); err != nil {
				continue // plain string content is fine
			}
			for _, b := range blocks {
				switch b.Type {
				case "tool_use":
					if b.ID == "" || b.Name == "" {
						t.Errorf("%s: messages[%d] tool_use id=%q name=%q", f, i, b.ID, b.Name)
						problems++
					}
					emitted[b.ID] = true
				case "tool_result":
					// tool_result lives in user messages; id field differs
				}
			}
			if m.Role == "user" {
				var b2 []struct {
					Type       string `json:"type"`
					ToolUseID  string `json:"tool_use_id"`
				}
				_ = json.Unmarshal(m.Content, &b2)
				for _, b := range b2 {
					if b.Type == "tool_result" && !emitted[b.ToolUseID] {
						t.Errorf("%s: messages[%d] orphan tool_result tool_use_id=%s", f, i, b.ToolUseID)
						problems++
					}
				}
			}
		}
		t.Logf("%s: %d messages, %d problems", f, len(req.Messages), problems)
	}
}
