package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"cloud.google.com/go/pubsub"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/utils"
	chat "google.golang.org/api/chat/v1"
	"google.golang.org/api/option"
)

type GoogleChatChannel struct {
	*BaseChannel
	config       config.GoogleChatConfig
	pubsubClient *pubsub.Client
	chatService  *chat.Service
	ctx          context.Context
	cancel       context.CancelFunc
	activeThreads map[string]string // threadName -> messageName (for updating status)
}

// PubSubMessagePayload represents the structure of the message data received from Pub/Sub
// The actual payload is inside the "data" field of the PubSub message, which is a JSON string of the event.
// However, the Google Chat event is sent as the message body directly if configured as "Cloud Pub/Sub" endpoint.
// Let's assume the standard Google Chat Event format.


// GoogleChatUser defines the user structure with Email field which might be missing in chat.User
type GoogleChatUser struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	Type        string `json:"type"`
	DomainID    string `json:"domainId"`
}

// GoogleChatMessage defines the message structure to capture Sender with Email
type GoogleChatMessage struct {
	Name         string          `json:"name"`
	Sender       *GoogleChatUser `json:"sender"`
	Text         string          `json:"text"`
	ArgumentText string          `json:"argumentText"`
	Thread       *chat.Thread    `json:"thread"`
}

// GoogleChatEvent supports both Classic and Google Workspace event formats
type GoogleChatEvent struct {
	// Classic format fields
	Name    string             `json:"name"`
	Type    string             `json:"type"`
	Space   *chat.Space        `json:"space"`
	Message *GoogleChatMessage `json:"message"`
	User    *GoogleChatUser    `json:"user"`

	// Google Workspace Event format fields
	Chat *struct {
		MessagePayload *struct {
			Message *GoogleChatMessage `json:"message"`
			Space   *chat.Space        `json:"space"`
		} `json:"messagePayload"`
		AddedToSpacePayload *struct {
			Space *chat.Space `json:"space"`
		} `json:"addedToSpacePayload"`
		User *GoogleChatUser `json:"user"`
	} `json:"chat"`
}

func NewGoogleChatChannel(cfg config.GoogleChatConfig, messageBus *bus.MessageBus) (*GoogleChatChannel, error) {
	if cfg.SubscriptionID == "" {
		return nil, fmt.Errorf("google chat subscription_id is required")
	}

	base := NewBaseChannel("googlechat", cfg, messageBus, cfg.AllowFrom,
		WithReasoningChannelID(cfg.ReasoningChannelID),
	)

	return &GoogleChatChannel{
		BaseChannel:   base,
		config:        cfg,
		activeThreads: make(map[string]string),
	}, nil
}

func (c *GoogleChatChannel) Start(ctx context.Context) error {
	logger.InfoC("googlechat", "Starting Google Chat channel")

	c.ctx, c.cancel = context.WithCancel(ctx)

	// Initialize Chat Service
	// We use ADC (Application Default Credentials)
	if c.config.Debug {
		logger.InfoC("googlechat", "Starting Google Chat channel in DEBUG mode - skipping Chat API initialization")
	} else {
		chatService, err := chat.NewService(ctx, option.WithScopes("https://www.googleapis.com/auth/chat.bot"))
		if err != nil {
			c.cancel()
			return fmt.Errorf("failed to create google chat service: %w", err)
		}
		c.chatService = chatService
	}

	// Initialize Pub/Sub Client
	projectID := c.config.ProjectID
	if projectID == "" {
		// If project ID is not specified, we can try to detect it, OR just require it.
		// Pubsub client requires a project ID.
		// Let's try to parse it from the subscription ID if it's a full path
		// projects/{project}/subscriptions/{sub}
		if strings.HasPrefix(c.config.SubscriptionID, "projects/") {
			parts := strings.Split(c.config.SubscriptionID, "/")
			if len(parts) >= 4 && parts[0] == "projects" && parts[2] == "subscriptions" {
				projectID = parts[1]
			}
		}
	}

	if projectID == "" {
		// Fallback to detection
		projectID = pubsub.DetectProjectID
	}

	pubsubClient, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		c.cancel()
		return fmt.Errorf("failed to create pubsub client: %w", err)
	}
	c.pubsubClient = pubsubClient

	// Start receiving messages
	go c.receiveLoop()

	c.SetRunning(true)
	logger.InfoC("googlechat", "Google Chat channel started")
	return nil
}

func (c *GoogleChatChannel) Stop(ctx context.Context) error {
	logger.InfoC("googlechat", "Stopping Google Chat channel")

	if c.cancel != nil {
		c.cancel()
	}

	if c.pubsubClient != nil {
		c.pubsubClient.Close()
	}

	c.SetRunning(false)
	logger.InfoC("googlechat", "Google Chat channel stopped")
	return nil
}

func (c *GoogleChatChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return fmt.Errorf("googlechat channel not running")
	}

	if c.config.Debug {
		logger.InfoCF("googlechat", "DEBUG: Sending Google Chat message", map[string]any{
			"chat_id": msg.ChatID,
			"content": msg.Content,
			"type":    msg.Type,
		})
		return nil
	}

	// msg.ChatID is expected to be the Thread Name or Space Name
	// Format: "spaces/SPACE_ID/threads/THREAD_ID" or "spaces/SPACE_ID"
	
	var spaceName string
	var threadName string

	if strings.Contains(msg.ChatID, "/threads/") {
		parts := strings.Split(msg.ChatID, "/threads/")
		if len(parts) == 2 {
			spaceName = parts[0]
			threadName = msg.ChatID
		}
	} else {
		spaceName = msg.ChatID
	}

	// Status Update Handling
	if msg.Type == "status" {
		if threadName == "" {
			// Cannot update status without a thread
			return nil
		}

		// Check if we already have an active status message for this thread
		activeMsgName, exists := c.activeThreads[threadName]
		if exists {
			// Update existing message
			chatMsg := &chat.Message{
				Text: toChatFormat(msg.Content),
			}
			_, err := c.chatService.Spaces.Messages.Update(activeMsgName, chatMsg).UpdateMask("text").Context(ctx).Do()
			if err != nil {
				logger.ErrorCF("googlechat", "Failed to update status message", map[string]any{
					"error": err.Error(),
					"thread": threadName,
				})
				// If update fails (e.g. message deleted), maybe clear it and create new?
				// For now just return error or log
				delete(c.activeThreads, threadName)
			}
			return nil
		} else {
			// Create new status message
			chatMsg := &chat.Message{
				Text: toChatFormat(msg.Content),
			}
			if threadName != "" {
				chatMsg.Thread = &chat.Thread{
					Name: threadName,
				}
			}
			resp, err := c.chatService.Spaces.Messages.Create(spaceName, chatMsg).Context(ctx).Do()
			if err != nil {
				return fmt.Errorf("failed to create status message: %w", err)
			}
			c.activeThreads[threadName] = resp.Name
			return nil
		}
	}

	// Final Message Handling (Type == "message" or empty)
	chatMsg := &chat.Message{
		Text: toChatFormat(msg.Content),
	}

	if threadName != "" {
		// If we have an active status message, update it with the final response
		if activeMsgName, exists := c.activeThreads[threadName]; exists {
			chatMsg.Thread = nil // Update doesn't need thread info, it targets message name
			_, err := c.chatService.Spaces.Messages.Update(activeMsgName, chatMsg).UpdateMask("text").Context(ctx).Do()
			if err == nil {
				delete(c.activeThreads, threadName)
				return nil
			}
			// If update failed, fall through to create new message
			logger.ErrorCF("googlechat", "Failed to update final message, creating new one", map[string]any{
				"error": err.Error(),
			})
			delete(c.activeThreads, threadName)
		}
		
		chatMsg.Thread = &chat.Thread{
			Name: threadName,
		}
	}

	_, err := c.chatService.Spaces.Messages.Create(spaceName, chatMsg).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to send google chat message: %w", err)
	}

	logger.DebugCF("googlechat", "Message sent", map[string]any{
		"space":  spaceName,
		"thread": threadName,
	})

	return nil
}

func (c *GoogleChatChannel) receiveLoop() {
	// Handle full subscription path vs just ID
	subID := c.config.SubscriptionID
	if strings.Contains(subID, "/") {
		// It's likely a full path involved, but the client expects just the ID if we created the client with ProjectID?
		// No, client.Subscription(id) documentation says:
		// "id is the name of the subscription to create a handle for. It must be a valid subscription name."
		// If it contains slashes, it might be interpreted as full path?
		// Actually, `pubsub.NewClient` takes a projectID. `client.Subscription` takes a specific subscription ID *within* that project,
		// OR a full path `projects/P/subscriptions/S` (if the library supports it, which it usually does in recent versions via `SubscriptionInProject` or if `id` is fully qualified).
		// Wait, `Subscription` method usually expects just the ID relative to the client's project.
		// If the user provided a full path `projects/X/subscriptions/Y`, we should probably parse it.
		
		if strings.HasPrefix(subID, "projects/") {
			parts := strings.Split(subID, "/")
			if len(parts) >= 4 {
				// We can use the subscription ID part if the project matches.
				// If the project is different, we might be in trouble if we initialized client with a different project.
				// But let's assume standard usage: Client(ProjectA) -> Subscription(SubA)
				subID = parts[3]
			}
		}
	}

	sub := c.pubsubClient.Subscription(subID)
	sub.ReceiveSettings.MaxOutstandingMessages = 10

	err := sub.Receive(c.ctx, func(ctx context.Context, msg *pubsub.Message) {
		msg.Ack() // Ack immediately. If we crash, we lose the message, but prevents loops.

		logger.DebugCF("googlechat", "Raw PubSub payload", map[string]any{
			"data": string(msg.Data),
			"attributes": msg.Attributes,
		})

		var event GoogleChatEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			logger.ErrorCF("googlechat", "Failed to unmarshal pubsub message event", map[string]any{
				"error": err.Error(),
			})
			return
		}

		c.handleEvent(event)
	})

	if err != nil && c.ctx.Err() == nil {
		logger.ErrorCF("googlechat", "PubSub receive error", map[string]any{
			"error": err.Error(),
		})
	}
}

func (c *GoogleChatChannel) handleEvent(event GoogleChatEvent) {
	// Normalize Google Workspace Event to Classic format if needed
	if event.Type == "" && event.Chat != nil {
		if payload := event.Chat.MessagePayload; payload != nil {
			event.Type = "MESSAGE"
			event.Message = payload.Message
			event.Space = payload.Space
			event.User = event.Chat.User
		} else if payload := event.Chat.AddedToSpacePayload; payload != nil {
			event.Type = "ADDED_TO_SPACE"
			event.Space = payload.Space
			event.User = event.Chat.User
		}
	}

	logger.DebugCF("googlechat", "Received event", map[string]any{
		"type": event.Type,
		"name": event.Name,
	})

	switch event.Type {
	case "MESSAGE":
		c.handleMessage(event)
	case "ADDED_TO_SPACE":
		c.handleAddedToSpace(event)
	}
}

func (c *GoogleChatChannel) handleMessage(event GoogleChatEvent) {
	if event.Message == nil || event.Message.Sender == nil {
		return
	}

	senderName := event.Message.Sender.Name
	senderEmail := event.Message.Sender.Email
	messageID := event.Message.Name

	// Access Control
	if !c.IsAllowed(senderName) && (senderEmail == "" || !c.IsAllowed(senderEmail)) {
		logger.DebugCF("googlechat", "Message rejected by allowlist", map[string]any{
			"sender_name": senderName,
			"email":       senderEmail,
		})
		return
	}

	spaceName := ""
	if event.Space != nil {
		spaceName = event.Space.Name
	}

	threadName := ""
	if event.Message.Thread != nil {
		threadName = event.Message.Thread.Name
	}

	// Canonical ChatID for reply: The Thread Name (which includes space)
	chatID := threadName
	if chatID == "" {
		chatID = spaceName
	}

	content := event.Message.ArgumentText
	if content == "" {
		content = event.Message.Text
	}
	content = strings.TrimSpace(content)

	if content == "" {
		return
	}

	// Metadata
	metadata := map[string]string{
		"platform":     "googlechat",
		"space_name":   spaceName,
		"thread_name":  threadName,
		"sender_name":  event.Message.Sender.DisplayName,
		"sender_email": senderEmail,
		"user_type":    event.Message.Sender.Type,
	}

	logger.DebugCF("googlechat", "Processing message", map[string]any{
		"sender":  senderEmail,
		"chat_id": chatID,
		"text":    utils.Truncate(content, 50),
	})

	c.HandleMessage(c.ctx, bus.Peer{}, messageID, senderName, chatID, content, nil, metadata)
}

func (c *GoogleChatChannel) handleAddedToSpace(event GoogleChatEvent) {
	if event.Space == nil || event.User == nil {
		return
	}
	logger.InfoCF("googlechat", "Bot added to space", map[string]any{
		"space": event.Space.Name,
		"user":  event.User.DisplayName,
	})
}

var (
	chatBoldRegex = regexp.MustCompile(`\*\*(.*?)\*\*`)
	chatLinkRegex = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
)

func toChatFormat(text string) string {
	if text == "" {
		return ""
	}
	// Bold: **text** -> *text*
	text = chatBoldRegex.ReplaceAllString(text, "*$1*")

	// Links: [text](url) -> <url|text>
	text = chatLinkRegex.ReplaceAllString(text, "<$2|$1>")

	return text
}
