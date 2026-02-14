package devices

import (
	"context"
	"strings"
	"sync"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/constants"
	"github.com/sipeed/picoclaw/pkg/devices/events"
	"github.com/sipeed/picoclaw/pkg/devices/sources"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/state"
)

type Service struct {
	bus     *bus.MessageBus
	state   *state.Manager
	sources []events.EventSource
	enabled bool
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.RWMutex
}

type Config struct {
	Enabled    bool
	MonitorUSB bool // When true, monitor USB hotplug (Linux only)
	// Future: MonitorBluetooth, MonitorPCI, etc.
}

func NewService(cfg Config, stateMgr *state.Manager) *Service {
	s := &Service{
		state:   stateMgr,
		enabled: cfg.Enabled,
		sources: make([]EventSource, 0),
	}

	if cfg.Enabled && cfg.MonitorUSB {
		s.sources = append(s.sources, sources.NewUSBMonitor())
	}

	return s
}

func (s *Service) SetBus(msgBus *bus.MessageBus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bus = msgBus
}

func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.enabled || len(s.sources) == 0 {
		logger.InfoC("devices", "Device event service disabled or no sources")
		return nil
	}

	s.ctx, s.cancel = context.WithCancel(ctx)

	for _, src := range s.sources {
		eventCh, err := src.Start(s.ctx)
		if err != nil {
			logger.ErrorCF("devices", "Failed to start source", map[string]interface{}{
				"kind":  src.Kind(),
				"error": err.Error(),
			})
			continue
		}
		go s.handleEvents(src.Kind(), eventCh)
		logger.InfoCF("devices", "Device source started", map[string]interface{}{
			"kind": src.Kind(),
		})
	}

	logger.InfoC("devices", "Device event service started")
	return nil
}

func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}

	for _, src := range s.sources {
		src.Stop()
	}

	logger.InfoC("devices", "Device event service stopped")
}

func (s *Service) handleEvents(kind events.Kind, eventCh <-chan *events.DeviceEvent) {
	for ev := range eventCh {
		if ev == nil {
			continue
		}
		s.sendNotification(ev)
	}
}

func (s *Service) sendNotification(ev *events.DeviceEvent) {
	s.mu.RLock()
	msgBus := s.bus
	s.mu.RUnlock()

	if msgBus == nil {
		return
	}

	lastChannel := s.state.GetLastChannel()
	if lastChannel == "" {
		logger.DebugCF("devices", "No last channel, skipping notification", map[string]interface{}{
			"event": ev.FormatMessage(),
		})
		return
	}

	platform, userID := parseLastChannel(lastChannel)
	if platform == "" || userID == "" || constants.IsInternalChannel(platform) {
		return
	}

	msg := ev.FormatMessage()
	msgBus.PublishOutbound(bus.OutboundMessage{
		Channel: platform,
		ChatID:  userID,
		Content: msg,
	})

	logger.InfoCF("devices", "Device notification sent", map[string]interface{}{
		"kind":   ev.Kind,
		"action": ev.Action,
		"to":     platform,
	})
}

func parseLastChannel(lastChannel string) (platform, userID string) {
	if lastChannel == "" {
		return "", ""
	}
	parts := strings.SplitN(lastChannel, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", ""
	}
	return parts[0], parts[1]
}
