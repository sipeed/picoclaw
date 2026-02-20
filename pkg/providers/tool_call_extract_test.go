package providers

import "testing"

func TestExtractToolCallsFromText_WebAPIFormat(t *testing.T) {
	text := `I'll search for USD/EUR.
/WebAPI
{"name": "web_search", "arguments": {"query": "current USD to EUR exchange rate"}}
</tool_call>`

	calls := extractToolCallsFromText(text)
	if len(calls) != 1 {
		t.Fatalf("len(calls) = %d, want 1", len(calls))
	}
	if calls[0].Name != "web_search" {
		t.Fatalf("calls[0].Name = %q, want %q", calls[0].Name, "web_search")
	}
	if calls[0].Arguments["query"] != "current USD to EUR exchange rate" {
		t.Fatalf("query arg mismatch: %+v", calls[0].Arguments)
	}

	stripped := stripToolCallsFromText(text)
	if stripped == text {
		t.Fatalf("expected stripped text to remove webapi tool call block")
	}
}

func TestExtractToolCallsFromText_JSONWrapperFormat(t *testing.T) {
	text := `before {"tool_calls":[{"id":"call_1","type":"function","function":{"name":"web_search","arguments":"{\"query\":\"btc price\"}"}}]} after`

	calls := extractToolCallsFromText(text)
	if len(calls) != 1 {
		t.Fatalf("len(calls) = %d, want 1", len(calls))
	}
	if calls[0].Name != "web_search" {
		t.Fatalf("calls[0].Name = %q, want %q", calls[0].Name, "web_search")
	}
	if calls[0].Arguments["query"] != "btc price" {
		t.Fatalf("query arg mismatch: %+v", calls[0].Arguments)
	}
}
