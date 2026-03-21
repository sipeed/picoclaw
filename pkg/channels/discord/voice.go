package discord

import (
	"fmt"
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
