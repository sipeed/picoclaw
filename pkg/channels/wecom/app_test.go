package wecom

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

// generateTestAESKeyApp generates a valid test AES key for WeCom App
func generateTestAESKeyApp() string {
	// AES key needs to be 32 bytes (256 bits) for AES-256
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	// Return base64 encoded key without padding
	return base64.StdEncoding.EncodeToString(key)[:43]
}

// encryptTestMessageApp encrypts a message for testing WeCom App
func encryptTestMessageApp(message, aesKey string) (string, error) {
	// Decode AES key
	key, err := base64.StdEncoding.DecodeString(aesKey + "=")
	if err != nil {
		return "", err
	}

	// Prepare message: random(16) + msg_len(4) + msg + corp_id
	random := make([]byte, 0, 16)
	for i := range 16 {
		random = append(random, byte(i+1))
	}

	msgBytes := []byte(message)
	corpID := []byte("test_corp_id")

	msgLen := uint32(len(msgBytes))
	lenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBytes, msgLen)

	plainText := append(random, lenBytes...)
	plainText = append(plainText, msgBytes...)
	plainText = append(plainText, corpID...)

	// PKCS7 padding
	blockSize := aes.BlockSize
	padding := blockSize - len(plainText)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	plainText = append(plainText, padText...)

	// Encrypt
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	mode := cipher.NewCBCEncrypter(block, key[:aes.BlockSize])
	cipherText := make([]byte, len(plainText))
	mode.CryptBlocks(cipherText, plainText)

	return base64.StdEncoding.EncodeToString(cipherText), nil
}

// generateSignatureApp generates a signature for testing WeCom App
func generateSignatureApp(token, timestamp, nonce, msgEncrypt string) string {
	params := []string{token, timestamp, nonce, msgEncrypt}
	sort.Strings(params)
	str := strings.Join(params, "")
	hash := sha1.Sum([]byte(str))
	return fmt.Sprintf("%x", hash)
}

func TestNewWeComAppChannel(t *testing.T) {
	msgBus := bus.NewMessageBus()

	t.Run("missing corp_id", func(t *testing.T) {
		cfg := config.WeComAppConfig{
			CorpID:     "",
			CorpSecret: "test_secret",
			AgentID:    1000002,
		}
		_, err := NewWeComAppChannel(cfg, msgBus)
		if err == nil {
			t.Error("expected error for missing corp_id, got nil")
		}
	})

	t.Run("missing corp_secret", func(t *testing.T) {
		cfg := config.WeComAppConfig{
			CorpID:     "test_corp_id",
			CorpSecret: "",
			AgentID:    1000002,
		}
		_, err := NewWeComAppChannel(cfg, msgBus)
		if err == nil {
			t.Error("expected error for missing corp_secret, got nil")
		}
	})

	t.Run("missing agent_id", func(t *testing.T) {
		cfg := config.WeComAppConfig{
			CorpID:     "test_corp_id",
			CorpSecret: "test_secret",
			AgentID:    0,
		}
		_, err := NewWeComAppChannel(cfg, msgBus)
		if err == nil {
			t.Error("expected error for missing agent_id, got nil")
		}
	})

	t.Run("valid config", func(t *testing.T) {
		cfg := config.WeComAppConfig{
			CorpID:     "test_corp_id",
			CorpSecret: "test_secret",
			AgentID:    1000002,
			AllowFrom:  []string{"user1", "user2"},
		}
		ch, err := NewWeComAppChannel(cfg, msgBus)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ch.Name() != "wecom_app" {
			t.Errorf("Name() = %q, want %q", ch.Name(), "wecom_app")
		}
		if ch.IsRunning() {
			t.Error("new channel should not be running")
		}
	})
}

func TestWeComAppChannelIsAllowed(t *testing.T) {
	msgBus := bus.NewMessageBus()

	t.Run("empty allowlist allows all", func(t *testing.T) {
		cfg := config.WeComAppConfig{
			CorpID:     "test_corp_id",
			CorpSecret: "test_secret",
			AgentID:    1000002,
			AllowFrom:  []string{},
		}
		ch, _ := NewWeComAppChannel(cfg, msgBus)
		if !ch.IsAllowed("any_user") {
			t.Error("empty allowlist should allow all users")
		}
	})

	t.Run("allowlist restricts users", func(t *testing.T) {
		cfg := config.WeComAppConfig{
			CorpID:     "test_corp_id",
			CorpSecret: "test_secret",
			AgentID:    1000002,
			AllowFrom:  []string{"allowed_user"},
		}
		ch, _ := NewWeComAppChannel(cfg, msgBus)
		if !ch.IsAllowed("allowed_user") {
			t.Error("allowed user should pass allowlist check")
		}
		if ch.IsAllowed("blocked_user") {
			t.Error("non-allowed user should be blocked")
		}
	})
}

func TestWeComAppVerifySignature(t *testing.T) {
	msgBus := bus.NewMessageBus()
	cfg := config.WeComAppConfig{
		CorpID:     "test_corp_id",
		CorpSecret: "test_secret",
		AgentID:    1000002,
		Token:      "test_token",
	}
	ch, _ := NewWeComAppChannel(cfg, msgBus)

	t.Run("valid signature", func(t *testing.T) {
		timestamp := "1234567890"
		nonce := "test_nonce"
		msgEncrypt := "test_message"
		expectedSig := generateSignatureApp("test_token", timestamp, nonce, msgEncrypt)

		if !verifySignature(ch.config.Token, expectedSig, timestamp, nonce, msgEncrypt) {
			t.Error("valid signature should pass verification")
		}
	})

	t.Run("invalid signature", func(t *testing.T) {
		timestamp := "1234567890"
		nonce := "test_nonce"
		msgEncrypt := "test_message"

		if verifySignature(ch.config.Token, "invalid_sig", timestamp, nonce, msgEncrypt) {
			t.Error("invalid signature should fail verification")
		}
	})

	t.Run("empty token skips verification", func(t *testing.T) {
		cfgEmpty := config.WeComAppConfig{
			CorpID:     "test_corp_id",
			CorpSecret: "test_secret",
			AgentID:    1000002,
			Token:      "",
		}
		chEmpty, _ := NewWeComAppChannel(cfgEmpty, msgBus)

		if !verifySignature(chEmpty.config.Token, "any_sig", "any_ts", "any_nonce", "any_msg") {
			t.Error("empty token should skip verification and return true")
		}
	})
}

func TestWeComAppDecryptMessage(t *testing.T) {
	msgBus := bus.NewMessageBus()

	t.Run("decrypt without AES key", func(t *testing.T) {
		cfg := config.WeComAppConfig{
			CorpID:         "test_corp_id",
			CorpSecret:     "test_secret",
			AgentID:        1000002,
			EncodingAESKey: "",
		}
		ch, _ := NewWeComAppChannel(cfg, msgBus)

		// Without AES key, message should be base64 decoded only
		plainText := "hello world"
		encoded := base64.StdEncoding.EncodeToString([]byte(plainText))

		result, err := decryptMessage(encoded, ch.config.EncodingAESKey)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != plainText {
			t.Errorf("decryptMessage() = %q, want %q", result, plainText)
		}
	})

	t.Run("decrypt with AES key", func(t *testing.T) {
		aesKey := generateTestAESKeyApp()
		cfg := config.WeComAppConfig{
			CorpID:         "test_corp_id",
			CorpSecret:     "test_secret",
			AgentID:        1000002,
			EncodingAESKey: aesKey,
		}
		ch, _ := NewWeComAppChannel(cfg, msgBus)

		originalMsg := "<xml><Content>Hello</Content></xml>"
		encrypted, err := encryptTestMessageApp(originalMsg, aesKey)
		if err != nil {
			t.Fatalf("failed to encrypt test message: %v", err)
		}

		result, err := decryptMessage(encrypted, ch.config.EncodingAESKey)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != originalMsg {
			t.Errorf("WeComDecryptMessage() = %q, want %q", result, originalMsg)
		}
	})

	t.Run("invalid base64", func(t *testing.T) {
		cfg := config.WeComAppConfig{
			CorpID:         "test_corp_id",
			CorpSecret:     "test_secret",
			AgentID:        1000002,
			EncodingAESKey: "",
		}
		ch, _ := NewWeComAppChannel(cfg, msgBus)

		_, err := decryptMessage("invalid_base64!!!", ch.config.EncodingAESKey)
		if err == nil {
			t.Error("expected error for invalid base64, got nil")
		}
	})

	t.Run("invalid AES key", func(t *testing.T) {
		cfg := config.WeComAppConfig{
			CorpID:         "test_corp_id",
			CorpSecret:     "test_secret",
			AgentID:        1000002,
			EncodingAESKey: "invalid_key",
		}
		ch, _ := NewWeComAppChannel(cfg, msgBus)

		_, err := decryptMessage(base64.StdEncoding.EncodeToString([]byte("test")), ch.config.EncodingAESKey)
		if err == nil {
			t.Error("expected error for invalid AES key, got nil")
		}
	})

	t.Run("ciphertext too short", func(t *testing.T) {
		aesKey := generateTestAESKeyApp()
		cfg := config.WeComAppConfig{
			CorpID:         "test_corp_id",
			CorpSecret:     "test_secret",
			AgentID:        1000002,
			EncodingAESKey: aesKey,
		}
		ch, _ := NewWeComAppChannel(cfg, msgBus)

		// Encrypt a very short message that results in ciphertext less than block size
		shortData := make([]byte, 8)
		_, err := decryptMessage(base64.StdEncoding.EncodeToString(shortData), ch.config.EncodingAESKey)
		if err == nil {
			t.Error("expected error for short ciphertext, got nil")
		}
	})
}

func TestWeComAppPKCS7Unpad(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{
			name:     "empty input",
			input:    []byte{},
			expected: []byte{},
		},
		{
			name:     "valid padding 3 bytes",
			input:    append([]byte("hello"), bytes.Repeat([]byte{3}, 3)...),
			expected: []byte("hello"),
		},
		{
			name:     "valid padding 16 bytes (full block)",
			input:    append([]byte("123456789012345"), bytes.Repeat([]byte{16}, 16)...),
			expected: []byte("123456789012345"),
		},
		{
			name:     "invalid padding larger than data",
			input:    []byte{20},
			expected: nil, // should return error
		},
		{
			name:     "invalid padding zero",
			input:    append([]byte("test"), byte(0)),
			expected: nil, // should return error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := pkcs7Unpad(tt.input)
			if tt.expected == nil {
				// This case should return an error
				if err == nil {
					t.Errorf("pkcs7Unpad() expected error for invalid padding, got result: %v", result)
				}
				return
			}
			if err != nil {
				t.Errorf("pkcs7Unpad() unexpected error: %v", err)
				return
			}
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("pkcs7Unpad() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestWeComAppHandleVerification(t *testing.T) {
	msgBus := bus.NewMessageBus()
	aesKey := generateTestAESKeyApp()
	cfg := config.WeComAppConfig{
		CorpID:         "test_corp_id",
		CorpSecret:     "test_secret",
		AgentID:        1000002,
		Token:          "test_token",
		EncodingAESKey: aesKey,
	}
	ch, _ := NewWeComAppChannel(cfg, msgBus)

	t.Run("valid verification request", func(t *testing.T) {
		echostr := "test_echostr_123"
		encryptedEchostr, _ := encryptTestMessageApp(echostr, aesKey)
		timestamp := "1234567890"
		nonce := "test_nonce"
		signature := generateSignatureApp("test_token", timestamp, nonce, encryptedEchostr)

		req := httptest.NewRequest(
			http.MethodGet,
			"/webhook/wecom-app?msg_signature="+signature+"&timestamp="+timestamp+"&nonce="+nonce+"&echostr="+encryptedEchostr,
			nil,
		)
		w := httptest.NewRecorder()

		ch.handleVerification(context.Background(), w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
		}
		if w.Body.String() != echostr {
			t.Errorf("response body = %q, want %q", w.Body.String(), echostr)
		}
	})

	t.Run("missing parameters", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/webhook/wecom-app?msg_signature=sig&timestamp=ts", nil)
		w := httptest.NewRecorder()

		ch.handleVerification(context.Background(), w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("invalid signature", func(t *testing.T) {
		echostr := "test_echostr"
		encryptedEchostr, _ := encryptTestMessageApp(echostr, aesKey)
		timestamp := "1234567890"
		nonce := "test_nonce"

		req := httptest.NewRequest(
			http.MethodGet,
			"/webhook/wecom-app?msg_signature=invalid_sig&timestamp="+timestamp+"&nonce="+nonce+"&echostr="+encryptedEchostr,
			nil,
		)
		w := httptest.NewRecorder()

		ch.handleVerification(context.Background(), w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusForbidden)
		}
	})
}

func TestWeComAppHandleMessageCallback(t *testing.T) {
	msgBus := bus.NewMessageBus()
	aesKey := generateTestAESKeyApp()
	cfg := config.WeComAppConfig{
		CorpID:         "test_corp_id",
		CorpSecret:     "test_secret",
		AgentID:        1000002,
		Token:          "test_token",
		EncodingAESKey: aesKey,
	}
	ch, _ := NewWeComAppChannel(cfg, msgBus)

	t.Run("valid message callback", func(t *testing.T) {
		// Create XML message
		xmlMsg := WeComXMLMessage{
			ToUserName:   "corp_id",
			FromUserName: "user123",
			CreateTime:   1234567890,
			MsgType:      "text",
			Content:      "Hello World",
			MsgId:        123456,
			AgentID:      1000002,
		}
		xmlData, _ := xml.Marshal(xmlMsg)

		// Encrypt message
		encrypted, _ := encryptTestMessageApp(string(xmlData), aesKey)

		// Create encrypted XML wrapper
		encryptedWrapper := struct {
			XMLName xml.Name `xml:"xml"`
			Encrypt string   `xml:"Encrypt"`
		}{
			Encrypt: encrypted,
		}
		wrapperData, _ := xml.Marshal(encryptedWrapper)

		timestamp := "1234567890"
		nonce := "test_nonce"
		signature := generateSignatureApp("test_token", timestamp, nonce, encrypted)

		req := httptest.NewRequest(
			http.MethodPost,
			"/webhook/wecom-app?msg_signature="+signature+"&timestamp="+timestamp+"&nonce="+nonce,
			bytes.NewReader(wrapperData),
		)
		w := httptest.NewRecorder()

		ch.handleMessageCallback(context.Background(), w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
		}
		if w.Body.String() != "success" {
			t.Errorf("response body = %q, want %q", w.Body.String(), "success")
		}
	})

	t.Run("missing parameters", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhook/wecom-app?msg_signature=sig", nil)
		w := httptest.NewRecorder()

		ch.handleMessageCallback(context.Background(), w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("invalid XML", func(t *testing.T) {
		timestamp := "1234567890"
		nonce := "test_nonce"
		signature := generateSignatureApp("test_token", timestamp, nonce, "")

		req := httptest.NewRequest(
			http.MethodPost,
			"/webhook/wecom-app?msg_signature="+signature+"&timestamp="+timestamp+"&nonce="+nonce,
			strings.NewReader("invalid xml"),
		)
		w := httptest.NewRecorder()

		ch.handleMessageCallback(context.Background(), w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("invalid signature", func(t *testing.T) {
		encryptedWrapper := struct {
			XMLName xml.Name `xml:"xml"`
			Encrypt string   `xml:"Encrypt"`
		}{
			Encrypt: "encrypted_data",
		}
		wrapperData, _ := xml.Marshal(encryptedWrapper)

		timestamp := "1234567890"
		nonce := "test_nonce"

		req := httptest.NewRequest(
			http.MethodPost,
			"/webhook/wecom-app?msg_signature=invalid_sig&timestamp="+timestamp+"&nonce="+nonce,
			bytes.NewReader(wrapperData),
		)
		w := httptest.NewRecorder()

		ch.handleMessageCallback(context.Background(), w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusForbidden)
		}
	})
}

func TestWeComAppProcessMessage(t *testing.T) {
	msgBus := bus.NewMessageBus()
	cfg := config.WeComAppConfig{
		CorpID:     "test_corp_id",
		CorpSecret: "test_secret",
		AgentID:    1000002,
	}
	ch, _ := NewWeComAppChannel(cfg, msgBus)

	t.Run("process text message", func(t *testing.T) {
		msg := WeComXMLMessage{
			ToUserName:   "corp_id",
			FromUserName: "user123",
			CreateTime:   1234567890,
			MsgType:      "text",
			Content:      "Hello World",
			MsgId:        123456,
			AgentID:      1000002,
		}

		// Should not panic
		ch.processMessage(context.Background(), msg)
	})

	t.Run("process image message", func(t *testing.T) {
		msg := WeComXMLMessage{
			ToUserName:   "corp_id",
			FromUserName: "user123",
			CreateTime:   1234567890,
			MsgType:      "image",
			PicUrl:       "https://example.com/image.jpg",
			MediaId:      "media_123",
			MsgId:        123456,
			AgentID:      1000002,
		}

		// Should not panic
		ch.processMessage(context.Background(), msg)
	})

	t.Run("process voice message", func(t *testing.T) {
		msg := WeComXMLMessage{
			ToUserName:   "corp_id",
			FromUserName: "user123",
			CreateTime:   1234567890,
			MsgType:      "voice",
			MediaId:      "media_123",
			Format:       "amr",
			MsgId:        123456,
			AgentID:      1000002,
		}

		// Should not panic
		ch.processMessage(context.Background(), msg)
	})

	t.Run("skip unsupported message type", func(t *testing.T) {
		msg := WeComXMLMessage{
			ToUserName:   "corp_id",
			FromUserName: "user123",
			CreateTime:   1234567890,
			MsgType:      "video",
			MsgId:        123456,
			AgentID:      1000002,
		}

		// Should not panic
		ch.processMessage(context.Background(), msg)
	})

	t.Run("process event message", func(t *testing.T) {
		msg := WeComXMLMessage{
			ToUserName:   "corp_id",
			FromUserName: "user123",
			CreateTime:   1234567890,
			MsgType:      "event",
			Event:        "subscribe",
			MsgId:        123456,
			AgentID:      1000002,
		}

		// Should not panic
		ch.processMessage(context.Background(), msg)
	})
}

func TestWeComAppHandleWebhook(t *testing.T) {
	msgBus := bus.NewMessageBus()
	cfg := config.WeComAppConfig{
		CorpID:     "test_corp_id",
		CorpSecret: "test_secret",
		AgentID:    1000002,
		Token:      "test_token",
	}
	ch, _ := NewWeComAppChannel(cfg, msgBus)

	t.Run("GET request calls verification", func(t *testing.T) {
		echostr := "test_echostr"
		encoded := base64.StdEncoding.EncodeToString([]byte(echostr))
		timestamp := "1234567890"
		nonce := "test_nonce"
		signature := generateSignatureApp("test_token", timestamp, nonce, encoded)

		req := httptest.NewRequest(
			http.MethodGet,
			"/webhook/wecom-app?msg_signature="+signature+"&timestamp="+timestamp+"&nonce="+nonce+"&echostr="+encoded,
			nil,
		)
		w := httptest.NewRecorder()

		ch.handleWebhook(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
		}
	})

	t.Run("POST request calls message callback", func(t *testing.T) {
		encryptedWrapper := struct {
			XMLName xml.Name `xml:"xml"`
			Encrypt string   `xml:"Encrypt"`
		}{
			Encrypt: base64.StdEncoding.EncodeToString([]byte("test")),
		}
		wrapperData, _ := xml.Marshal(encryptedWrapper)

		timestamp := "1234567890"
		nonce := "test_nonce"
		signature := generateSignatureApp("test_token", timestamp, nonce, encryptedWrapper.Encrypt)

		req := httptest.NewRequest(
			http.MethodPost,
			"/webhook/wecom-app?msg_signature="+signature+"&timestamp="+timestamp+"&nonce="+nonce,
			bytes.NewReader(wrapperData),
		)
		w := httptest.NewRecorder()

		ch.handleWebhook(w, req)

		// Should not be method not allowed
		if w.Code == http.StatusMethodNotAllowed {
			t.Error("POST request should not return Method Not Allowed")
		}
	})

	t.Run("unsupported method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/webhook/wecom-app", nil)
		w := httptest.NewRecorder()

		ch.handleWebhook(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusMethodNotAllowed)
		}
	})
}

func TestWeComAppHandleHealth(t *testing.T) {
	msgBus := bus.NewMessageBus()
	cfg := config.WeComAppConfig{
		CorpID:     "test_corp_id",
		CorpSecret: "test_secret",
		AgentID:    1000002,
	}
	ch, _ := NewWeComAppChannel(cfg, msgBus)

	req := httptest.NewRequest(http.MethodGet, "/health/wecom-app", nil)
	w := httptest.NewRecorder()

	ch.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %q, want %q", contentType, "application/json")
	}

	body := w.Body.String()
	if !strings.Contains(body, "status") || !strings.Contains(body, "running") || !strings.Contains(body, "has_token") {
		t.Errorf("response body should contain status, running, and has_token fields, got: %s", body)
	}
}

func TestWeComAppAccessToken(t *testing.T) {
	msgBus := bus.NewMessageBus()
	cfg := config.WeComAppConfig{
		CorpID:     "test_corp_id",
		CorpSecret: "test_secret",
		AgentID:    1000002,
	}
	ch, _ := NewWeComAppChannel(cfg, msgBus)

	t.Run("get empty access token initially", func(t *testing.T) {
		token := ch.getAccessToken()
		if token != "" {
			t.Errorf("getAccessToken() = %q, want empty string", token)
		}
	})

	t.Run("set and get access token", func(t *testing.T) {
		ch.tokenMu.Lock()
		ch.accessToken = "test_token_123"
		ch.tokenExpiry = time.Now().Add(1 * time.Hour)
		ch.tokenMu.Unlock()

		token := ch.getAccessToken()
		if token != "test_token_123" {
			t.Errorf("getAccessToken() = %q, want %q", token, "test_token_123")
		}
	})

	t.Run("expired token returns empty", func(t *testing.T) {
		ch.tokenMu.Lock()
		ch.accessToken = "expired_token"
		ch.tokenExpiry = time.Now().Add(-1 * time.Hour)
		ch.tokenMu.Unlock()

		token := ch.getAccessToken()
		if token != "" {
			t.Errorf("getAccessToken() = %q, want empty string for expired token", token)
		}
	})
}

func TestWeComAppMessageStructures(t *testing.T) {
	t.Run("WeComTextMessage structure", func(t *testing.T) {
		msg := WeComTextMessage{
			ToUser:  "user123",
			MsgType: "text",
			AgentID: 1000002,
		}
		msg.Text.Content = "Hello World"

		if msg.ToUser != "user123" {
			t.Errorf("ToUser = %q, want %q", msg.ToUser, "user123")
		}
		if msg.MsgType != "text" {
			t.Errorf("MsgType = %q, want %q", msg.MsgType, "text")
		}
		if msg.AgentID != 1000002 {
			t.Errorf("AgentID = %d, want %d", msg.AgentID, 1000002)
		}
		if msg.Text.Content != "Hello World" {
			t.Errorf("Text.Content = %q, want %q", msg.Text.Content, "Hello World")
		}

		// Test JSON marshaling
		jsonData, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("failed to marshal JSON: %v", err)
		}

		var unmarshaled WeComTextMessage
		err = json.Unmarshal(jsonData, &unmarshaled)
		if err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		if unmarshaled.ToUser != msg.ToUser {
			t.Errorf("JSON round-trip failed for ToUser")
		}
	})

	t.Run("WeComMarkdownMessage structure", func(t *testing.T) {
		msg := WeComMarkdownMessage{
			ToUser:  "user123",
			MsgType: "markdown",
			AgentID: 1000002,
		}
		msg.Markdown.Content = "# Hello\nWorld"

		if msg.Markdown.Content != "# Hello\nWorld" {
			t.Errorf("Markdown.Content = %q, want %q", msg.Markdown.Content, "# Hello\nWorld")
		}

		// Test JSON marshaling
		jsonData, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("failed to marshal JSON: %v", err)
		}

		if !bytes.Contains(jsonData, []byte("markdown")) {
			t.Error("JSON should contain 'markdown' field")
		}
	})

	t.Run("WeComImageMessage structure", func(t *testing.T) {
		msg := WeComImageMessage{
			ToUser:  "user123",
			MsgType: "image",
			AgentID: 1000002,
		}
		msg.Image.MediaID = "media_123456"

		if msg.Image.MediaID != "media_123456" {
			t.Errorf("Image.MediaID = %q, want %q", msg.Image.MediaID, "media_123456")
		}
		if msg.ToUser != "user123" {
			t.Errorf("ToUser = %q, want %q", msg.ToUser, "user123")
		}
		if msg.MsgType != "image" {
			t.Errorf("MsgType = %q, want %q", msg.MsgType, "image")
		}
		if msg.AgentID != 1000002 {
			t.Errorf("AgentID = %d, want %d", msg.AgentID, 1000002)
		}
	})

	t.Run("WeComAccessTokenResponse structure", func(t *testing.T) {
		jsonData := `{
			"errcode": 0,
			"errmsg": "ok",
			"access_token": "test_access_token",
			"expires_in": 7200
		}`

		var resp WeComAccessTokenResponse
		err := json.Unmarshal([]byte(jsonData), &resp)
		if err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		if resp.ErrCode != 0 {
			t.Errorf("ErrCode = %d, want %d", resp.ErrCode, 0)
		}
		if resp.ErrMsg != "ok" {
			t.Errorf("ErrMsg = %q, want %q", resp.ErrMsg, "ok")
		}
		if resp.AccessToken != "test_access_token" {
			t.Errorf("AccessToken = %q, want %q", resp.AccessToken, "test_access_token")
		}
		if resp.ExpiresIn != 7200 {
			t.Errorf("ExpiresIn = %d, want %d", resp.ExpiresIn, 7200)
		}
	})

	t.Run("WeComSendMessageResponse structure", func(t *testing.T) {
		jsonData := `{
			"errcode": 0,
			"errmsg": "ok",
			"invaliduser": "",
			"invalidparty": "",
			"invalidtag": ""
		}`

		var resp WeComSendMessageResponse
		err := json.Unmarshal([]byte(jsonData), &resp)
		if err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		if resp.ErrCode != 0 {
			t.Errorf("ErrCode = %d, want %d", resp.ErrCode, 0)
		}
		if resp.ErrMsg != "ok" {
			t.Errorf("ErrMsg = %q, want %q", resp.ErrMsg, "ok")
		}
	})
}

func TestWeComAppXMLMessageStructure(t *testing.T) {
	xmlData := `<?xml version="1.0"?>
<xml>
	<ToUserName><![CDATA[corp_id]]></ToUserName>
	<FromUserName><![CDATA[user123]]></FromUserName>
	<CreateTime>1234567890</CreateTime>
	<MsgType><![CDATA[text]]></MsgType>
	<Content><![CDATA[Hello World]]></Content>
	<MsgId>1234567890123456</MsgId>
	<AgentID>1000002</AgentID>
</xml>`

	var msg WeComXMLMessage
	err := xml.Unmarshal([]byte(xmlData), &msg)
	if err != nil {
		t.Fatalf("failed to unmarshal XML: %v", err)
	}

	if msg.ToUserName != "corp_id" {
		t.Errorf("ToUserName = %q, want %q", msg.ToUserName, "corp_id")
	}
	if msg.FromUserName != "user123" {
		t.Errorf("FromUserName = %q, want %q", msg.FromUserName, "user123")
	}
	if msg.CreateTime != 1234567890 {
		t.Errorf("CreateTime = %d, want %d", msg.CreateTime, 1234567890)
	}
	if msg.MsgType != "text" {
		t.Errorf("MsgType = %q, want %q", msg.MsgType, "text")
	}
	if msg.Content != "Hello World" {
		t.Errorf("Content = %q, want %q", msg.Content, "Hello World")
	}
	if msg.MsgId != 1234567890123456 {
		t.Errorf("MsgId = %d, want %d", msg.MsgId, 1234567890123456)
	}
	if msg.AgentID != 1000002 {
		t.Errorf("AgentID = %d, want %d", msg.AgentID, 1000002)
	}
}

func TestWeComAppXMLMessageImage(t *testing.T) {
	xmlData := `<?xml version="1.0"?>
<xml>
	<ToUserName><![CDATA[corp_id]]></ToUserName>
	<FromUserName><![CDATA[user123]]></FromUserName>
	<CreateTime>1234567890</CreateTime>
	<MsgType><![CDATA[image]]></MsgType>
	<PicUrl><![CDATA[https://example.com/image.jpg]]></PicUrl>
	<MediaId><![CDATA[media_123]]></MediaId>
	<MsgId>1234567890123456</MsgId>
	<AgentID>1000002</AgentID>
</xml>`

	var msg WeComXMLMessage
	err := xml.Unmarshal([]byte(xmlData), &msg)
	if err != nil {
		t.Fatalf("failed to unmarshal XML: %v", err)
	}

	if msg.MsgType != "image" {
		t.Errorf("MsgType = %q, want %q", msg.MsgType, "image")
	}
	if msg.PicUrl != "https://example.com/image.jpg" {
		t.Errorf("PicUrl = %q, want %q", msg.PicUrl, "https://example.com/image.jpg")
	}
	if msg.MediaId != "media_123" {
		t.Errorf("MediaId = %q, want %q", msg.MediaId, "media_123")
	}
}

func TestWeComAppXMLMessageVoice(t *testing.T) {
	xmlData := `<?xml version="1.0"?>
<xml>
	<ToUserName><![CDATA[corp_id]]></ToUserName>
	<FromUserName><![CDATA[user123]]></FromUserName>
	<CreateTime>1234567890</CreateTime>
	<MsgType><![CDATA[voice]]></MsgType>
	<MediaId><![CDATA[media_123]]></MediaId>
	<Format><![CDATA[amr]]></Format>
	<MsgId>1234567890123456</MsgId>
	<AgentID>1000002</AgentID>
</xml>`

	var msg WeComXMLMessage
	err := xml.Unmarshal([]byte(xmlData), &msg)
	if err != nil {
		t.Fatalf("failed to unmarshal XML: %v", err)
	}

	if msg.MsgType != "voice" {
		t.Errorf("MsgType = %q, want %q", msg.MsgType, "voice")
	}
	if msg.Format != "amr" {
		t.Errorf("Format = %q, want %q", msg.Format, "amr")
	}
}

func TestWeComAppXMLMessageLocation(t *testing.T) {
	xmlData := `<?xml version="1.0"?>
<xml>
	<ToUserName><![CDATA[corp_id]]></ToUserName>
	<FromUserName><![CDATA[user123]]></FromUserName>
	<CreateTime>1234567890</CreateTime>
	<MsgType><![CDATA[location]]></MsgType>
	<Location_X>39.9042</Location_X>
	<Location_Y>116.4074</Location_Y>
	<Scale>16</Scale>
	<Label><![CDATA[Beijing]]></Label>
	<MsgId>1234567890123456</MsgId>
	<AgentID>1000002</AgentID>
</xml>`

	var msg WeComXMLMessage
	err := xml.Unmarshal([]byte(xmlData), &msg)
	if err != nil {
		t.Fatalf("failed to unmarshal XML: %v", err)
	}

	if msg.MsgType != "location" {
		t.Errorf("MsgType = %q, want %q", msg.MsgType, "location")
	}
	if msg.LocationX != 39.9042 {
		t.Errorf("LocationX = %f, want %f", msg.LocationX, 39.9042)
	}
	if msg.LocationY != 116.4074 {
		t.Errorf("LocationY = %f, want %f", msg.LocationY, 116.4074)
	}
	if msg.Scale != 16 {
		t.Errorf("Scale = %d, want %d", msg.Scale, 16)
	}
	if msg.Label != "Beijing" {
		t.Errorf("Label = %q, want %q", msg.Label, "Beijing")
	}
}

func TestWeComAppXMLMessageLink(t *testing.T) {
	xmlData := `<?xml version="1.0"?>
<xml>
	<ToUserName><![CDATA[corp_id]]></ToUserName>
	<FromUserName><![CDATA[user123]]></FromUserName>
	<CreateTime>1234567890</CreateTime>
	<MsgType><![CDATA[link]]></MsgType>
	<Title><![CDATA[Link Title]]></Title>
	<Description><![CDATA[Link Description]]></Description>
	<Url><![CDATA[https://example.com]]></Url>
	<MsgId>1234567890123456</MsgId>
	<AgentID>1000002</AgentID>
</xml>`

	var msg WeComXMLMessage
	err := xml.Unmarshal([]byte(xmlData), &msg)
	if err != nil {
		t.Fatalf("failed to unmarshal XML: %v", err)
	}

	if msg.MsgType != "link" {
		t.Errorf("MsgType = %q, want %q", msg.MsgType, "link")
	}
	if msg.Title != "Link Title" {
		t.Errorf("Title = %q, want %q", msg.Title, "Link Title")
	}
	if msg.Description != "Link Description" {
		t.Errorf("Description = %q, want %q", msg.Description, "Link Description")
	}
	if msg.Url != "https://example.com" {
		t.Errorf("Url = %q, want %q", msg.Url, "https://example.com")
	}
}

func TestWeComAppXMLMessageEvent(t *testing.T) {
	xmlData := `<?xml version="1.0"?>
<xml>
	<ToUserName><![CDATA[corp_id]]></ToUserName>
	<FromUserName><![CDATA[user123]]></FromUserName>
	<CreateTime>1234567890</CreateTime>
	<MsgType><![CDATA[event]]></MsgType>
	<Event><![CDATA[subscribe]]></Event>
	<EventKey><![CDATA[event_key_123]]></EventKey>
	<AgentID>1000002</AgentID>
</xml>`

	var msg WeComXMLMessage
	err := xml.Unmarshal([]byte(xmlData), &msg)
	if err != nil {
		t.Fatalf("failed to unmarshal XML: %v", err)
	}

	if msg.MsgType != "event" {
		t.Errorf("MsgType = %q, want %q", msg.MsgType, "event")
	}
	if msg.Event != "subscribe" {
		t.Errorf("Event = %q, want %q", msg.Event, "subscribe")
	}
	if msg.EventKey != "event_key_123" {
		t.Errorf("EventKey = %q, want %q", msg.EventKey, "event_key_123")
	}
}
