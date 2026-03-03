package server

import "sync"

// LogBuffer is a thread-safe ring buffer that stores the most recent N log lines.
// It supports incremental reads via LinesSince and tracks a runID that increments
// on each Reset (used to detect gateway restarts).
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

// Total returns the total number of lines appended in the current run.
func (b *LogBuffer) Total() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.total
}
