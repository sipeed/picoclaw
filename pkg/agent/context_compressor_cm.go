package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// structuredContextManager wraps ContextCompressor as a ContextManager
// implementation. It uses the 6-phase compression algorithm instead of
// the legacy drop-oldest approach.
//
// Activate by setting: agents.defaults.context_manager = "structured"
type structuredContextManager struct {
	al         *AgentLoop
	compressor *ContextCompressor
}

// structuredCMConfig is the JSON config for the structured context manager.
type structuredCMConfig struct {
	ThresholdPercent int `json:"threshold_percent"`
	ProtectFirstN    int `json:"protect_first_n"`
	ProtectLastN     int `json:"protect_last_n"`
}

func init() {
	_ = RegisterContextManager("structured", func(cfg json.RawMessage, al *AgentLoop) (ContextManager, error) {
		agent := al.registry.GetDefaultAgent()
		if agent == nil {
			return nil, fmt.Errorf("structured context manager: no default agent")
		}

		var opts []CompressorOption

		if cfg != nil {
			var c structuredCMConfig
			if err := json.Unmarshal(cfg, &c); err == nil {
				if c.ThresholdPercent > 0 {
					opts = append(opts, WithThresholdPercent(c.ThresholdPercent))
				}
				if c.ProtectFirstN > 0 {
					opts = append(opts, WithProtectFirstN(c.ProtectFirstN))
				}
				if c.ProtectLastN > 0 {
					opts = append(opts, WithProtectLastN(c.ProtectLastN))
				}
			}
		}

		compressor := NewContextCompressor(agent.ContextWindow, opts...)

		logger.InfoCF("agent", "structured context manager initialized", map[string]any{
			"context_window": agent.ContextWindow,
		})

		return &structuredContextManager{
			al:         al,
			compressor: compressor,
		}, nil
	})
}

func (m *structuredContextManager) Assemble(_ context.Context, req *AssembleRequest) (*AssembleResponse, error) {
	// Same as legacy: read history from session.
	agent := m.al.registry.GetDefaultAgent()
	if agent == nil {
		return &AssembleResponse{}, nil
	}
	history := agent.Sessions.GetHistory(req.SessionKey)
	summary := agent.Sessions.GetSummary(req.SessionKey)
	return &AssembleResponse{
		History: history,
		Summary: summary,
	}, nil
}

func (m *structuredContextManager) Compact(_ context.Context, req *CompactRequest) error {
	agent := m.al.registry.GetDefaultAgent()
	if agent == nil {
		return nil
	}

	history := agent.Sessions.GetHistory(req.SessionKey)
	if len(history) <= 4 {
		return nil
	}

	compressed, summaryPrompt := m.compressor.Compress(history)

	if summaryPrompt == "" {
		// Nothing to compress — too few messages.
		return nil
	}

	// Use the summary prompt as the session summary.
	// In a full integration the caller would send summaryPrompt to an LLM
	// and store the response. For now, store a structured note.
	existingSummary := agent.Sessions.GetSummary(req.SessionKey)
	droppedCount := len(history) - len(compressed)

	summaryNote := fmt.Sprintf(
		"[Structured compression #%d: %d messages compressed using 6-phase algorithm]",
		m.compressor.compressionCount, droppedCount,
	)
	if existingSummary != "" {
		summaryNote = existingSummary + "\n\n" + summaryNote
	}

	agent.Sessions.SetSummary(req.SessionKey, summaryNote)
	agent.Sessions.SetHistory(req.SessionKey, compressed)
	agent.Sessions.Save(req.SessionKey)

	m.al.emitEvent(
		EventKindContextCompress,
		m.al.newTurnEventScope("", req.SessionKey).meta(0, "structuredCompression", "turn.context.compress"),
		ContextCompressPayload{
			Reason:            req.Reason,
			DroppedMessages:   droppedCount,
			RemainingMessages: len(compressed),
		},
	)

	logger.InfoCF("agent", "structured compression complete", map[string]any{
		"session_key":    req.SessionKey,
		"original_msgs":  len(history),
		"compressed_msgs": len(compressed),
		"dropped":        droppedCount,
		"reason":         req.Reason,
	})

	return nil
}

func (m *structuredContextManager) Ingest(_ context.Context, _ *IngestRequest) error {
	// No-op: messages are persisted by Sessions JSONL.
	return nil
}
