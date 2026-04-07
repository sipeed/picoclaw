package gateway

import (
	"fmt"
	"os"
	"runtime"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/pid"
)

func newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show gateway process status",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return gatewayStatusCmd(internal.GetPicoclawHome())
		},
	}
}

func newStopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop a running gateway process",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return gatewayStopCmd(internal.GetPicoclawHome())
		},
	}
}

func gatewayStatusCmd(homePath string) error {
	data := pid.ReadPidFileWithCheck(homePath)
	if data == nil {
		fmt.Println("Gateway status: stopped")
		return nil
	}

	fmt.Printf(
		"Gateway status: running (PID: %d, host: %s, port: %d)\n",
		data.PID,
		data.Host,
		data.Port,
	)
	return nil
}

func gatewayStopCmd(homePath string) error {
	data := pid.ReadPidFileWithCheck(homePath)
	if data == nil {
		return fmt.Errorf("gateway is not running")
	}

	process, err := os.FindProcess(data.PID)
	if err != nil {
		return fmt.Errorf("failed to find gateway process (PID: %d): %w", data.PID, err)
	}

	if runtime.GOOS == "windows" {
		err = process.Kill()
	} else {
		err = process.Signal(syscall.SIGTERM)
	}
	if err != nil {
		return fmt.Errorf("failed to stop gateway (PID: %d): %w", data.PID, err)
	}

	fmt.Printf("Sent stop signal to gateway (PID: %d)\n", data.PID)
	return nil
}
