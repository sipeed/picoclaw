package weixin

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	basechannels "github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/media"
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
		config: &config.WeixinSettings{
			CDNBaseURL: "https://cdn.example.com",
		},
		typingCache: make(map[string]typingTicketCacheEntry),
	}

	got, err := ch.downloadAndDecryptCDNBuffer(context.Background(), "token", "", key)
	if err != nil {
		t.Fatalf("downloadAndDecryptCDNBuffer() error = %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("downloadAndDecryptCDNBuffer() = %q, want %q", got, plaintext)
	}
}

func TestDownloadAndDecryptCDNBufferUsesFullURLWhenProvided(t *testing.T) {
	key := []byte("1234567890abcdef")
	plaintext := []byte("hello weixin")
	ciphertext, err := encryptAESECB(plaintext, key)
	if err != nil {
		t.Fatalf("encryptAESECB() error = %v", err)
	}

	fullURLAttempts := 0
	ch := &WeixinChannel{
		api: &ApiClient{
			HttpClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.String() == "https://full.example.com/download" {
					fullURLAttempts++
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader(ciphertext)),
						Header:     make(http.Header),
					}, nil
				}
				t.Fatalf("unexpected fallback request: %s", r.URL.String())
				return nil, nil
			})},
		},
		config: &config.WeixinSettings{
			CDNBaseURL: "https://cdn.example.com",
		},
		typingCache: make(map[string]typingTicketCacheEntry),
	}

	got, err := ch.downloadAndDecryptCDNBuffer(context.Background(), "token", "https://full.example.com/download", key)
	if err != nil {
		t.Fatalf("downloadAndDecryptCDNBuffer() error = %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("downloadAndDecryptCDNBuffer() = %q, want %q", got, plaintext)
	}
	if fullURLAttempts == 0 {
		t.Fatalf("fullURLAttempts = %d, want > 0", fullURLAttempts)
	}
}

func TestDownloadAndDecryptCDNBufferFallsBackToConstructedURLWhenFullURLFails(t *testing.T) {
	key := []byte("1234567890abcdef")
	plaintext := []byte("hello weixin")
	ciphertext, err := encryptAESECB(plaintext, key)
	if err != nil {
		t.Fatalf("encryptAESECB() error = %v", err)
	}

	fullURLAttempts := 0
	constructedAttempts := 0
	ch := &WeixinChannel{
		api: &ApiClient{
			HttpClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.String() == "https://full.example.com/download?encrypted_query_param=token&taskid=123" {
					fullURLAttempts++
					return &http.Response{
						StatusCode: http.StatusInternalServerError,
						Body:       io.NopCloser(bytes.NewReader(nil)),
						Header:     make(http.Header),
					}, nil
				}
				if r.URL.String() != "https://cdn.example.com/download?encrypted_query_param=token" {
					t.Fatalf("unexpected fallback request: %s", r.URL.String())
				}
				constructedAttempts++
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(ciphertext)),
					Header:     make(http.Header),
				}, nil
			})},
		},
		config: &config.WeixinSettings{
			CDNBaseURL: "https://cdn.example.com",
		},
		typingCache: make(map[string]typingTicketCacheEntry),
	}

	got, err := ch.downloadAndDecryptCDNBuffer(
		context.Background(),
		"token",
		"https://full.example.com/download?encrypted_query_param=token&taskid=123",
		key,
	)
	if err != nil {
		t.Fatalf("downloadAndDecryptCDNBuffer() error = %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("downloadAndDecryptCDNBuffer() = %q, want %q", got, plaintext)
	}
	if fullURLAttempts == 0 {
		t.Fatalf("fullURLAttempts = %d, want > 0", fullURLAttempts)
	}
	if constructedAttempts == 0 {
		t.Fatalf("constructedAttempts = %d, want > 0", constructedAttempts)
	}
}

func TestBuildCDNDownloadURLEscapesOpaqueToken(t *testing.T) {
	token := "MFcCAQAESzBJAgEAAgSieMV9AgM9CcwCBEoKPqICBGnHZB0EJDk4OWY5YWU0LTc4OGItNGQ5Ni1iMjZhLWU4YjhlMmEwOWVkZgIEIR0IAgIBAAQFAExUPQA%3D"

	got := buildCDNDownloadURL("https://cdn.example.com", token)

	if got != "https://cdn.example.com/download?encrypted_query_param=MFcCAQAESzBJAgEAAgSieMV9AgM9CcwCBEoKPqICBGnHZB0EJDk4OWY5YWU0LTc4OGItNGQ5Ni1iMjZhLWU4YjhlMmEwOWVkZgIEIR0IAgIBAAQFAExUPQA%253D" {
		t.Fatalf("buildCDNDownloadURL() = %q", got)
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
		config: &config.WeixinSettings{
			CDNBaseURL: "https://cdn.example.com",
		},
		typingCache: make(map[string]typingTicketCacheEntry),
	}

	got, err := ch.uploadBufferToCDN(context.Background(), plaintext, "upload-param", "", "file-key", key)
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

	wxCfg := &config.WeixinSettings{
		BaseURL: "https://ilinkai.weixin.qq.com/",
	}
	wxCfg.SetToken("token-123")
	got := buildWeixinSyncBufPath(wxCfg)
	if filepath.Dir(got) != filepath.Join(home, "channels", "weixin", "sync") {
		t.Fatalf("sync path dir = %q", filepath.Dir(got))
	}
}

func TestBuildWeixinSyncBufPathUsesAccountIDWhenAvailable(t *testing.T) {
	home := t.TempDir()
	t.Setenv(config.EnvHome, home)

	wxCfg := &config.WeixinSettings{
		AccountID: "wx.account/one",
		BaseURL:   "https://ilinkai.weixin.qq.com/",
	}
	wxCfg.SetToken("token-123")

	got := buildWeixinSyncBufPath(wxCfg)
	if filepath.Base(got) != "wx.account_one.json" {
		t.Fatalf("sync path base = %q, want wx.account_one.json", filepath.Base(got))
	}
}

func TestHandleInboundMessageUsesChannelInstanceAndAccount(t *testing.T) {
	home := t.TempDir()
	t.Setenv(config.EnvHome, home)

	msgBus := bus.NewMessageBus()
	wxCfg := &config.WeixinSettings{
		AccountID: "account-a",
		BaseURL:   "https://ilinkai.weixin.qq.com/",
	}
	wxCfg.SetToken("token-a")
	bc := &config.Channel{Enabled: true, Type: config.ChannelWeixin}
	bc.SetName("weixin_a")
	ch, err := NewWeixinChannel(bc, wxCfg, msgBus)
	if err != nil {
		t.Fatalf("NewWeixinChannel() error = %v", err)
	}
	ch.SetName("weixin_a")

	ch.handleInboundMessage(context.Background(), WeixinMessage{
		FromUserID:   "user-1",
		ClientID:     "msg-1",
		ContextToken: "ctx-1",
		SessionID:    "session-1",
		ItemList: []MessageItem{
			{
				Type:     MessageItemTypeText,
				TextItem: &TextItem{Text: "hello"},
			},
		},
	})

	select {
	case got := <-msgBus.InboundChan():
		if got.Context.Channel != "weixin_a" {
			t.Fatalf("Context.Channel = %q, want weixin_a", got.Context.Channel)
		}
		if got.Context.Account != "account-a" {
			t.Fatalf("Context.Account = %q, want account-a", got.Context.Account)
		}
		if got.Context.ChatID != "user-1" {
			t.Fatalf("Context.ChatID = %q, want user-1", got.Context.ChatID)
		}
		if got.Context.ChatType != "direct" {
			t.Fatalf("Context.ChatType = %q, want direct", got.Context.ChatType)
		}
		if got.Content != "hello" {
			t.Fatalf("Content = %q, want hello", got.Content)
		}
		if got.Context.Raw["account_id"] != "account-a" {
			t.Fatalf("raw account_id = %q, want account-a", got.Context.Raw["account_id"])
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for inbound message")
	}
}

type recordingMediaStore struct {
	scope string
}

func (s *recordingMediaStore) Store(_ string, _ media.MediaMeta, scope string) (string, error) {
	s.scope = scope
	return "media://ref", nil
}

func (s *recordingMediaStore) Resolve(ref string) (string, error) {
	return "", nil
}

func (s *recordingMediaStore) ResolveWithMeta(ref string) (string, media.MediaMeta, error) {
	return "", media.MediaMeta{}, nil
}

func (s *recordingMediaStore) ReleaseAll(scope string) error {
	return nil
}

func TestStoreInboundBytesUsesChannelInstanceInScope(t *testing.T) {
	msgBus := bus.NewMessageBus()
	wxCfg := &config.WeixinSettings{BaseURL: "https://ilinkai.weixin.qq.com/"}
	wxCfg.SetToken("token-a")
	bc := &config.Channel{Enabled: true, Type: config.ChannelWeixin}
	bc.SetName("weixin_a")
	ch, err := NewWeixinChannel(bc, wxCfg, msgBus)
	if err != nil {
		t.Fatalf("NewWeixinChannel() error = %v", err)
	}
	ch.SetName("weixin_a")
	store := &recordingMediaStore{}
	ch.SetMediaStore(store)

	if _, err := ch.storeInboundBytes("user-1", "msg-1", "file.txt", "text/plain", []byte("hello")); err != nil {
		t.Fatalf("storeInboundBytes() error = %v", err)
	}
	if !strings.HasPrefix(store.scope, "weixin_a:user-1:msg-1") {
		t.Fatalf("media scope = %q, want weixin_a:user-1:msg-1 prefix", store.scope)
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
