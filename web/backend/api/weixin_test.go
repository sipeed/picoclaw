package api

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestSaveWeixinBindingReturnsSuccessWhenRestartFails(t *testing.T) {
	resetGatewayTestState(t)

	configPath := filepath.Join(t.TempDir(), "config.json")
	cfg := config.DefaultConfig()
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	originalHealthGet := gatewayHealthGet
	gatewayHealthGet = func(url string, timeout time.Duration) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(
				`{"status":"ok","uptime":"1s","pid":` + strconv.Itoa(os.Getpid()) + `}`,
			)),
		}, nil
	}
	t.Cleanup(func() {
		gatewayHealthGet = originalHealthGet
	})

	h := NewHandler(configPath)
	if err := h.saveWeixinBinding("bot-token", "bot-account"); err != nil {
		t.Fatalf("saveWeixinBinding() error = %v, want nil after config save succeeds", err)
	}

	savedCfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if got := savedCfg.Channels.Weixin.Token.String(); got != "bot-token" {
		t.Fatalf("Weixin.Token() = %q, want %q", got, "bot-token")
	}
	if got := savedCfg.Channels.Weixin.AccountID; got != "bot-account" {
		t.Fatalf("Weixin.AccountID = %q, want %q", got, "bot-account")
	}
	if !savedCfg.Channels.Weixin.Enabled {
		t.Fatalf("Weixin.Enabled = false, want true")
	}
}
