package dingtalk

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

// mockTransport is a custom http.RoundTripper for testing
type mockTransport struct {
	response     *http.Response
	responseBody any
	requestErr   error
	requests     []*http.Request
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	m.requests = append(m.requests, req)

	if m.requestErr != nil {
		return nil, m.requestErr
	}

	if m.response != nil {
		return m.response, nil
	}

	// Generate response from responseBody
	bodyBytes, _ := json.Marshal(m.responseBody)
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(bodyBytes)),
	}, nil
}

// TestRefreshAccessToken tests the access token refresh functionality
func TestRefreshAccessToken(t *testing.T) {
	tests := []struct {
		name           string
		clientID       string
		clientSecret   string
		serverResponse any
		serverStatus   int
		expectError    bool
		expectToken    string
	}{
		{
			name:         "successful token refresh",
			clientID:     "test_app_key",
			clientSecret: "test_app_secret",
			serverResponse: map[string]any{
				"accessToken": "test_access_token_123",
				"expireIn":    7200,
			},
			serverStatus: http.StatusOK,
			expectError:  false,
			expectToken:  "test_access_token_123",
		},
		{
			name:         "invalid credentials",
			clientID:     "invalid_client",
			clientSecret: "invalid_secret",
			serverResponse: map[string]any{
				"code":    "invalid.client",
				"message": "Invalid client credentials",
			},
			serverStatus: http.StatusBadRequest,
			expectError:  true,
			expectToken:  "",
		},
		{
			name:         "server error",
			clientID:     "test_app_key",
			clientSecret: "test_app_secret",
			serverResponse: map[string]any{
				"code":    "server.error",
				"message": "Internal server error",
			},
			serverStatus: http.StatusInternalServerError,
			expectError:  true,
			expectToken:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock transport
			mock := &mockTransport{}

			if tt.serverStatus > 0 {
				bodyBytes, _ := json.Marshal(tt.serverResponse)
				mock.response = &http.Response{
					StatusCode: tt.serverStatus,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(bytes.NewReader(bodyBytes)),
				}
			}

			// Create HTTP client with mock transport
			httpClient := &http.Client{
				Transport: mock,
			}

			// Create channel
			cfg := config.DingTalkConfig{
				Enabled:      true,
				ClientID:     tt.clientID,
				ClientSecret: tt.clientSecret,
			}

			channel := &DingTalkChannel{
				config:       cfg,
				clientID:     tt.clientID,
				clientSecret: tt.clientSecret,
				httpClient:   httpClient,
				ctx:          context.Background(),
			}

			// Test the token refresh
			err := channel.refreshAccessToken()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if channel.accessToken != tt.expectToken {
					t.Errorf("Expected token %s, got %s", tt.expectToken, channel.accessToken)
				}
			}

			// Verify request format if a request was made
			if len(mock.requests) > 0 {
				req := mock.requests[0]

				// Verify method
				if req.Method != "POST" {
					t.Errorf("Expected POST request, got %s", req.Method)
				}

				// Verify URL path is correct
				expectedPath := "/v1.0/oauth2/accessToken"
				if req.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, req.URL.Path)
				}

				// Verify Content-Type
				if req.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type: application/json, got %s", req.Header.Get("Content-Type"))
				}

				// Parse and verify request body
				bodyBytes, _ := io.ReadAll(req.Body)
				var reqBody map[string]string
				json.Unmarshal(bodyBytes, &reqBody)

				// Verify new API format: appKey and appSecret
				if reqBody["appKey"] != tt.clientID {
					t.Errorf("Expected appKey %s, got %s", tt.clientID, reqBody["appKey"])
				}
				if reqBody["appSecret"] != tt.clientSecret {
					t.Errorf("Expected appSecret %s, got %s", tt.clientSecret, reqBody["appSecret"])
				}
				// Verify old fields are NOT present
				if _, exists := reqBody["client_id"]; exists {
					t.Error("client_id should not be present in request body")
				}
				if _, exists := reqBody["client_secret"]; exists {
					t.Error("client_secret should not be present in request body")
				}
				if _, exists := reqBody["grant_type"]; exists {
					t.Error("grant_type should not be present in request body")
				}
			}
		})
	}
}

// TestChatInfoStorage tests that chat info is properly stored
func TestChatInfoStorage(t *testing.T) {
	channel := &DingTalkChannel{
		chatInfos: sync.Map{},
	}

	now := time.Now()
	info := &chatInfo{
		sessionWebhook:     "https://webhook.example.com/test",
		sessionWebhookExp:  now.Add(2 * time.Hour),
		senderStaffId:      "staff123",
		openConversationId: "conv456",
		conversationType:   "1",
	}

	// Store the info
	channel.chatInfos.Store("test_chat_id", info)

	// Retrieve and verify
	retrieved, ok := channel.getChatInfo("test_chat_id")
	if !ok {
		t.Fatal("Failed to retrieve chat info")
	}

	if retrieved.sessionWebhook != info.sessionWebhook {
		t.Errorf("Expected sessionWebhook %s, got %s", info.sessionWebhook, retrieved.sessionWebhook)
	}
	if retrieved.senderStaffId != info.senderStaffId {
		t.Errorf("Expected senderStaffId %s, got %s", info.senderStaffId, retrieved.senderStaffId)
	}
	if retrieved.conversationType != info.conversationType {
		t.Errorf("Expected conversationType %s, got %s", info.conversationType, retrieved.conversationType)
	}

	// Test non-existent chat
	_, ok = channel.getChatInfo("non_existent")
	if ok {
		t.Error("Expected false for non-existent chat")
	}
}

// TestBuildMarkdownMsgParam tests the markdown message parameter builder
func TestBuildMarkdownMsgParam(t *testing.T) {
	result := buildMarkdownMsgParam("Test Title", "Test Content")

	var parsed map[string]string
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	if parsed["title"] != "Test Title" {
		t.Errorf("Expected title 'Test Title', got %s", parsed["title"])
	}
	if parsed["text"] != "Test Content" {
		t.Errorf("Expected text 'Test Content', got %s", parsed["text"])
	}
}

// TestTokenExpiry tests the token expiry logic
func TestTokenExpiry(t *testing.T) {
	channel := &DingTalkChannel{
		accessToken: "test_token",
		tokenExpiry: time.Now().Add(1 * time.Hour),
	}

	// Token should be valid
	token := channel.getAccessToken()
	if token != "test_token" {
		t.Errorf("Expected token 'test_token', got %s", token)
	}

	// Set token as expired
	channel.tokenExpiry = time.Now().Add(-1 * time.Hour)

	// Token should be empty (expired)
	token = channel.getAccessToken()
	if token != "" {
		t.Errorf("Expected empty token for expired, got %s", token)
	}
}

// TestSendProactiveWithoutToken tests that proactive send fails gracefully without token
func TestSendProactiveWithoutToken(t *testing.T) {
	channel := &DingTalkChannel{
		chatInfos: sync.Map{},
	}

	// Store some chat info
	channel.chatInfos.Store("test_chat", &chatInfo{
		conversationType: "1",
		senderStaffId:    "staff123",
	})

	// No token set, should fail
	err := channel.sendProactive(context.Background(), "test_chat", "test message")
	if err == nil {
		t.Error("Expected error when no access token available")
	}
}

// TestSendProactiveWithoutChatInfo tests that proactive send works without stored chat info
// When no chatInfo is stored, it assumes single chat and uses chatID as staffId
func TestSendProactiveWithoutChatInfo(t *testing.T) {
	mock := &mockTransport{
		responseBody: map[string]any{},
	}
	httpClient := &http.Client{
		Transport: mock,
	}

	channel := &DingTalkChannel{
		chatInfos:   sync.Map{},
		accessToken: "test_token",
		tokenExpiry: time.Now().Add(1 * time.Hour),
		httpClient:  httpClient,
		clientID:    "test_app_key", // robotCode = clientID
	}

	// No chat info stored - should try to send as single chat using clientID as robotCode
	err := channel.sendProactive(context.Background(), "staff123", "test message")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify the request was made to the correct endpoint
	if len(mock.requests) != 1 {
		t.Fatalf("Expected 1 request, got %d", len(mock.requests))
	}

	req := mock.requests[0]
	expectedPath := "/v1.0/robot/oToMessages/batchSend"
	if req.URL.Path != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, req.URL.Path)
	}

	// Verify the header
	if req.Header.Get("X-Acs-Dingtalk-Access-Token") != "test_token" {
		t.Errorf("Expected X-Acs-Dingtalk-Access-Token header, got %s", req.Header.Get("X-Acs-Dingtalk-Access-Token"))
	}
}
