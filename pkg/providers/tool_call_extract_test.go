package providers

import "testing"

func TestExtractToolCallsFromText_IndentedObjectWithExtraText(t *testing.T) {
	text := `Planning tool execution...
{
  "message": "calling tools",
  "tool_calls": [
    {
      "id": "call_1",
      "type": "function",
      "function": {
        "name": "read_file",
        "arguments": "{\"path\":\"/tmp/test.txt\"}"
      }
    }
  ],
  "status": "ok"
}
Done.`

	got := extractToolCallsFromText(text)
	if len(got) != 1 {
		t.Fatalf("extractToolCallsFromText() len = %d, want 1", len(got))
	}
	if got[0].Name != "read_file" {
		t.Fatalf("tool name = %q, want %q", got[0].Name, "read_file")
	}
	if got[0].Arguments["path"] != "/tmp/test.txt" {
		t.Fatalf("tool args[path] = %v, want /tmp/test.txt", got[0].Arguments["path"])
	}
}

func TestExtractToolCallsFromText_ToolCallsNotFirstField(t *testing.T) {
	text := `{"kind":"assistant_result","metadata":{"provider":"x"},"tool_calls":[{"id":"call_7","type":"function","function":{"name":"write_file","arguments":"{\"path\":\"a.txt\",\"content\":\"hello\"}"}}]}`

	got := extractToolCallsFromText(text)
	if len(got) != 1 {
		t.Fatalf("extractToolCallsFromText() len = %d, want 1", len(got))
	}
	if got[0].ID != "call_7" {
		t.Fatalf("tool id = %q, want %q", got[0].ID, "call_7")
	}
	if got[0].Name != "write_file" {
		t.Fatalf("tool name = %q, want %q", got[0].Name, "write_file")
	}
}

func TestExtractToolCallsFromText_SkipsInvalidCandidateAndFindsValidObject(t *testing.T) {
	text := `prefix {"tool_calls":invalid} middle {"note":"valid-json-without-tools"} tail {
  "note": "valid-json-with-tools",
  "tool_calls": [
    {
      "id": "call_2",
      "type": "function",
      "function": {
        "name": "get_weather",
        "arguments": "{\"city\":\"Tokyo\"}"
      }
    }
  ]
}`

	got := extractToolCallsFromText(text)
	if len(got) != 1 {
		t.Fatalf("extractToolCallsFromText() len = %d, want 1", len(got))
	}
	if got[0].Name != "get_weather" {
		t.Fatalf("tool name = %q, want %q", got[0].Name, "get_weather")
	}
}

func TestStripToolCallsFromText_DoesNotStripInvalidToolCallsObject(t *testing.T) {
	text := `before {"tool_calls":"not-an-array","other":1} after`

	got := stripToolCallsFromText(text)
	if got != text {
		t.Fatalf("stripToolCallsFromText() = %q, want unchanged", got)
	}
}

func TestStripToolCallsFromText_StripsValidIndentedObject(t *testing.T) {
	text := `before
{
  "message": "tool call follows",
  "tool_calls": [
    {
      "id": "call_3",
      "type": "function",
      "function": {
        "name": "list_files",
        "arguments": "{}"
      }
    }
  ]
}
after`

	got := stripToolCallsFromText(text)
	want := "before\n\nafter"
	if got != want {
		t.Fatalf("stripToolCallsFromText() = %q, want %q", got, want)
	}
}
