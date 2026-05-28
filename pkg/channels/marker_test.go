// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package channels

import (
	"testing"
)

func TestSplitByMarker_Basic(t *testing.T) {
	content := "Hello <|[SPLIT]|>World"
	chunks := SplitByMarker(content)

	if len(chunks) != 2 {
		t.Fatalf("Expected 2 chunks, got %d: %q", len(chunks), chunks)
	}
	if chunks[0] != "Hello" {
		t.Errorf("Expected first chunk 'Hello', got %q", chunks[0])
	}
	if chunks[1] != "World" {
		t.Errorf("Expected second chunk 'World', got %q", chunks[1])
	}
}

func TestSplitByMarker_NoMarker(t *testing.T) {
	content := "Hello World"
	chunks := SplitByMarker(content)

	if len(chunks) != 1 {
		t.Fatalf("Expected 1 chunk, got %d: %q", len(chunks), chunks)
	}
	if chunks[0] != "Hello World" {
		t.Errorf("Expected chunk 'Hello World', got %q", chunks[0])
	}
}

func TestSplitByMarker_MultipleMarkers(t *testing.T) {
	content := "Part1 <|[SPLIT]|> Part2 <|[SPLIT]|> Part3"
	chunks := SplitByMarker(content)

	if len(chunks) != 3 {
		t.Fatalf("Expected 3 chunks, got %d: %q", len(chunks), chunks)
	}
	if chunks[0] != "Part1" || chunks[1] != "Part2" || chunks[2] != "Part3" {
		t.Errorf("Unexpected chunks: %q", chunks)
	}
}

func TestSplitByMarker_EmptyParts(t *testing.T) {
	// Test consecutive markers and leading/trailing markers
	content := "<|[SPLIT]|>Hello <|[SPLIT]|><|[SPLIT]|>World<|[SPLIT]|>"
	chunks := SplitByMarker(content)

	if len(chunks) != 2 {
		t.Fatalf("Expected 2 chunks, got %d: %q", len(chunks), chunks)
	}
	if chunks[0] != "Hello" || chunks[1] != "World" {
		t.Errorf("Unexpected chunks: %q", chunks)
	}
}

func TestSplitByMarker_WhitespaceTrimmed(t *testing.T) {
	content := "  Hello   <|[SPLIT]|>   World  "
	chunks := SplitByMarker(content)

	if len(chunks) != 2 {
		t.Fatalf("Expected 2 chunks, got %d: %q", len(chunks), chunks)
	}
	if chunks[0] != "Hello" || chunks[1] != "World" {
		t.Errorf("Whitespace should be trimmed: %q", chunks)
	}
}

func TestSplitByMarker_EmptyInput(t *testing.T) {
	chunks := SplitByMarker("")
	if len(chunks) != 0 {
		t.Errorf("Expected empty slice for empty input, got %d chunks", len(chunks))
	}
}

// TestMarkerAndLengthSplitIntegration tests that SplitByMarker and SplitMessage work together correctly.
// Marker splitting happens first (per-agent config), then length splitting happens (per-channel config).
func TestMarkerAndLengthSplitIntegration(t *testing.T) {
	maxLen := 10

	// Original content: "Short <|[SPLIT]|> ThisIsAVeryLongString"
	content := "Short <|[SPLIT]|> ThisIsAVeryLongString"
	markerChunks := SplitByMarker(content)

	// Step 1: Marker split should give us 2 chunks
	if len(markerChunks) != 2 {
		t.Fatalf("Expected 2 marker chunks, got %d: %q", len(markerChunks), markerChunks)
	}

	// Step 2: Length split should be applied to each marker chunk
	var finalChunks []string
	for _, chunk := range markerChunks {
		if len([]rune(chunk)) > maxLen {
			lengthChunks := SplitMessage(chunk, maxLen)
			finalChunks = append(finalChunks, lengthChunks...)
		} else {
			finalChunks = append(finalChunks, chunk)
		}
	}

	// "Short" is 6 chars, within limit
	// "ThisIsAVeryLongString" is 22 chars, should be split into multiple chunks
	// SplitMessage with maxLen=10 splits: "ThisIsAVeryLongString" -> ["ThisI", "sAVer", "yLong", "String"] (5 chunks)
	if len(finalChunks) != 5 {
		t.Errorf("Expected 5 final chunks, got %d: %q", len(finalChunks), finalChunks)
	}

	// Verify first chunk is unchanged
	if finalChunks[0] != "Short" {
		t.Errorf("First chunk should be 'Short', got %q", finalChunks[0])
	}

	// Verify all length-split chunks are within limit
	for i, chunk := range finalChunks[1:] {
		if len([]rune(chunk)) > maxLen {
			t.Errorf("Chunk %d exceeds maxLen: %q (%d chars)", i+1, chunk, len([]rune(chunk)))
		}
	}
}

// TestMarkerSplitPreservesCodeBlockIntegrity tests that marker split preserves code block boundaries
func TestMarkerSplitPreservesCodeBlockIntegrity(t *testing.T) {
	content := "Hello <|[SPLIT]|>```go\npackage main\n```<|[SPLIT]|>World"
	chunks := SplitByMarker(content)

	if len(chunks) != 3 {
		t.Fatalf("Expected 3 chunks, got %d: %q", len(chunks), chunks)
	}

	// Verify code block is intact in middle chunk
	if chunks[1] != "```go\npackage main\n```" {
		t.Errorf("Code block not preserved correctly: %q", chunks[1])
	}
}
