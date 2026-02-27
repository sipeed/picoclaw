//go:build amd64 || arm64 || riscv64 || mips64 || ppc64

package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

type FeishuCard struct {
	Schema string           `json:"schema"`
	Config FeishuCardConfig `json:"config"`
	Body   FeishuCardBody   `json:"body"`
}

type FeishuCardConfig struct {
	WideScreenMode bool                `json:"wide_screen_mode"`
	StreamingMode  bool                `json:"streaming_mode,omitempty"`
	Summary        *FeishuCardSummary  `json:"summary,omitempty"`
	Streaming      *FeishuStreamingCfg `json:"streaming_config,omitempty"`
}

type FeishuCardSummary struct {
	Content string `json:"content"`
}

type FeishuStreamingCfg struct {
	PrintFrequencyMs int `json:"print_frequency_ms,omitempty"`
	PrintStep        int `json:"print_step,omitempty"`
}

type FeishuCardBody struct {
	Elements []FeishuCardElement `json:"elements"`
}

type FeishuCardElement struct {
	Tag       string `json:"tag"`
	Content   string `json:"content,omitempty"`
	ElementID string `json:"element_id,omitempty"`
}

func BuildMarkdownCard(text string) *FeishuCard {
	return &FeishuCard{
		Schema: "2.0",
		Config: FeishuCardConfig{
			WideScreenMode: true,
		},
		Body: FeishuCardBody{
			Elements: []FeishuCardElement{
				{Tag: "markdown", Content: text, ElementID: "content"},
			},
		},
	}
}

func BuildStreamingCard(initialText string) *FeishuCard {
	return &FeishuCard{
		Schema: "2.0",
		Config: FeishuCardConfig{
			WideScreenMode: true,
			StreamingMode:  true,
			Summary:        &FeishuCardSummary{Content: "[Generating...]"},
			Streaming:      &FeishuStreamingCfg{PrintFrequencyMs: 50, PrintStep: 2},
		},
		Body: FeishuCardBody{
			Elements: []FeishuCardElement{
				{Tag: "markdown", Content: initialText, ElementID: "content"},
			},
		},
	}
}

func ShouldUseCard(text string) bool {
	hasCodeBlock := strings.Contains(text, "```")
	lines := strings.Split(text, "\n")
	hasTable := false
	if len(lines) >= 3 {
		count := 0
		for _, line := range lines {
			if strings.Contains(line, "|") {
				count++
			}
		}
		hasTable = count >= 2
	}
	return hasCodeBlock || hasTable
}

func TruncateSummary(text string, maxLen int) string {
	if maxLen <= 0 {
		maxLen = 50
	}
	clean := strings.ReplaceAll(text, "\n", " ")
	clean = strings.TrimSpace(clean)
	if len(clean) <= maxLen {
		return clean
	}
	return clean[:maxLen-3] + "..."
}

type tokenEntry struct {
	token     string
	expiresAt time.Time
}

type tokenCache struct {
	mu     sync.RWMutex
	tokens map[string]tokenEntry
}

var globalTokenCache = &tokenCache{
	tokens: make(map[string]tokenEntry),
}

func (tc *tokenCache) Get(key string) (string, bool) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	if cached, ok := tc.tokens[key]; ok {
		if time.Now().Before(cached.expiresAt.Add(-time.Minute)) {
			return cached.token, true
		}
	}
	return "", false
}

func (tc *tokenCache) Set(key, token string, expiresIn int) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.tokens[key] = tokenEntry{
		token:     token,
		expiresAt: time.Now().Add(time.Duration(expiresIn) * time.Second),
	}
}

func getFeishuToken(appID, appSecret, domain string) (string, error) {
	cacheKey := fmt.Sprintf("%s|%s", domain, appID)
	if token, ok := globalTokenCache.Get(cacheKey); ok {
		return token, nil
	}

	apiBase := "https://open.feishu.cn/open-apis"
	if domain == "lark" {
		apiBase = "https://open.larksuite.com/open-apis"
	}

	reqBody := map[string]string{
		"app_id":     appID,
		"app_secret": appSecret,
	}
	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal token request: %w", err)
	}

	req, err := http.NewRequest("POST", apiBase+"/auth/v3/tenant_access_token/internal", bytes.NewReader(reqJSON))
	if err != nil {
		return "", fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request token: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read token response: %w", err)
	}

	var tokenResp struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
		Expire            int    `json:"expire"`
	}
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return "", fmt.Errorf("unmarshal token response: %w", err)
	}
	if tokenResp.Code != 0 {
		return "", fmt.Errorf("token error: code=%d msg=%s", tokenResp.Code, tokenResp.Msg)
	}

	globalTokenCache.Set(cacheKey, tokenResp.TenantAccessToken, tokenResp.Expire)
	return tokenResp.TenantAccessToken, nil
}

type FeishuStreamingSession struct {
	client      *lark.Client
	appID       string
	appSecret   string
	domain      string
	cardID      string
	messageID   string
	sequence    int
	currentText string
	closed      bool
	mu          sync.Mutex
	updateCh    chan string
	lastUpdate  time.Time
	throttleMs  int
	pendingText string
	done        chan struct{}
	apiBase     string
}

func NewStreamingSession(client *lark.Client, appID, appSecret, domain string, throttleMs int) *FeishuStreamingSession {
	if throttleMs <= 0 {
		throttleMs = 100
	}
	apiBase := "https://open.feishu.cn/open-apis"
	if domain == "lark" {
		apiBase = "https://open.larksuite.com/open-apis"
	}
	return &FeishuStreamingSession{
		client:     client,
		appID:      appID,
		appSecret:  appSecret,
		domain:     domain,
		updateCh:   make(chan string, 100),
		throttleMs: throttleMs,
		done:       make(chan struct{}),
		apiBase:    apiBase,
	}
}

func (s *FeishuStreamingSession) Start(ctx context.Context, chatID string) error {
	token, err := getFeishuToken(s.appID, s.appSecret, s.domain)
	if err != nil {
		return fmt.Errorf("get token: %w", err)
	}

	card := BuildStreamingCard("⏳ Thinking...")
	cardJSON, _ := json.Marshal(map[string]string{"type": "card_json", "data": JSONMarshal(card)})

	createReq, _ := http.NewRequest("POST", s.apiBase+"/cardkit/v1/cards", bytes.NewReader(cardJSON))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createReq.Header.Set("Content-Type", "application/json")

	createResp, err := (&http.Client{Timeout: 10 * time.Second}).Do(createReq)
	if err != nil {
		return fmt.Errorf("create card: %w", err)
	}
	defer createResp.Body.Close()

	var createResult struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			CardID string `json:"card_id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(createResp.Body).Decode(&createResult); err != nil {
		return fmt.Errorf("decode create response: %w", err)
	}
	if createResult.Code != 0 {
		return fmt.Errorf("create card error: code=%d msg=%s", createResult.Code, createResult.Msg)
	}
	s.cardID = createResult.Data.CardID

	content := fmt.Sprintf(`{"type":"card","data":{"card_id":"%s"}}`, s.cardID)
	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(chatID).
			MsgType(larkim.MsgTypeInteractive).
			Content(content).
			Build()).
		Build()

	resp, err := s.client.Im.V1.Message.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("send card message: %w", err)
	}
	if !resp.Success() {
		return fmt.Errorf("send error: code=%d msg=%s", resp.Code, resp.Msg)
	}
	if resp.Data == nil || resp.Data.MessageId == nil || *resp.Data.MessageId == "" {
		return fmt.Errorf("no message_id returned")
	}
	s.messageID = *resp.Data.MessageId
	s.sequence = 1

	go s.runUpdateLoop()

	return nil
}

func (s *FeishuStreamingSession) runUpdateLoop() {
	for {
		select {
		case text := <-s.updateCh:
			s.doUpdate(text)
		case <-s.done:
			return
		}
	}
}

func (s *FeishuStreamingSession) doUpdate(text string) {
	s.mu.Lock()
	s.currentText = text
	s.lastUpdate = time.Now()
	seq := s.sequence + 1
	s.sequence = seq
	s.mu.Unlock()

	token, err := getFeishuToken(s.appID, s.appSecret, s.domain)
	if err != nil {
		return
	}

	updateURL := fmt.Sprintf("%s/cardkit/v1/cards/%s/elements/content/content", s.apiBase, s.cardID)
	updateBody := map[string]interface{}{
		"content":  text,
		"sequence": seq,
		"uuid":     fmt.Sprintf("s_%s_%d", s.cardID, seq),
	}
	updateJSON, _ := json.Marshal(updateBody)

	req, _ := http.NewRequest("PUT", updateURL, bytes.NewReader(updateJSON))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	(&http.Client{Timeout: 10 * time.Second}).Do(req)
}

func (s *FeishuStreamingSession) Update(text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}

	now := time.Now()
	if now.Sub(s.lastUpdate) < time.Duration(s.throttleMs)*time.Millisecond {
		s.pendingText = text
		return
	}

	select {
	case s.updateCh <- text:
	default:
	}
}

func (s *FeishuStreamingSession) Close(finalText string) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	close(s.done)

	text := finalText
	if text == "" {
		text = s.pendingText
	}
	if text == "" {
		text = s.currentText
	}
	s.mu.Unlock()

	if text != s.currentText {
		s.doUpdate(text)
	}

	return s.closeStreamingMode(text)
}

func (s *FeishuStreamingSession) closeStreamingMode(finalText string) error {
	token, err := getFeishuToken(s.appID, s.appSecret, s.domain)
	if err != nil {
		return err
	}

	s.mu.Lock()
	seq := s.sequence + 1
	s.sequence = seq
	s.mu.Unlock()

	summary := TruncateSummary(finalText, 50)
	settings := map[string]interface{}{
		"settings": JSONMarshal(map[string]interface{}{
			"config": map[string]interface{}{
				"streaming_mode": false,
				"summary":        map[string]string{"content": summary},
			},
		}),
		"sequence": seq,
		"uuid":     fmt.Sprintf("c_%s_%d", s.cardID, seq),
	}
	settingsJSON, _ := json.Marshal(settings)

	url := fmt.Sprintf("%s/cardkit/v1/cards/%s/settings", s.apiBase, s.cardID)
	req, _ := http.NewRequest("PATCH", url, bytes.NewReader(settingsJSON))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	(&http.Client{Timeout: 10 * time.Second}).Do(req)
	return nil
}

func (s *FeishuStreamingSession) IsActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cardID != "" && !s.closed
}

func JSONMarshal(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
