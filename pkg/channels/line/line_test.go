package line

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebhookRejectsOversizedBody(t *testing.T) {
	ch := &LINEChannel{}

	// Create a body larger than maxWebhookBodySize (1 MB)
	oversized := bytes.Repeat([]byte("A"), (1<<20)+1)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(oversized))
	rec := httptest.NewRecorder()

	ch.webhookHandler(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status %d, got %d", http.StatusRequestEntityTooLarge, rec.Code)
	}
}

func TestWebhookRejectsNonPostMethod(t *testing.T) {
	ch := &LINEChannel{}

	req := httptest.NewRequest(http.MethodGet, "/webhook", nil)
	rec := httptest.NewRecorder()

	ch.webhookHandler(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestWebhookRejectsInvalidSignature(t *testing.T) {
	ch := &LINEChannel{}

	body := `{"events":[]}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Line-Signature", "invalidsignature")
	rec := httptest.NewRecorder()

	ch.webhookHandler(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}
