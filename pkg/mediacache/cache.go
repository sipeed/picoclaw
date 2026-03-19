package mediacache

import (
	"database/sql"
	"fmt"
	"hash/fnv"
	"time"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS media_cache (
	hash        TEXT NOT NULL,
	type        TEXT NOT NULL,
	result      TEXT NOT NULL DEFAULT '',
	file_path   TEXT NOT NULL DEFAULT '',
	pages       INTEGER NOT NULL DEFAULT 0,
	created_at  TEXT NOT NULL,
	accessed_at TEXT NOT NULL,
	PRIMARY KEY (hash, type)
);
CREATE INDEX IF NOT EXISTS idx_media_cache_accessed ON media_cache(accessed_at);
`

// Entry types for the cache.
const (
	TypeImageDesc = "image_desc"
	TypePDFOCR    = "pdf_ocr"
)

// Cache provides SQLite-backed caching for media processing results.
// Image descriptions are stored as text directly in the result column.
// PDF OCR results store the path to the generated markdown file.
type Cache struct {
	db *sql.DB
}

// Open opens (or creates) a media cache database at dbPath.
func Open(dbPath string) (*Cache, error) {
	connStr := "file:" + dbPath + "?_journal_mode=WAL&_busy_timeout=5000"
	db, err := sql.Open("sqlite", connStr)
	if err != nil {
		return nil, fmt.Errorf("open media cache: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("init media cache schema: %w", err)
	}
	return &Cache{db: db}, nil
}

// Close closes the database connection.
func (c *Cache) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// Get retrieves a cached result by hash and type.
// Returns the result and true if found, or empty string and false if not.
func (c *Cache) Get(hash, entryType string) (string, bool) {
	var result string
	err := c.db.QueryRow(
		`SELECT result FROM media_cache WHERE hash = ? AND type = ?`,
		hash, entryType,
	).Scan(&result)
	if err != nil {
		return "", false
	}
	// Update accessed_at
	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = c.db.Exec(
		`UPDATE media_cache SET accessed_at = ? WHERE hash = ? AND type = ?`,
		now, hash, entryType,
	)
	return result, true
}

// Put stores a result in the cache.
func (c *Cache) Put(hash, entryType, result string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := c.db.Exec(
		`INSERT INTO media_cache (hash, type, result, created_at, accessed_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(hash, type) DO UPDATE SET result = ?, accessed_at = ?`,
		hash, entryType, result, now, now,
		result, now,
	)
	return err
}

// Entry represents a full cache entry with optional file path and page count.
// Used for PDF OCR where result holds a preview and file_path holds the full MD.
type Entry struct {
	Result   string // preview text (image desc) or first-page excerpt (PDF)
	FilePath string // path to full OCR markdown (empty for image_desc)
	Pages    int    // number of pages (0 for non-PDF)
}

// GetEntry retrieves a full cache entry by hash and type.
func (c *Cache) GetEntry(hash, entryType string) (Entry, bool) {
	var e Entry
	err := c.db.QueryRow(
		`SELECT result, file_path, pages FROM media_cache WHERE hash = ? AND type = ?`,
		hash, entryType,
	).Scan(&e.Result, &e.FilePath, &e.Pages)
	if err != nil {
		return Entry{}, false
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = c.db.Exec(
		`UPDATE media_cache SET accessed_at = ? WHERE hash = ? AND type = ?`,
		now, hash, entryType,
	)
	return e, true
}

// PutEntry stores a full entry in the cache.
func (c *Cache) PutEntry(hash, entryType string, entry Entry) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := c.db.Exec(
		`INSERT INTO media_cache (hash, type, result, file_path, pages, created_at, accessed_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(hash, type) DO UPDATE SET result = ?, file_path = ?, pages = ?, accessed_at = ?`,
		hash, entryType, entry.Result, entry.FilePath, entry.Pages, now, now,
		entry.Result, entry.FilePath, entry.Pages, now,
	)
	return err
}

// ListEntry represents a full row from the media_cache table.
type ListEntry struct {
	Hash       string
	Type       string
	Result     string
	FilePath   string
	Pages      int
	CreatedAt  string
	AccessedAt string
}

// List returns all cache entries, optionally filtered by type.
// Pass empty string to list all types. Ordered by accessed_at desc.
func (c *Cache) List(entryType string) ([]ListEntry, error) {
	var rows *sql.Rows
	var err error
	if entryType != "" {
		rows, err = c.db.Query(
			`SELECT hash, type, result, file_path, pages, created_at, accessed_at
			 FROM media_cache WHERE type = ? ORDER BY accessed_at DESC`, entryType)
	} else {
		rows, err = c.db.Query(
			`SELECT hash, type, result, file_path, pages, created_at, accessed_at
			 FROM media_cache ORDER BY accessed_at DESC`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []ListEntry
	for rows.Next() {
		var e ListEntry
		if err := rows.Scan(&e.Hash, &e.Type, &e.Result, &e.FilePath, &e.Pages, &e.CreatedAt, &e.AccessedAt); err != nil {
			return entries, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// Prune removes entries not accessed within the given duration.
// Returns the number of entries removed.
func (c *Cache) Prune(ttl time.Duration) (int64, error) {
	cutoff := time.Now().Add(-ttl).UTC().Format(time.RFC3339)
	res, err := c.db.Exec(
		`DELETE FROM media_cache WHERE accessed_at < ?`, cutoff,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// HashData computes a fast FNV-1a 64-bit hash of the given data,
// returned as a hex string.
func HashData(data []byte) string {
	h := fnv.New64a()
	h.Write(data)
	return fmt.Sprintf("%016x", h.Sum64())
}
