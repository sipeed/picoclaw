package channels

import (
	"errors"
	"fmt"
	"testing"
)

func TestClassifySendError(t *testing.T) {
	raw := fmt.Errorf("some API error")

	tests := []struct {
		name       string
		statusCode int
		wantIs     error
		wantNil    bool
	}{
		{"429 -> ErrRateLimit", 429, ErrRateLimit, false},
		{"500 -> ErrTemporary", 500, ErrTemporary, false},
		{"502 -> ErrTemporary", 502, ErrTemporary, false},
		{"503 -> ErrTemporary", 503, ErrTemporary, false},
		{"400 -> ErrSendFailed", 400, ErrSendFailed, false},
		{"403 -> ErrSendFailed", 403, ErrSendFailed, false},
		{"404 -> ErrSendFailed", 404, ErrSendFailed, false},
		{"200 -> raw error", 200, nil, false},
		{"201 -> raw error", 201, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ClassifySendError(tt.statusCode, raw)
			if err == nil {
				t.Fatal("expected non-nil error")
			}
			if tt.wantIs != nil {
				if !errors.Is(err, tt.wantIs) {
					t.Errorf("errors.Is(err, %v) = false, want true; err = %v", tt.wantIs, err)
				}
			} else {
				// Should return the raw error unchanged
				if err != raw {
					t.Errorf("expected raw error to be returned unchanged for status %d, got %v", tt.statusCode, err)
				}
			}
		})
	}
}

func TestClassifySendErrorNoFalsePositive(t *testing.T) {
	raw := fmt.Errorf("some error")

	// 429 should NOT match ErrTemporary or ErrSendFailed
	err := ClassifySendError(429, raw)
	if errors.Is(err, ErrTemporary) {
		t.Error("429 should not match ErrTemporary")
	}
	if errors.Is(err, ErrSendFailed) {
		t.Error("429 should not match ErrSendFailed")
	}

	// 500 should NOT match ErrRateLimit or ErrSendFailed
	err = ClassifySendError(500, raw)
	if errors.Is(err, ErrRateLimit) {
		t.Error("500 should not match ErrRateLimit")
	}
	if errors.Is(err, ErrSendFailed) {
		t.Error("500 should not match ErrSendFailed")
	}

	// 400 should NOT match ErrRateLimit or ErrTemporary
	err = ClassifySendError(400, raw)
	if errors.Is(err, ErrRateLimit) {
		t.Error("400 should not match ErrRateLimit")
	}
	if errors.Is(err, ErrTemporary) {
		t.Error("400 should not match ErrTemporary")
	}
}

func TestClassifyNetError(t *testing.T) {
	t.Run("nil error returns nil", func(t *testing.T) {
		if err := ClassifyNetError(nil); err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("non-nil error wraps as ErrTemporary", func(t *testing.T) {
		raw := fmt.Errorf("connection refused")
		err := ClassifyNetError(raw)
		if err == nil {
			t.Fatal("expected non-nil error")
		}
		if !errors.Is(err, ErrTemporary) {
			t.Errorf("errors.Is(err, ErrTemporary) = false, want true; err = %v", err)
		}
	})
}
