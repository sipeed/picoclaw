package discord

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/sipeed/picoclaw/pkg/audio"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/logger"
)

func (c *DiscordChannel) handleVoiceCommand(s *discordgo.Session, m *discordgo.MessageCreate) bool {
	if m.Content == "!vc join" {
		vs, err := s.State.VoiceState(m.GuildID, m.Author.ID)
		if err != nil || vs == nil {
			if _, sendErr := s.ChannelMessageSend(m.ChannelID, "You need to be in a voice channel first!"); sendErr != nil {
				logger.InfoCF("discord", "Failed to send voice channel requirement message", map[string]any{
					"channel": m.ChannelID,
					"error":   sendErr,
				})
			}
			return true
		}

		logger.InfoCF("discord", "Joining voice channel", map[string]any{"channel": vs.ChannelID})
		vc, err := s.ChannelVoiceJoin(c.ctx, m.GuildID, vs.ChannelID, false, false)
		if err != nil {
			if _, sendErr := s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Failed to join voice channel: %v", err)); sendErr != nil {
				logger.InfoCF("discord", "Failed to send voice join error message", map[string]any{
					"channel": m.ChannelID,
					"error":   sendErr,
				})
			}
			return true
		}

		go c.receiveVoice(vc, m.GuildID, m.ChannelID)
		if _, sendErr := s.ChannelMessageSend(m.ChannelID, "Joined Voice Channel! Listening for audio..."); sendErr != nil {
			logger.InfoCF("discord", "Failed to send voice join success message", map[string]any{
				"channel": m.ChannelID,
				"error":   sendErr,
			})
		}
		return true
	} else if m.Content == "!vc leave" {
		vc, exists := s.VoiceConnections[m.GuildID]
		if exists && vc != nil {
			if err := vc.Disconnect(c.ctx); err != nil {
				logger.InfoCF("discord", "Failed to disconnect from voice channel", map[string]any{
					"guild":  m.GuildID,
					"error":  err,
				})
			}
			if _, sendErr := s.ChannelMessageSend(m.ChannelID, "Left Voice Channel."); sendErr != nil {
				logger.InfoCF("discord", "Failed to send voice leave success message", map[string]any{
					"channel": m.ChannelID,
					"error":   sendErr,
				})
			}
		} else {
			if _, sendErr := s.ChannelMessageSend(m.ChannelID, "Not in a voice channel."); sendErr != nil {
				logger.InfoCF("discord", "Failed to send voice not-in-channel message", map[string]any{
					"channel": m.ChannelID,
					"error":   sendErr,
				})
			}
		}
		return true
	}
	return false
}

func VoiceReceiveActive(vc *discordgo.VoiceConnection) bool {
	return vc != nil && vc.OpusRecv != nil
}

func streamOggOpusToDiscord(ctx context.Context, vc *discordgo.VoiceConnection, r io.Reader) (retErr error) {
	// Recover from panic if vc.OpusSend is closed mid-send (e.g. on disconnect)
	defer func() {
		if rec := recover(); rec != nil {
			retErr = fmt.Errorf("voice connection closed during playback")
		}
	}()

	// Wait for the speaking transition to register
	vc.Speaking(true)
	defer vc.Speaking(false)

	return audio.DecodeOggOpus(r, func(frame []byte) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case vc.OpusSend <- frame:
			return nil
		}
	})
}

func (c *DiscordChannel) receiveVoice(vc *discordgo.VoiceConnection, guildID string, chatID string) {
	logger.InfoCF("discord", "Started listening for voice", map[string]any{"guild": guildID})

	go func(ctx context.Context, vc *discordgo.VoiceConnection) {
		// Recover from potential panics if OpusSend is closed mid-send.
		defer func() {
			if rec := recover(); rec != nil {
				logger.WarnCF("discord", "Recovered from panic while sending wake-up frames", map[string]any{
					"error": rec,
					"guild": guildID,
				})
			}
		}()

		// If the voice connection or OpusSend are not available, nothing to do.
		if vc == nil || vc.OpusSend == nil {
			return
		}

		time.Sleep(250 * time.Millisecond) // Wait a bit for connection to settle

		// Abort if the context has already been canceled.
		select {
		case <-ctx.Done():
			return
		default:
		}

		vc.Speaking(true)
		defer vc.Speaking(false)

		silenceFrame := []byte{0xF8, 0xFF, 0xFE}
		for i := 0; i < 5; i++ {
			select {
			case <-ctx.Done():
				return
			case vc.OpusSend <- silenceFrame:
			}
			time.Sleep(20 * time.Millisecond)
		}

		logger.DebugCF("discord", "Sent wake-up silence frames", map[string]any{"guild": guildID})
	}(c.ctx, vc)
	sessionID := fmt.Sprintf("discord_vc_%s", guildID)

	c.bus.PublishVoiceControl(c.ctx, bus.VoiceControl{
		SessionID: sessionID,
		Type:      "state",
		Action:    "listening",
	})

	var sequence uint64 = 0
	var interruptCount int
	var lastInterruptAt time.Time

	for {
		select {
		case <-c.ctx.Done():
			return
		case p, ok := <-vc.OpusRecv:
			if !ok {
				logger.InfoCF("discord", "Voice channel closed", map[string]any{"guild": guildID})
				// Cancel any TTS that may still be playing
				c.ttsMu.Lock()
				if c.cancelTTS != nil {
					c.cancelTTS()
					c.cancelTTS = nil
				}
				c.ttsMu.Unlock()
				return
			}

			if p == nil {
				logger.DebugCF("discord", "Received nil Opus packet", nil)
				continue
			}

			if len(p.Opus) == 0 {
				logger.DebugCF("discord", "Received empty Opus packet", map[string]any{
					"seq":  p.Sequence,
					"ssrc": p.SSRC,
				})
				continue
			}

			logger.DebugCF("discord", "Received Opus packet", map[string]any{
				"seq":  p.Sequence,
				"len":  len(p.Opus),
				"ssrc": p.SSRC,
			})
			// Interruption detection: if user sends voice while TTS is playing,
			// cancel TTS after a short debounce (3 packets in 200ms)
			now := time.Now()
			if now.Sub(lastInterruptAt) > 500*time.Millisecond {
				interruptCount = 0
			}
			interruptCount++
			lastInterruptAt = now

			if interruptCount >= 3 {
				c.ttsMu.Lock()
				if c.cancelTTS != nil {
					c.cancelTTS()
					c.cancelTTS = nil
					logger.InfoCF("discord", "TTS interrupted by user voice", nil)
				}
				c.ttsMu.Unlock()
				interruptCount = 0
			}

			sequence++

			chunk := bus.AudioChunk{
				SessionID:  sessionID,
				SpeakerID:  fmt.Sprintf("%d", p.SSRC),
				ChatID:     chatID,
				Channel:    "discord",
				Sequence:   sequence,
				Timestamp:  p.Timestamp,
				SampleRate: 48000,
				Channels:   2,
				Format:     "opus",
				Data:       p.Opus,
			}

			ctx, cancel := context.WithTimeout(c.ctx, 100*time.Millisecond)
			err := c.bus.PublishAudioChunk(ctx, chunk)
			cancel()
			if err != nil {
				logger.ErrorCF("discord", "Failed to publish audio chunk", map[string]any{
					"guild":     guildID,
					"sessionID": sessionID,
					"sequence":  sequence,
					"error":     err.Error(),
				})
			}
		}
	}
}
