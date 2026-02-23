package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// NoteMeta represents parsed frontmatter metadata from a single markdown note.
type NoteMeta struct {
	Title   string
	Created string
	Updated string
	Tags    []string
	Aliases []string
	RelPath string
	Links   []string
}

// Vault manages the memory vault: scanning, indexing, and searching notes.
type Vault struct {
	memoryDir string
}

// NewVault creates a new Vault rooted at the given memory directory.
func NewVault(memoryDir string) *Vault {
	return &Vault{memoryDir: memoryDir}
}

// ParseFrontmatter extracts YAML frontmatter from markdown content.
// Frontmatter is delimited by "---\n" at the start and a subsequent "\n---".
// Returns the parsed metadata and the body (everything after the closing delimiter).
// If no valid frontmatter is found, returns a zero NoteMeta and the full content as body.
func ParseFrontmatter(content string) (NoteMeta, string) {
	if !strings.HasPrefix(content, "---\n") && !strings.HasPrefix(content, "---\r\n") {
		return NoteMeta{}, content
	}

	// Find closing delimiter after the opening "---\n"
	rest := content[4:]
	endIdx := strings.Index(rest, "\n---")
	if endIdx < 0 {
		return NoteMeta{}, content
	}

	fmBlock := rest[:endIdx]
	body := rest[endIdx+4:] // skip "\n---"

	meta := NoteMeta{}
	for _, line := range strings.Split(fmBlock, "\n") {
		line = strings.TrimSpace(line)
		colonIdx := strings.Index(line, ":")
		if colonIdx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:colonIdx])
		val := strings.TrimSpace(line[colonIdx+1:])

		switch key {
		case "title":
			meta.Title = val
		case "created":
			meta.Created = val
		case "updated":
			meta.Updated = val
		case "tags":
			meta.Tags = parseBracketList(val)
		case "aliases":
			meta.Aliases = parseBracketList(val)
		}
	}

	return meta, body
}

// parseBracketList parses a bracket-delimited list like "[a, b, c]" into a string slice.
// Also handles plain comma-separated values without brackets.
// Returns nil for empty input or empty brackets.
func parseBracketList(s string) []string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// ScanAll walks the memory directory recursively and returns metadata for all
// markdown notes. It skips _index.md and non-.md files. Notes without
// frontmatter get their title inferred from the filename.
func (v *Vault) ScanAll() ([]NoteMeta, error) {
	var notes []NoteMeta
	err := filepath.WalkDir(v.memoryDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		rel, _ := filepath.Rel(v.memoryDir, path)
		rel = filepath.ToSlash(rel)
		if rel == "_index.md" {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil // skip unreadable files
		}
		meta, body := ParseFrontmatter(string(content))
		meta.RelPath = rel
		meta.Links = ExtractWikilinks(body)
		if meta.Title == "" {
			base := filepath.Base(rel)
			meta.Title = strings.TrimSuffix(base, ".md")
		}
		notes = append(notes, meta)
		return nil
	})
	return notes, err
}

// RebuildIndex performs a full scan of all notes and regenerates _index.md.
func (v *Vault) RebuildIndex() error {
	notes, err := v.ScanAll()
	if err != nil {
		return err
	}
	return v.writeIndex(notes)
}

// ReadIndex reads and returns the current _index.md content.
// Returns an empty string if the index file does not exist.
func (v *Vault) ReadIndex() string {
	data, err := os.ReadFile(filepath.Join(v.memoryDir, "_index.md"))
	if err != nil {
		return ""
	}
	return string(data)
}

// writeIndex generates _index.md from the given notes list.
func (v *Vault) writeIndex(notes []NoteMeta) error {
	// Sort notes by updated date (newest first), then by title
	sort.Slice(notes, func(i, j int) bool {
		if notes[i].Updated != notes[j].Updated {
			return notes[i].Updated > notes[j].Updated
		}
		return notes[i].Title < notes[j].Title
	})

	// Collect unique tags and count
	tagNotes := make(map[string][]string) // tag -> list of paths
	for _, n := range notes {
		for _, tag := range n.Tags {
			tagNotes[tag] = append(tagNotes[tag], n.RelPath)
		}
	}

	// Collect aliases
	type aliasEntry struct {
		alias string
		path  string
	}
	var aliases []aliasEntry
	for _, n := range notes {
		for _, a := range n.Aliases {
			aliases = append(aliases, aliasEntry{alias: a, path: n.RelPath})
		}
	}

	var sb strings.Builder
	sb.WriteString("# Memory Vault Index\n")
	sb.WriteString(fmt.Sprintf("<!-- Auto-generated. Do not edit manually. -->\n"))
	sb.WriteString(fmt.Sprintf("<!-- Last rebuilt: %s -->\n", time.Now().Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("<!-- Notes: %d | Tags: %d -->\n", len(notes), len(tagNotes)))

	// Notes table
	sb.WriteString("\n## Notes\n\n")
	sb.WriteString("| Title | Path | Tags | Updated |\n")
	sb.WriteString("|-------|------|------|----------|\n")
	for _, n := range notes {
		tags := strings.Join(n.Tags, ", ")
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", n.Title, n.RelPath, tags, n.Updated))
	}

	// Tags section
	sb.WriteString("\n## Tags\n\n")
	sortedTags := make([]string, 0, len(tagNotes))
	for tag := range tagNotes {
		sortedTags = append(sortedTags, tag)
	}
	sort.Strings(sortedTags)
	for _, tag := range sortedTags {
		paths := tagNotes[tag]
		sb.WriteString(fmt.Sprintf("- **%s** (%d): %s\n", tag, len(paths), strings.Join(paths, ", ")))
	}

	// Aliases section
	if len(aliases) > 0 {
		sb.WriteString("\n## Aliases\n\n")
		sort.Slice(aliases, func(i, j int) bool {
			return aliases[i].alias < aliases[j].alias
		})
		for _, a := range aliases {
			sb.WriteString(fmt.Sprintf("- %s -> %s\n", a.alias, a.path))
		}
	}

	indexPath := filepath.Join(v.memoryDir, "_index.md")
	return os.WriteFile(indexPath, []byte(sb.String()), 0o644)
}

// ExtractWikilinks finds all [[target]] references in body text.
// Returns a slice of link targets with the brackets stripped.
func ExtractWikilinks(body string) []string {
	var links []string
	for {
		start := strings.Index(body, "[[")
		if start < 0 {
			break
		}
		end := strings.Index(body[start+2:], "]]")
		if end < 0 {
			break
		}
		link := body[start+2 : start+2+end]
		links = append(links, link)
		body = body[start+2+end+2:]
	}
	return links
}
