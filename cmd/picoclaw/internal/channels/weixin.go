package channels

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal/onboard"
)

func newWeixinCommand() *cobra.Command {
	var baseURL string
	var proxy string
	var timeout int

	cmd := &cobra.Command{
		Use:     "weixin",
		Aliases: []string{"wechat", "wx"},
		Short:   "Manage Weixin (WeChat personal) channel login",
	}

	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "Start QR login for Weixin (WeChat personal)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return onboard.RunWeixinOnboard(baseURL, proxy, time.Duration(timeout)*time.Second)
		},
	}

	loginCmd.Flags().StringVar(&baseURL, "base-url", "https://ilinkai.weixin.qq.com/", "iLink API base URL")
	loginCmd.Flags().StringVar(&proxy, "proxy", "", "HTTP proxy URL (e.g. http://localhost:7890)")
	loginCmd.Flags().IntVar(&timeout, "timeout", 300, "Login timeout in seconds")

	cmd.AddCommand(loginCmd)

	return cmd
}
