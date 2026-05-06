package api

import "sync"

// LogBuffer is a thread-safe ring buffer that stores the most recent N log lines.
// It supports incremental reads via LinesSince and tracks a runID that increments
// whenever the buffer is reset or cleared so clients can detect log history resets.
type LogBuffer struct {
	mu    sync.RWMutex
	lines []string
	cap   int
	total int // total lines ever appended in current run
	runID int
}

// NewLogBuffer creates a LogBuffer with the given capacity.
func NewLogBuffer(capacity int) *LogBuffer {
	return &LogBuffer{
		lines: make([]string, 0, capacity),
		cap:   capacity,
	}
}

// Append adds a line to the buffer. If the buffer is full, the oldest line is evicted.
func (b *LogBuffer) Append(line string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.lines) < b.cap {
		b.lines = append(b.lines, line)
	} else {
		b.lines[b.total%b.cap] = line
	}

	b.total++
}

// Reset clears the buffer and increments the runID. Call this when starting a new gateway process.
func (b *LogBuffer) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.lines = b.lines[:0]
	b.total = 0
	b.runID++
}

// Clear removes all buffered lines and increments the runID so clients treat
// subsequent reads as a new log stream.
func (b *LogBuffer) Clear() {
	b.Reset()
}

// LinesSince returns lines appended after the given offset, the current total count, and the runID.
// If offset >= total, no lines are returned. If offset is too old (evicted), all buffered lines are returned.
func (b *LogBuffer) LinesSince(offset int) (lines []string, total int, runID int) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	total = b.total
	runID = b.runID

	if offset >= b.total {
		return nil, total, runID
	}

	buffered := len(b.lines)

	// How many new lines since offset
	newCount := b.total - offset
	if newCount > buffered {
		newCount = buffered
	}

	result := make([]string, newCount)

	if b.total <= b.cap {
		// Buffer hasn't wrapped yet — simple slice
		copy(result, b.lines[buffered-newCount:])
	} else {
		// Buffer has wrapped — read from ring
		start := (b.total - newCount) % b.cap
		for i := range newCount {
			result[i] = b.lines[(start+i)%b.cap]
		}
	}

	return result, total, runID
}

// RunID returns the current run identifier.
func (b *LogBuffer) RunID() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.runID
}

// Tail returns up to the last n buffered lines (newest last).
// If n <= 0, it returns nil.
func (b *LogBuffer) Tail(n int) []string {
	if n <= 0 {
		return nil
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.total == 0 || len(b.lines) == 0 {
		return nil
	}

	if n > len(b.lines) {
		n = len(b.lines)
	}

	result := make([]string, n)

	if b.total <= b.cap {
		copy(result, b.lines[len(b.lines)-n:])
		return result
	}

	start := (b.total - n) % b.cap
	for i := 0; i < n; i++ {
		result[i] = b.lines[(start+i)%b.cap]
	}
	return result
}
