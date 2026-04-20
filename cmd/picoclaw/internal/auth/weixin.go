package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/channels/weixin"
	"github.com/sipeed/picoclaw/pkg/config"
)

func newWeixinCommand() *cobra.Command {
	var baseURL string
	var channelName string
	var proxy string
	var timeout int

	cmd := &cobra.Command{
		Use:   "weixin",
		Short: "Connect a WeChat personal account via QR code",
		Long: `Start the interactive Weixin (WeChat personal) QR code login flow.

A QR code is displayed in the terminal. Scan it with the WeChat mobile app
to authorize your account. On success, the bot token is saved to the picoclaw
config so you can start the gateway immediately.

Example:
  picoclaw auth weixin`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runWeixinOnboard(channelName, baseURL, proxy, time.Duration(timeout)*time.Second)
		},
	}

	cmd.Flags().StringVar(&baseURL, "base-url", "https://ilinkai.weixin.qq.com/", "iLink API base URL")
	cmd.Flags().StringVar(&channelName, "channel", config.ChannelWeixin, "Channel name to create or update")
	cmd.Flags().StringVar(&proxy, "proxy", "", "HTTP proxy URL (e.g. http://localhost:7890)")
	cmd.Flags().IntVar(&timeout, "timeout", 300, "Login timeout in seconds")

	return cmd
}

func runWeixinOnboard(channelName, baseURL, proxy string, timeout time.Duration) error {
	fmt.Println("Starting Weixin (WeChat personal) login...")
	fmt.Println()

	botToken, userID, accountID, returnedBaseURL, err := weixin.PerformLoginInteractive(
		context.Background(),
		weixin.AuthFlowOpts{
			BaseURL: baseURL,
			Timeout: timeout,
			Proxy:   proxy,
		},
	)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	fmt.Println()
	fmt.Println("✅ Login successful!")
	fmt.Printf("   Account ID : %s\n", accountID)
	if userID != "" {
		fmt.Printf("   User ID    : %s\n", userID)
	}
	fmt.Println()

	// Prefer the server-returned base URL (may be region-specific)
	effectiveBaseURL := returnedBaseURL
	if effectiveBaseURL == "" {
		effectiveBaseURL = baseURL
	}

	if err := saveWeixinConfig(channelName, botToken, accountID, effectiveBaseURL, proxy); err != nil {
		fmt.Printf("⚠️  Could not auto-save to config: %v\n", err)
		printManualWeixinConfig(channelName, botToken, accountID, effectiveBaseURL)
		return nil
	}

	fmt.Println("✓ Config updated. Start the gateway with:")
	fmt.Println()
	fmt.Println("  picoclaw gateway")
	fmt.Println()
	fmt.Println("To restrict which WeChat users can send messages, add their user IDs")
	channelName = normalizeWeixinChannelName(channelName)
	fmt.Printf("to channels.%s.allow_from in your config.\n", channelName)

	return nil
}

func normalizeWeixinChannelName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return config.ChannelWeixin
	}
	return name
}

// saveWeixinConfig patches the named Weixin channel in the config and saves it.
func saveWeixinConfig(channelName, token, accountID, baseURL, proxy string) error {
	cfgPath := internal.GetConfigPath()
	channelName = normalizeWeixinChannelName(channelName)

	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if cfg.Channels == nil {
		cfg.Channels = config.ChannelsConfig{}
	}

	bc := cfg.Channels.Get(channelName)
	if bc == nil {
		bc = &config.Channel{Type: config.ChannelWeixin}
		cfg.Channels[channelName] = bc
	}
	if bc.Type != "" && bc.Type != config.ChannelWeixin {
		return fmt.Errorf("channel %q already exists with type %q", channelName, bc.Type)
	}
	bc.Type = config.ChannelWeixin
	bc.Enabled = true

	if decoded, err := bc.GetDecoded(); err == nil && decoded != nil {
		if weixinCfg, ok := decoded.(*config.WeixinSettings); ok {
			weixinCfg.Token = *config.NewSecureString(token)
			weixinCfg.AccountID = accountID
			if baseURL != "" {
				weixinCfg.BaseURL = baseURL
			}
			if proxy != "" {
				weixinCfg.Proxy = proxy
			}
		}
	}

	return config.SaveConfig(cfgPath, cfg)
}

func printManualWeixinConfig(channelName, token, accountID, baseURL string) {
	channelName = normalizeWeixinChannelName(channelName)
	fmt.Println()
	fmt.Println("Add the following to the channels section of your picoclaw config:")
	fmt.Println()
	fmt.Printf("  %q: {\n", channelName)
	fmt.Println(`    "enabled": true,`)
	fmt.Println(`    "type": "weixin",`)
	fmt.Printf("    \"token\": %q,\n", token)
	if accountID != "" {
		fmt.Printf("    \"account_id\": %q,\n", accountID)
	}
	const defaultBase = "https://ilinkai.weixin.qq.com/"
	if baseURL != "" && baseURL != defaultBase {
		fmt.Printf("    \"base_url\": %q,\n", baseURL)
	}
	fmt.Println(`    "allow_from": []`)
	fmt.Println(`  }`)
}
