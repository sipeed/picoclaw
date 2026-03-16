package matrix

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"jane/pkg/bus"
	"jane/pkg/channels"
	"jane/pkg/config"
	"jane/pkg/logger"
)

const (
	typingRefreshInterval      = 20 * time.Second
	typingServerTTL            = 30 * time.Second
	roomKindCacheTTL           = 5 * time.Minute
	roomKindCacheCleanupPeriod = 1 * time.Minute
	roomKindCacheMaxEntries    = 2048
)

// MatrixChannel implements the Channel interface for Matrix.
type MatrixChannel struct {
	*channels.BaseChannel

	client *mautrix.Client
	config config.MatrixConfig
	syncer *mautrix.DefaultSyncer

	ctx       context.Context
	cancel    context.CancelFunc
	startTime time.Time

	typingMu       sync.Mutex
	typingSessions map[string]*typingSession // roomID -> session

	roomKindCache     *roomKindCache
	localpartMentionR *regexp.Regexp
}

func NewMatrixChannel(cfg config.MatrixConfig, messageBus *bus.MessageBus) (*MatrixChannel, error) {
	homeserver := strings.TrimSpace(cfg.Homeserver)
	userID := strings.TrimSpace(cfg.UserID)
	accessToken := strings.TrimSpace(cfg.AccessToken)
	if homeserver == "" {
		return nil, fmt.Errorf("matrix homeserver is required")
	}
	if userID == "" {
		return nil, fmt.Errorf("matrix user_id is required")
	}
	if accessToken == "" {
		return nil, fmt.Errorf("matrix access_token is required")
	}

	client, err := mautrix.NewClient(homeserver, id.UserID(userID), accessToken)
	if err != nil {
		return nil, fmt.Errorf("create matrix client: %w", err)
	}
	if cfg.DeviceID != "" {
		client.DeviceID = id.DeviceID(cfg.DeviceID)
	}

	syncer, ok := client.Syncer.(*mautrix.DefaultSyncer)
	if !ok {
		return nil, fmt.Errorf("matrix syncer is not *mautrix.DefaultSyncer")
	}

	base := channels.NewBaseChannel(
		"matrix",
		cfg,
		messageBus,
		cfg.AllowFrom,
		channels.WithMaxMessageLength(65536),
		channels.WithGroupTrigger(cfg.GroupTrigger),
		channels.WithReasoningChannelID(cfg.ReasoningChannelID),
	)

	return &MatrixChannel{
		BaseChannel:       base,
		client:            client,
		config:            cfg,
		syncer:            syncer,
		typingSessions:    make(map[string]*typingSession),
		startTime:         time.Now(),
		roomKindCache:     newRoomKindCache(roomKindCacheMaxEntries, roomKindCacheTTL),
		localpartMentionR: localpartMentionRegexp(matrixLocalpart(client.UserID)),
		typingMu:          sync.Mutex{},
	}, nil
}

func (c *MatrixChannel) Start(ctx context.Context) error {
	logger.InfoC("matrix", "Starting Matrix channel")

	c.ctx, c.cancel = context.WithCancel(ctx)
	c.startTime = time.Now()

	c.syncer.OnEventType(event.EventMessage, c.handleMessageEvent)
	c.syncer.OnEventType(event.StateMember, c.handleMemberEvent)

	c.SetRunning(true)
	go c.runRoomKindCacheJanitor(c.ctx)

	go func() {
		if err := c.client.SyncWithContext(c.ctx); err != nil && c.ctx.Err() == nil {
			logger.ErrorCF("matrix", "Matrix sync stopped unexpectedly", map[string]any{
				"error": err.Error(),
			})
		}
	}()

	logger.InfoC("matrix", "Matrix channel started")
	return nil
}

func (c *MatrixChannel) Stop(ctx context.Context) error {
	logger.InfoC("matrix", "Stopping Matrix channel")
	c.SetRunning(false)

	if c.cancel != nil {
		c.cancel()
	}
	c.stopTypingSessions(ctx)

	logger.InfoC("matrix", "Matrix channel stopped")
	return nil
}

func (c *MatrixChannel) isGroupRoom(ctx context.Context, roomID id.RoomID) bool {
	now := time.Now()
	if isGroup, ok := c.roomKindCache.get(roomID.String(), now); ok {
		return isGroup
	}

	qctx := c.baseContext()
	if ctx != nil {
		qctx = ctx
	}
	reqCtx, cancel := context.WithTimeout(qctx, 5*time.Second)
	defer cancel()

	resp, err := c.client.JoinedMembers(reqCtx, roomID)
	if err != nil {
		logger.DebugCF("matrix", "Failed to query room members; assume direct", map[string]any{
			"room_id": roomID.String(),
			"error":   err.Error(),
		})
		return false
	}

	isGroup := len(resp.Joined) > 2
	c.roomKindCache.set(roomID.String(), isGroup, now)
	return isGroup
}

func (c *MatrixChannel) baseContext() context.Context {
	if c.ctx != nil {
		return c.ctx
	}
	return context.Background()
}
