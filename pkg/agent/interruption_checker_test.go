// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"sync"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/stretchr/testify/assert"
)

func TestNewInterruptionChecker(t *testing.T) {
	checker := NewInterruptionChecker()
	assert.NotNil(t, checker)
	assert.False(t, checker.HasPending())
	assert.Equal(t, 0, checker.Len())
}

func TestInterruptionChecker_Signal(t *testing.T) {
	checker := NewInterruptionChecker()

	msg := bus.InboundMessage{
		Channel: "test",
		ChatID:  "123",
		Content: "interrupt",
	}

	checker.Signal(msg)
	assert.True(t, checker.HasPending())
	assert.Equal(t, 1, checker.Len())
}

func TestInterruptionChecker_DrainAll(t *testing.T) {
	checker := NewInterruptionChecker()

	msg1 := bus.InboundMessage{Content: "msg1"}
	msg2 := bus.InboundMessage{Content: "msg2"}
	msg3 := bus.InboundMessage{Content: "msg3"}

	checker.Signal(msg1)
	checker.Signal(msg2)
	checker.Signal(msg3)

	assert.Equal(t, 3, checker.Len())

	drained := checker.DrainAll()
	assert.Len(t, drained, 3)
	assert.Equal(t, "msg1", drained[0].Content)
	assert.Equal(t, "msg2", drained[1].Content)
	assert.Equal(t, "msg3", drained[2].Content)

	// Queue should be empty after drain
	assert.False(t, checker.HasPending())
	assert.Equal(t, 0, checker.Len())

	// Drain empty queue should return nil
	drained2 := checker.DrainAll()
	assert.Nil(t, drained2)
}

func TestInterruptionChecker_Peek(t *testing.T) {
	checker := NewInterruptionChecker()

	// Peek empty queue
	peeked := checker.Peek()
	assert.Nil(t, peeked)

	// Add messages
	msg1 := bus.InboundMessage{Content: "first"}
	msg2 := bus.InboundMessage{Content: "second"}

	checker.Signal(msg1)
	checker.Signal(msg2)

	// Peek should return first message without removing
	peeked = checker.Peek()
	assert.NotNil(t, peeked)
	assert.Equal(t, "first", peeked.Content)

	// Queue should still have 2 messages
	assert.Equal(t, 2, checker.Len())

	// Peek again should return same message
	peeked2 := checker.Peek()
	assert.Equal(t, "first", peeked2.Content)
}

func TestInterruptionChecker_Clear(t *testing.T) {
	checker := NewInterruptionChecker()

	checker.Signal(bus.InboundMessage{Content: "msg1"})
	checker.Signal(bus.InboundMessage{Content: "msg2"})

	assert.Equal(t, 2, checker.Len())

	checker.Clear()
	assert.Equal(t, 0, checker.Len())
	assert.False(t, checker.HasPending())
}

func TestInterruptionChecker_ConcurrentAccess(t *testing.T) {
	checker := NewInterruptionChecker()
	var wg sync.WaitGroup

	// Concurrent signal
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			checker.Signal(bus.InboundMessage{
				Content: "msg",
			})
		}(i)
	}

	// Concurrent drain
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			checker.DrainAll()
		}()
	}

	// Concurrent peek
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			checker.Peek()
		}()
	}

	// Concurrent HasPending
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			checker.HasPending()
		}()
	}

	wg.Wait()

	// Should not panic - test passes if we get here
	assert.True(t, true, "Concurrent access should not cause race conditions")
}

func TestInterruptionChecker_PreservesOrder(t *testing.T) {
	checker := NewInterruptionChecker()

	// Signal messages in order
	for i := 0; i < 10; i++ {
		checker.Signal(bus.InboundMessage{
			Content: string(rune('A' + i)), // A, B, C, ...
		})
	}

	drained := checker.DrainAll()
	assert.Len(t, drained, 10)

	// Verify order is preserved (FIFO)
	for i := 0; i < 10; i++ {
		expected := string(rune('A' + i))
		assert.Equal(t, expected, drained[i].Content)
	}
}

func TestInterruptionChecker_MultipleSignalAndDrain(t *testing.T) {
	checker := NewInterruptionChecker()

	// First batch
	checker.Signal(bus.InboundMessage{Content: "msg1"})
	checker.Signal(bus.InboundMessage{Content: "msg2"})

	batch1 := checker.DrainAll()
	assert.Len(t, batch1, 2)
	assert.False(t, checker.HasPending())

	// Second batch
	checker.Signal(bus.InboundMessage{Content: "msg3"})
	checker.Signal(bus.InboundMessage{Content: "msg4"})
	checker.Signal(bus.InboundMessage{Content: "msg5"})

	batch2 := checker.DrainAll()
	assert.Len(t, batch2, 3)
	assert.False(t, checker.HasPending())

	// Third drain should return nil
	batch3 := checker.DrainAll()
	assert.Nil(t, batch3)
}

func TestInterruptionChecker_SignalAfterDrain(t *testing.T) {
	checker := NewInterruptionChecker()

	// Initial messages
	checker.Signal(bus.InboundMessage{Content: "before1"})
	checker.Signal(bus.InboundMessage{Content: "before2"})

	// Drain
	drained1 := checker.DrainAll()
	assert.Len(t, drained1, 2)

	// Signal new messages after drain
	checker.Signal(bus.InboundMessage{Content: "after1"})
	checker.Signal(bus.InboundMessage{Content: "after2"})

	// Should have new messages
	assert.True(t, checker.HasPending())
	assert.Equal(t, 2, checker.Len())

	drained2 := checker.DrainAll()
	assert.Len(t, drained2, 2)
	assert.Equal(t, "after1", drained2[0].Content)
	assert.Equal(t, "after2", drained2[1].Content)
}

// Benchmark tests
func BenchmarkInterruptionChecker_Signal(b *testing.B) {
	checker := NewInterruptionChecker()
	msg := bus.InboundMessage{Content: "test"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.Signal(msg)
	}
}

func BenchmarkInterruptionChecker_DrainAll(b *testing.B) {
	checker := NewInterruptionChecker()

	// Pre-populate
	for i := 0; i < 10; i++ {
		checker.Signal(bus.InboundMessage{Content: "msg"})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.DrainAll()
		// Repopulate for next iteration
		for j := 0; j < 10; j++ {
			checker.Signal(bus.InboundMessage{Content: "msg"})
		}
	}
}

func BenchmarkInterruptionChecker_HasPending(b *testing.B) {
	checker := NewInterruptionChecker()
	checker.Signal(bus.InboundMessage{Content: "test"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.HasPending()
	}
}

func BenchmarkInterruptionChecker_Concurrent(b *testing.B) {
	checker := NewInterruptionChecker()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			checker.Signal(bus.InboundMessage{Content: "test"})
			checker.HasPending()
			if checker.Len() > 100 {
				checker.DrainAll()
			}
		}
	})
}
