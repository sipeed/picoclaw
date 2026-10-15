// Package qqc2c implements a first-party QQ single-chat (C2C) channel for PicoClaw.
//
// It deliberately does NOT depend on github.com/tencent-connect/botgo: the whole
// protocol (Access Token, WebSocket gateway, C2C send, streaming, keyboard and
// interaction callbacks) is implemented here against the public QQ Bot API v2:
//
//	https://bot.q.qq.com/wiki/develop/api-v2/
//
// Design notes
//   - Auth uses Access Token (the legacy Bot Token is deprecated):
//     POST https://bots.qq.com/app/getAppAccessToken {appId, clientSecret}
//     then every openapi call sends  Authorization: QQBot <access_token>
//   - Messages are sent to  POST /v2/users/{user_openid}/messages  as *passive
//     replies* (msg_id + incrementing msg_seq). QQ removed proactive push on
//     2025-04-21, so the channel is answer-oriented by design.
//   - Streaming uses POST /v2/users/{user_openid}/stream_messages.
package qqc2c

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// ---------------------------------------------------------------------------
// endpoints
// ---------------------------------------------------------------------------

const (
	tokenURL     = "https://bots.qq.com/app/getAppAccessToken"
	defaultAPIBase = "https://api.sgroup.qq.com"
	// wsGatewayPath is the JSON endpoint that hands out the WebSocket URL.
	// NOTE: /websocket is the upgrade endpoint itself (it answers 426 without
	// Upgrade headers), the gateway lookup lives at /gateway.
	wsGatewayPath = "/gateway"
)

// ---------------------------------------------------------------------------
// wire types
// ---------------------------------------------------------------------------

type accessTokenResp struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   any    `json:"expires_in"` // QQ sometimes returns a string
}

func (r accessTokenResp) expiresSeconds() int {
	switch v := r.ExpiresIn.(type) {
	case float64:
		return int(v)
	case string:
		var n int
		_, _ = fmt.Sscanf(v, "%d", &n)
		return n
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	}
	return 0
}

type wsGatewayResp struct {
	URL string `json:"url"`
}

// websocket payload (op codes per the v2 spec)
type wsPayload struct {
	Op int             `json:"op"`
	D  json.RawMessage `json:"d,omitempty"`
	S  int64           `json:"s,omitempty"`
	T  string          `json:"t,omitempty"`
}

type wsHello struct {
	HeartbeatInterval int `json:"heartbeat_interval"`
}

type wsIdentify struct {
	Token      string   `json:"token"`
	Intents    int      `json:"intents"`
	Shard      [2]int   `json:"shard"`
	Properties struct{} `json:"properties,omitempty"`
}

type wsResume struct {
	Token     string `json:"token"`
	SessionID string `json:"session_id"`
	Seq       int64  `json:"seq"`
}

// C2CMessageData is the `d` of a C2C_MESSAGE_CREATE event.
type c2cMessageData struct {
	ID          string `json:"id"`
	Content     string `json:"content"`
	Timestamp   string `json:"timestamp"`
	Author      struct {
		UserOpenID string `json:"user_openid"`
		UnionOpenID string `json:"union_openid"`
	} `json:"author"`
	MessageScene struct {
		Ext []struct {
			MsgIdx    string `json:"msg_idx"`
			RefMsgIdx string `json:"ref_msg_idx"`
		} `json:"ext"`
	} `json:"message_scene"`
}

// InteractionData is the `d` of an INTERACTION_CREATE event.
type interactionData struct {
	ID       string `json:"id"`
	Type     int    `json:"type"` // 11 = message button, 12 = single-chat quick menu
	Scene    string `json:"scene"`
	ChatType int    `json:"chat_type"` // 2 = c2c
	UserOpenID string `json:"user_openid"`
	Data     struct {
		Resolved struct {
			ButtonData string `json:"button_data"`
			ButtonID   string `json:"button_id"`
		} `json:"resolved"`
	} `json:"data"`
}

// MessageToCreate is the body of a C2C send.
type MessageToCreate struct {
	MsgType   int             `json:"msg_type,omitempty"` // 0 text, 2 markdown, 6 input notify, 7 media
	Content   string          `json:"content,omitempty"`
	Markdown  *Markdown       `json:"markdown,omitempty"`
	Keyboard  *Keyboard       `json:"keyboard,omitempty"`
	Media     *Media          `json:"media,omitempty"`
	InputNotify *InputNotify  `json:"input_notify,omitempty"`
	MsgID     string          `json:"msg_id,omitempty"`
	EventID   string          `json:"event_id,omitempty"`
	MsgSeq    int             `json:"msg_seq,omitempty"`
	IsWakeup  bool            `json:"is_wakeup,omitempty"`
}

type Markdown struct {
	Content string `json:"content"`
}

type InputNotify struct {
	InputType   int `json:"input_type"`   // 1 = typing
	InputSecond int `json:"input_second"` // seconds
}

type Media struct {
	FileInfo string `json:"file_info"`
}

// StreamRequest is the body of a streaming C2C send.
type StreamRequest struct {
	InputMode   string `json:"input_mode,omitempty"`   // append | replace
	InputState  int    `json:"input_state,omitempty"`  // 1 generating, 10 finished
	Index       int    `json:"index"`                  // starts at 0
	ContentType string `json:"content_type,omitempty"` // text | markdown
	ContentRaw  string `json:"content_raw,omitempty"`
	MsgID       string `json:"msg_id,omitempty"`
	EventID     string `json:"event_id,omitempty"`
	StreamMsgID string `json:"stream_msg_id,omitempty"`
	MsgSeq      int    `json:"msg_seq,omitempty"`
}

// SentMessage is the common response of send/stream calls.
type SentMessage struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	RemainMsgLen int `json:"remain_msg_len,omitempty"`
}

// apiError carries the platform error payload so callers can classify it.
type apiError struct {
	Status int
	Body   string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("qq api http %d: %s", e.Status, e.Body)
}

func isNotFoundErr(err error) bool {
	ae, ok := err.(*apiError)
	if !ok {
		return false
	}
	for _, needle := range []string{"11255", "40011028", "请求的资源不存在", "not found"} {
		if bytes.Contains([]byte(ae.Body), []byte(needle)) {
			return true
		}
	}
	return false
}

func isMsgSeqErr(err error) bool {
	ae, ok := err.(*apiError)
	if !ok {
		return false
	}
	for _, needle := range []string{"msg_seq", "重复", "duplicate", "11244"} {
		if bytes.Contains([]byte(ae.Body), []byte(needle)) {
			return true
		}
	}
	return false
}

// isUserRejectedErr matches QQ error 40054013 (用户拒收消息): the user turned the
// "允许主动消息" switch off, so proactive messages to them keep failing.
func isUserRejectedErr(err error) bool {
	ae, ok := err.(*apiError)
	if !ok {
		return false
	}
	for _, needle := range []string{"40054013", "用户拒收"} {
		if bytes.Contains([]byte(ae.Body), []byte(needle)) {
			return true
		}
	}
	return false
}

// isNoPermissionErr matches QQ error 40034105 (主动消息发送失败，无权限): the bot
// itself lacks the permission/quota for active messages.
func isNoPermissionErr(err error) bool {
	ae, ok := err.(*apiError)
	if !ok {
		return false
	}
	for _, needle := range []string{"40034105", "无权限"} {
		if bytes.Contains([]byte(ae.Body), []byte(needle)) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// token manager
// ---------------------------------------------------------------------------

type tokenManager struct {
	appID     string
	appSecret string
	hc        *http.Client

	mu        sync.Mutex
	token     string
	expiresAt time.Time
}

func newTokenManager(appID, appSecret string) *tokenManager {
	return &tokenManager{
		appID:     appID,
		appSecret: appSecret,
		hc:        &http.Client{Timeout: 10 * time.Second},
	}
}

// Access returns a valid access token, refreshing it when it is close to expiring.
func (t *tokenManager) Access(ctx context.Context) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.token != "" && time.Now().Add(60*time.Second).Before(t.expiresAt) {
		return t.token, nil
	}
	body, _ := json.Marshal(map[string]string{
		"appId":        t.appID,
		"clientSecret": t.appSecret,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := t.hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("qq token request: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode != http.StatusOK {
		return "", &apiError{Status: resp.StatusCode, Body: string(raw)}
	}
	var out accessTokenResp
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("qq token decode: %w", err)
	}
	if out.AccessToken == "" {
		return "", fmt.Errorf("qq token empty: %s", string(raw))
	}
	ttl := out.expiresSeconds()
	if ttl <= 0 {
		ttl = 7200
	}
	t.token = out.AccessToken
	t.expiresAt = time.Now().Add(time.Duration(ttl) * time.Second)
	logger.InfoCF("qq_c2c", "access token refreshed", map[string]any{"ttl_seconds": ttl})
	return t.token, nil
}

// Invalidate drops the cached token so the next call refetches it.
func (t *tokenManager) Invalidate() {
	t.mu.Lock()
	t.token = ""
	t.expiresAt = time.Time{}
	t.mu.Unlock()
}

// ---------------------------------------------------------------------------
// openapi client
// ---------------------------------------------------------------------------

type apiClient struct {
	base string
	tm   *tokenManager
	hc   *http.Client
}

func newAPIClient(base string, tm *tokenManager) *apiClient {
	if base == "" {
		base = defaultAPIBase
	}
	return &apiClient{base: base, tm: tm, hc: &http.Client{Timeout: 15 * time.Second}}
}

func (a *apiClient) do(ctx context.Context, method, path string, in any, out any) error {
	token, err := a.tm.Access(ctx)
	if err != nil {
		return err
	}
	var rdr io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, a.base+path, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "QQBot "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.hc.Do(req)
	if err != nil {
		return fmt.Errorf("qq request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == http.StatusUnauthorized {
		a.tm.Invalidate()
		return &apiError{Status: resp.StatusCode, Body: string(raw)}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &apiError{Status: resp.StatusCode, Body: string(raw)}
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("qq decode %s: %w", path, err)
		}
	}
	return nil
}

// Gateway returns the WebSocket gateway URL.
func (a *apiClient) Gateway(ctx context.Context) (string, error) {
	var out wsGatewayResp
	if err := a.do(ctx, http.MethodGet, wsGatewayPath, nil, &out); err != nil {
		return "", err
	}
	return out.URL, nil
}

// PostC2C sends a (passive) single-chat message.
func (a *apiClient) PostC2C(ctx context.Context, openID string, body *MessageToCreate) (*SentMessage, error) {
	var out SentMessage
	err := a.do(ctx, http.MethodPost, "/v2/users/"+openID+"/messages", body, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// StreamC2C sends one streaming chunk.
func (a *apiClient) StreamC2C(ctx context.Context, openID string, body *StreamRequest) (*SentMessage, error) {
	var out SentMessage
	err := a.do(ctx, http.MethodPost, "/v2/users/"+openID+"/stream_messages", body, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// PutInteraction acknowledges a button interaction (must happen within 3s).
func (a *apiClient) PutInteraction(ctx context.Context, interactionID string, code int) error {
	body := map[string]int{"code": code}
	return a.do(ctx, http.MethodPut, "/interactions/"+interactionID, body, nil)
}
