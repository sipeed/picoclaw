package memory

import (
	"testing"
)

// --- ParseFrontmatter tests ---

func TestParseFrontmatter_Valid(t *testing.T) {
	content := `---
title: Go Error Patterns
created: 2026-02-23
updated: 2026-02-23
tags: [go, patterns, errors]
aliases: [error-handling, go-errors]
---

# Go Error Patterns

Content here.`

	meta, body := ParseFrontmatter(content)

	if meta.Title != "Go Error Patterns" {
		t.Errorf("Title = %q, want %q", meta.Title, "Go Error Patterns")
	}
	if meta.Created != "2026-02-23" {
		t.Errorf("Created = %q, want %q", meta.Created, "2026-02-23")
	}
	if meta.Updated != "2026-02-23" {
		t.Errorf("Updated = %q, want %q", meta.Updated, "2026-02-23")
	}
	assertStringSlice(t, "Tags", meta.Tags, []string{"go", "patterns", "errors"})
	assertStringSlice(t, "Aliases", meta.Aliases, []string{"error-handling", "go-errors"})

	expectedBody := "\n\n# Go Error Patterns\n\nContent here."
	if body != expectedBody {
		t.Errorf("Body = %q, want %q", body, expectedBody)
	}
}

func TestParseFrontmatter_Empty(t *testing.T) {
	content := "# Just a heading\n\nSome content."

	meta, body := ParseFrontmatter(content)

	if meta.Title != "" {
		t.Errorf("Title = %q, want empty", meta.Title)
	}
	if body != content {
		t.Errorf("Body should equal original content")
	}
}

func TestParseFrontmatter_Partial(t *testing.T) {
	content := `---
title: Only Title
tags: [single]
---

Body text.`

	meta, body := ParseFrontmatter(content)

	if meta.Title != "Only Title" {
		t.Errorf("Title = %q, want %q", meta.Title, "Only Title")
	}
	if meta.Created != "" {
		t.Errorf("Created = %q, want empty", meta.Created)
	}
	if meta.Updated != "" {
		t.Errorf("Updated = %q, want empty", meta.Updated)
	}
	assertStringSlice(t, "Tags", meta.Tags, []string{"single"})
	if len(meta.Aliases) != 0 {
		t.Errorf("Aliases = %v, want empty", meta.Aliases)
	}

	expectedBody := "\n\nBody text."
	if body != expectedBody {
		t.Errorf("Body = %q, want %q", body, expectedBody)
	}
}

func TestParseFrontmatter_NoClosingDelimiter(t *testing.T) {
	content := `---
title: Broken
tags: [oops]

No closing delimiter here.`

	meta, body := ParseFrontmatter(content)

	if meta.Title != "" {
		t.Errorf("Title = %q, want empty for malformed frontmatter", meta.Title)
	}
	if body != content {
		t.Errorf("Body should equal original content for malformed frontmatter")
	}
}

func TestParseFrontmatter_EmptyContent(t *testing.T) {
	meta, body := ParseFrontmatter("")

	if meta.Title != "" {
		t.Errorf("Title = %q, want empty", meta.Title)
	}
	if body != "" {
		t.Errorf("Body = %q, want empty", body)
	}
}

func TestParseFrontmatter_OnlyFrontmatter(t *testing.T) {
	content := `---
title: No Body
---
`
	meta, body := ParseFrontmatter(content)

	if meta.Title != "No Body" {
		t.Errorf("Title = %q, want %q", meta.Title, "No Body")
	}
	if body != "\n" {
		t.Errorf("Body = %q, want %q", body, "\n")
	}
}

// --- parseBracketList tests ---

func TestParseBracketList(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"multiple items", "[a, b, c]", []string{"a", "b", "c"}},
		{"single item", "[only]", []string{"only"}},
		{"empty brackets", "[]", nil},
		{"empty string", "", nil},
		{"spaces around items", "[ foo , bar , baz ]", []string{"foo", "bar", "baz"}},
		{"no brackets", "plain, text", []string{"plain", "text"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseBracketList(tt.input)
			assertStringSlice(t, "result", got, tt.want)
		})
	}
}

// --- ExtractWikilinks tests ---

func TestExtractWikilinks_None(t *testing.T) {
	links := ExtractWikilinks("No links here, just plain text.")
	if len(links) != 0 {
		t.Errorf("got %v, want empty", links)
	}
}

func TestExtractWikilinks_Single(t *testing.T) {
	links := ExtractWikilinks("See [[go-patterns]] for details.")
	assertStringSlice(t, "links", links, []string{"go-patterns"})
}

func TestExtractWikilinks_Multiple(t *testing.T) {
	body := `Check [[error-handling]] and [[testing-guide]].
Also see [[deployment]].`
	links := ExtractWikilinks(body)
	assertStringSlice(t, "links", links, []string{"error-handling", "testing-guide", "deployment"})
}

func TestExtractWikilinks_Unclosed(t *testing.T) {
	links := ExtractWikilinks("Broken [[link without closing.")
	if len(links) != 0 {
		t.Errorf("got %v, want empty for unclosed link", links)
	}
}

func TestExtractWikilinks_Empty(t *testing.T) {
	links := ExtractWikilinks("")
	if len(links) != 0 {
		t.Errorf("got %v, want empty", links)
	}
}

func TestExtractWikilinks_Adjacent(t *testing.T) {
	links := ExtractWikilinks("[[one]][[two]]")
	assertStringSlice(t, "links", links, []string{"one", "two"})
}

// --- Test helpers ---

func assertStringSlice(t *testing.T, name string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: got %v (len %d), want %v (len %d)", name, got, len(got), want, len(want))
		return
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("%s[%d] = %q, want %q", name, i, got[i], want[i])
		}
	}
}
