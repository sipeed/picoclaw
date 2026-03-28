//go:build windows
// +build windows

package ui

import (
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"testing"
)

func TestIsGatewayProcessRunning(t *testing.T) {
	origExecCommand := execCommand
	defer func() { execCommand = origExecCommand }()

	tests := []struct {
		name       string
		exitCode   int
		wantResult bool
	}{
		{
			name:       "running",
			exitCode:   0,
			wantResult: true,
		},
		{
			name:       "not running",
			exitCode:   1,
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotName string
			var gotArgs []string

			execCommand = func(name string, args ...string) *exec.Cmd {
				gotName = name
				gotArgs = args

				cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess", "--")
				cmd.Env = append(os.Environ(),
					"GO_WANT_HELPER_PROCESS=1",
					"GO_WANT_HELPER_PROCESS_EXIT_CODE="+strconv.Itoa(tt.exitCode),
				)
				return cmd
			}

			got := isGatewayProcessRunning()
			if got != tt.wantResult {
				t.Errorf("isGatewayProcessRunning() = %v, want %v", got, tt.wantResult)
			}
			if gotName != "cmd" {
				t.Errorf("expected command name cmd, got %s", gotName)
			}
			expectedArgs := []string{"/C", `tasklist /FI "IMAGENAME eq jane-ai.exe" | findstr /I /C:"jane-ai.exe" >NUL || tasklist /FI "IMAGENAME eq picoclaw.exe" | findstr /I /C:"picoclaw.exe" >NUL`}
			if !reflect.DeepEqual(gotArgs, expectedArgs) {
				t.Errorf("expected args %v, got %v", expectedArgs, gotArgs)
			}
		})
	}
}

func TestStopGatewayProcess(t *testing.T) {
	origExecCommand := execCommand
	defer func() { execCommand = origExecCommand }()

	tests := []struct {
		name     string
		exitCode int
		wantErr  bool
	}{
		{
			name:     "success",
			exitCode: 0,
			wantErr:  false,
		},
		{
			name:     "failure",
			exitCode: 1,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotName string
			var gotArgs []string

			execCommand = func(name string, args ...string) *exec.Cmd {
				gotName = name
				gotArgs = args

				cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess", "--")
				cmd.Env = append(os.Environ(),
					"GO_WANT_HELPER_PROCESS=1",
					"GO_WANT_HELPER_PROCESS_EXIT_CODE="+strconv.Itoa(tt.exitCode),
				)
				return cmd
			}

			err := stopGatewayProcess()
			if (err != nil) != tt.wantErr {
				t.Errorf("stopGatewayProcess() error = %v, wantErr %v", err, tt.wantErr)
			}
			if gotName != "cmd" {
				t.Errorf("expected command name cmd, got %s", gotName)
			}
			expectedArgs := []string{"/C", `taskkill /F /IM jane-ai.exe >NUL 2>&1 || taskkill /F /IM picoclaw.exe >NUL 2>&1`}
			if !reflect.DeepEqual(gotArgs, expectedArgs) {
				t.Errorf("expected args %v, got %v", expectedArgs, gotArgs)
			}
		})
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	if code := os.Getenv("GO_WANT_HELPER_PROCESS_EXIT_CODE"); code != "" {
		c, _ := strconv.Atoi(code)
		os.Exit(c)
	}
	os.Exit(0)
}
