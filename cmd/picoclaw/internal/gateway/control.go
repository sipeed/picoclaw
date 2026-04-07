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

type gatewayTarget struct {
	data    *pid.PidFileData
	process *os.Process
}

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
	target, err := resolveGatewayTarget(homePath)
	if err != nil {
		return err
	}
	if target == nil {
		fmt.Println("Gateway status: stopped")
		return nil
	}

	fmt.Printf(
		"Gateway status: running (PID: %d, host: %s, port: %d)\n",
		target.data.PID,
		target.data.Host,
		target.data.Port,
	)
	return nil
}

func gatewayStopCmd(homePath string) error {
	target, err := resolveGatewayTarget(homePath)
	if err != nil {
		return err
	}
	if target == nil {
		return fmt.Errorf("gateway is not running")
	}

	process := target.process
	if process == nil {
		return fmt.Errorf("failed to find gateway process (PID: %d)", target.data.PID)
	}

	if runtime.GOOS == "windows" {
		err = process.Kill()
	} else {
		err = process.Signal(syscall.SIGTERM)
	}
	if err != nil {
		return fmt.Errorf("failed to stop gateway (PID: %d): %w", target.data.PID, err)
	}

	fmt.Printf("Sent stop signal to gateway (PID: %d)\n", target.data.PID)
	return nil
}

func resolveGatewayTarget(homePath string) (*gatewayTarget, error) {
	data := pid.ReadPidFileWithCheck(homePath)
	if data == nil {
		return nil, nil
	}

	process, err := os.FindProcess(data.PID)
	if err != nil {
		return nil, fmt.Errorf("failed to find gateway process (PID: %d): %w", data.PID, err)
	}

	// Hardening: when possible, ensure the PID file still points to a picoclaw
	// gateway process before we report or signal it. Currently we only have a
	// reliable, dependency-free implementation on Linux (/proc).
	if runtime.GOOS == "linux" {
		err = verifyGatewayProcessIdentity(data.PID)
		if err != nil {
			return nil, err
		}
	}

	return &gatewayTarget{
		data:    data,
		process: process,
	}, nil
}
