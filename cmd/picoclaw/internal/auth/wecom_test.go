package auth

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/config"
)

func newIPv4TestServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()

	server := httptest.NewUnstartedServer(handler)
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)

	server.Listener = listener
	server.Start()
	t.Cleanup(server.Close)
	return server
}

func TestNewWeComCommand(t *testing.T) {
	cmd := newWeComCommand()

	require.NotNil(t, cmd)
	assert.Equal(t, "wecom", cmd.Use)
	assert.Equal(t, "Scan a WeCom QR code and configure channels.wecom", cmd.Short)
	assert.NotNil(t, cmd.Flags().Lookup("timeout"))
}

func TestBuildWeComQRGenerateURL(t *testing.T) {
	rawURL, err := buildWeComQRGenerateURL("https://example.com/ai/qc/generate", wecomQRSourceID, 3)
	require.NoError(t, err)

	parsed, err := url.Parse(rawURL)
	require.NoError(t, err)

	assert.Equal(t, wecomQRSourceID, parsed.Query().Get("source"))
	assert.Equal(t, wecomQRSourceID, parsed.Query().Get("sourceID"))
	assert.Equal(t, "3", parsed.Query().Get("plat"))
}

func TestBuildWeComQRCodePageURL(t *testing.T) {
	rawURL, err := buildWeComQRCodePageURL("https://example.com/ai/qc/gen", wecomQRSourceID, "scode-1")
	require.NoError(t, err)

	parsed, err := url.Parse(rawURL)
	require.NoError(t, err)

	assert.Equal(t, wecomQRSourceID, parsed.Query().Get("source"))
	assert.Equal(t, wecomQRSourceID, parsed.Query().Get("sourceID"))
	assert.Equal(t, "scode-1", parsed.Query().Get("scode"))
}

func TestFetchWeComQRCode(t *testing.T) {
	server := newIPv4TestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/generate", r.URL.Path)
		assert.Equal(t, wecomQRSourceID, r.URL.Query().Get("source"))
		assert.Equal(t, wecomQRSourceID, r.URL.Query().Get("sourceID"))
		assert.Equal(t, strconv.Itoa(wecomPlatformCode()), r.URL.Query().Get("plat"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"scode":"scode-1","auth_url":"https://example.com/qr"}}`))
	}))

	opts := normalizeWeComQRFlowOptions(wecomQRFlowOptions{
		HTTPClient:  server.Client(),
		GenerateURL: server.URL + "/generate",
		Writer:      bytes.NewBuffer(nil),
	})

	session, err := fetchWeComQRCode(context.Background(), opts)
	require.NoError(t, err)
	assert.Equal(t, "scode-1", session.SCode)
	assert.Equal(t, "https://example.com/qr", session.AuthURL)
}

func TestPollWeComQRCodeResult(t *testing.T) {
	var calls atomic.Int32

	server := newIPv4TestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := calls.Add(1)
		assert.Equal(t, "/query", r.URL.Path)
		assert.Equal(t, "scode-1", r.URL.Query().Get("scode"))
		w.Header().Set("Content-Type", "application/json")
		switch call {
		case 1:
			_, _ = w.Write([]byte(`{"data":{"status":"wait"}}`))
		case 2:
			_, _ = w.Write([]byte(`{"data":{"status":"scaned"}}`))
		default:
			_, _ = w.Write([]byte(`{"data":{"status":"success","bot_info":{"botid":"bot-1","secret":"secret-1"}}}`))
		}
	}))

	var output bytes.Buffer
	opts := normalizeWeComQRFlowOptions(wecomQRFlowOptions{
		HTTPClient:   server.Client(),
		QueryURL:     server.URL + "/query",
		PollInterval: time.Millisecond,
		PollTimeout:  time.Second,
		Writer:       &output,
	})

	botInfo, err := pollWeComQRCodeResult(context.Background(), opts, "scode-1")
	require.NoError(t, err)
	assert.Equal(t, "bot-1", botInfo.BotID)
	assert.Equal(t, "secret-1", botInfo.Secret)
	assert.Contains(t, output.String(), "QR code scanned. Confirm the login in WeCom.")
}

func TestApplyWeComAuthResult(t *testing.T) {
	cfg := config.DefaultConfig()
	require.NoError(t, config.InitChannelList(cfg.Channels))
	wecom := cfg.Channels["wecom"]
	t.Logf("wecom: %+v", wecom)
	decoded, err := wecom.GetDecoded()
	require.NoError(t, err)
	weCfg := decoded.(*config.WeComSettings)
	weCfg.WebSocketURL = ""

	applyWeComAuthResult(cfg, wecomQRBotInfo{
		BotID:  "bot-1",
		Secret: "secret-1",
	})

	assert.True(t, wecom.Enabled)
	assert.Equal(t, "bot-1", weCfg.BotID)
	assert.Equal(t, "secret-1", weCfg.Secret.String())
	assert.Equal(t, wecomDefaultWebSocketURL, weCfg.WebSocketURL)
}

func TestAuthWeComCmdWithScanner(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	t.Setenv(config.EnvHome, tmpDir)
	t.Setenv(config.EnvConfig, configPath)

	var output bytes.Buffer
	err := authWeComCmdWithScanner(
		context.Background(),
		&output,
		time.Second,
		func(_ context.Context, opts wecomQRFlowOptions) (wecomQRBotInfo, error) {
			assert.Equal(t, wecomQRSourceID, opts.SourceID)
			return wecomQRBotInfo{
				BotID:  "bot-1",
				Secret: "secret-1",
			}, nil
		},
	)
	require.NoError(t, err)

	cfg, err := config.LoadConfig(internal.GetConfigPath())
	require.NoError(t, err)
	wecom := cfg.Channels["wecom"]
	decoded, err := wecom.GetDecoded()
	require.NoError(t, err)
	weCfg := decoded.(*config.WeComSettings)
	assert.True(t, wecom.Enabled)
	assert.Equal(t, "bot-1", weCfg.BotID)
	assert.Equal(t, "secret-1", weCfg.Secret.String())
	assert.Equal(t, wecomDefaultWebSocketURL, weCfg.WebSocketURL)
	assert.Contains(t, output.String(), "WeCom connected.")
}
