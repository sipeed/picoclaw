package vecstore

import (
	"strings"
	"testing"
)

func TestChunkMarkdownByHeaders(t *testing.T) {
	md := `# Title

Intro paragraph.

## Section A

Content A here.

## Section B

Content B here.
`
	chunks := ChunkMarkdown("test.md", md, 800)
	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}

	// First chunk should contain "Title" and "Intro"
	if !strings.Contains(chunks[0].Text, "Title") {
		t.Errorf("first chunk should contain Title: %q", chunks[0].Text)
	}

	// Should have sections A and B as separate chunks
	foundA, foundB := false, false
	for _, c := range chunks {
		if strings.Contains(c.Text, "Section A") {
			foundA = true
		}
		if strings.Contains(c.Text, "Section B") {
			foundB = true
		}
	}
	if !foundA || !foundB {
		t.Errorf("expected sections A and B in separate chunks, foundA=%v foundB=%v", foundA, foundB)
	}
}

func TestChunkMarkdownLongSection(t *testing.T) {
	// Create a long section that exceeds maxChars
	long := "## Big Section\n\n"
	for i := 0; i < 20; i++ {
		long += "This is paragraph number " + string(rune('A'+i)) + ". It has some content.\n\n"
	}

	chunks := ChunkMarkdown("test.md", long, 200)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks for long section, got %d", len(chunks))
	}

	for _, c := range chunks {
		if c.Source != "test.md" {
			t.Errorf("expected source 'test.md', got %q", c.Source)
		}
		if c.ID == "" {
			t.Error("chunk ID should not be empty")
		}
	}
}

func TestChunkMarkdownDeterministicIDs(t *testing.T) {
	md := "## Hello\n\nWorld"
	c1 := ChunkMarkdown("src.md", md, 800)
	c2 := ChunkMarkdown("src.md", md, 800)

	if len(c1) != len(c2) {
		t.Fatalf("chunk counts differ: %d vs %d", len(c1), len(c2))
	}
	for i := range c1 {
		if c1[i].ID != c2[i].ID {
			t.Errorf("chunk %d: IDs differ %q vs %q", i, c1[i].ID, c2[i].ID)
		}
	}
}

func TestChunkMarkdownEmpty(t *testing.T) {
	chunks := ChunkMarkdown("test.md", "", 800)
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for empty text, got %d", len(chunks))
	}
}

func TestChunkIDUniqueness(t *testing.T) {
	// Same text, different source → different ID
	id1 := chunkID("a.md", "hello")
	id2 := chunkID("b.md", "hello")
	if id1 == id2 {
		t.Error("IDs should differ for different sources")
	}

	// Same source, different text → different ID
	id3 := chunkID("a.md", "hello")
	id4 := chunkID("a.md", "world")
	if id3 == id4 {
		t.Error("IDs should differ for different text")
	}
}
