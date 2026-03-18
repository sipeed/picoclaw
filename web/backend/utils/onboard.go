package utils

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var execCommand = exec.Command

func EnsureOnboarded(configPath string) error {
	_, err := os.Stat(configPath)
	if err == nil {
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("stat config: %w", err)
	}

	cmd := execCommand(FindPicoclawBinary(), "onboard")
	cmd.Env = append(os.Environ(), "PICOCLAW_CONFIG="+configPath)
	cmd.Stdin = strings.NewReader("n\n")

	output, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			return fmt.Errorf("run onboard: %w", err)
		}
		return fmt.Errorf("run onboard: %w: %s", err, trimmed)
	}

	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("onboard completed but did not create config %s", configPath)
		}
		return fmt.Errorf("verify config after onboard: %w", err)
	}

	return nil
}
