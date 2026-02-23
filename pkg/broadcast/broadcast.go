package broadcast

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/KarakuriAgent/clawdroid/pkg/logger"
)

const (
	// Action is the intent action the Android app listens for.
	Action = "io.clawdroid.AGENT_MESSAGE"
	// Package is the Android app package name.
	Package = "io.clawdroid"
)

// Message represents a message to send via Android broadcast.
type Message struct {
	Content string `json:"content"`
	Type    string `json:"type,omitempty"`
}

// Send sends a message to the Android app via am broadcast.
// This works because the Go server runs inside Termux on the same device.
func Send(msg Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal broadcast message: %w", err)
	}

	cmd := exec.Command("am", "broadcast",
		"-a", Action,
		"-p", Package,
		"--es", "message", string(data),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.ErrorCF("broadcast", "am broadcast failed", map[string]interface{}{
			"error":  err.Error(),
			"output": string(output),
		})
		return fmt.Errorf("am broadcast failed: %w (%s)", err, string(output))
	}

	logger.InfoCF("broadcast", "Broadcast sent", map[string]interface{}{
		"content_len": len(msg.Content),
	})
	return nil
}
