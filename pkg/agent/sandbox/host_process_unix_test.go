//go:build !windows

package sandbox

import (
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPrepareCommandForTermination(t *testing.T) {
	// Should not panic on nil
	prepareCommandForTermination(nil)

	cmd := exec.Command("echo", "test")
	prepareCommandForTermination(cmd)

	if cmd.SysProcAttr == nil {
		t.Fatal("expected SysProcAttr to be initialized")
	}
	if !cmd.SysProcAttr.Setpgid {
		t.Fatal("expected Setpgid to be true")
	}
}

func TestTerminateProcessTree(t *testing.T) {
	// Should not panic on nil cmd or nil process
	if err := terminateProcessTree(nil); err != nil {
		t.Fatalf("expected nil error for nil cmd, got: %v", err)
	}

	cmdUnstarted := exec.Command("echo", "test")
	if err := terminateProcessTree(cmdUnstarted); err != nil {
		t.Fatalf("expected nil error for unstarted cmd, got: %v", err)
	}

	// Start a real dummy process to test killing
	cmd := exec.Command("sleep", "1")
	prepareCommandForTermination(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start cmd: %v", err)
	}

	errChan := make(chan error, 1)
	go func() {
		errChan <- cmd.Wait()
	}()

	if err := terminateProcessTree(cmd); err != nil {
		t.Fatalf("terminateProcessTree failed: %v", err)
	}

	// Verify the process is dead by waiting for Wait() to return
	require.Eventually(t, func() bool {
		select {
		case err := <-errChan:
			// Process died (killed), err should not be nil
			return err != nil
		default:
			return false
		}
	}, 2*time.Second, 50*time.Millisecond, "expected process Wait to finish after termination")
}
