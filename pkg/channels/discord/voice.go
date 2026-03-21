package discord

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/logger"
)

func (c *DiscordChannel) handleVoiceCommand(s *discordgo.Session, m *discordgo.MessageCreate) bool {
	if m.Content == "!vc join" {
		vs, err := s.State.VoiceState(m.GuildID, m.Author.ID)
		if err != nil || vs == nil {
			s.ChannelMessageSend(m.ChannelID, "You need to be in a voice channel first!")
			return true
		}

		logger.InfoCF("discord", "Joining voice channel", map[string]any{"channel": vs.ChannelID})
		vc, err := s.ChannelVoiceJoin(c.ctx, m.GuildID, vs.ChannelID, false, false)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Failed to join voice channel: %v", err))
			return true
		}

		go c.receiveVoice(vc, m.GuildID, m.ChannelID)
		s.ChannelMessageSend(m.ChannelID, "Joined Voice Channel! Listening for audio...")
		return true
	} else if m.Content == "!vc leave" {
		vc, exists := s.VoiceConnections[m.GuildID]
		if exists && vc != nil {
			vc.Disconnect(c.ctx)
			s.ChannelMessageSend(m.ChannelID, "Left Voice Channel.")
		} else {
			s.ChannelMessageSend(m.ChannelID, "Not in a voice channel.")
		}
		return true
	}
	return false
}

func VoiceReceiveActive(vc *discordgo.VoiceConnection) bool {
	return vc != nil && vc.OpusRecv != nil
}

func streamOggOpusToDiscord(ctx context.Context, vc *discordgo.VoiceConnection, r io.Reader) error {
	// Wait for the speaking transition to register
	vc.Speaking(true)
	defer vc.Speaking(false)

	var packet []byte
	header := make([]byte, 27)

	for {
		// Check for interruption
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if _, err := io.ReadFull(r, header); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return nil
			}
			return fmt.Errorf("failed to read ogg header: %w", err)
		}
		if string(header[:4]) != "OggS" {
			return fmt.Errorf("invalid ogg magic string")
		}

		pageSegments := int(header[26])
		segmentTable := make([]byte, pageSegments)
		if _, err := io.ReadFull(r, segmentTable); err != nil {
			return fmt.Errorf("failed to read segment table: %w", err)
		}

		for _, lacing := range segmentTable {
			segment := make([]byte, lacing)
			if _, err := io.ReadFull(r, segment); err != nil {
				return fmt.Errorf("failed to read segment data: %w", err)
			}

			packet = append(packet, segment...)

			// If lacing is less than 255, the packet is complete
			if lacing < 255 {
				if len(packet) > 0 {
					// Ignore Ogg Opus headers
					if !bytes.HasPrefix(packet, []byte("OpusHead")) && !bytes.HasPrefix(packet, []byte("OpusTags")) {
						select {
						case <-ctx.Done():
							return ctx.Err()
						case vc.OpusSend <- packet:
						}
					}
					// Start new packet
					packet = nil
				}
			}
		}
	}
}

func (c *DiscordChannel) receiveVoice(vc *discordgo.VoiceConnection, guildID string, chatID string) {
	logger.InfoCF("discord", "Started listening for voice", map[string]any{"guild": guildID})

	go func() {
		time.Sleep(250 * time.Millisecond) // Wait a bit for connection to settle
		vc.Speaking(true)
		for i := 0; i < 5; i++ {
			vc.OpusSend <- []byte{0xF8, 0xFF, 0xFE}
			time.Sleep(20 * time.Millisecond)
		}
		vc.Speaking(false)
		logger.DebugCF("discord", "Sent wake-up silence frames", nil)
	}()

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
				return
			}

			logger.DebugCF("discord", "Received Opus packet", map[string]any{
				"seq":  p.Sequence,
				"len":  len(p.Opus),
				"ssrc": p.SSRC,
			})

			if p == nil || len(p.Opus) == 0 {
				continue
			}

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
				Sequence:   sequence,
				Timestamp:  p.Timestamp,
				SampleRate: 48000,
				Channels:   2,
				Format:     "opus",
				Data:       p.Opus,
			}

			c.bus.PublishAudioChunk(c.ctx, chunk)
		}
	}
}
