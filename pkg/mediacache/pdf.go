package mediacache

import (
	"os"
	"regexp"
	"strconv"
)

// pdfPageCountRe matches /Type /Pages ... /Count N in PDF cross-reference.
// This covers the vast majority of well-formed PDFs.
var pdfPageCountRe = regexp.MustCompile(`/Type\s*/Pages\b[^>]*/Count\s+(\d+)`)

// PDFPageCount extracts the total page count from a PDF file by parsing
// the /Type /Pages dictionary. Returns 0 if the count cannot be determined
// (encrypted, malformed, or unusual structure). This is a best-effort
// extraction that avoids heavy PDF library dependencies.
func PDFPageCount(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	// Search from the end of the file where the root Pages dict typically lives.
	// Limit search to last 64KB for performance on large files.
	searchStart := 0
	if len(data) > 64*1024 {
		searchStart = len(data) - 64*1024
	}
	matches := pdfPageCountRe.FindAllSubmatch(data[searchStart:], -1)
	if len(matches) == 0 {
		// Fallback: search entire file
		matches = pdfPageCountRe.FindAllSubmatch(data, -1)
	}
	if len(matches) == 0 {
		return 0
	}
	// Use the largest /Count found (root Pages object has the total).
	var maxCount int
	for _, m := range matches {
		if n, err := strconv.Atoi(string(m[1])); err == nil && n > maxCount {
			maxCount = n
		}
	}
	return maxCount
}

// FormatPageCount returns pages as a string, or "?" if unknown.
func FormatPageCount(pages int) string {
	if pages > 0 {
		return strconv.Itoa(pages)
	}
	return "?"
}
