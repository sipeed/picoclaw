package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/utils"
)

type ScreenshotTool struct {
	workspace      string
	restrict       bool
	maxFileBytes   int64
	timeout        time.Duration
	sendCallback   SendFileCallback
	defaultChannel string
	defaultChatID  string
	lookPath       func(string) (string, error)
	runCommand     func(ctx context.Context, command string, args ...string) error
	now            func() time.Time
}

func NewScreenshotTool(workspace string, restrict bool, maxFileBytes int64) *ScreenshotTool {
	return &ScreenshotTool{
		workspace:    workspace,
		restrict:     restrict,
		maxFileBytes: maxFileBytes,
		timeout:      45 * time.Second,
		lookPath:     exec.LookPath,
		runCommand: func(ctx context.Context, command string, args ...string) error {
			cmd := exec.CommandContext(ctx, command, args...)
			output, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
			}
			return nil
		},
		now: time.Now,
	}
}

func (t *ScreenshotTool) Name() string {
	return "screenshot"
}

func (t *ScreenshotTool) Description() string {
	return "Capture a screenshot (web URL or desktop) and optionally send it to user."
}

func (t *ScreenshotTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"url": map[string]interface{}{
				"type":        "string",
				"description": "Web page URL to capture. If omitted, captures local desktop screen.",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Optional output image path. Defaults to workspace/tmp/screenshots/*.png",
			},
			"caption": map[string]interface{}{
				"type":        "string",
				"description": "Optional caption when sending screenshot.",
			},
			"send": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether to send the screenshot to chat. Default true.",
			},
			"keep_local": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether to keep local screenshot file after sending. Default false.",
			},
			"width": map[string]interface{}{
				"type":        "integer",
				"description": "Browser viewport width for URL screenshot. Default 1366.",
			},
			"height": map[string]interface{}{
				"type":        "integer",
				"description": "Browser viewport height for URL screenshot. Default 768.",
			},
			"channel": map[string]interface{}{
				"type":        "string",
				"description": "Optional target channel override.",
			},
			"chat_id": map[string]interface{}{
				"type":        "string",
				"description": "Optional target chat id override.",
			},
		},
	}
}

func (t *ScreenshotTool) SetContext(channel, chatID string) {
	t.defaultChannel = channel
	t.defaultChatID = chatID
}

func (t *ScreenshotTool) SetSendCallback(callback SendFileCallback) {
	t.sendCallback = callback
}

func (t *ScreenshotTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	outputPath, err := t.resolveOutputPath(args)
	if err != nil {
		return ErrorResult(err.Error())
	}

	width := intFromArgs(args, "width", 1366)
	height := intFromArgs(args, "height", 768)
	targetURL, _ := args["url"].(string)

	cmdCtx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	if strings.TrimSpace(targetURL) != "" {
		if err := t.captureURL(cmdCtx, targetURL, outputPath, width, height); err != nil {
			return ErrorResult(fmt.Sprintf("capture URL screenshot failed: %v", err))
		}
	} else {
		if err := t.captureDesktop(cmdCtx, outputPath); err != nil {
			return ErrorResult(fmt.Sprintf("capture desktop screenshot failed: %v", err))
		}
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		return ErrorResult(fmt.Sprintf("screenshot not created: %v", err))
	}
	if info.Size() == 0 {
		return ErrorResult("screenshot is empty")
	}
	if t.maxFileBytes > 0 && info.Size() > t.maxFileBytes {
		return ErrorResult(fmt.Sprintf("screenshot too large: %d bytes (limit %d bytes)", info.Size(), t.maxFileBytes))
	}

	send := boolFromArgs(args, "send", true)
	keepLocal := boolFromArgs(args, "keep_local", false)
	caption, _ := args["caption"].(string)

	if !send {
		return UserResult(fmt.Sprintf("Screenshot saved: %s", outputPath))
	}

	channel, _ := args["channel"].(string)
	chatID, _ := args["chat_id"].(string)
	if channel == "" {
		channel = t.defaultChannel
	}
	if chatID == "" {
		chatID = t.defaultChatID
	}
	if channel == "" || chatID == "" {
		return ErrorResult("No target channel/chat specified")
	}
	if t.sendCallback == nil {
		return ErrorResult("Screenshot sending not configured")
	}

	if err := t.sendCallback(bus.OutboundMessage{
		Channel: channel,
		ChatID:  chatID,
		Content: caption,
		Attachments: []bus.Attachment{
			{
				Type:     "image",
				Path:     outputPath,
				FileName: filepath.Base(outputPath),
				MIMEType: "image/png",
			},
		},
	}); err != nil {
		return ErrorResult(fmt.Sprintf("sending screenshot: %v", err))
	}

	if !keepLocal {
		utils.ScheduleFileCleanup(outputPath, 15*time.Minute, "screenshot")
	}

	return SilentResult(fmt.Sprintf("Screenshot sent to %s:%s", channel, chatID))
}

func (t *ScreenshotTool) resolveOutputPath(args map[string]interface{}) (string, error) {
	rawPath, _ := args["path"].(string)
	if strings.TrimSpace(rawPath) == "" {
		rawPath = filepath.Join("tmp", "screenshots", t.now().Format("20060102_150405")+".png")
	}

	resolvedPath, err := validatePath(rawPath, t.workspace, t.restrict)
	if err != nil {
		return "", err
	}
	if ext := strings.ToLower(filepath.Ext(resolvedPath)); ext == "" {
		resolvedPath += ".png"
	}

	if err := os.MkdirAll(filepath.Dir(resolvedPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create screenshot directory: %w", err)
	}
	return resolvedPath, nil
}

func (t *ScreenshotTool) captureURL(ctx context.Context, targetURL, outputPath string, width, height int) error {
	browsers := []string{"chromium-browser", "chromium", "google-chrome", "google-chrome-stable"}
	var browser string
	for _, candidate := range browsers {
		if _, err := t.lookPath(candidate); err == nil {
			browser = candidate
			break
		}
	}
	if browser == "" {
		return fmt.Errorf("no headless browser found (tried: %s)", strings.Join(browsers, ", "))
	}

	args := []string{
		"--headless",
		"--disable-gpu",
		"--hide-scrollbars",
		"--no-sandbox",
		fmt.Sprintf("--window-size=%d,%d", width, height),
		fmt.Sprintf("--screenshot=%s", outputPath),
		targetURL,
	}
	return t.runCommand(ctx, browser, args...)
}

func (t *ScreenshotTool) captureDesktop(ctx context.Context, outputPath string) error {
	type command struct {
		name string
		args []string
	}

	candidates := []command{
		{name: "scrot", args: []string{outputPath}},
		{name: "grim", args: []string{outputPath}},
		{name: "import", args: []string{"-window", "root", outputPath}},
	}

	var available []command
	for _, candidate := range candidates {
		if _, err := t.lookPath(candidate.name); err == nil {
			available = append(available, candidate)
		}
	}
	if len(available) == 0 {
		return fmt.Errorf("no desktop screenshot command found (tried: scrot, grim, import)")
	}

	var lastErr error
	for _, cmd := range available {
		if err := t.runCommand(ctx, cmd.name, cmd.args...); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	return lastErr
}

func intFromArgs(args map[string]interface{}, key string, fallback int) int {
	v, ok := args[key]
	if !ok {
		return fallback
	}
	switch n := v.(type) {
	case int:
		if n > 0 {
			return n
		}
	case float64:
		if int(n) > 0 {
			return int(n)
		}
	}
	return fallback
}

func boolFromArgs(args map[string]interface{}, key string, fallback bool) bool {
	v, ok := args[key]
	if !ok {
		return fallback
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return fallback
}
