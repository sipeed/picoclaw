package credential

import (
	"sync"
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

func TestSecureStore_ConcurrentSetGet(t *testing.T) {
	s := NewSecureStore()
	const goroutines = 10
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if id%2 == 0 {
					s.SetString("even")
				} else {
					s.SetString("odd")
				}
				_ = s.Get()
			}
		}(i)
	}
	wg.Wait()

	final := s.Get()
	if final != "" && final != "even" && final != "odd" {
		t.Errorf("Get() returned unexpected value %q after concurrent Set/Get", final)
	}
}
