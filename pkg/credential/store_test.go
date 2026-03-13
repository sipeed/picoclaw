package credential

import (
	"testing"
)

func TestSecureStore_SetGet(t *testing.T) {
	s := NewSecureStore()
	if s.IsSet() {
		t.Error("expected empty store")
	}

	s.SetString("hunter2")
	if !s.IsSet() {
		t.Error("expected store to be set")
	}
	if got := s.Get(); got != "hunter2" {
		t.Errorf("Get() = %q, want %q", got, "hunter2")
	}
}

func TestSecureStore_Clear(t *testing.T) {
	s := NewSecureStore()
	s.SetString("secret")
	s.Clear()

	if s.IsSet() {
		t.Error("expected store to be empty after Clear()")
	}
	if got := s.Get(); got != "" {
		t.Errorf("Get() after Clear() = %q, want empty", got)
	}
}

func TestSecureStore_SetOverwrites(t *testing.T) {
	s := NewSecureStore()
	s.SetString("first")
	s.SetString("second")

	if got := s.Get(); got != "second" {
		t.Errorf("Get() = %q, want %q", got, "second")
	}
}

func TestSecureStore_EmptyPassphrase(t *testing.T) {
	s := NewSecureStore()
	s.SetString("") // empty → should not mark as set

	if s.IsSet() {
		t.Error("empty passphrase should not mark store as set")
	}
}

func TestSecureStore_GetReturnsCopy(t *testing.T) {
	s := NewSecureStore()
	s.SetString("abc")

	// Mutating the returned string bytes (not possible in Go, but we verify
	// that a second Get() is not affected by the first).
	got1 := s.Get()
	got2 := s.Get()
	if got1 != got2 {
		t.Errorf("successive Get() calls returned different values: %q vs %q", got1, got2)
	}
}
