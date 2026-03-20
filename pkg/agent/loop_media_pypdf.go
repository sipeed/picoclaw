package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/mediacache"
)

// minTextDensity is the minimum ratio of non-whitespace characters to total
// runes for extracted text to be considered "real" (not garbage/metadata).
const minTextDensity = 0.3

// minTextLength is the minimum total rune count of extracted text for the
// fast path to succeed. Very short texts likely mean the PDF is image-based.
const minTextLength = 100

// pdftotextTimeout is the maximum time for pdftotext text extraction.
const pdftotextTimeout = 30 * time.Second

// tryPdftotextExtract attempts to extract text from a PDF using pdftotext
// (poppler-utils). Returns the extracted text, page count, and true if
// successful. Returns "", 0, false if pdftotext is unavailable, PDF has
// no text layer, or the extracted text is too short/garbage.
func tryPdftotextExtract(ctx context.Context, pdfPath string) (text string, pages int, ok bool) {
	cmdCtx, cancel := context.WithTimeout(ctx, pdftotextTimeout)
	defer cancel()

	// Extract text to stdout. -layout preserves reading order.
	cmd := exec.CommandContext(cmdCtx, "pdftotext", "-layout", pdfPath, "-")
	out, err := cmd.Output()
	if err != nil {
		logger.DebugCF("agent", "pdftotext extraction failed", map[string]any{
			"path":  pdfPath,
			"error": err.Error(),
		})
		return "", 0, false
	}

	extracted := strings.TrimSpace(string(out))

	// Get page count via pdfinfo
	pages = pdfinfoPageCount(ctx, pdfPath)

	// Check quality: too short or too sparse → probably image-based PDF
	runes := []rune(extracted)
	if len(runes) < minTextLength {
		logger.DebugCF("agent", "pdftotext: text too short, falling back to OCR", map[string]any{
			"path":  pdfPath,
			"runes": len(runes),
		})
		return "", pages, false
	}

	nonSpace := 0
	for _, r := range runes {
		if !unicode.IsSpace(r) {
			nonSpace++
		}
	}
	density := float64(nonSpace) / float64(len(runes))
	if density < minTextDensity {
		logger.DebugCF("agent", "pdftotext: low text density, falling back to OCR", map[string]any{
			"path":    pdfPath,
			"density": fmt.Sprintf("%.2f", density),
		})
		return "", pages, false
	}

	logger.InfoCF("agent", "pdftotext: text extraction successful", map[string]any{
		"path":  pdfPath,
		"pages": pages,
		"runes": len(runes),
	})

	return extracted, pages, true
}

// pdfinfoPageCount runs `pdfinfo` to get the page count of a PDF.
// Returns 0 if pdfinfo is unavailable or fails.
func pdfinfoPageCount(ctx context.Context, pdfPath string) int {
	cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "pdfinfo", pdfPath)
	out, err := cmd.Output()
	if err != nil {
		return 0
	}

	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "Pages:") {
			field := strings.TrimSpace(strings.TrimPrefix(line, "Pages:"))
			if n, err := strconv.Atoi(field); err == nil {
				return n
			}
		}
	}
	return 0
}

// savePdftotextResult writes extracted text to a .md file in the per-PDF
// output directory and caches it. Returns the document tag string.
func (al *AgentLoop) savePdftotextResult(
	pdfPath, text string, pages int, hash, outputDir string,
) string {
	os.MkdirAll(outputDir, 0o755)

	mdPath := filepath.Join(outputDir, "document.md")

	if err := os.WriteFile(mdPath, []byte(text), 0o644); err != nil {
		logger.WarnCF("agent", "Failed to write pdftotext output", map[string]any{
			"path":  mdPath,
			"error": err.Error(),
		})
		return fmt.Sprintf("[file:%s]", pdfPath)
	}

	preview := extractPreview(text, maxPreviewRunes)

	// Cache the result
	if al.mediaCache != nil {
		_ = al.mediaCache.PutEntry(hash, mediacache.TypePDFText, mediacache.Entry{
			Result:   preview,
			FilePath: mdPath,
			Pages:    pages,
		})
	}

	return formatDocumentTag(preview, mdPath, pages)
}
