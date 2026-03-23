package weixin

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	basechannels "github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestParseWeixinMediaAESKey(t *testing.T) {
	raw := []byte("1234567890abcdef")

	got, err := parseWeixinMediaAESKey(base64.StdEncoding.EncodeToString(raw))
	if err != nil {
		t.Fatalf("parseWeixinMediaAESKey(raw) error = %v", err)
	}
	if !bytes.Equal(got, raw) {
		t.Fatalf("parseWeixinMediaAESKey(raw) = %x, want %x", got, raw)
	}

	hexEncoded := base64.StdEncoding.EncodeToString([]byte("31323334353637383930616263646566"))
	got, err = parseWeixinMediaAESKey(hexEncoded)
	if err != nil {
		t.Fatalf("parseWeixinMediaAESKey(hex-string) error = %v", err)
	}
	if !bytes.Equal(got, raw) {
		t.Fatalf("parseWeixinMediaAESKey(hex-string) = %x, want %x", got, raw)
	}
}

func TestDownloadAndDecryptCDNBuffer(t *testing.T) {
	key := []byte("1234567890abcdef")
	plaintext := []byte("hello weixin")
	ciphertext, err := encryptAESECB(plaintext, key)
	if err != nil {
		t.Fatalf("encryptAESECB() error = %v", err)
	}

	ch := &WeixinChannel{
		api: &ApiClient{
			HttpClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path != "/download" {
					t.Fatalf("download path = %q, want /download", r.URL.Path)
				}
				if r.URL.Query().Get("encrypted_query_param") != "token" {
					t.Fatalf("encrypted_query_param = %q, want token", r.URL.Query().Get("encrypted_query_param"))
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(ciphertext)),
					Header:     make(http.Header),
				}, nil
			})},
		},
		config: config.WeixinConfig{
			CDNBaseURL: "https://cdn.example.com",
		},
		typingCache: make(map[string]typingTicketCacheEntry),
	}

	got, err := ch.downloadAndDecryptCDNBuffer(context.Background(), "token", key)
	if err != nil {
		t.Fatalf("downloadAndDecryptCDNBuffer() error = %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("downloadAndDecryptCDNBuffer() = %q, want %q", got, plaintext)
	}
}

func TestUploadBufferToCDN(t *testing.T) {
	key := []byte("1234567890abcdef")
	plaintext := []byte("upload me")
	wantCipher, err := encryptAESECB(plaintext, key)
	if err != nil {
		t.Fatalf("encryptAESECB() error = %v", err)
	}

	ch := &WeixinChannel{
		api: &ApiClient{
			HttpClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path != "/upload" {
					t.Fatalf("upload path = %q, want /upload", r.URL.Path)
				}
				if got := r.URL.Query().Get("encrypted_query_param"); got != "upload-param" {
					t.Fatalf("encrypted_query_param = %q, want upload-param", got)
				}
				if got := r.URL.Query().Get("filekey"); got != "file-key" {
					t.Fatalf("filekey = %q, want file-key", got)
				}
				body, _ := io.ReadAll(r.Body)
				if !bytes.Equal(body, wantCipher) {
					t.Fatalf("upload body = %x, want %x", body, wantCipher)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(nil)),
					Header: http.Header{
						"X-Encrypted-Param": []string{"download-param"},
					},
				}, nil
			})},
		},
		config: config.WeixinConfig{
			CDNBaseURL: "https://cdn.example.com",
		},
		typingCache: make(map[string]typingTicketCacheEntry),
	}

	got, err := ch.uploadBufferToCDN(context.Background(), plaintext, "upload-param", "file-key", key)
	if err != nil {
		t.Fatalf("uploadBufferToCDN() error = %v", err)
	}
	if got != "download-param" {
		t.Fatalf("uploadBufferToCDN() = %q, want download-param", got)
	}
}

func TestLoadSaveGetUpdatesBuf(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sync.json")

	if err := saveGetUpdatesBuf(path, "cursor-123"); err != nil {
		t.Fatalf("saveGetUpdatesBuf() error = %v", err)
	}

	got, err := loadGetUpdatesBuf(path)
	if err != nil {
		t.Fatalf("loadGetUpdatesBuf() error = %v", err)
	}
	if got != "cursor-123" {
		t.Fatalf("loadGetUpdatesBuf() = %q, want cursor-123", got)
	}
}

func TestBuildWeixinSyncBufPathUsesPicoclawHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv(config.EnvHome, home)

	wxCfg := config.WeixinConfig{
		BaseURL: "https://ilinkai.weixin.qq.com/",
	}
	wxCfg.SetToken("token-123")
	got := buildWeixinSyncBufPath(wxCfg)
	if filepath.Dir(got) != filepath.Join(home, "channels", "weixin", "sync") {
		t.Fatalf("sync path dir = %q", filepath.Dir(got))
	}
}

func TestSessionPauseGuard(t *testing.T) {
	ch := &WeixinChannel{
		typingCache: make(map[string]typingTicketCacheEntry),
	}

	ch.pauseSession("getupdates", 0, weixinSessionExpiredCode, "expired")

	if err := ch.ensureSessionActive(); !errors.Is(err, basechannels.ErrSendFailed) {
		t.Fatalf("ensureSessionActive() error = %v, want ErrSendFailed", err)
	}

	ch.pauseMu.Lock()
	ch.pauseUntil = time.Now().Add(-time.Second)
	ch.pauseMu.Unlock()

	if err := ch.ensureSessionActive(); err != nil {
		t.Fatalf("ensureSessionActive() after expiry error = %v, want nil", err)
	}
}

func TestSelectInboundMediaItemFallsBackToRefMessage(t *testing.T) {
	msg := WeixinMessage{
		ItemList: []MessageItem{
			{
				Type: MessageItemTypeText,
				TextItem: &TextItem{
					Text: "look",
				},
				RefMsg: &RefMessage{
					MessageItem: &MessageItem{
						Type: MessageItemTypeImage,
						ImageItem: &ImageItem{
							Media: &CDNMedia{
								EncryptQueryParam: "abc",
							},
						},
					},
				},
			},
		},
	}

	item := selectInboundMediaItem(msg)
	if item == nil {
		t.Fatal("selectInboundMediaItem() = nil, want ref media item")
	}
	if item.Type != MessageItemTypeImage {
		t.Fatalf("selectInboundMediaItem().Type = %d, want %d", item.Type, MessageItemTypeImage)
	}
}
