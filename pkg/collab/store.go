package collab

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/fileutil"
)

type stateFile struct {
	Threads   map[string]*Thread   `json:"threads,omitempty"`
	Messages  map[string]*Envelope `json:"messages,omitempty"`
	Mailboxes map[string]*Mailbox  `json:"mailboxes,omitempty"`
}

type Store struct {
	mu    sync.RWMutex
	path  string
	state stateFile
}

func NewStore(dir string) (*Store, error) {
	store := &Store{
		state: stateFile{
			Threads:   make(map[string]*Thread),
			Messages:  make(map[string]*Envelope),
			Mailboxes: make(map[string]*Mailbox),
		},
	}

	dir = strings.TrimSpace(dir)
	if dir == "" {
		return store, nil
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	store.path = filepath.Join(dir, "state.json")
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) load() error {
	if strings.TrimSpace(s.path) == "" {
		return nil
	}
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}

	var state stateFile
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}
	if state.Threads == nil {
		state.Threads = make(map[string]*Thread)
	}
	if state.Messages == nil {
		state.Messages = make(map[string]*Envelope)
	}
	if state.Mailboxes == nil {
		state.Mailboxes = make(map[string]*Mailbox)
	}
	s.state = state
	return nil
}

func (s *Store) saveLocked() error {
	if strings.TrimSpace(s.path) == "" {
		return nil
	}
	data, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return err
	}
	return fileutil.WriteFileAtomic(s.path, data, 0o600)
}

func (s *Store) EnsureThread(thread Thread) (Thread, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	thread.ID = strings.TrimSpace(thread.ID)
	if thread.ID == "" {
		return Thread{}, errors.New("thread id is required")
	}

	existing, ok := s.state.Threads[thread.ID]
	if !ok {
		cloned := CloneThread(thread)
		if cloned.Status == "" {
			cloned.Status = ThreadStatusOpen
		}
		if cloned.CreatedAt.IsZero() {
			cloned.CreatedAt = now
		}
		cloned.UpdatedAt = now
		existing = &cloned
		s.state.Threads[thread.ID] = existing
	} else {
		existing.ParticipantAgentIDs = MergeParticipants(existing.ParticipantAgentIDs, thread.ParticipantAgentIDs...)
		if thread.Status != "" {
			existing.Status = thread.Status
		}
		if strings.TrimSpace(thread.CloseReason) != "" {
			existing.CloseReason = thread.CloseReason
		}
		existing.UpdatedAt = now
	}

	if err := s.saveLocked(); err != nil {
		return Thread{}, err
	}
	return CloneThread(*existing), nil
}

func (s *Store) AddMessage(env Envelope) (Envelope, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	env.ID = strings.TrimSpace(env.ID)
	env.ThreadID = strings.TrimSpace(env.ThreadID)
	env.FromAgentID = strings.TrimSpace(env.FromAgentID)
	if env.ID == "" || env.ThreadID == "" || env.FromAgentID == "" {
		return Envelope{}, errors.New("id, thread_id, and from_agent_id are required")
	}
	if env.CreatedAt.IsZero() {
		env.CreatedAt = now
	}
	env.UpdatedAt = now
	switch env.Status {
	case StatusDelivered:
		if env.DeliveredAt == nil {
			delivered := now
			env.DeliveredAt = &delivered
		}
	case StatusReceived:
		if env.DeliveredAt == nil {
			delivered := now
			env.DeliveredAt = &delivered
		}
		if env.ReceivedAt == nil {
			received := now
			env.ReceivedAt = &received
		}
	case StatusReplied:
		if env.RepliedAt == nil {
			replied := now
			env.RepliedAt = &replied
		}
	}

	cloned := CloneEnvelope(env)
	s.state.Messages[env.ID] = &cloned

	thread := s.state.Threads[env.ThreadID]
	if thread == nil {
		thread = &Thread{
			ID:        env.ThreadID,
			Status:    ThreadStatusOpen,
			CreatedAt: env.CreatedAt,
		}
		s.state.Threads[env.ThreadID] = thread
	}
	thread.ParticipantAgentIDs = MergeParticipants(thread.ParticipantAgentIDs, env.FromAgentID)
	thread.ParticipantAgentIDs = MergeParticipants(thread.ParticipantAgentIDs, env.ToAgentIDs...)
	if !contains(thread.MessageIDs, env.ID) {
		thread.MessageIDs = append(thread.MessageIDs, env.ID)
	}
	thread.LastMessageID = env.ID
	thread.UpdatedAt = now

	for _, agentID := range env.ToAgentIDs {
		agentID = strings.TrimSpace(agentID)
		if agentID == "" {
			continue
		}
		mailbox := s.state.Mailboxes[agentID]
		if mailbox == nil {
			mailbox = &Mailbox{AgentID: agentID}
			s.state.Mailboxes[agentID] = mailbox
		}
		if !contains(mailbox.MessageIDs, env.ID) {
			mailbox.MessageIDs = append(mailbox.MessageIDs, env.ID)
		}
		mailbox.UpdatedAt = now
	}

	if err := s.saveLocked(); err != nil {
		return Envelope{}, err
	}
	return CloneEnvelope(cloned), nil
}

func (s *Store) UpdateMessageStatus(id string, status MessageStatus, errorSummary string) (Envelope, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	msg := s.state.Messages[strings.TrimSpace(id)]
	if msg == nil {
		return Envelope{}, errors.New("message not found")
	}

	now := time.Now().UTC()
	msg.Status = status
	msg.UpdatedAt = now
	if strings.TrimSpace(errorSummary) != "" {
		msg.ErrorSummary = strings.TrimSpace(errorSummary)
	}
	switch status {
	case StatusDelivered:
		msg.DeliveredAt = &now
	case StatusReceived:
		msg.ReceivedAt = &now
	case StatusReplied:
		msg.RepliedAt = &now
	}

	if err := s.saveLocked(); err != nil {
		return Envelope{}, err
	}
	return CloneEnvelope(*msg), nil
}

func (s *Store) GetMessage(id string) (Envelope, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	msg := s.state.Messages[strings.TrimSpace(id)]
	if msg == nil {
		return Envelope{}, false
	}
	return CloneEnvelope(*msg), true
}

func (s *Store) GetThread(id string) (Thread, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	thread := s.state.Threads[strings.TrimSpace(id)]
	if thread == nil {
		return Thread{}, false
	}
	return CloneThread(*thread), true
}

func (s *Store) ListThreadMessages(threadID string) []Envelope {
	s.mu.RLock()
	defer s.mu.RUnlock()

	thread := s.state.Threads[strings.TrimSpace(threadID)]
	if thread == nil {
		return nil
	}

	out := make([]Envelope, 0, len(thread.MessageIDs))
	for _, messageID := range thread.MessageIDs {
		if msg := s.state.Messages[messageID]; msg != nil {
			out = append(out, CloneEnvelope(*msg))
		}
	}
	return out
}

func (s *Store) ListMailbox(agentID, threadID, status string) []Envelope {
	s.mu.RLock()
	defer s.mu.RUnlock()

	mailbox := s.state.Mailboxes[strings.TrimSpace(agentID)]
	if mailbox == nil {
		return nil
	}

	threadID = strings.TrimSpace(threadID)
	status = strings.TrimSpace(strings.ToLower(status))

	out := make([]Envelope, 0, len(mailbox.MessageIDs))
	for _, messageID := range mailbox.MessageIDs {
		msg := s.state.Messages[messageID]
		if msg == nil {
			continue
		}
		if threadID != "" && msg.ThreadID != threadID {
			continue
		}
		if status != "" && status != "all" && strings.ToLower(string(msg.Status)) != status {
			continue
		}
		out = append(out, CloneEnvelope(*msg))
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out
}

func (s *Store) FindResponseTo(threadID, parentID string) (Envelope, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	thread := s.state.Threads[strings.TrimSpace(threadID)]
	if thread == nil {
		return Envelope{}, false
	}
	for _, messageID := range thread.MessageIDs {
		msg := s.state.Messages[messageID]
		if msg == nil || msg.ParentID != parentID {
			continue
		}
		switch msg.Kind {
		case KindReply, KindStatus, KindCancel:
			return CloneEnvelope(*msg), true
		}
	}
	return Envelope{}, false
}

func (s *Store) ThreadMessageCount(threadID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	thread := s.state.Threads[strings.TrimSpace(threadID)]
	if thread == nil {
		return 0
	}
	return len(thread.MessageIDs)
}

func (s *Store) CloseThread(threadID, reason string) (Thread, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	thread := s.state.Threads[strings.TrimSpace(threadID)]
	if thread == nil {
		return Thread{}, errors.New("thread not found")
	}
	thread.Status = ThreadStatusClosed
	thread.CloseReason = strings.TrimSpace(reason)
	thread.UpdatedAt = time.Now().UTC()
	if err := s.saveLocked(); err != nil {
		return Thread{}, err
	}
	return CloneThread(*thread), nil
}

func contains(items []string, needle string) bool {
	for _, item := range items {
		if item == needle {
			return true
		}
	}
	return false
}
