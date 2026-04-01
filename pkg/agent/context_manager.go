package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// ContextManager manages conversation context via a pluggable strategy.
// Exactly ONE ContextManager is active per AgentLoop, selected by config.
// The default ("legacy") preserves current summarization behavior.
type ContextManager interface {
	// Assemble builds budget-aware context from the ContextManager's own storage.
	// Called before BuildMessages. Returns assembled messages ready for LLM.
	Assemble(ctx context.Context, req *AssembleRequest) (*AssembleResponse, error)

	// Compact compresses conversation history.
	// Called after turn completes (may be async internally) and on context overflow (sync).
	Compact(ctx context.Context, req *CompactRequest) error

	// Ingest records a message into the ContextManager's own storage.
	// Called after each message is persisted to session JSONL.
	Ingest(ctx context.Context, req *IngestRequest) error
}

// AssembleRequest is the input to Assemble.
type AssembleRequest struct {
	SessionKey string // session identifier
	Budget     int    // context window in tokens
	MaxTokens  int    // max response tokens
}

// AssembleResponse is the output of Assemble.
type AssembleResponse struct {
	History []providers.Message // assembled conversation history for BuildMessages
	Summary string              // conversation summary embedded into system prompt by BuildMessages
}

// CompactRequest is the input to Compact.
type CompactRequest struct {
	SessionKey string                // session identifier
	Reason     ContextCompressReason // proactive_budget | llm_retry | summarize
}

// IngestRequest is the input to Ingest.
type IngestRequest struct {
	SessionKey string            // session identifier
	Message    providers.Message // the message just persisted
}

// ContextManagerFactory constructs a ContextManager from config.
// al provides access to the AgentLoop's runtime resources (provider, model, workspace, etc.)
// cfg is the raw JSON configuration from config.json (may be nil).
type ContextManagerFactory func(cfg json.RawMessage, al *AgentLoop) (ContextManager, error)

var (
	cmRegistryMu sync.RWMutex
	cmRegistry   = map[string]ContextManagerFactory{}
)

// RegisterContextManager registers a named ContextManager factory.
func RegisterContextManager(name string, factory ContextManagerFactory) error {
	if name == "" {
		return fmt.Errorf("context manager name is required")
	}
	if factory == nil {
		return fmt.Errorf("context manager %q factory is nil", name)
	}

	cmRegistryMu.Lock()
	defer cmRegistryMu.Unlock()

	if _, exists := cmRegistry[name]; exists {
		return fmt.Errorf("context manager %q is already registered", name)
	}
	cmRegistry[name] = factory
	return nil
}

func lookupContextManager(name string) (ContextManagerFactory, bool) {
	cmRegistryMu.RLock()
	defer cmRegistryMu.RUnlock()

	f, ok := cmRegistry[name]
	return f, ok
}
