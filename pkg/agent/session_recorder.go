package agent

import (
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/session"
	"github.com/sipeed/picoclaw/pkg/tools"
)

// sessionRecorderImpl bridges tools.SessionRecorder → session.SessionStore.
type sessionRecorderImpl struct {
	adapter *session.LegacyAdapter
}

var _ tools.SessionRecorder = (*sessionRecorderImpl)(nil)

func newSessionRecorder(adapter *session.LegacyAdapter) *sessionRecorderImpl {
	return &sessionRecorderImpl{adapter: adapter}
}

func (r *sessionRecorderImpl) RecordFork(conductorKey, subagentKey, taskID, label string) error {
	store := r.adapter.Store()
	return store.Fork(conductorKey, subagentKey, &session.CreateOpts{
		ForkTurnID: taskID,
		Label:      label,
	})
}

func (r *sessionRecorderImpl) RecordSubagentTurn(subagentKey string, messages []providers.Message) error {
	store := r.adapter.Store()
	turn := &session.Turn{
		Kind:     session.TurnNormal,
		Messages: messages,
	}
	return store.Append(subagentKey, turn)
}

func (r *sessionRecorderImpl) RecordCompletion(subagentKey, status, result string) error {
	store := r.adapter.Store()
	return store.SetStatus(subagentKey, status)
}

func (r *sessionRecorderImpl) RecordReport(conductorKey, subagentKey, senderID, content string) error {
	store := r.adapter.Store()
	turn := &session.Turn{
		Kind:      session.TurnReport,
		OriginKey: subagentKey,
		Author:    senderID,
		Messages: []providers.Message{
			{Role: "user", Content: content},
		},
	}
	if err := store.Append(conductorKey, turn); err != nil {
		return err
	}
	// Advance LegacyAdapter's stored counter so flush loop doesn't double-write.
	r.adapter.AdvanceStored(conductorKey, 1)
	return nil
}
