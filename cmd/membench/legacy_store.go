package main

import (
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/session"
)

// LegacyStore wraps session.SessionManager for legacy baseline.
type LegacyStore struct {
	sm *session.SessionManager
}

// NewLegacyStore creates a new in-memory session manager.
func NewLegacyStore() *LegacyStore {
	return &LegacyStore{
		sm: session.NewSessionManager(""),
	}
}

// IngestSample loads all turns from a LOCOMO sample into the legacy session store.
func (ls *LegacyStore) IngestSample(sample *LocomoSample) {
	sessionKey := "locomo-" + sample.SampleID
	turns := GetTurns(sample)
	for _, turn := range turns {
		content := turn.Speaker + ": " + turn.Text
		ls.sm.AddMessage(sessionKey, "user", content)
	}
}

// GetHistory returns all messages for a sample's session.
func (ls *LegacyStore) GetHistory(sampleID string) []providers.Message {
	sessionKey := "locomo-" + sampleID
	return ls.sm.GetHistory(sessionKey)
}
