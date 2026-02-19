package multiagent

import (
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// AnnounceMode determines how a spawn result is delivered to the parent session.
// Inspired by Google Cloud Pub/Sub delivery modes and Microsoft Azure Service Bus.
type AnnounceMode string

const (
	// AnnounceQueue buffers the result until the parent requests it (default).
	// Like Apple's GCD serial queue — ordered, non-blocking.
	AnnounceQueue AnnounceMode = "queue"

	// AnnounceDirect sends the result immediately to the parent's chat channel.
	// Like Google Pub/Sub push subscription — immediate delivery.
	AnnounceDirect AnnounceMode = "direct"
)

// Announcement is a completion notice from a child spawn to its parent.
type Announcement struct {
	FromSessionKey string
	ToSessionKey   string
	RunID          string
	AgentID        string
	Content        string
	Outcome        *SpawnOutcome
	Mode           AnnounceMode
	CreatedAt      time.Time
}

// Announcer manages per-session announcement delivery using Go channels
// (inspired by Apple's Grand Central Dispatch work queues).
// Thread-safe for concurrent producers (multiple child spawns completing
// simultaneously) and a single consumer (parent agent).
type Announcer struct {
	// Per-session buffered channels (Google Pub/Sub topic model).
	channels sync.Map // sessionKey -> chan *Announcement
	bufSize  int
}

// NewAnnouncer creates an announcer with the given per-session buffer size.
// Buffer size follows NVIDIA's double-buffering pattern: enough to absorb
// burst completions without blocking producers.
func NewAnnouncer(bufSize int) *Announcer {
	if bufSize <= 0 {
		bufSize = 32 // default: buffer up to 32 pending announcements
	}
	return &Announcer{bufSize: bufSize}
}

// Deliver sends an announcement to the target session's channel.
// Non-blocking: if the channel is full, the oldest announcement is dropped
// (Meta's back-pressure pattern for high-throughput systems).
func (a *Announcer) Deliver(targetSessionKey string, ann *Announcement) {
	ann.CreatedAt = time.Now()
	if ann.Mode == "" {
		ann.Mode = AnnounceQueue
	}

	ch := a.getOrCreateChan(targetSessionKey)

	select {
	case ch <- ann:
		logger.DebugCF("announce", "Announcement delivered", map[string]interface{}{
			"from":    ann.FromSessionKey,
			"to":      targetSessionKey,
			"run_id":  ann.RunID,
			"agent":   ann.AgentID,
			"mode":    string(ann.Mode),
		})
	default:
		// Channel full — drop oldest to make room (back-pressure).
		select {
		case <-ch:
			logger.WarnCF("announce", "Dropped oldest announcement (buffer full)", map[string]interface{}{
				"session": targetSessionKey,
			})
		default:
		}
		// Retry delivery.
		select {
		case ch <- ann:
		default:
			logger.WarnCF("announce", "Failed to deliver announcement", map[string]interface{}{
				"session": targetSessionKey,
				"run_id":  ann.RunID,
			})
		}
	}
}

// Drain returns all pending announcements for a session, clearing the buffer.
// The parent agent calls this between LLM iterations to collect spawn results.
// Follows Google's batch-pull pattern from Cloud Pub/Sub.
func (a *Announcer) Drain(sessionKey string) []*Announcement {
	v, ok := a.channels.Load(sessionKey)
	if !ok {
		return nil
	}
	ch := v.(chan *Announcement)

	var results []*Announcement
	for {
		select {
		case ann := <-ch:
			results = append(results, ann)
		default:
			return results
		}
	}
}

// Pending returns the number of pending announcements for a session.
func (a *Announcer) Pending(sessionKey string) int {
	v, ok := a.channels.Load(sessionKey)
	if !ok {
		return 0
	}
	return len(v.(chan *Announcement))
}

// Cleanup removes the channel for a session (called on session end).
func (a *Announcer) Cleanup(sessionKey string) {
	a.channels.Delete(sessionKey)
}

func (a *Announcer) getOrCreateChan(sessionKey string) chan *Announcement {
	if v, ok := a.channels.Load(sessionKey); ok {
		return v.(chan *Announcement)
	}
	ch := make(chan *Announcement, a.bufSize)
	actual, _ := a.channels.LoadOrStore(sessionKey, ch)
	return actual.(chan *Announcement)
}
