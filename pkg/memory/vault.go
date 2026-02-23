package memory

import "strings"

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
