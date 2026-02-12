package vecstore

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"time"
)

// ChunkMarkdown splits markdown text into chunks at semantic boundaries.
// Splits first by ## headers, then sub-splits long sections by paragraphs.
// Each chunk gets a deterministic ID: sha256(source + ":" + text)[:12].
func ChunkMarkdown(source, text string, maxChars int) []Chunk {
	if maxChars <= 0 {
		maxChars = 800
	}

	sections := splitByHeaders(text)
	now := time.Now()

	var chunks []Chunk
	for _, section := range sections {
		section = strings.TrimSpace(section)
		if section == "" {
			continue
		}

		if len(section) <= maxChars {
			chunks = append(chunks, makeChunk(source, section, now))
			continue
		}

		// Sub-split long sections by paragraphs
		for _, part := range splitByParagraphs(section, maxChars) {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			chunks = append(chunks, makeChunk(source, part, now))
		}
	}
	return chunks
}

// splitByHeaders splits text at ## header boundaries, keeping the header with its content.
func splitByHeaders(text string) []string {
	lines := strings.Split(text, "\n")
	var sections []string
	var current strings.Builder

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") && current.Len() > 0 {
			sections = append(sections, current.String())
			current.Reset()
		}
		current.WriteString(line)
		current.WriteByte('\n')
	}
	if current.Len() > 0 {
		sections = append(sections, current.String())
	}
	return sections
}

// splitByParagraphs splits text at double-newline boundaries, respecting maxChars.
func splitByParagraphs(text string, maxChars int) []string {
	paragraphs := strings.Split(text, "\n\n")
	var parts []string
	var current strings.Builder

	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		// If adding this paragraph would exceed max, flush current
		if current.Len() > 0 && current.Len()+len(p)+2 > maxChars {
			parts = append(parts, current.String())
			current.Reset()
		}

		// If a single paragraph exceeds max, just add it as-is
		if current.Len() == 0 && len(p) > maxChars {
			parts = append(parts, p)
			continue
		}

		if current.Len() > 0 {
			current.WriteString("\n\n")
		}
		current.WriteString(p)
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

func makeChunk(source, text string, now time.Time) Chunk {
	return Chunk{
		ID:        chunkID(source, text),
		Text:      text,
		Source:    source,
		UpdatedAt: now,
	}
}

func chunkID(source, text string) string {
	h := sha256.Sum256([]byte(source + ":" + text))
	return fmt.Sprintf("%x", h[:6]) // 12 hex chars
}
