package gateway

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func captureGatewayStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	runErr := fn()

	require.NoError(t, w.Close())
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, copyErr := io.Copy(&buf, r)
	require.NoError(t, copyErr)

	return buf.String(), runErr
}

func writeGatewayPidFile(t *testing.T, homePath string, processID int) {
	t.Helper()

	data := map[string]any{
		"pid":     processID,
		"token":   "test-token",
		"version": "test",
		"host":    "127.0.0.1",
		"port":    18790,
	}
	raw, err := json.Marshal(data)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(homePath, ".picoclaw.pid"), raw, 0o600)
	require.NoError(t, err)
}

func TestGatewayStatusCmdStopped(t *testing.T) {
	homePath := t.TempDir()

	output, err := captureGatewayStdout(t, func() error {
		return gatewayStatusCmd(homePath)
	})

	require.NoError(t, err)
	assert.Contains(t, output, "Gateway status: stopped")
}

func TestGatewayStopCmdNotRunning(t *testing.T) {
	homePath := t.TempDir()

	err := gatewayStopCmd(homePath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "gateway is not running")
}

func TestGatewayStopCmdRunningProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX signal semantics")
	}

	homePath := t.TempDir()
	sleepCmd := exec.Command("sleep", "30")
	require.NoError(t, sleepCmd.Start())
	t.Cleanup(func() {
		if sleepCmd.Process != nil {
			_ = sleepCmd.Process.Kill()
		}
		_ = sleepCmd.Wait()
	})

	writeGatewayPidFile(t, homePath, sleepCmd.Process.Pid)

	output, err := captureGatewayStdout(t, func() error {
		return gatewayStopCmd(homePath)
	})

	require.NoError(t, err)
	assert.Contains(t, output, "Sent stop signal to gateway")

	done := make(chan error, 1)
	go func() {
		done <- sleepCmd.Wait()
	}()

	select {
	case waitErr := <-done:
		if waitErr != nil {
			assert.True(t, strings.Contains(waitErr.Error(), "signal"))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("gateway process did not exit after stop signal")
	}
}
