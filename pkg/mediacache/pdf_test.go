package mediacache

import (
	"os"
	"path/filepath"
	"testing"
)

// minimalPDF is a valid PDF with 3 pages.
// This is the smallest possible multi-page PDF structure.
const minimalPDF = `%PDF-1.4
1 0 obj <</Type /Catalog /Pages 2 0 R>> endobj
2 0 obj <</Type /Pages /Kids [3 0 R 4 0 R 5 0 R] /Count 3>> endobj
3 0 obj <</Type /Page /Parent 2 0 R /MediaBox [0 0 612 792]>> endobj
4 0 obj <</Type /Page /Parent 2 0 R /MediaBox [0 0 612 792]>> endobj
5 0 obj <</Type /Page /Parent 2 0 R /MediaBox [0 0 612 792]>> endobj
xref
0 6
trailer <</Size 6 /Root 1 0 R>>
startxref
0
%%EOF`

func TestPDFPageCount_ValidPDF(t *testing.T) {
	path := writeTempFile(t, "test.pdf", minimalPDF)
	count := PDFPageCount(path)
	if count != 3 {
		t.Errorf("PDFPageCount = %d, want 3", count)
	}
}

func TestPDFPageCount_NonExistent(t *testing.T) {
	count := PDFPageCount("/nonexistent/file.pdf")
	if count != 0 {
		t.Errorf("PDFPageCount = %d, want 0 for missing file", count)
	}
}

func TestPDFPageCount_NotPDF(t *testing.T) {
	path := writeTempFile(t, "test.txt", "hello world")
	count := PDFPageCount(path)
	if count != 0 {
		t.Errorf("PDFPageCount = %d, want 0 for non-PDF", count)
	}
}

func TestPDFPageCount_SinglePage(t *testing.T) {
	pdf := `%PDF-1.4
1 0 obj <</Type /Catalog /Pages 2 0 R>> endobj
2 0 obj <</Type /Pages /Kids [3 0 R] /Count 1>> endobj
3 0 obj <</Type /Page /Parent 2 0 R /MediaBox [0 0 612 792]>> endobj
xref
0 4
trailer <</Size 4 /Root 1 0 R>>
startxref
0
%%EOF`
	path := writeTempFile(t, "single.pdf", pdf)
	count := PDFPageCount(path)
	if count != 1 {
		t.Errorf("PDFPageCount = %d, want 1", count)
	}
}

func TestFormatPageCount(t *testing.T) {
	if s := FormatPageCount(0); s != "?" {
		t.Errorf("FormatPageCount(0) = %q, want %q", s, "?")
	}
	if s := FormatPageCount(18); s != "18" {
		t.Errorf("FormatPageCount(18) = %q, want %q", s, "18")
	}
}

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}
