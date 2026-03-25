package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/mdp/qrterminal/v3"
	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/config"
)

const (
	wecomQRSourceID          = "picoclaw"
	wecomQRGenerateEndpoint  = "https://work.weixin.qq.com/ai/qc/generate"
	wecomQRQueryEndpoint     = "https://work.weixin.qq.com/ai/qc/query_result"
	wecomQRPageEndpoint      = "https://work.weixin.qq.com/ai/qc/gen"
	wecomQRHTTPTimeout       = 15 * time.Second
	wecomQRPollInterval      = 3 * time.Second
	wecomQRPollTimeout       = 5 * time.Minute
	wecomDefaultWebSocketURL = "wss://openws.work.weixin.qq.com"
)

type wecomQRScanner func(context.Context, wecomQRFlowOptions) (wecomQRBotInfo, error)

type wecomQRFlowOptions struct {
	HTTPClient    *http.Client
	GenerateURL   string
	QueryURL      string
	QRCodePageURL string
	SourceID      string
	PollInterval  time.Duration
	PollTimeout   time.Duration
	Writer        io.Writer
}

type wecomQRBotInfo struct {
	BotID  string
	Secret string
}

type wecomQRSession struct {
	SCode   string
	AuthURL string
}

type wecomQRGenerateResponse struct {
	ErrCode int    `json:"errcode,omitempty"`
	ErrMsg  string `json:"errmsg,omitempty"`
	Data    struct {
		SCode   string `json:"scode"`
		AuthURL string `json:"auth_url"`
	} `json:"data"`
}

type wecomQRQueryResponse struct {
	ErrCode int    `json:"errcode,omitempty"`
	ErrMsg  string `json:"errmsg,omitempty"`
	Data    struct {
		Status  string `json:"status"`
		BotInfo struct {
			BotID  string `json:"botid"`
			Secret string `json:"secret"`
		} `json:"bot_info"`
	} `json:"data"`
}

func newWeComCommand() *cobra.Command {
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "wecom",
		Short: "Scan a WeCom QR code and configure channels.wecom",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return authWeComCmd(timeout)
		},
	}

	cmd.Flags().DurationVar(&timeout, "timeout", wecomQRPollTimeout, "How long to wait for QR confirmation")

	return cmd
}

func authWeComCmd(timeout time.Duration) error {
	return authWeComCmdWithScanner(context.Background(), os.Stdout, timeout, scanWeComQRCodeInteractive)
}

func authWeComCmdWithScanner(
	ctx context.Context,
	writer io.Writer,
	timeout time.Duration,
	scanner wecomQRScanner,
) error {
	if scanner == nil {
		return fmt.Errorf("wecom QR scanner is nil")
	}
	if writer == nil {
		writer = os.Stdout
	}

	cfg, err := internal.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	opts := defaultWeComQRFlowOptions(timeout)
	opts.Writer = writer

	botInfo, err := scanner(ctx, opts)
	if err != nil {
		return err
	}

	applyWeComAuthResult(cfg, botInfo)

	if saveErr := config.SaveConfig(internal.GetConfigPath(), cfg); saveErr != nil {
		return fmt.Errorf("failed to save config: %w", saveErr)
	}

	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "WeCom connected.")
	fmt.Fprintf(writer, "Bot ID: %s\n", botInfo.BotID)
	fmt.Fprintf(writer, "Config: %s\n", internal.GetConfigPath())

	return nil
}

func defaultWeComQRFlowOptions(timeout time.Duration) wecomQRFlowOptions {
	if timeout <= 0 {
		timeout = wecomQRPollTimeout
	}

	return wecomQRFlowOptions{
		HTTPClient:    &http.Client{Timeout: wecomQRHTTPTimeout},
		GenerateURL:   wecomQRGenerateEndpoint,
		QueryURL:      wecomQRQueryEndpoint,
		QRCodePageURL: wecomQRPageEndpoint,
		SourceID:      wecomQRSourceID,
		PollInterval:  wecomQRPollInterval,
		PollTimeout:   timeout,
		Writer:        os.Stdout,
	}
}

func applyWeComAuthResult(cfg *config.Config, botInfo wecomQRBotInfo) {
	cfg.Channels.WeCom.Enabled = true
	cfg.Channels.WeCom.BotID = botInfo.BotID
	cfg.Channels.WeCom.SetSecret(botInfo.Secret)
	if strings.TrimSpace(cfg.Channels.WeCom.WebSocketURL) == "" {
		cfg.Channels.WeCom.WebSocketURL = wecomDefaultWebSocketURL
	}
}

func scanWeComQRCodeInteractive(ctx context.Context, opts wecomQRFlowOptions) (wecomQRBotInfo, error) {
	opts = normalizeWeComQRFlowOptions(opts)

	fmt.Fprintln(opts.Writer, "Requesting WeCom QR code...")

	session, err := fetchWeComQRCode(ctx, opts)
	if err != nil {
		return wecomQRBotInfo{}, err
	}

	fmt.Fprintln(opts.Writer)
	fmt.Fprintln(opts.Writer, "=======================================================")
	fmt.Fprintln(opts.Writer, "Please scan the following QR code with WeCom:")
	fmt.Fprintln(opts.Writer, "=======================================================")
	fmt.Fprintln(opts.Writer)

	qrterminal.GenerateWithConfig(session.AuthURL, qrterminal.Config{
		Level:      qrterminal.L,
		Writer:     opts.Writer,
		HalfBlocks: true,
	})

	pageURL, err := buildWeComQRCodePageURL(opts.QRCodePageURL, opts.SourceID, session.SCode)
	if err != nil {
		return wecomQRBotInfo{}, err
	}

	fmt.Fprintln(opts.Writer)
	fmt.Fprintf(opts.Writer, "QR Code Link: %s\n", pageURL)
	fmt.Fprintln(opts.Writer)
	fmt.Fprintln(opts.Writer, "Waiting for scan...")

	return pollWeComQRCodeResult(ctx, opts, session.SCode)
}

func normalizeWeComQRFlowOptions(opts wecomQRFlowOptions) wecomQRFlowOptions {
	if opts.HTTPClient == nil {
		opts.HTTPClient = &http.Client{Timeout: wecomQRHTTPTimeout}
	}
	if strings.TrimSpace(opts.GenerateURL) == "" {
		opts.GenerateURL = wecomQRGenerateEndpoint
	}
	if strings.TrimSpace(opts.QueryURL) == "" {
		opts.QueryURL = wecomQRQueryEndpoint
	}
	if strings.TrimSpace(opts.QRCodePageURL) == "" {
		opts.QRCodePageURL = wecomQRPageEndpoint
	}
	if strings.TrimSpace(opts.SourceID) == "" {
		opts.SourceID = wecomQRSourceID
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = wecomQRPollInterval
	}
	if opts.PollTimeout <= 0 {
		opts.PollTimeout = wecomQRPollTimeout
	}
	if opts.Writer == nil {
		opts.Writer = os.Stdout
	}

	return opts
}

func fetchWeComQRCode(ctx context.Context, opts wecomQRFlowOptions) (wecomQRSession, error) {
	generateURL, err := buildWeComQRGenerateURL(opts.GenerateURL, opts.SourceID, wecomPlatformCode())
	if err != nil {
		return wecomQRSession{}, err
	}

	var resp wecomQRGenerateResponse
	if err := doWeComJSONGet(ctx, opts.HTTPClient, generateURL, &resp); err != nil {
		return wecomQRSession{}, fmt.Errorf("failed to get WeCom QR code: %w", err)
	}
	if resp.ErrCode != 0 {
		return wecomQRSession{}, fmt.Errorf(
			"failed to get WeCom QR code: errcode=%d errmsg=%s",
			resp.ErrCode,
			resp.ErrMsg,
		)
	}
	if resp.Data.SCode == "" || resp.Data.AuthURL == "" {
		return wecomQRSession{}, fmt.Errorf("failed to get WeCom QR code: response missing scode or auth_url")
	}

	return wecomQRSession{
		SCode:   resp.Data.SCode,
		AuthURL: resp.Data.AuthURL,
	}, nil
}

func pollWeComQRCodeResult(ctx context.Context, opts wecomQRFlowOptions, scode string) (wecomQRBotInfo, error) {
	if strings.TrimSpace(scode) == "" {
		return wecomQRBotInfo{}, fmt.Errorf("missing WeCom QR scode")
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, opts.PollTimeout)
	defer cancel()

	var scannedPrinted bool

	for {
		status, err := queryWeComQRCodeStatus(timeoutCtx, opts, scode)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(timeoutCtx.Err(), context.DeadlineExceeded) {
				return wecomQRBotInfo{}, fmt.Errorf("WeCom QR scan timed out after %s", opts.PollTimeout)
			}
			return wecomQRBotInfo{}, err
		}

		switch strings.ToLower(status.Data.Status) {
		case "success":
			if status.Data.BotInfo.BotID == "" || status.Data.BotInfo.Secret == "" {
				return wecomQRBotInfo{}, fmt.Errorf("WeCom QR scan succeeded but bot credentials are missing")
			}
			return wecomQRBotInfo{
				BotID:  status.Data.BotInfo.BotID,
				Secret: status.Data.BotInfo.Secret,
			}, nil
		case "expired":
			return wecomQRBotInfo{}, fmt.Errorf("WeCom QR code expired, please retry")
		case "scaned", "scanned":
			if !scannedPrinted {
				fmt.Fprintln(opts.Writer, "QR code scanned. Confirm the login in WeCom.")
				scannedPrinted = true
			}
		}

		select {
		case <-timeoutCtx.Done():
			if errors.Is(timeoutCtx.Err(), context.DeadlineExceeded) {
				return wecomQRBotInfo{}, fmt.Errorf("WeCom QR scan timed out after %s", opts.PollTimeout)
			}
			return wecomQRBotInfo{}, timeoutCtx.Err()
		case <-time.After(opts.PollInterval):
		}
	}
}

func queryWeComQRCodeStatus(ctx context.Context, opts wecomQRFlowOptions, scode string) (wecomQRQueryResponse, error) {
	queryURL, err := buildWeComQRQueryURL(opts.QueryURL, scode)
	if err != nil {
		return wecomQRQueryResponse{}, err
	}

	var resp wecomQRQueryResponse
	if err := doWeComJSONGet(ctx, opts.HTTPClient, queryURL, &resp); err != nil {
		return wecomQRQueryResponse{}, fmt.Errorf("failed to query WeCom QR result: %w", err)
	}
	if resp.ErrCode != 0 {
		return wecomQRQueryResponse{}, fmt.Errorf(
			"failed to query WeCom QR result: errcode=%d errmsg=%s",
			resp.ErrCode,
			resp.ErrMsg,
		)
	}

	return resp, nil
}

func buildWeComQRGenerateURL(baseURL, sourceID string, platformCode int) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid WeCom QR generate URL: %w", err)
	}

	query := u.Query()
	query.Set("source", sourceID)
	query.Set("sourceID", sourceID)
	query.Set("plat", strconv.Itoa(platformCode))
	u.RawQuery = query.Encode()

	return u.String(), nil
}

func buildWeComQRQueryURL(baseURL, scode string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid WeCom QR query URL: %w", err)
	}

	query := u.Query()
	query.Set("scode", scode)
	u.RawQuery = query.Encode()

	return u.String(), nil
}

func buildWeComQRCodePageURL(baseURL, sourceID, scode string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid WeCom QR page URL: %w", err)
	}

	query := u.Query()
	query.Set("source", sourceID)
	query.Set("sourceID", sourceID)
	query.Set("scode", scode)
	u.RawQuery = query.Encode()

	return u.String(), nil
}

func doWeComJSONGet(ctx context.Context, client *http.Client, targetURL string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 8192))
		if readErr != nil {
			return fmt.Errorf("unexpected status %s", resp.Status)
		}
		return fmt.Errorf("unexpected status %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode JSON response: %w", err)
	}

	return nil
}

func wecomPlatformCode() int {
	switch runtime.GOOS {
	case "darwin":
		return 1
	case "windows":
		return 2
	case "linux":
		return 3
	default:
		return 0
	}
}
