package zalo

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
)

const zaloAPIBase = "https://bot-api.zaloplatforms.com/bot"

type ZaloChannel struct {
	*channels.BaseChannel
	cfg        config.ZaloConfig
	httpClient *http.Client
}

func NewZaloChannel(cfg *config.Config, messageBus *bus.MessageBus) (*ZaloChannel, error) {
	zc := cfg.Channels.Zalo
	if strings.TrimSpace(zc.Token) == "" {
		return nil, fmt.Errorf("zalo token is required")
	}
	if strings.TrimSpace(zc.SecretToken) == "" {
		return nil, fmt.Errorf("zalo secret_token is required")
	}

	base := channels.NewBaseChannel(
		"zalo",
		zc,
		messageBus,
		zc.AllowFrom,
		channels.WithMaxMessageLength(2000),
		channels.WithGroupTrigger(zc.GroupTrigger),
		channels.WithReasoningChannelID(zc.ReasoningChannelID),
	)

	return &ZaloChannel{
		BaseChannel: base,
		cfg:         zc,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (c *ZaloChannel) Start(ctx context.Context) error {
	if _, err := c.getMe(ctx); err != nil {
		return err
	}
	c.SetRunning(true)
	return nil
}

func (c *ZaloChannel) Stop(ctx context.Context) error {
	c.SetRunning(false)
	return nil
}

func (c *ZaloChannel) WebhookPath() string {
	if p := strings.TrimSpace(c.cfg.WebhookPath); p != "" {
		if strings.HasPrefix(p, "/") {
			return p
		}
		return "/" + p
	}
	return "/webhook/zalo"
}

func (c *ZaloChannel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	secret := r.Header.Get("X-Bot-Api-Secret-Token")
	if subtle.ConstantTimeCompare([]byte(secret), []byte(c.cfg.SecretToken)) != 1 {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	go func() {
		defer cancel()
		c.processPayload(ctx, payload)
	}()
}

func (c *ZaloChannel) processPayload(ctx context.Context, payload map[string]any) {
	if v, ok := payload["result"]; ok {
		if arr, ok := v.([]any); ok {
			for _, item := range arr {
				if m, ok := item.(map[string]any); ok {
					c.processUpdate(ctx, m)
				}
			}
			return
		}
	}
	c.processUpdate(ctx, payload)
}

func (c *ZaloChannel) processUpdate(ctx context.Context, upd map[string]any) {
	msgAny, _ := upd["message"]
	msg, _ := msgAny.(map[string]any)
	if msg == nil {
		msg = upd
	}

	content := getString(msg, "text")
	if strings.TrimSpace(content) == "" {
		return
	}

	messageID := getString(msg, "message_id")
	if messageID == "" {
		messageID = getString(msg, "id")
	}

	chatID := getString(msg, "chat_id")
	if chatID == "" {
		chatID = getStringPath(msg, "chat", "id")
	}
	if chatID == "" {
		return
	}

	senderID := getStringPath(msg, "from", "id")
	if senderID == "" {
		senderID = getString(msg, "from_id")
	}
	if senderID == "" {
		return
	}

	peerKind := "direct"
	if t := getStringPath(msg, "chat", "type"); t != "" {
		if strings.Contains(strings.ToLower(t), "group") {
			peerKind = "group"
		}
	}

	if peerKind == "group" {
		should, cleaned := c.ShouldRespondInGroup(false, content)
		if !should {
			return
		}
		content = cleaned
	}

	sender := bus.SenderInfo{
		Platform:    "zalo",
		PlatformID:  senderID,
		CanonicalID: identity.BuildCanonicalID("zalo", senderID),
	}

	peer := bus.Peer{
		Kind: peerKind,
		ID:   chatID,
	}

	c.HandleMessage(ctx, peer, messageID, senderID, chatID, content, nil, nil, sender)
}

func (c *ZaloChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}
	return c.sendMessage(ctx, msg.ChatID, msg.Content)
}

func (c *ZaloChannel) StartTyping(ctx context.Context, chatID string) (func(), error) {
	if !c.IsRunning() {
		return func() {}, channels.ErrNotRunning
	}
	err := c.sendChatAction(ctx, chatID, "typing")
	return func() {}, err
}

func (c *ZaloChannel) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	for _, part := range msg.Parts {
		if part.Type != "image" {
			continue
		}
		ref := strings.TrimSpace(part.Ref)
		if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
			if err := c.sendPhoto(ctx, msg.ChatID, ref, part.Caption); err != nil {
				return err
			}
			continue
		}

		caption := strings.TrimSpace(part.Caption)
		if caption == "" {
			caption = part.Filename
		}
		if caption != "" {
			if err := c.sendMessage(ctx, msg.ChatID, caption); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *ZaloChannel) apiURL(method string) string {
	return strings.TrimRight(zaloAPIBase, "/") + c.cfg.Token + "/" + strings.TrimLeft(method, "/")
}

func (c *ZaloChannel) doPOST(ctx context.Context, method string, payload any) (int, []byte, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL(method), bytes.NewReader(b))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, channels.ClassifyNetError(err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, channels.ClassifyNetError(err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, data, channels.ClassifySendError(
			resp.StatusCode,
			fmt.Errorf("http %d", resp.StatusCode),
		)
	}

	return resp.StatusCode, data, nil
}

func (c *ZaloChannel) getMe(ctx context.Context) (map[string]any, error) {
	_, data, err := c.doPOST(ctx, "getMe", map[string]any{})
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	if ok, _ := out["ok"].(bool); !ok {
		return out, fmt.Errorf("zalo getMe returned ok=false")
	}
	return out, nil
}

func (c *ZaloChannel) sendMessage(ctx context.Context, chatID, text string) error {
	_, _, err := c.doPOST(ctx, "sendMessage", map[string]any{
		"chat_id": chatID,
		"text":    text,
	})
	return err
}

func (c *ZaloChannel) sendPhoto(ctx context.Context, chatID, photoURL, caption string) error {
	payload := map[string]any{
		"chat_id": chatID,
		"photo":   photoURL,
	}
	if strings.TrimSpace(caption) != "" {
		payload["caption"] = caption
	}
	_, _, err := c.doPOST(ctx, "sendPhoto", payload)
	return err
}

func (c *ZaloChannel) sendChatAction(ctx context.Context, chatID, action string) error {
	_, _, err := c.doPOST(ctx, "sendChatAction", map[string]any{
		"chat_id": chatID,
		"action":  action,
	})
	return err
}

func getString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		if t == float64(int64(t)) {
			return fmt.Sprintf("%.0f", t)
		}
		return fmt.Sprintf("%v", t)
	default:
		return fmt.Sprintf("%v", t)
	}
}

func getStringPath(m map[string]any, k1, k2 string) string {
	v, ok := m[k1]
	if !ok || v == nil {
		return ""
	}
	m2, ok := v.(map[string]any)
	if !ok {
		return ""
	}
	return getString(m2, k2)
}
