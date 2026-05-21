package tasks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/fileutil"
)

type Runtime string

const (
	RuntimeSubagent Runtime = "subagent"
	RuntimeDelegate Runtime = "delegate"
	RuntimeTool     Runtime = "tool"
	RuntimeCron     Runtime = "cron"
)

type Status string

const (
	StatusQueued    Status = "queued"
	StatusRunning   Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusTimedOut  Status = "timed_out"
	StatusCancelled Status = "cancelled"
	StatusLost      Status = "lost"
)

type DeliveryStatus string

const (
	DeliveryPending       DeliveryStatus = "pending"
	DeliveryDelivered     DeliveryStatus = "delivered"
	DeliverySessionQueued DeliveryStatus = "session_queued"
	DeliveryFailed        DeliveryStatus = "failed"
	DeliveryParentMissing DeliveryStatus = "parent_missing"
	DeliveryNotApplicable DeliveryStatus = "not_applicable"
)

type NotifyPolicy string

const (
	NotifyDoneOnly     NotifyPolicy = "done_only"
	NotifyStateChanges NotifyPolicy = "state_changes"
	NotifySilent       NotifyPolicy = "silent"
)

const (
	DefaultTerminalRetention = 7 * 24 * time.Hour
	DefaultMaxRecords        = 1000
)

type CompletionPayload struct {
	Text  string            `json:"text,omitempty"`
	Media []CompletionMedia `json:"media,omitempty"`
}

type CompletionMedia struct {
	Ref         string `json:"ref"`
	Type        string `json:"type,omitempty"`
	Filename    string `json:"filename,omitempty"`
	ContentType string `json:"content_type,omitempty"`
}

type Record struct {
	TaskID              string             `json:"task_id"`
	Runtime             Runtime            `json:"runtime"`
	TaskKind            string             `json:"task_kind,omitempty"`
	RequesterSessionKey string             `json:"requester_session_key,omitempty"`
	OwnerKey            string             `json:"owner_key,omitempty"`
	ScopeKind           string             `json:"scope_kind,omitempty"`
	Channel             string             `json:"channel,omitempty"`
	ChatID              string             `json:"chat_id,omitempty"`
	TopicID             string             `json:"topic_id,omitempty"`
	AgentID             string             `json:"agent_id,omitempty"`
	Label               string             `json:"label,omitempty"`
	Task                string             `json:"task"`
	Status              Status             `json:"status"`
	DeliveryStatus      DeliveryStatus     `json:"delivery_status"`
	NotifyPolicy        NotifyPolicy       `json:"notify_policy"`
	DeliveryMode        string             `json:"delivery_mode,omitempty"`
	LastCompletionID    string             `json:"last_completion_id,omitempty"`
	DeliveredAt         int64              `json:"delivered_at,omitempty"`
	DeliveryError       string             `json:"delivery_error,omitempty"`
	CreatedAt           int64              `json:"created_at"`
	StartedAt           int64              `json:"started_at,omitempty"`
	EndedAt             int64              `json:"ended_at,omitempty"`
	LastEventAt         int64              `json:"last_event_at,omitempty"`
	CleanupAfter        int64              `json:"cleanup_after,omitempty"`
	Error               string             `json:"error,omitempty"`
	ProgressSummary     string             `json:"progress_summary,omitempty"`
	TerminalSummary     string             `json:"terminal_summary,omitempty"`
	Completion          *CompletionPayload `json:"completion,omitempty"`
}

type Options struct {
	TerminalRetention time.Duration
	MaxRecords        int
}

type Registry struct {
	mu       sync.RWMutex
	store    string
	options  Options
	records  map[string]Record
	lastLoad error
}

type Snapshot struct {
	Tasks []Record `json:"tasks"`
}

func NewRegistry(storePath string) *Registry {
	return NewRegistryWithOptions(storePath, Options{})
}

func NewRegistryWithOptions(storePath string, opts Options) *Registry {
	if opts.TerminalRetention <= 0 {
		opts.TerminalRetention = DefaultTerminalRetention
	}
	if opts.MaxRecords <= 0 {
		opts.MaxRecords = DefaultMaxRecords
	}
	r := &Registry{
		store:   strings.TrimSpace(storePath),
		options: opts,
		records: make(map[string]Record),
	}
	if r.store != "" {
		r.lastLoad = r.load()
		if r.lastLoad == nil && r.pruneLocked(time.Now().UnixMilli()) {
			_ = r.saveLocked()
		}
	}
	return r
}

func WorkspaceStorePath(workspace string) string {
	workspace = strings.TrimSpace(workspace)
	if workspace == "" {
		return ""
	}
	return filepath.Join(workspace, "state", "task_registry.json")
}

func (r *Registry) LastLoadError() error {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.lastLoad
}

func (r *Registry) Upsert(rec Record) error {
	if r == nil || strings.TrimSpace(rec.TaskID) == "" {
		return nil
	}
	now := time.Now().UnixMilli()
	if rec.CreatedAt == 0 {
		rec.CreatedAt = now
	}
	if rec.LastEventAt == 0 {
		rec.LastEventAt = now
	}
	if rec.Status == "" {
		rec.Status = StatusQueued
	}
	if rec.DeliveryStatus == "" {
		rec.DeliveryStatus = DeliveryPending
	}
	if rec.NotifyPolicy == "" {
		rec.NotifyPolicy = NotifyDoneOnly
	}
	if rec.Runtime == "" {
		rec.Runtime = RuntimeTool
	}
	rec = r.normalizeRecord(rec, now)

	r.mu.Lock()
	r.records[rec.TaskID] = rec
	r.pruneLocked(now)
	err := r.saveLocked()
	r.mu.Unlock()
	return err
}

func (r *Registry) Update(taskID string, mutate func(*Record)) error {
	if r == nil || strings.TrimSpace(taskID) == "" || mutate == nil {
		return nil
	}
	r.mu.Lock()
	rec, ok := r.records[taskID]
	if !ok {
		r.mu.Unlock()
		return fmt.Errorf("task %q not found", taskID)
	}
	mutate(&rec)
	if rec.LastEventAt == 0 {
		rec.LastEventAt = time.Now().UnixMilli()
	}
	rec = r.normalizeRecord(rec, rec.LastEventAt)
	r.records[taskID] = rec
	r.pruneLocked(rec.LastEventAt)
	err := r.saveLocked()
	r.mu.Unlock()
	return err
}

func (r *Registry) Get(taskID string) (Record, bool) {
	if r == nil {
		return Record{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	rec, ok := r.records[taskID]
	return rec, ok
}

func (r *Registry) List() []Record {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Record, 0, len(r.records))
	for _, rec := range r.records {
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt != out[j].CreatedAt {
			return out[i].CreatedAt < out[j].CreatedAt
		}
		return out[i].TaskID < out[j].TaskID
	})
	return out
}

func (r *Registry) ListPendingTerminalDelivery() []Record {
	if r == nil {
		return nil
	}
	records := r.List()
	out := make([]Record, 0)
	for _, rec := range records {
		if rec.DeliveryStatus == DeliveryPending && isTerminalStatus(rec.Status) {
			out = append(out, rec)
		}
	}
	return out
}

func (r *Registry) MaxNumericSuffix(prefix string) int {
	maxSeq := 0
	for _, rec := range r.List() {
		if !strings.HasPrefix(rec.TaskID, prefix) {
			continue
		}
		n, err := strconv.Atoi(strings.TrimPrefix(rec.TaskID, prefix))
		if err == nil && n > maxSeq {
			maxSeq = n
		}
	}
	return maxSeq
}

func (r *Registry) normalizeRecord(rec Record, now int64) Record {
	if r == nil {
		return rec
	}
	if now == 0 {
		now = time.Now().UnixMilli()
	}
	if isTerminalStatus(rec.Status) && rec.EndedAt == 0 {
		rec.EndedAt = rec.LastEventAt
		if rec.EndedAt == 0 {
			rec.EndedAt = now
		}
	}
	if isTerminalStatus(rec.Status) && rec.CleanupAfter == 0 {
		base := recordReferenceAt(rec)
		if base == 0 {
			base = now
		}
		rec.CleanupAfter = base + int64(r.options.TerminalRetention/time.Millisecond)
	}
	return rec
}

func (r *Registry) pruneLocked(now int64) bool {
	if r == nil || len(r.records) == 0 {
		return false
	}
	changed := false
	for id, rec := range r.records {
		if shouldPruneExpired(rec, now) {
			delete(r.records, id)
			changed = true
		}
	}
	if r.options.MaxRecords <= 0 || len(r.records) <= r.options.MaxRecords {
		return changed
	}
	terminal := make([]Record, 0, len(r.records))
	for _, rec := range r.records {
		if isTerminalStatus(rec.Status) {
			terminal = append(terminal, rec)
		}
	}
	sort.Slice(terminal, func(i, j int) bool {
		return recordReferenceAt(terminal[i]) < recordReferenceAt(terminal[j])
	})
	for len(r.records) > r.options.MaxRecords && len(terminal) > 0 {
		victim := terminal[0]
		terminal = terminal[1:]
		delete(r.records, victim.TaskID)
		changed = true
	}
	return changed
}

func shouldPruneExpired(rec Record, now int64) bool {
	return isTerminalStatus(rec.Status) && rec.CleanupAfter > 0 && now >= rec.CleanupAfter
}

func isTerminalStatus(status Status) bool {
	switch status {
	case StatusSucceeded, StatusFailed, StatusTimedOut, StatusCancelled, StatusLost:
		return true
	default:
		return false
	}
}

func recordReferenceAt(rec Record) int64 {
	for _, value := range []int64{rec.EndedAt, rec.LastEventAt, rec.StartedAt, rec.CreatedAt} {
		if value > 0 {
			return value
		}
	}
	return 0
}

func (r *Registry) load() error {
	data, err := os.ReadFile(r.store)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	for _, rec := range snap.Tasks {
		if strings.TrimSpace(rec.TaskID) == "" {
			continue
		}
		r.records[rec.TaskID] = r.normalizeRecord(rec, now)
	}
	return nil
}

func (r *Registry) saveLocked() error {
	if r.store == "" {
		return nil
	}
	tasks := make([]Record, 0, len(r.records))
	for _, rec := range r.records {
		tasks = append(tasks, rec)
	}
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].CreatedAt != tasks[j].CreatedAt {
			return tasks[i].CreatedAt < tasks[j].CreatedAt
		}
		return tasks[i].TaskID < tasks[j].TaskID
	})
	data, err := json.MarshalIndent(Snapshot{Tasks: tasks}, "", "  ")
	if err != nil {
		return err
	}
	return fileutil.WriteFileAtomic(r.store, data, 0o600)
}
