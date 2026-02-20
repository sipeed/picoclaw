// PicoClaw - Ultra-lightweight personal AI agent
// WeCom Bot (企业微信智能机器人) channel tests

package channels

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

// generateTestAESKey generates a valid test AES key
func generateTestAESKey() string {
	// AES key needs to be 32 bytes (256 bits) for AES-256
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	// Return base64 encoded key without padding
	return base64.StdEncoding.EncodeToString(key)[:43]
}

// encryptTestMessage encrypts a message for testing
func encryptTestMessage(message, aesKey string) (string, error) {
	// Decode AES key
	key, err := base64.StdEncoding.DecodeString(aesKey + "=")
	if err != nil {
		return "", err
	}

	// Prepare message: random(16) + msg_len(4) + msg + corp_id
	random := make([]byte, 0, 16)
	for i := 0; i < 16; i++ {
		random = append(random, byte(i))
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

// generateSignature generates a signature for testing
func generateSignature(token, timestamp, nonce, msgEncrypt string) string {
	params := []string{token, timestamp, nonce, msgEncrypt}
	sort.Strings(params)
	str := strings.Join(params, "")
	hash := sha1.Sum([]byte(str))
	return fmt.Sprintf("%x", hash)
}

func TestNewWeComBotChannel(t *testing.T) {
	msgBus := bus.NewMessageBus()

	t.Run("missing token", func(t *testing.T) {
		cfg := config.WeComConfig{
			Token:      "",
			WebhookURL: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test",
		}
		_, err := NewWeComBotChannel(cfg, msgBus)
		if err == nil {
			t.Error("expected error for missing token, got nil")
		}
	})

	t.Run("missing webhook_url", func(t *testing.T) {
		cfg := config.WeComConfig{
			Token:      "test_token",
			WebhookURL: "",
		}
		_, err := NewWeComBotChannel(cfg, msgBus)
		if err == nil {
			t.Error("expected error for missing webhook_url, got nil")
		}
	})

	t.Run("valid config", func(t *testing.T) {
		cfg := config.WeComConfig{
			Token:      "test_token",
			WebhookURL: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test",
			AllowFrom:  []string{"user1", "user2"},
		}
		ch, err := NewWeComBotChannel(cfg, msgBus)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ch.Name() != "wecom" {
			t.Errorf("Name() = %q, want %q", ch.Name(), "wecom")
		}
		if ch.IsRunning() {
			t.Error("new channel should not be running")
		}
	})
}

func TestWeComBotChannelIsAllowed(t *testing.T) {
	msgBus := bus.NewMessageBus()

	t.Run("empty allowlist allows all", func(t *testing.T) {
		cfg := config.WeComConfig{
			Token:      "test_token",
			WebhookURL: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test",
			AllowFrom:  []string{},
		}
		ch, _ := NewWeComBotChannel(cfg, msgBus)
		if !ch.IsAllowed("any_user") {
			t.Error("empty allowlist should allow all users")
		}
	})

	t.Run("allowlist restricts users", func(t *testing.T) {
		cfg := config.WeComConfig{
			Token:      "test_token",
			WebhookURL: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test",
			AllowFrom:  []string{"allowed_user"},
		}
		ch, _ := NewWeComBotChannel(cfg, msgBus)
		if !ch.IsAllowed("allowed_user") {
			t.Error("allowed user should pass allowlist check")
		}
		if ch.IsAllowed("blocked_user") {
			t.Error("non-allowed user should be blocked")
		}
	})
}

func TestWeComBotVerifySignature(t *testing.T) {
	msgBus := bus.NewMessageBus()
	cfg := config.WeComConfig{
		Token:      "test_token",
		WebhookURL: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test",
	}
	ch, _ := NewWeComBotChannel(cfg, msgBus)

	t.Run("valid signature", func(t *testing.T) {
		timestamp := "1234567890"
		nonce := "test_nonce"
		msgEncrypt := "test_message"
		expectedSig := generateSignature("test_token", timestamp, nonce, msgEncrypt)

		if !ch.verifySignature(expectedSig, timestamp, nonce, msgEncrypt) {
			t.Error("valid signature should pass verification")
		}
	})

	t.Run("invalid signature", func(t *testing.T) {
		timestamp := "1234567890"
		nonce := "test_nonce"
		msgEncrypt := "test_message"

		if ch.verifySignature("invalid_sig", timestamp, nonce, msgEncrypt) {
			t.Error("invalid signature should fail verification")
		}
	})

	t.Run("empty token skips verification", func(t *testing.T) {
		// Create a channel manually with empty token to test the behavior
		cfgEmpty := config.WeComConfig{
			Token:      "",
			WebhookURL: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test",
		}
		base := NewBaseChannel("wecom", cfgEmpty, msgBus, cfgEmpty.AllowFrom)
		chEmpty := &WeComBotChannel{
			BaseChannel: base,
			config:      cfgEmpty,
		}

		if !chEmpty.verifySignature("any_sig", "any_ts", "any_nonce", "any_msg") {
			t.Error("empty token should skip verification and return true")
		}
	})
}

func TestWeComBotDecryptMessage(t *testing.T) {
	msgBus := bus.NewMessageBus()

	t.Run("decrypt without AES key", func(t *testing.T) {
		cfg := config.WeComConfig{
			Token:          "test_token",
			WebhookURL:     "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test",
			EncodingAESKey: "",
		}
		ch, _ := NewWeComBotChannel(cfg, msgBus)

		// Without AES key, message should be base64 decoded only
		plainText := "hello world"
		encoded := base64.StdEncoding.EncodeToString([]byte(plainText))

		result, err := ch.decryptMessage(encoded)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != plainText {
			t.Errorf("decryptMessage() = %q, want %q", result, plainText)
		}
	})

	t.Run("decrypt with AES key", func(t *testing.T) {
		aesKey := generateTestAESKey()
		cfg := config.WeComConfig{
			Token:          "test_token",
			WebhookURL:     "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test",
			EncodingAESKey: aesKey,
		}
		ch, _ := NewWeComBotChannel(cfg, msgBus)

		originalMsg := "<xml><Content>Hello</Content></xml>"
		encrypted, err := encryptTestMessage(originalMsg, aesKey)
		if err != nil {
			t.Fatalf("failed to encrypt test message: %v", err)
		}

		result, err := ch.decryptMessage(encrypted)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != originalMsg {
			t.Errorf("decryptMessage() = %q, want %q", result, originalMsg)
		}
	})

	t.Run("invalid base64", func(t *testing.T) {
		cfg := config.WeComConfig{
			Token:          "test_token",
			WebhookURL:     "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test",
			EncodingAESKey: "",
		}
		ch, _ := NewWeComBotChannel(cfg, msgBus)

		_, err := ch.decryptMessage("invalid_base64!!!")
		if err == nil {
			t.Error("expected error for invalid base64, got nil")
		}
	})

	t.Run("invalid AES key", func(t *testing.T) {
		cfg := config.WeComConfig{
			Token:          "test_token",
			WebhookURL:     "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test",
			EncodingAESKey: "invalid_key",
		}
		ch, _ := NewWeComBotChannel(cfg, msgBus)

		_, err := ch.decryptMessage(base64.StdEncoding.EncodeToString([]byte("test")))
		if err == nil {
			t.Error("expected error for invalid AES key, got nil")
		}
	})
}

func TestWeComBotPKCS7Unpad(t *testing.T) {
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
			result, err := pkcs7UnpadWeCom(tt.input)
			if tt.expected == nil {
				// This case should return an error
				if err == nil {
					t.Errorf("pkcs7UnpadWeCom() expected error for invalid padding, got result: %v", result)
				}
				return
			}
			if err != nil {
				t.Errorf("pkcs7UnpadWeCom() unexpected error: %v", err)
				return
			}
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("pkcs7UnpadWeCom() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestWeComBotHandleVerification(t *testing.T) {
	msgBus := bus.NewMessageBus()
	aesKey := generateTestAESKey()
	cfg := config.WeComConfig{
		Token:          "test_token",
		EncodingAESKey: aesKey,
		WebhookURL:     "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test",
	}
	ch, _ := NewWeComBotChannel(cfg, msgBus)

	t.Run("valid verification request", func(t *testing.T) {
		echostr := "test_echostr_123"
		encryptedEchostr, _ := encryptTestMessage(echostr, aesKey)
		timestamp := "1234567890"
		nonce := "test_nonce"
		signature := generateSignature("test_token", timestamp, nonce, encryptedEchostr)

		req := httptest.NewRequest(http.MethodGet, "/webhook/wecom?msg_signature="+signature+"&timestamp="+timestamp+"&nonce="+nonce+"&echostr="+encryptedEchostr, nil)
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
		req := httptest.NewRequest(http.MethodGet, "/webhook/wecom?msg_signature=sig&timestamp=ts", nil)
		w := httptest.NewRecorder()

		ch.handleVerification(context.Background(), w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("invalid signature", func(t *testing.T) {
		echostr := "test_echostr"
		encryptedEchostr, _ := encryptTestMessage(echostr, aesKey)
		timestamp := "1234567890"
		nonce := "test_nonce"

		req := httptest.NewRequest(http.MethodGet, "/webhook/wecom?msg_signature=invalid_sig&timestamp="+timestamp+"&nonce="+nonce+"&echostr="+encryptedEchostr, nil)
		w := httptest.NewRecorder()

		ch.handleVerification(context.Background(), w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusForbidden)
		}
	})
}

func TestWeComBotHandleMessageCallback(t *testing.T) {
	msgBus := bus.NewMessageBus()
	aesKey := generateTestAESKey()
	cfg := config.WeComConfig{
		Token:          "test_token",
		EncodingAESKey: aesKey,
		WebhookURL:     "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test",
	}
	ch, _ := NewWeComBotChannel(cfg, msgBus)

	t.Run("valid message callback", func(t *testing.T) {
		// Create XML message
		xmlMsg := WeComBotXMLMessage{
			ToUserName:   "corp_id",
			FromUserName: "user123",
			CreateTime:   1234567890,
			MsgType:      "text",
			Content:      "Hello World",
			MsgId:        123456,
		}
		xmlData, _ := xml.Marshal(xmlMsg)

		// Encrypt message
		encrypted, _ := encryptTestMessage(string(xmlData), aesKey)

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
		signature := generateSignature("test_token", timestamp, nonce, encrypted)

		req := httptest.NewRequest(http.MethodPost, "/webhook/wecom?msg_signature="+signature+"&timestamp="+timestamp+"&nonce="+nonce, bytes.NewReader(wrapperData))
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
		req := httptest.NewRequest(http.MethodPost, "/webhook/wecom?msg_signature=sig", nil)
		w := httptest.NewRecorder()

		ch.handleMessageCallback(context.Background(), w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("invalid XML", func(t *testing.T) {
		timestamp := "1234567890"
		nonce := "test_nonce"
		signature := generateSignature("test_token", timestamp, nonce, "")

		req := httptest.NewRequest(http.MethodPost, "/webhook/wecom?msg_signature="+signature+"&timestamp="+timestamp+"&nonce="+nonce, strings.NewReader("invalid xml"))
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

		req := httptest.NewRequest(http.MethodPost, "/webhook/wecom?msg_signature=invalid_sig&timestamp="+timestamp+"&nonce="+nonce, bytes.NewReader(wrapperData))
		w := httptest.NewRecorder()

		ch.handleMessageCallback(context.Background(), w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusForbidden)
		}
	})
}

func TestWeComBotProcessMessage(t *testing.T) {
	msgBus := bus.NewMessageBus()
	cfg := config.WeComConfig{
		Token:      "test_token",
		WebhookURL: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test",
	}
	ch, _ := NewWeComBotChannel(cfg, msgBus)

	t.Run("process text message", func(t *testing.T) {
		msg := WeComBotXMLMessage{
			ToUserName:   "corp_id",
			FromUserName: "user123",
			CreateTime:   1234567890,
			MsgType:      "text",
			Content:      "Hello World",
			MsgId:        123456,
		}

		// Should not panic
		ch.processMessage(context.Background(), msg)
	})

	t.Run("process voice message with recognition", func(t *testing.T) {
		msg := WeComBotXMLMessage{
			ToUserName:   "corp_id",
			FromUserName: "user123",
			CreateTime:   1234567890,
			MsgType:      "voice",
			Recognition:  "Voice message text",
			MsgId:        123456,
		}

		// Should not panic
		ch.processMessage(context.Background(), msg)
	})

	t.Run("skip unsupported message type", func(t *testing.T) {
		msg := WeComBotXMLMessage{
			ToUserName:   "corp_id",
			FromUserName: "user123",
			CreateTime:   1234567890,
			MsgType:      "video",
			MsgId:        123456,
		}

		// Should not panic
		ch.processMessage(context.Background(), msg)
	})
}

func TestWeComBotHandleWebhook(t *testing.T) {
	msgBus := bus.NewMessageBus()
	cfg := config.WeComConfig{
		Token:      "test_token",
		WebhookURL: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test",
	}
	ch, _ := NewWeComBotChannel(cfg, msgBus)

	t.Run("GET request calls verification", func(t *testing.T) {
		echostr := "test_echostr"
		encoded := base64.StdEncoding.EncodeToString([]byte(echostr))
		timestamp := "1234567890"
		nonce := "test_nonce"
		signature := generateSignature("test_token", timestamp, nonce, encoded)

		req := httptest.NewRequest(http.MethodGet, "/webhook/wecom?msg_signature="+signature+"&timestamp="+timestamp+"&nonce="+nonce+"&echostr="+encoded, nil)
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
		signature := generateSignature("test_token", timestamp, nonce, encryptedWrapper.Encrypt)

		req := httptest.NewRequest(http.MethodPost, "/webhook/wecom?msg_signature="+signature+"&timestamp="+timestamp+"&nonce="+nonce, bytes.NewReader(wrapperData))
		w := httptest.NewRecorder()

		ch.handleWebhook(w, req)

		// Should not be method not allowed
		if w.Code == http.StatusMethodNotAllowed {
			t.Error("POST request should not return Method Not Allowed")
		}
	})

	t.Run("unsupported method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/webhook/wecom", nil)
		w := httptest.NewRecorder()

		ch.handleWebhook(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusMethodNotAllowed)
		}
	})
}

func TestWeComBotHandleHealth(t *testing.T) {
	msgBus := bus.NewMessageBus()
	cfg := config.WeComConfig{
		Token:      "test_token",
		WebhookURL: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test",
	}
	ch, _ := NewWeComBotChannel(cfg, msgBus)

	req := httptest.NewRequest(http.MethodGet, "/health/wecom", nil)
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
	if !strings.Contains(body, "status") || !strings.Contains(body, "running") {
		t.Errorf("response body should contain status and running fields, got: %s", body)
	}
}

func TestWeComBotWebhookReplyMessage(t *testing.T) {
	msg := WeComBotWebhookReply{
		MsgType: "text",
	}
	msg.Text.Content = "Hello World"

	if msg.MsgType != "text" {
		t.Errorf("MsgType = %q, want %q", msg.MsgType, "text")
	}
	if msg.Text.Content != "Hello World" {
		t.Errorf("Text.Content = %q, want %q", msg.Text.Content, "Hello World")
	}
}

func TestWeComBotXMLMessageStructure(t *testing.T) {
	xmlData := `<?xml version="1.0"?>
<xml>
	<ToUserName><![CDATA[corp_id]]></ToUserName>
	<FromUserName><![CDATA[user123]]></FromUserName>
	<CreateTime>1234567890</CreateTime>
	<MsgType><![CDATA[text]]></MsgType>
	<Content><![CDATA[Hello World]]></Content>
	<MsgId>1234567890123456</MsgId>
</xml>`

	var msg WeComBotXMLMessage
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
}
