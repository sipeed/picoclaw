package weixin

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/mdp/qrterminal/v3"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// AuthFlowOpts configures the interactive QR login flow.
type AuthFlowOpts struct {
	BaseURL string
	BotType string
	Timeout time.Duration
	Proxy   string
}

// PerformLoginInteractive starts the Weixin QR login flow and blocks until login is successful or times out.
// It prints a QR code to the terminal for the user to scan.
// Returns the BotToken, UserID, AccountID, and BaseUrl on success.
func PerformLoginInteractive(
	ctx context.Context,
	opts AuthFlowOpts,
) (botToken, userID, accountID, baseUrl string, err error) {
	if opts.BaseURL == "" {
		opts.BaseURL = "https://ilinkai.weixin.qq.com/"
	}
	if opts.BotType == "" {
		opts.BotType = "3" // Default iLink Bot Type
	}
	if opts.Timeout == 0 {
		opts.Timeout = 5 * time.Minute
	}

	api, err := NewApiClient(opts.BaseURL, "", opts.Proxy)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to create api client: %w", err)
	}
	pollAPI := api

	logger.InfoC("weixin", "Requesting Weixin QR code...")
	qrResp, err := api.GetQRCode(ctx, opts.BotType)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to get qrcode: %w", err)
	}

	fmt.Println("\n=======================================================")
	fmt.Println("Please scan the following QR code with WeChat to login:")
	fmt.Println("=======================================================")
	fmt.Println()

	// Create Small QR
	qrconfig := qrterminal.Config{
		Level:      qrterminal.L,
		Writer:     os.Stdout,
		HalfBlocks: true,
	}
	qrterminal.GenerateWithConfig(qrResp.QrcodeImgContent, qrconfig)

	fmt.Printf("\nQR Code Link: %s\n\n", qrResp.QrcodeImgContent)
	fmt.Println("Waiting for scan...")

	timeoutCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	pollTicker := time.NewTicker(2 * time.Second)
	defer pollTicker.Stop()

	scannedPrinted := false

	for {
		select {
		case <-timeoutCtx.Done():
			return "", "", "", "", fmt.Errorf("login timeout")
		case <-pollTicker.C:
			statusResp, err := pollAPI.GetQRCodeStatus(timeoutCtx, qrResp.Qrcode)
			if err != nil {
				// Long poll timeout or temporary error
				continue
			}

			switch statusResp.Status {
			case "wait":
				// still waiting
			case "scaned":
				if !scannedPrinted {
					fmt.Println("👀 QR Code scanned! Please confirm login on your WeChat app...")
					scannedPrinted = true
				}
			case "confirmed":
				if statusResp.BotToken == "" || statusResp.IlinkBotID == "" {
					return "", "", "", "", fmt.Errorf("login confirmed but missing bot_token or ilink_bot_id")
				}
				logger.InfoCF("weixin", "Login successful", map[string]any{
					"account_id": statusResp.IlinkBotID,
				})

				return statusResp.BotToken, statusResp.IlinkUserID, statusResp.IlinkBotID, statusResp.Baseurl, nil
			case "scaned_but_redirect":
				if statusResp.RedirectHost == "" {
					logger.WarnC(
						"weixin",
						"scaned_but_redirect received without redirect_host; continuing on current host",
					)
					continue
				}
				nextBaseURL := "https://" + statusResp.RedirectHost + "/"
				nextAPI, nextErr := NewApiClient(nextBaseURL, "", opts.Proxy)
				if nextErr != nil {
					logger.WarnCF("weixin", "Failed to switch QR polling host", map[string]any{
						"redirect_host": statusResp.RedirectHost,
						"error":         nextErr.Error(),
					})
					continue
				}
				pollAPI = nextAPI
				logger.InfoCF("weixin", "Switched QR polling host", map[string]any{
					"redirect_host": statusResp.RedirectHost,
				})
			case "expired":
				return "", "", "", "", fmt.Errorf("qrcode expired, please try again")
			default:
				logger.WarnCF("weixin", "Unknown QR code status", map[string]any{
					"status": statusResp.Status,
				})
			}
		}
	}
}
