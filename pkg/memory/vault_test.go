package memory

import (
	"os"
	"path/filepath"
	"strings"
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

// --- ScanAll tests ---

func TestScanAll_MultipleNotes(t *testing.T) {
	dir := t.TempDir()

	// Note with frontmatter in a subfolder
	writeTestFile(t, dir, "topics/go-errors.md", `---
title: Go Error Patterns
created: 2026-02-20
updated: 2026-02-23
tags: [go, errors]
aliases: [error-handling]
---

Content about Go errors. See [[testing-guide]].`)

	// Note without frontmatter at root
	writeTestFile(t, dir, "quick-note.md", "# Quick Note\n\nJust some text.")

	// Daily note
	writeTestFile(t, dir, "202602/20260223.md", `---
title: 2026-02-23
created: 2026-02-23
updated: 2026-02-23
tags: [daily]
---

Today's notes.`)

	// _index.md should be skipped
	writeTestFile(t, dir, "_index.md", "# Index\nShould be skipped.")

	// Non-md file should be skipped
	writeTestFile(t, dir, "notes.txt", "Not a markdown file.")

	vault := NewVault(dir)
	notes, err := vault.ScanAll()
	if err != nil {
		t.Fatalf("ScanAll error: %v", err)
	}

	if len(notes) != 3 {
		t.Fatalf("got %d notes, want 3", len(notes))
	}

	// Find the go-errors note and verify metadata
	var goErrors *NoteMeta
	for i := range notes {
		if strings.Contains(notes[i].RelPath, "go-errors") {
			goErrors = &notes[i]
			break
		}
	}
	if goErrors == nil {
		t.Fatal("go-errors note not found in scan results")
	}
	if goErrors.Title != "Go Error Patterns" {
		t.Errorf("Title = %q, want %q", goErrors.Title, "Go Error Patterns")
	}
	assertStringSlice(t, "Tags", goErrors.Tags, []string{"go", "errors"})
	assertStringSlice(t, "Links", goErrors.Links, []string{"testing-guide"})

	// Note without frontmatter should use filename as title
	var quickNote *NoteMeta
	for i := range notes {
		if strings.Contains(notes[i].RelPath, "quick-note") {
			quickNote = &notes[i]
			break
		}
	}
	if quickNote == nil {
		t.Fatal("quick-note not found in scan results")
	}
	if quickNote.Title != "quick-note" {
		t.Errorf("Title = %q, want %q (inferred from filename)", quickNote.Title, "quick-note")
	}
}

func TestScanAll_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	vault := NewVault(dir)
	notes, err := vault.ScanAll()
	if err != nil {
		t.Fatalf("ScanAll error: %v", err)
	}
	if len(notes) != 0 {
		t.Errorf("got %d notes, want 0", len(notes))
	}
}

// --- RebuildIndex tests ---

func TestRebuildIndex(t *testing.T) {
	dir := t.TempDir()

	writeTestFile(t, dir, "go-patterns.md", `---
title: Go Patterns
created: 2026-02-20
updated: 2026-02-23
tags: [go, patterns]
aliases: [golang-patterns]
---

Go patterns content.`)

	writeTestFile(t, dir, "hardware.md", `---
title: Hardware Setup
created: 2026-02-18
updated: 2026-02-22
tags: [hardware, setup]
---

Hardware content.`)

	vault := NewVault(dir)
	err := vault.RebuildIndex()
	if err != nil {
		t.Fatalf("RebuildIndex error: %v", err)
	}

	// Verify _index.md was created
	indexPath := filepath.Join(dir, "_index.md")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("Failed to read _index.md: %v", err)
	}

	index := string(data)

	// Check structure
	if !strings.Contains(index, "# Memory Vault Index") {
		t.Error("Index missing header")
	}
	if !strings.Contains(index, "Auto-generated") {
		t.Error("Index missing auto-generated comment")
	}
	if !strings.Contains(index, "## Notes") {
		t.Error("Index missing Notes section")
	}
	if !strings.Contains(index, "## Tags") {
		t.Error("Index missing Tags section")
	}

	// Check note entries
	if !strings.Contains(index, "Go Patterns") {
		t.Error("Index missing Go Patterns entry")
	}
	if !strings.Contains(index, "Hardware Setup") {
		t.Error("Index missing Hardware Setup entry")
	}

	// Check tags
	if !strings.Contains(index, "**go**") {
		t.Error("Index missing 'go' tag")
	}
	if !strings.Contains(index, "**hardware**") {
		t.Error("Index missing 'hardware' tag")
	}

	// Check aliases
	if !strings.Contains(index, "## Aliases") {
		t.Error("Index missing Aliases section")
	}
	if !strings.Contains(index, "golang-patterns") {
		t.Error("Index missing golang-patterns alias")
	}
}

// --- ReadIndex tests ---

func TestReadIndex_Exists(t *testing.T) {
	dir := t.TempDir()
	indexContent := "# Memory Vault Index\n\nSome index content."
	writeTestFile(t, dir, "_index.md", indexContent)

	vault := NewVault(dir)
	got := vault.ReadIndex()
	if got != indexContent {
		t.Errorf("ReadIndex = %q, want %q", got, indexContent)
	}
}

func TestReadIndex_Missing(t *testing.T) {
	dir := t.TempDir()
	vault := NewVault(dir)
	got := vault.ReadIndex()
	if got != "" {
		t.Errorf("ReadIndex = %q, want empty for missing index", got)
	}
}

// --- SaveNote tests ---

func TestSaveNote_New(t *testing.T) {
	dir := t.TempDir()
	vault := NewVault(dir)

	meta := NoteMeta{
		Title:   "Test Note",
		Tags:    []string{"test", "example"},
		Aliases: []string{"test-alias"},
	}
	err := vault.SaveNote("topics/test-note.md", meta, "This is the body content.")
	if err != nil {
		t.Fatalf("SaveNote error: %v", err)
	}

	// Verify file was created with correct content
	data, err := os.ReadFile(filepath.Join(dir, "topics", "test-note.md"))
	if err != nil {
		t.Fatalf("Failed to read saved note: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "title: Test Note") {
		t.Error("Saved note missing title in frontmatter")
	}
	if !strings.Contains(content, "tags: [test, example]") {
		t.Error("Saved note missing tags in frontmatter")
	}
	if !strings.Contains(content, "aliases: [test-alias]") {
		t.Error("Saved note missing aliases in frontmatter")
	}
	if !strings.Contains(content, "created:") {
		t.Error("Saved note missing created date")
	}
	if !strings.Contains(content, "This is the body content.") {
		t.Error("Saved note missing body content")
	}

	// Verify index was updated
	index := vault.ReadIndex()
	if !strings.Contains(index, "Test Note") {
		t.Error("Index not updated after SaveNote")
	}
}

func TestSaveNote_UpdateExisting(t *testing.T) {
	dir := t.TempDir()
	vault := NewVault(dir)

	// Save initial note
	meta1 := NoteMeta{
		Title: "Original Title",
		Tags:  []string{"v1"},
	}
	vault.SaveNote("note.md", meta1, "Original body.")

	// Read back to get the created date
	data1, _ := os.ReadFile(filepath.Join(dir, "note.md"))
	origMeta, _ := ParseFrontmatter(string(data1))
	origCreated := origMeta.Created

	// Update the same note
	meta2 := NoteMeta{
		Title: "Updated Title",
		Tags:  []string{"v2"},
	}
	err := vault.SaveNote("note.md", meta2, "Updated body.")
	if err != nil {
		t.Fatalf("SaveNote update error: %v", err)
	}

	// Verify created date was preserved
	data2, _ := os.ReadFile(filepath.Join(dir, "note.md"))
	updatedMeta, _ := ParseFrontmatter(string(data2))

	if updatedMeta.Created != origCreated {
		t.Errorf("Created date changed: %q -> %q", origCreated, updatedMeta.Created)
	}
	if updatedMeta.Title != "Updated Title" {
		t.Errorf("Title = %q, want %q", updatedMeta.Title, "Updated Title")
	}
}

// --- ReadNote tests ---

func TestReadNote(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "test.md", "# Test\nContent.")

	vault := NewVault(dir)
	content, err := vault.ReadNote("test.md")
	if err != nil {
		t.Fatalf("ReadNote error: %v", err)
	}
	if content != "# Test\nContent." {
		t.Errorf("ReadNote = %q, want %q", content, "# Test\nContent.")
	}
}

func TestReadNote_Missing(t *testing.T) {
	dir := t.TempDir()
	vault := NewVault(dir)
	_, err := vault.ReadNote("nonexistent.md")
	if err == nil {
		t.Error("ReadNote should error for missing file")
	}
}

// --- Search tests ---

func TestSearch_ByTags(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "a.md", "---\ntitle: Note A\ntags: [go, errors]\n---\nContent A.")
	writeTestFile(t, dir, "b.md", "---\ntitle: Note B\ntags: [go, testing]\n---\nContent B.")
	writeTestFile(t, dir, "c.md", "---\ntitle: Note C\ntags: [python]\n---\nContent C.")

	vault := NewVault(dir)
	results, err := vault.Search("", []string{"go"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}

	// Search with AND logic: must have both tags
	results2, err := vault.Search("", []string{"go", "errors"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results2) != 1 {
		t.Fatalf("got %d results for AND search, want 1", len(results2))
	}
	if results2[0].Title != "Note A" {
		t.Errorf("Title = %q, want %q", results2[0].Title, "Note A")
	}
}

func TestSearch_ByQuery(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "go-errors.md", "---\ntitle: Go Error Patterns\ntags: [go]\n---\nContent.")
	writeTestFile(t, dir, "python.md", "---\ntitle: Python Basics\ntags: [python]\n---\nContent.")

	vault := NewVault(dir)
	results, err := vault.Search("Error", nil)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Title != "Go Error Patterns" {
		t.Errorf("Title = %q, want %q", results[0].Title, "Go Error Patterns")
	}
}

func TestSearch_Combined(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "a.md", "---\ntitle: Go Errors\ntags: [go, errors]\n---\nContent.")
	writeTestFile(t, dir, "b.md", "---\ntitle: Go Testing\ntags: [go, testing]\n---\nContent.")
	writeTestFile(t, dir, "c.md", "---\ntitle: Python Errors\ntags: [python, errors]\n---\nContent.")

	vault := NewVault(dir)
	results, err := vault.Search("Go", []string{"errors"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Title != "Go Errors" {
		t.Errorf("Title = %q, want %q", results[0].Title, "Go Errors")
	}
}

func TestSearch_NoResults(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "a.md", "---\ntitle: Note A\ntags: [go]\n---\nContent.")

	vault := NewVault(dir)
	results, err := vault.Search("nonexistent", nil)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}

// --- Test helpers ---

func writeTestFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	fullPath := filepath.Join(dir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("Failed to create dir for %s: %v", relPath, err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write %s: %v", relPath, err)
	}
}

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
