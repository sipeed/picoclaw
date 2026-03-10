package tools

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/sipeed/picoclaw/pkg/auth"
	"golang.org/x/crypto/ssh"
)

type VPSTool struct {
	Host string
	User string
}

func (t *VPSTool) Name() string {
	return "vps_exec"
}

func (t *VPSTool) Description() string {
	return "Execute commands on the high-compute VPS for heavy tasks like video editing (ffmpeg), ASR (whisper), and Google Drive operations (gws)."
}

func (t *VPSTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "The shell command to execute on the VPS.",
			},
		},
		"required": []string{"command"},
	}
}

func (t *VPSTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	cmdStr, _ := args["command"].(string)
	if cmdStr == "" {
		return ErrorResult("Command is required")
	}

	cred, err := auth.GetCredential("vps")
	if err != nil || cred == nil || cred.AccessToken == "" {
		return ErrorResult("VPS credentials not found. Please set them using the vps provider.")
	}

	config := &ssh.ClientConfig{
		User: t.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(cred.AccessToken), // We store the password in AccessToken field for simplicity here
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}

	client, err := ssh.Dial("tcp", t.Host+":22", config)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to connect to VPS: %v", err))
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to create SSH session: %v", err))
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmdStr)
	if err != nil {
		if err == io.EOF {
			return SilentResult(string(output))
		}
		return ErrorResult(fmt.Sprintf("VPS execution failed: %v\nOutput: %s", err, string(output)))
	}

	return SilentResult(string(output))
}
