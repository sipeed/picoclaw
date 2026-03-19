package mediacache

import (
	"path/filepath"
	"testing"
	"time"
)

func openTestCache(t *testing.T) *Cache {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test_cache.db")
	c, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

func TestCache_PutAndGet(t *testing.T) {
	c := openTestCache(t)

	hash := HashData([]byte("test-image-data"))

	// Miss
	if _, ok := c.Get(hash, TypeImageDesc); ok {
		t.Fatal("expected miss on empty cache")
	}

	// Put
	if err := c.Put(hash, TypeImageDesc, "a photo of a cat"); err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Hit
	result, ok := c.Get(hash, TypeImageDesc)
	if !ok {
		t.Fatal("expected hit after Put")
	}
	if result != "a photo of a cat" {
		t.Errorf("result = %q, want %q", result, "a photo of a cat")
	}
}

func TestCache_DifferentTypes(t *testing.T) {
	c := openTestCache(t)
	hash := HashData([]byte("same-data"))

	_ = c.Put(hash, TypeImageDesc, "image description")
	_ = c.Put(hash, TypePDFOCR, "/path/to/doc.md")

	desc, ok := c.Get(hash, TypeImageDesc)
	if !ok || desc != "image description" {
		t.Errorf("image_desc: got %q, ok=%v", desc, ok)
	}

	ocr, ok := c.Get(hash, TypePDFOCR)
	if !ok || ocr != "/path/to/doc.md" {
		t.Errorf("pdf_ocr: got %q, ok=%v", ocr, ok)
	}
}

func TestCache_Upsert(t *testing.T) {
	c := openTestCache(t)
	hash := HashData([]byte("data"))

	_ = c.Put(hash, TypeImageDesc, "first")
	_ = c.Put(hash, TypeImageDesc, "updated")

	result, ok := c.Get(hash, TypeImageDesc)
	if !ok || result != "updated" {
		t.Errorf("expected updated value, got %q", result)
	}
}

func TestCache_Prune(t *testing.T) {
	c := openTestCache(t)

	hash := HashData([]byte("old-data"))
	_ = c.Put(hash, TypeImageDesc, "old description")

	// Backdate the accessed_at
	_, _ = c.db.Exec(
		`UPDATE media_cache SET accessed_at = ? WHERE hash = ?`,
		time.Now().Add(-48*time.Hour).UTC().Format(time.RFC3339),
		hash,
	)

	// Put a fresh entry
	freshHash := HashData([]byte("fresh-data"))
	_ = c.Put(freshHash, TypeImageDesc, "fresh description")

	// Prune with 24h TTL
	n, err := c.Prune(24 * time.Hour)
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if n != 1 {
		t.Errorf("pruned %d, want 1", n)
	}

	// Old should be gone
	if _, ok := c.Get(hash, TypeImageDesc); ok {
		t.Error("old entry should have been pruned")
	}

	// Fresh should remain
	if _, ok := c.Get(freshHash, TypeImageDesc); !ok {
		t.Error("fresh entry should remain")
	}
}

func TestHashData_Deterministic(t *testing.T) {
	data := []byte("hello world")
	h1 := HashData(data)
	h2 := HashData(data)
	if h1 != h2 {
		t.Errorf("non-deterministic: %q != %q", h1, h2)
	}
	if len(h1) != 16 {
		t.Errorf("hash length = %d, want 16", len(h1))
	}
}

func TestHashData_DifferentInputs(t *testing.T) {
	h1 := HashData([]byte("input-a"))
	h2 := HashData([]byte("input-b"))
	if h1 == h2 {
		t.Error("different inputs should produce different hashes")
	}
}

func TestCache_PutEntryAndGetEntry(t *testing.T) {
	c := openTestCache(t)
	hash := HashData([]byte("pdf-content"))

	entry := Entry{
		Result:   "契約書 — 第1条 本契約は甲乙間の...",
		FilePath: "/workspace/.mediacache/abc123.md",
		Pages:    8,
	}
	if err := c.PutEntry(hash, TypePDFOCR, entry); err != nil {
		t.Fatalf("PutEntry: %v", err)
	}

	got, ok := c.GetEntry(hash, TypePDFOCR)
	if !ok {
		t.Fatal("expected hit")
	}
	if got.Result != entry.Result {
		t.Errorf("Result = %q", got.Result)
	}
	if got.FilePath != entry.FilePath {
		t.Errorf("FilePath = %q", got.FilePath)
	}
	if got.Pages != 8 {
		t.Errorf("Pages = %d, want 8", got.Pages)
	}
}

func TestCache_GetEntry_Miss(t *testing.T) {
	c := openTestCache(t)
	_, ok := c.GetEntry("nonexistent", TypePDFOCR)
	if ok {
		t.Error("expected miss")
	}
}

func TestCache_SimpleGetIgnoresFilePath(t *testing.T) {
	// Simple Get/Put should still work with the new schema
	c := openTestCache(t)
	hash := HashData([]byte("img"))

	if err := c.Put(hash, TypeImageDesc, "a sunset"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	result, ok := c.Get(hash, TypeImageDesc)
	if !ok || result != "a sunset" {
		t.Errorf("Get = %q, ok=%v", result, ok)
	}

	// GetEntry should also work, with empty file_path
	entry, ok := c.GetEntry(hash, TypeImageDesc)
	if !ok {
		t.Fatal("GetEntry miss")
	}
	if entry.FilePath != "" {
		t.Errorf("FilePath should be empty for image_desc, got %q", entry.FilePath)
	}
}
