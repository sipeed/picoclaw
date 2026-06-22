package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

const (
	AndroidToolName = "android"

	defaultAndroidADBPath             = "adb"
	defaultAndroidTimeoutMS           = 5000
	defaultAndroidMaxActionsPerMinute = 20
	defaultAndroidScreenshotMaxBytes  = 2 * 1024 * 1024
	maxAndroidTextInputRunes          = 512
	maxAndroidUITreeNodes             = 80
)

var (
	androidWindowPackageRe = regexp.MustCompile(`(?m)(?:mCurrentFocus|mFocusedApp).*?\s([A-Za-z0-9_.$]+)/(?:[A-Za-z0-9_.$]+|\.)`)
	androidSizeRe          = regexp.MustCompile(`(?m)(?:Physical size|Override size):\s*(\d+)x(\d+)`)
	androidXMLNodeRe       = regexp.MustCompile(`<node\s+([^>]*)>`)
	androidXMLAttrRe       = regexp.MustCompile(`([A-Za-z0-9_-]+)="([^"]*)"`)
)

type AndroidToolOptions struct {
	ADBPath             string
	DefaultTimeoutMS    int
	MaxActionsPerMinute int
	ScreenshotMaxBytes  int
	Devices             []AndroidDeviceConfig
}

type AndroidDeviceConfig struct {
	ID            string
	Serial        string
	AllowPackages []string
	BlockPackages []string
}

type androidRunner interface {
	Run(ctx context.Context, timeout time.Duration, name string, args ...string) ([]byte, error)
}

type execAndroidRunner struct{}

type AndroidTool struct {
	adbPath             string
	defaultTimeout      time.Duration
	maxActionsPerMinute int
	screenshotMaxBytes  int
	devices             []AndroidDeviceConfig
	runner              androidRunner

	mu             sync.Mutex
	actionTimestamps []time.Time
}

func NewAndroidTool(opts AndroidToolOptions) *AndroidTool {
	adbPath := strings.TrimSpace(opts.ADBPath)
	if adbPath == "" {
		adbPath = defaultAndroidADBPath
	}

	timeoutMS := opts.DefaultTimeoutMS
	if timeoutMS <= 0 {
		timeoutMS = defaultAndroidTimeoutMS
	}

	maxActions := opts.MaxActionsPerMinute
	if maxActions <= 0 {
		maxActions = defaultAndroidMaxActionsPerMinute
	}

	maxScreenshotBytes := opts.ScreenshotMaxBytes
	if maxScreenshotBytes <= 0 {
		maxScreenshotBytes = defaultAndroidScreenshotMaxBytes
	}

	return &AndroidTool{
		adbPath:             adbPath,
		defaultTimeout:      time.Duration(timeoutMS) * time.Millisecond,
		maxActionsPerMinute: maxActions,
		screenshotMaxBytes:  maxScreenshotBytes,
		devices:             normalizeAndroidDevices(opts.Devices),
		runner:              execAndroidRunner{},
	}
}

func AndroidToolEnabledFromEnv() bool {
	return parseAndroidBoolEnv("PICOCLAW_TOOLS_ANDROID_ENABLED", false)
}

func NewAndroidToolFromEnv() *AndroidTool {
	deviceID := strings.TrimSpace(os.Getenv("PICOCLAW_TOOLS_ANDROID_DEVICE_ID"))
	deviceSerial := strings.TrimSpace(os.Getenv("PICOCLAW_TOOLS_ANDROID_DEVICE_SERIAL"))
	var devices []AndroidDeviceConfig
	if deviceID != "" || deviceSerial != "" {
		if deviceID == "" {
			deviceID = "default"
		}
		devices = append(devices, AndroidDeviceConfig{
			ID:            deviceID,
			Serial:        deviceSerial,
			AllowPackages: splitAndroidCSV(os.Getenv("PICOCLAW_TOOLS_ANDROID_ALLOW_PACKAGES")),
			BlockPackages: splitAndroidCSV(os.Getenv("PICOCLAW_TOOLS_ANDROID_BLOCK_PACKAGES")),
		})
	}

	return NewAndroidTool(AndroidToolOptions{
		ADBPath:             os.Getenv("PICOCLAW_TOOLS_ANDROID_ADB_PATH"),
		DefaultTimeoutMS:    parseAndroidIntEnv("PICOCLAW_TOOLS_ANDROID_TIMEOUT_MS", defaultAndroidTimeoutMS),
		MaxActionsPerMinute: parseAndroidIntEnv("PICOCLAW_TOOLS_ANDROID_MAX_ACTIONS_PER_MINUTE", defaultAndroidMaxActionsPerMinute),
		ScreenshotMaxBytes:  parseAndroidIntEnv("PICOCLAW_TOOLS_ANDROID_SCREENSHOT_MAX_BYTES", defaultAndroidScreenshotMaxBytes),
		Devices:             devices,
	})
}

func (t *AndroidTool) Name() string {
	return AndroidToolName
}

func (t *AndroidTool) Description() string {
	return "Control an explicitly allowlisted Android device through safe ADB primitives. Actions: devices, status, screenshot, ui_tree, tap, swipe, text, key, wake. No arbitrary shell execution."
}

func (t *AndroidTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"devices", "status", "screenshot", "ui_tree", "tap", "swipe", "text", "key", "wake"},
				"description": "Android action to execute through ADB.",
			},
			"device_id": map[string]any{
				"type":        "string",
				"description": "Configured Android device ID. Required when more than one device is configured.",
			},
			"x":            map[string]any{"type": "integer", "description": "X coordinate for tap."},
			"y":            map[string]any{"type": "integer", "description": "Y coordinate for tap."},
			"x1":           map[string]any{"type": "integer", "description": "Swipe start X coordinate."},
			"y1":           map[string]any{"type": "integer", "description": "Swipe start Y coordinate."},
			"x2":           map[string]any{"type": "integer", "description": "Swipe end X coordinate."},
			"y2":           map[string]any{"type": "integer", "description": "Swipe end Y coordinate."},
			"duration_ms":  map[string]any{"type": "integer", "description": "Swipe duration in milliseconds. Default: 300. Max: 5000."},
			"text":         map[string]any{"type": "string", "description": "Text for Android input text. Spaces are converted to %s. Max 512 runes."},
			"key":          map[string]any{"type": "string", "enum": []string{"HOME", "BACK", "ENTER", "WAKEUP", "POWER"}},
			"confirm":      map[string]any{"type": "boolean", "description": "Must be true for mutating actions unless dry_run is true."},
			"dry_run":      map[string]any{"type": "boolean", "description": "Validate the action without sending it to the Android device."},
			"include_image": map[string]any{"type": "boolean", "description": "For screenshot, include base64 PNG bytes in the result. Default: true."},
			"timeout_ms":   map[string]any{"type": "integer", "description": "Per-action timeout in milliseconds. Default comes from tool config."},
		},
		"required": []string{"action"},
	}
}

func (t *AndroidTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	action := strings.ToLower(strings.TrimSpace(stringArg(args, "action")))
	if action == "" {
		return ErrorResult("action is required")
	}

	switch action {
	case "devices":
		return t.devicesAction(ctx, args)
	case "status":
		return t.statusAction(ctx, args)
	case "screenshot":
		return t.screenshotAction(ctx, args)
	case "ui_tree":
		return t.uiTreeAction(ctx, args)
	case "tap":
		return t.tapAction(ctx, args)
	case "swipe":
		return t.swipeAction(ctx, args)
	case "text":
		return t.textAction(ctx, args)
	case "key":
		return t.keyAction(ctx, args)
	case "wake":
		return t.wakeAction(ctx, args)
	default:
		return ErrorResult("unknown android action: " + action)
	}
}

func (t *AndroidTool) devicesAction(ctx context.Context, args map[string]any) *ToolResult {
	out, err := t.runADB(ctx, timeoutFromArgs(args, t.defaultTimeout), "devices", "-l")
	if err != nil {
		return ErrorResult("adb devices failed: " + err.Error())
	}
	result, _ := json.MarshalIndent(map[string]any{
		"action":             "devices",
		"configured_devices": t.devices,
		"adb_devices":        parseADBDevices(string(out)),
		"raw":                strings.TrimSpace(string(out)),
	}, "", "  ")
	return SilentResult(string(result))
}

func (t *AndroidTool) statusAction(ctx context.Context, args map[string]any) *ToolResult {
	device, errResult := t.resolveDevice(args)
	if errResult != nil {
		return errResult
	}
	timeout := timeoutFromArgs(args, t.defaultTimeout)
	model, _ := t.runShellText(ctx, device, timeout, "getprop", "ro.product.model")
	androidVersion, _ := t.runShellText(ctx, device, timeout, "getprop", "ro.build.version.release")
	size, _ := t.runShellText(ctx, device, timeout, "wm", "size")
	battery, _ := t.runShellText(ctx, device, timeout, "dumpsys", "battery")
	currentPackage, _ := t.currentPackage(ctx, device, timeout)

	result, _ := json.MarshalIndent(map[string]any{
		"action":          "status",
		"device_id":       device.ID,
		"serial":          device.Serial,
		"model":           strings.TrimSpace(model),
		"android_version": strings.TrimSpace(androidVersion),
		"screen_size":     strings.TrimSpace(size),
		"current_package": currentPackage,
		"battery":         limitAndroidString(strings.TrimSpace(battery), 4096),
	}, "", "  ")
	return SilentResult(string(result))
}

func (t *AndroidTool) screenshotAction(ctx context.Context, args map[string]any) *ToolResult {
	device, errResult := t.resolveDevice(args)
	if errResult != nil {
		return errResult
	}
	if errResult = t.checkRateLimit(); errResult != nil {
		return errResult
	}
	timeout := timeoutFromArgs(args, t.defaultTimeout)
	if errResult = t.ensureScreenActionAllowed(ctx, device, timeout); errResult != nil {
		return errResult
	}

	adbArgs := append(device.adbArgs(), "exec-out", "screencap", "-p")
	png, err := t.runADB(ctx, timeout, adbArgs...)
	if err != nil {
		return ErrorResult("adb screenshot failed: " + err.Error())
	}
	if len(png) > t.screenshotMaxBytes {
		return ErrorResult(fmt.Sprintf("screenshot too large: %d bytes exceeds max %d", len(png), t.screenshotMaxBytes))
	}

	includeImage := true
	if v, ok := args["include_image"].(bool); ok {
		includeImage = v
	}
	payload := map[string]any{
		"action":       "screenshot",
		"device_id":    device.ID,
		"serial":       device.Serial,
		"content_type": "image/png",
		"bytes":        len(png),
	}
	if includeImage {
		payload["base64"] = base64.StdEncoding.EncodeToString(png)
	}
	result, _ := json.MarshalIndent(payload, "", "  ")
	logger.InfoCF("android", "Captured Android screenshot", map[string]any{"device_id": device.ID, "bytes": len(png)})
	return SilentResult(string(result))
}

func (t *AndroidTool) uiTreeAction(ctx context.Context, args map[string]any) *ToolResult {
	device, errResult := t.resolveDevice(args)
	if errResult != nil {
		return errResult
	}
	if errResult = t.checkRateLimit(); errResult != nil {
		return errResult
	}
	timeout := timeoutFromArgs(args, t.defaultTimeout)
	if errResult = t.ensureScreenActionAllowed(ctx, device, timeout); errResult != nil {
		return errResult
	}

	_, err := t.runShell(ctx, device, timeout, "uiautomator", "dump", "/sdcard/window.xml")
	if err != nil {
		return ErrorResult("uiautomator dump failed: " + err.Error())
	}
	xmlRaw, err := t.runShellText(ctx, device, timeout, "cat", "/sdcard/window.xml")
	if err != nil {
		return ErrorResult("reading UI hierarchy failed: " + err.Error())
	}

	result, _ := json.MarshalIndent(map[string]any{
		"action":    "ui_tree",
		"device_id": device.ID,
		"nodes":     summarizeAndroidUIXML(xmlRaw),
	}, "", "  ")
	logger.InfoCF("android", "Captured Android UI tree", map[string]any{"device_id": device.ID})
	return SilentResult(string(result))
}

func (t *AndroidTool) tapAction(ctx context.Context, args map[string]any) *ToolResult {
	device, timeout, dryRun, errResult := t.prepareMutatingAction(ctx, args, true)
	if errResult != nil {
		return errResult
	}
	x, ok := intArg(args, "x")
	if !ok {
		return ErrorResult("x is required for tap")
	}
	y, ok := intArg(args, "y")
	if !ok {
		return ErrorResult("y is required for tap")
	}
	if errResult = t.validatePoint(ctx, device, timeout, x, y); errResult != nil {
		return errResult
	}
	if dryRun {
		return SilentResult(fmt.Sprintf("dry_run: would tap %d,%d on %s", x, y, device.ID))
	}
	_, err := t.runShell(ctx, device, timeout, "input", "tap", strconv.Itoa(x), strconv.Itoa(y))
	if err != nil {
		return ErrorResult("adb tap failed: " + err.Error())
	}
	t.audit("tap", device, map[string]any{"x": x, "y": y})
	return SilentResult(fmt.Sprintf("tap sent to %s at (%d,%d)", device.ID, x, y))
}

func (t *AndroidTool) swipeAction(ctx context.Context, args map[string]any) *ToolResult {
	device, timeout, dryRun, errResult := t.prepareMutatingAction(ctx, args, true)
	if errResult != nil {
		return errResult
	}
	x1, ok := intArg(args, "x1")
	if !ok {
		return ErrorResult("x1 is required for swipe")
	}
	y1, ok := intArg(args, "y1")
	if !ok {
		return ErrorResult("y1 is required for swipe")
	}
	x2, ok := intArg(args, "x2")
	if !ok {
		return ErrorResult("x2 is required for swipe")
	}
	y2, ok := intArg(args, "y2")
	if !ok {
		return ErrorResult("y2 is required for swipe")
	}
	durationMS, ok := intArg(args, "duration_ms")
	if !ok || durationMS <= 0 {
		durationMS = 300
	}
	if durationMS > 5000 {
		return ErrorResult("duration_ms must be <= 5000")
	}
	if errResult = t.validatePoint(ctx, device, timeout, x1, y1); errResult != nil {
		return errResult
	}
	if errResult = t.validatePoint(ctx, device, timeout, x2, y2); errResult != nil {
		return errResult
	}
	if dryRun {
		return SilentResult(fmt.Sprintf("dry_run: would swipe %d,%d -> %d,%d on %s", x1, y1, x2, y2, device.ID))
	}
	_, err := t.runShell(ctx, device, timeout, "input", "swipe", strconv.Itoa(x1), strconv.Itoa(y1), strconv.Itoa(x2), strconv.Itoa(y2), strconv.Itoa(durationMS))
	if err != nil {
		return ErrorResult("adb swipe failed: " + err.Error())
	}
	t.audit("swipe", device, map[string]any{"x1": x1, "y1": y1, "x2": x2, "y2": y2, "duration_ms": durationMS})
	return SilentResult(fmt.Sprintf("swipe sent to %s", device.ID))
}

func (t *AndroidTool) textAction(ctx context.Context, args map[string]any) *ToolResult {
	device, timeout, dryRun, errResult := t.prepareMutatingAction(ctx, args, true)
	if errResult != nil {
		return errResult
	}
	text := stringArg(args, "text")
	if strings.TrimSpace(text) == "" {
		return ErrorResult("text is required")
	}
	if len([]rune(text)) > maxAndroidTextInputRunes {
		return ErrorResult(fmt.Sprintf("text too long: max %d runes", maxAndroidTextInputRunes))
	}
	encoded := strings.ReplaceAll(text, " ", "%s")
	if dryRun {
		return SilentResult(fmt.Sprintf("dry_run: would input %d runes on %s", len([]rune(text)), device.ID))
	}
	_, err := t.runShell(ctx, device, timeout, "input", "text", encoded)
	if err != nil {
		return ErrorResult("adb text input failed: " + err.Error())
	}
	t.audit("text", device, map[string]any{"runes": len([]rune(text))})
	return SilentResult(fmt.Sprintf("text input sent to %s", device.ID))
}

func (t *AndroidTool) keyAction(ctx context.Context, args map[string]any) *ToolResult {
	keyName := strings.ToUpper(strings.TrimSpace(stringArg(args, "key")))
	keyCode, ok := map[string]string{
		"HOME":   "KEYCODE_HOME",
		"BACK":   "KEYCODE_BACK",
		"ENTER":  "KEYCODE_ENTER",
		"WAKEUP": "KEYCODE_WAKEUP",
		"POWER":  "KEYCODE_POWER",
	}[keyName]
	if !ok {
		return ErrorResult("key must be one of HOME, BACK, ENTER, WAKEUP, POWER")
	}
	checkPackage := keyName != "WAKEUP" && keyName != "POWER"
	device, timeout, dryRun, errResult := t.prepareMutatingAction(ctx, args, checkPackage)
	if errResult != nil {
		return errResult
	}
	if dryRun {
		return SilentResult(fmt.Sprintf("dry_run: would send key %s to %s", keyName, device.ID))
	}
	_, err := t.runShell(ctx, device, timeout, "input", "keyevent", keyCode)
	if err != nil {
		return ErrorResult("adb key event failed: " + err.Error())
	}
	t.audit("key", device, map[string]any{"key": keyName})
	return SilentResult(fmt.Sprintf("key %s sent to %s", keyName, device.ID))
}

func (t *AndroidTool) wakeAction(ctx context.Context, args map[string]any) *ToolResult {
	device, timeout, dryRun, errResult := t.prepareMutatingAction(ctx, args, false)
	if errResult != nil {
		return errResult
	}
	if dryRun {
		return SilentResult("dry_run: would wake " + device.ID)
	}
	_, err := t.runShell(ctx, device, timeout, "input", "keyevent", "KEYCODE_WAKEUP")
	if err != nil {
		return ErrorResult("adb wake failed: " + err.Error())
	}
	t.audit("wake", device, nil)
	return SilentResult("wake sent to " + device.ID)
}

func (t *AndroidTool) prepareMutatingAction(ctx context.Context, args map[string]any, checkPackage bool) (AndroidDeviceConfig, time.Duration, bool, *ToolResult) {
	device, errResult := t.resolveDevice(args)
	if errResult != nil {
		return AndroidDeviceConfig{}, 0, false, errResult
	}
	dryRun, _ := args["dry_run"].(bool)
	confirm, _ := args["confirm"].(bool)
	if !dryRun && !confirm {
		return AndroidDeviceConfig{}, 0, false, ErrorResult("mutating Android actions require confirm: true")
	}
	if errResult = t.checkRateLimit(); errResult != nil {
		return AndroidDeviceConfig{}, 0, false, errResult
	}
	timeout := timeoutFromArgs(args, t.defaultTimeout)
	if checkPackage {
		if errResult = t.ensureScreenActionAllowed(ctx, device, timeout); errResult != nil {
			return AndroidDeviceConfig{}, 0, false, errResult
		}
	}
	return device, timeout, dryRun, nil
}

func (t *AndroidTool) resolveDevice(args map[string]any) (AndroidDeviceConfig, *ToolResult) {
	if len(t.devices) == 0 {
		return AndroidDeviceConfig{}, ErrorResult("no Android devices configured; add at least one allowlisted device before using android actions")
	}
	deviceID := strings.TrimSpace(stringArg(args, "device_id"))
	if deviceID == "" && len(t.devices) == 1 {
		return t.devices[0], nil
	}
	if deviceID == "" {
		return AndroidDeviceConfig{}, ErrorResult("device_id is required when multiple Android devices are configured")
	}
	for _, d := range t.devices {
		if d.ID == deviceID {
			return d, nil
		}
	}
	return AndroidDeviceConfig{}, ErrorResult("unknown or not allowlisted Android device_id: " + deviceID)
}

func (t *AndroidTool) ensureScreenActionAllowed(ctx context.Context, device AndroidDeviceConfig, timeout time.Duration) *ToolResult {
	pkg, err := t.currentPackage(ctx, device, timeout)
	if err != nil {
		return ErrorResult("failed to resolve current Android package: " + err.Error())
	}
	pkg = strings.TrimSpace(pkg)
	if pkg == "" && len(device.AllowPackages) > 0 {
		return ErrorResult("current package is unknown and allow_packages is configured")
	}
	for _, pattern := range device.effectiveBlockPackages() {
		if matchAndroidPackage(pattern, pkg) {
			return ErrorResult("blocked Android package is active: " + pkg)
		}
	}
	if len(device.AllowPackages) > 0 {
		for _, pattern := range device.AllowPackages {
			if matchAndroidPackage(pattern, pkg) {
				return nil
			}
		}
		return ErrorResult("current Android package is not in allow_packages: " + pkg)
	}
	return nil
}

func (t *AndroidTool) currentPackage(ctx context.Context, device AndroidDeviceConfig, timeout time.Duration) (string, error) {
	out, err := t.runShellText(ctx, device, timeout, "dumpsys", "window")
	if err != nil {
		return "", err
	}
	matches := androidWindowPackageRe.FindStringSubmatch(out)
	if len(matches) < 2 {
		return "", nil
	}
	return matches[1], nil
}

func (t *AndroidTool) validatePoint(ctx context.Context, device AndroidDeviceConfig, timeout time.Duration, x, y int) *ToolResult {
	width, height, ok := t.screenSize(ctx, device, timeout)
	if !ok {
		if x < 0 || y < 0 {
			return ErrorResult("coordinates must be non-negative")
		}
		return nil
	}
	if x < 0 || y < 0 || x >= width || y >= height {
		return ErrorResult(fmt.Sprintf("coordinates out of bounds: (%d,%d) outside %dx%d", x, y, width, height))
	}
	return nil
}

func (t *AndroidTool) screenSize(ctx context.Context, device AndroidDeviceConfig, timeout time.Duration) (int, int, bool) {
	out, err := t.runShellText(ctx, device, timeout, "wm", "size")
	if err != nil {
		return 0, 0, false
	}
	matches := androidSizeRe.FindStringSubmatch(out)
	if len(matches) < 3 {
		return 0, 0, false
	}
	width, errW := strconv.Atoi(matches[1])
	height, errH := strconv.Atoi(matches[2])
	if errW != nil || errH != nil || width <= 0 || height <= 0 {
		return 0, 0, false
	}
	return width, height, true
}

func (t *AndroidTool) runShellText(ctx context.Context, device AndroidDeviceConfig, timeout time.Duration, args ...string) (string, error) {
	out, err := t.runShell(ctx, device, timeout, args...)
	return string(out), err
}

func (t *AndroidTool) runShell(ctx context.Context, device AndroidDeviceConfig, timeout time.Duration, args ...string) ([]byte, error) {
	adbArgs := append(device.adbArgs(), "shell")
	adbArgs = append(adbArgs, args...)
	return t.runADB(ctx, timeout, adbArgs...)
}

func (t *AndroidTool) runADB(ctx context.Context, timeout time.Duration, args ...string) ([]byte, error) {
	if timeout <= 0 {
		timeout = t.defaultTimeout
	}
	return t.runner.Run(ctx, timeout, t.adbPath, args...)
}

func (r execAndroidRunner) Run(ctx context.Context, timeout time.Duration, name string, args ...string) ([]byte, error) {
	if timeout <= 0 {
		timeout = time.Duration(defaultAndroidTimeoutMS) * time.Millisecond
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, name, args...)
	out, err := cmd.CombinedOutput()
	if runCtx.Err() == context.DeadlineExceeded {
		return out, fmt.Errorf("timeout after %s", timeout)
	}
	if err != nil {
		return out, fmt.Errorf("%w: %s", err, limitAndroidString(strings.TrimSpace(string(out)), 4096))
	}
	return out, nil
}

func (t *AndroidTool) checkRateLimit() *ToolResult {
	if t.maxActionsPerMinute <= 0 {
		return nil
	}
	now := time.Now()
	cutoff := now.Add(-time.Minute)
	t.mu.Lock()
	defer t.mu.Unlock()
	kept := t.actionTimestamps[:0]
	for _, ts := range t.actionTimestamps {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	t.actionTimestamps = kept
	if len(t.actionTimestamps) >= t.maxActionsPerMinute {
		return ErrorResult(fmt.Sprintf("android action rate limit exceeded: max %d per minute", t.maxActionsPerMinute))
	}
	t.actionTimestamps = append(t.actionTimestamps, now)
	return nil
}

func (t *AndroidTool) audit(action string, device AndroidDeviceConfig, details map[string]any) {
	fields := map[string]any{"action": action, "device_id": device.ID, "serial": device.Serial}
	for k, v := range details {
		fields[k] = v
	}
	logger.InfoCF("android", "Android action executed", fields)
}

func (d AndroidDeviceConfig) adbArgs() []string {
	serial := strings.TrimSpace(d.Serial)
	if serial == "" || strings.EqualFold(serial, "usb") {
		return nil
	}
	return []string{"-s", serial}
}

func (d AndroidDeviceConfig) effectiveBlockPackages() []string {
	out := append([]string{}, d.BlockPackages...)
	out = append(out,
		"com.google.android.apps.walletnfcrel",
		"com.android.vending",
		"*bank*",
		"*wallet*",
		"*paypal*",
	)
	return out
}

func normalizeAndroidDevices(devices []AndroidDeviceConfig) []AndroidDeviceConfig {
	out := make([]AndroidDeviceConfig, 0, len(devices))
	for _, d := range devices {
		d.ID = strings.TrimSpace(d.ID)
		d.Serial = strings.TrimSpace(d.Serial)
		d.AllowPackages = trimAndroidStrings(d.AllowPackages)
		d.BlockPackages = trimAndroidStrings(d.BlockPackages)
		if d.ID == "" {
			continue
		}
		out = append(out, d)
	}
	return out
}

func parseADBDevices(raw string) []map[string]string {
	var devices []map[string]string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "List of devices") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		devices = append(devices, map[string]string{"serial": fields[0], "state": fields[1], "raw": line})
	}
	return devices
}

func summarizeAndroidUIXML(raw string) []map[string]string {
	var nodes []map[string]string
	matches := androidXMLNodeRe.FindAllStringSubmatch(raw, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		attrs := map[string]string{}
		for _, attr := range androidXMLAttrRe.FindAllStringSubmatch(match[1], -1) {
			if len(attr) == 3 {
				attrs[attr[1]] = attr[2]
			}
		}
		if attrs["text"] == "" && attrs["resource-id"] == "" && attrs["content-desc"] == "" && attrs["clickable"] != "true" {
			continue
		}
		node := map[string]string{}
		for _, key := range []string{"text", "content-desc", "resource-id", "class", "package", "bounds", "clickable", "enabled"} {
			if attrs[key] != "" {
				node[key] = limitAndroidString(attrs[key], 300)
			}
		}
		nodes = append(nodes, node)
		if len(nodes) >= maxAndroidUITreeNodes {
			break
		}
	}
	return nodes
}

func matchAndroidPackage(pattern, pkg string) bool {
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	pkg = strings.ToLower(strings.TrimSpace(pkg))
	if pattern == "" || pkg == "" {
		return false
	}
	if pattern == "*" || pattern == pkg {
		return true
	}
	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") && len(pattern) > 2 {
		return strings.Contains(pkg, strings.Trim(pattern, "*"))
	}
	if strings.HasPrefix(pattern, "*") {
		return strings.HasSuffix(pkg, strings.TrimPrefix(pattern, "*"))
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(pkg, strings.TrimSuffix(pattern, "*"))
	}
	return false
}

func timeoutFromArgs(args map[string]any, fallback time.Duration) time.Duration {
	if v, ok := intArg(args, "timeout_ms"); ok && v > 0 {
		return time.Duration(v) * time.Millisecond
	}
	return fallback
}

func intArg(args map[string]any, name string) (int, bool) {
	v, ok := args[name]
	if !ok {
		return 0, false
	}
	switch typed := v.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	case json.Number:
		n, err := typed.Int64()
		return int(n), err == nil
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(typed))
		return n, err == nil
	default:
		return 0, false
	}
}

func stringArg(args map[string]any, name string) string {
	if v, ok := args[name].(string); ok {
		return v
	}
	return ""
}

func trimAndroidStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func splitAndroidCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return trimAndroidStrings(strings.Split(value, ","))
}

func parseAndroidBoolEnv(name string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseAndroidIntEnv(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func limitAndroidString(value string, max int) string {
	if max <= 0 || len(value) <= max {
		return value
	}
	return value[:max] + "..."
}
