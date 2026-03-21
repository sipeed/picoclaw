package voice

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3/pkg/media/oggwriter"

	"github.com/sipeed/picoclaw/pkg/asr"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/logger"
)

type speechAccumulator struct {
	writer      *oggwriter.OggWriter
	file        string
	lastAudioAt time.Time
	mu          sync.Mutex
	closed      bool
	chatID      string
	speakerID   string
	sessionID   string
}

func (a *speechAccumulator) Push(chunk bus.AudioChunk) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.closed {
		return
	}

	a.lastAudioAt = time.Now()

	pkt := &rtp.Packet{
		Header: rtp.Header{
			SequenceNumber: uint16(chunk.Sequence),
			Timestamp:      chunk.Timestamp,
			SSRC:           uint32(chunk.Sequence), // Arbitrary dummy
		},
		Payload: chunk.Data,
	}

	if err := a.writer.WriteRTP(pkt); err != nil {
		logger.ErrorCF("voice-agent", "Failed to write RTP", map[string]any{"error": err})
	}
}

func (a *speechAccumulator) Close() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.closed {
		a.writer.Close()
		a.closed = true
	}
}

type Agent struct {
	bus         *bus.MessageBus
	transcriber asr.Transcriber

	mu       sync.Mutex
	sessions map[string]*speechAccumulator // keyed by sessionID_speakerID
}

func NewAgent(mb *bus.MessageBus, t asr.Transcriber) *Agent {
	return &Agent{
		bus:         mb,
		transcriber: t,
		sessions:    make(map[string]*speechAccumulator),
	}
}

func (a *Agent) Start(ctx context.Context) {
	logger.InfoCF("voice-agent", "Started Voice Agent orchestrator", nil)
	go a.listenChunks(ctx)
	go a.vadTick(ctx)
}

func (a *Agent) listenChunks(ctx context.Context) {
	chunks := a.bus.AudioChunksChan()
	for {
		select {
		case <-ctx.Done():
			return
		case chunk := <-chunks:
			a.handleChunk(chunk)
		}
	}
}

func (a *Agent) handleChunk(chunk bus.AudioChunk) {
	a.mu.Lock()
	defer a.mu.Unlock()

	key := fmt.Sprintf("%s_%s", chunk.SessionID, chunk.SpeakerID)

	acc, exists := a.sessions[key]
	if !exists {
		filename := filepath.Join(os.TempDir(), fmt.Sprintf("voice_%s_%d.ogg", key, time.Now().UnixNano()))
		writer, err := oggwriter.New(filename, uint32(chunk.SampleRate), uint16(chunk.Channels))
		if err != nil {
			logger.ErrorCF("voice-agent", "Failed to create OggWriter", map[string]any{"error": err})
			return
		}

		acc = &speechAccumulator{
			writer:      writer,
			file:        filename,
			lastAudioAt: time.Now(),
			chatID:      chunk.ChatID,
			speakerID:   chunk.SpeakerID,
			sessionID:   chunk.SessionID,
		}
		a.sessions[key] = acc
		logger.DebugCF("voice-agent", "Started accumulating voice", map[string]any{"key": key, "file": filename})
	}

	acc.Push(chunk)
}

func (a *Agent) vadTick(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.checkSilence(ctx)
		}
	}
}

func (a *Agent) checkSilence(ctx context.Context) {
	a.mu.Lock()
	now := time.Now()
	var finished []*speechAccumulator

	for key, acc := range a.sessions {
		acc.mu.Lock()
		last := acc.lastAudioAt
		acc.mu.Unlock()

		if now.Sub(last) > 1500*time.Millisecond {
			acc.Close()
			delete(a.sessions, key)
			finished = append(finished, acc)
		}
	}
	a.mu.Unlock()

	for _, acc := range finished {
		go a.processUtterance(ctx, acc)
	}
}

func (a *Agent) processUtterance(ctx context.Context, acc *speechAccumulator) {
	defer os.Remove(acc.file)

	logger.InfoCF("voice-agent", "User finished speaking, transcribing...", map[string]any{"file": acc.file})

	if a.transcriber == nil {
		logger.ErrorCF("voice-agent", "No STT configured!", nil)
		return
	}

	res, err := a.transcriber.Transcribe(ctx, acc.file)
	if err != nil {
		logger.ErrorCF("voice-agent", "Transcription failed", map[string]any{"error": err})
		return
	}

	if res.Text == "" {
		logger.DebugCF("voice-agent", "Ignored empty transcription", map[string]any{"file": acc.file})
		return
	}

	logger.InfoCF("voice-agent", "Transcription result", map[string]any{"text": res.Text, "duration": res.Duration})

	channelType := "discord"

	text := strings.ToLower(strings.TrimSpace(res.Text))
	if strings.Contains(text, "leave the voice channel") || strings.Contains(text, "leave voice") ||
		strings.Contains(text, "disconnect voice") {
		logger.InfoCF("voice-agent", "Voice command triggered: leave", nil)
		a.bus.PublishVoiceControl(ctx, bus.VoiceControl{
			SessionID: acc.sessionID,
			Type:      "command",
			Action:    "leave",
		})
		a.bus.PublishOutbound(ctx, bus.OutboundMessage{
			Channel: channelType,
			ChatID:  acc.chatID,
			Content: "Goodbye! Leaving the voice channel.",
		})
		return
	}

	oralPrompt := "\n\n[SYSTEM]: The user just spoke this to you over voice chat. Please reply in a highly concise, conversational, oral style suitable for text-to-speech. Do not use markdown, emojis, asterisks, or code blocks. Speak naturally."

	a.bus.PublishInbound(ctx, bus.InboundMessage{
		Channel:  channelType,
		SenderID: acc.speakerID,
		ChatID:   acc.chatID,
		Content:  res.Text + oralPrompt,
		Peer:     bus.Peer{Kind: "channel", ID: acc.chatID},
		Metadata: map[string]string{
			"is_voice": "true",
		},
	})
}
