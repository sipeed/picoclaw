package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/health"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/netbind"
	ppid "github.com/sipeed/picoclaw/pkg/pid"
	"github.com/sipeed/picoclaw/web/backend/utils"
)

// gateway holds the state for the managed gateway process.
var gateway = struct {
	mu                  sync.Mutex
	cmd                 *exec.Cmd
	owned               bool // true if we started the process, false if we attached to an existing one
	bootDefaultModel    string
	bootConfigSignature string
	runtimeStatus       string
	startupDeadline     time.Time
	logs                *LogBuffer
	pidData             *ppid.PidFileData // pid file data read from picoclaw.pid.json
	picoToken           string            // cached raw pico token for upstream gateway proxy injection
}{
	runtimeStatus: "stopped",
	logs:          NewLogBuffer(200),
}

// refreshPicoTokensLocked reads the pico token from config and caches it.
// Caller must hold gateway.mu (or be sole writer).
func refreshPicoTokensLocked(configPath string) {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return
	}
	var picoCfg config.PicoSettings
	if bc := cfg.Channels.GetByType(config.ChannelPico); bc != nil {
		decoded, err := bc.GetDecoded()
		if err == nil && decoded != nil {
			if p, ok := decoded.(*config.PicoSettings); ok {
				picoCfg = *p
			}
		}
	}
	gateway.picoToken = picoCfg.Token.String()
}

// ensurePicoTokenCachedLocked lazily fills the in-memory pico token cache when
// the launcher has already discovered a running gateway via pidData, but has
// not yet refreshed the token into memory.
func ensurePicoTokenCachedLocked(configPath string) {
	if gateway.picoToken != "" {
		return
	}
	refreshPicoTokensLocked(configPath)
}

func (h *Handler) gatewayCommandArgs() []string {
	args := []string{"gateway", "-E"}
	if h.debug {
		args = append(args, "-d")
	}
	return args
}

const (
	protocolKey = "Sec-Websocket-Protocol"
	tokenPrefix = "token."
)

// picoGatewayProtocol returns the gateway-facing pico subprotocol that the
// launcher should inject when proxying browser traffic upstream.
func picoGatewayProtocol() string {
	gateway.mu.Lock()
	defer gateway.mu.Unlock()
	if gateway.picoToken == "" {
		return ""
	}
	return tokenPrefix + gateway.picoToken
}

var (
	gatewayStartupWindow          = 15 * time.Second
	gatewayRestartGracePeriod     = 5 * time.Second
	gatewayRestartForceKillWindow = 3 * time.Second
	gatewayRestartPollInterval    = 100 * time.Millisecond
	gatewayExecCommand            = exec.Command
)

var gatewayHealthGet = func(url string, timeout time.Duration) (*http.Response, error) {
	client := http.Client{Timeout: timeout}
	return client.Get(url)
}

var gatewayProcessMatcher = isLikelyGatewayProcess

// getGatewayHealth checks the gateway health endpoint and returns the status response.
// Returns (*health.StatusResponse, statusCode, error). If error is not nil, the other values are not valid.
func (h *Handler) getGatewayHealth(cfg *config.Config, timeout time.Duration) (*health.StatusResponse, int, error) {
	// Prefer port/host from pidData when available.
	var port int
	var host string
	gateway.mu.Lock()
	if d := gateway.pidData; d != nil && d.Port > 0 {
		port = d.Port
		host = gatewayProbeHost(d.Host)
	}
	gateway.mu.Unlock()
	if port == 0 {
		port = 18790
		if cfg != nil && cfg.Gateway.Port != 0 {
			port = cfg.Gateway.Port
		}
	}
	if host == "" {
		host = gatewayProbeHost(h.effectiveGatewayBindHost(cfg))
	}

	url := "http://" + net.JoinHostPort(host, strconv.Itoa(port)) + "/health"

	return getGatewayHealthByURL(url, timeout)
}

func getGatewayHealthByURL(url string, timeout time.Duration) (*health.StatusResponse, int, error) {
	resp, err := gatewayHealthGet(url, timeout)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	var healthResponse health.StatusResponse
	if decErr := json.NewDecoder(resp.Body).Decode(&healthResponse); decErr != nil {
		return nil, resp.StatusCode, decErr
	}

	return &healthResponse, resp.StatusCode, nil
}

// isLikelyGatewayProcess returns whether PID appears to be a picoclaw gateway
// process plus whether inspection was conclusive on this platform/environment.
func isLikelyGatewayProcess(pid int) (bool, bool) {
	if pid <= 0 {
		return false, true
	}

	if runtime.GOOS == "windows" {
		psCmd := fmt.Sprintf(
			`$p=Get-CimInstance Win32_Process -Filter "ProcessId = %d"; if ($null -eq $p) { "" } else { $p.CommandLine }`,
			pid,
		)
		out, err := launcherExecCommand("powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).Output()
		if err == nil {
			cmdline := strings.TrimSpace(string(out))
			if cmdline != "" {
				return looksLikeGatewayCommandLine(cmdline), true
			}
		}

		// Fallback: determine only whether the process still exists.
		out, err = launcherExecCommand("tasklist", "/FI", "PID eq "+strconv.Itoa(pid), "/FO", "CSV", "/NH").Output()
		if err != nil {
			return false, false
		}
		line := strings.ToLower(strings.TrimSpace(string(out)))
		if line == "" {
			return false, true
		}
		// A CSV row means the process exists, but may have a custom executable
		// name we cannot classify here.
		if strings.HasPrefix(line, "\"") {
			if strings.Contains(line, "\"picoclaw.exe\"") {
				return true, true
			}
			return false, true
		}
		if strings.Contains(line, "no tasks are running") {
			return false, true
		}
		return false, true
	}

	out, err := launcherExecCommand("ps", "-o", "command=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return false, false
	}
	cmdline := strings.ToLower(strings.TrimSpace(string(out)))
	if cmdline == "" {
		return false, true
	}
	return looksLikeGatewayCommandLine(cmdline), true
}

// looksLikeGatewayCommandLine checks whether a process command line likely
// represents "picoclaw gateway ..." regardless of executable filename.
func looksLikeGatewayCommandLine(cmdline string) bool {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(cmdline)))
	if len(fields) == 0 {
		return false
	}
	for _, f := range fields {
		token := strings.Trim(f, `"'`)
		if token == "gateway" || strings.HasSuffix(token, "/gateway") || strings.HasSuffix(token, `\gateway`) {
			return true
		}
	}
	return false
}

func (h *Handler) getGatewayHealthForPidData(
	pidData *ppid.PidFileData,
	cfg *config.Config,
	timeout time.Duration,
) (*health.StatusResponse, int, error) {
	if pidData == nil {
		return nil, 0, errors.New("nil pid data")
	}

	port := pidData.Port
	if port == 0 {
		port = 18790
		if cfg != nil && cfg.Gateway.Port != 0 {
			port = cfg.Gateway.Port
		}
	}

	host := gatewayProbeHost(strings.TrimSpace(pidData.Host))
	if host == "" {
		host = gatewayProbeHost(h.effectiveGatewayBindHost(cfg))
	}
	if host == "" {
		host = netbind.ResolveAdaptiveLoopbackHost()
	}

	url := "http://" + net.JoinHostPort(host, strconv.Itoa(port)) + "/health"
	return getGatewayHealthByURL(url, timeout)
}

func (h *Handler) validateGatewayPidData(
	pidData *ppid.PidFileData,
	cfg *config.Config,
) (ok bool, decisive bool, reason string) {
	if pidData == nil || pidData.PID <= 0 {
		return false, true, "invalid pid data"
	}

	if gatewayProcess, inspected := gatewayProcessMatcher(pidData.PID); inspected {
		if !gatewayProcess {
			return false, true, "pid process command is not picoclaw gateway"
		}
		return true, true, ""
	}

	healthResp, statusCode, err := h.getGatewayHealthForPidData(pidData, cfg, 800*time.Millisecond)
	if err != nil {
		return false, false, fmt.Sprintf("health probe failed: %v", err)
	}
	if statusCode != http.StatusOK {
		return false, false, fmt.Sprintf("health endpoint returned status %d", statusCode)
	}
	if healthResp.PID > 0 && healthResp.PID != pidData.PID {
		return false, true, fmt.Sprintf("health pid mismatch: pidFile=%d, health=%d", pidData.PID, healthResp.PID)
	}
	return true, true, ""
}

func (h *Handler) sanitizeGatewayPidData(pidData *ppid.PidFileData, cfg *config.Config) *ppid.PidFileData {
	if pidData == nil {
		return nil
	}

	ok, decisive, reason := h.validateGatewayPidData(pidData, cfg)
	if ok {
		return pidData
	}

	logger.Warnf("ignore pid file for PID %d: %s", pidData.PID, reason)
	if decisive && ppid.RemovePidFileIfPID(globalConfigDir(), pidData.PID) {
		logger.Warnf("removed stale pid file for PID %d", pidData.PID)
	}
	return nil
}

// registerGatewayRoutes binds gateway lifecycle endpoints to the ServeMux.
func (h *Handler) registerGatewayRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/gateway/status", h.handleGatewayStatus)
	mux.HandleFunc("GET /api/gateway/logs", h.handleGatewayLogs)
	mux.HandleFunc("POST /api/gateway/logs/clear", h.handleGatewayClearLogs)
	mux.HandleFunc("POST /api/gateway/start", h.handleGatewayStart)
	mux.HandleFunc("POST /api/gateway/stop", h.handleGatewayStop)
	mux.HandleFunc("POST /api/gateway/restart", h.handleGatewayRestart)
}

// TryAutoStartGateway checks whether gateway start preconditions are met and
// starts it when possible. Intended to be called by the backend at startup.
func (h *Handler) TryAutoStartGateway() {
	// Check PID file first to detect an already-running gateway.
	pidData := h.sanitizeGatewayPidData(ppid.ReadPidFileWithCheck(globalConfigDir()), nil)
	if pidData != nil {
		gateway.mu.Lock()
		ready, reason, err := h.gatewayStartReady()
		if err != nil {
			logger.ErrorC("gateway", fmt.Sprintf("Skip auto-starting gateway: %v", err))
			gateway.mu.Unlock()
			return
		}
		logger.Infof("ready: %v, reason: %s", ready, reason)
		if !ready {
			logger.InfoC("gateway", fmt.Sprintf("Skip auto-starting gateway: %s", reason))
			gateway.mu.Unlock()
			return
		}
		pid := pidData.PID
		_, err = h.startGatewayLocked("starting", pid)
		if err != nil {
			logger.ErrorC("gateway", fmt.Sprintf("Failed to attach to running gateway (PID: %d): %v", pid, err))
		} else {
			gateway.pidData = pidData
			refreshPicoTokensLocked(h.configPath)
			logger.InfoC("gateway", fmt.Sprintf("Attached to running gateway via PID file (PID: %d)", pid))
		}
		gateway.mu.Unlock()
		return
	}

	gateway.mu.Lock()
	defer gateway.mu.Unlock()

	if gateway.cmd != nil && gateway.cmd.Process != nil {
		gateway.cmd = nil
	}

	ready, reason, err := h.gatewayStartReady()
	if err != nil {
		logger.ErrorC("gateway", fmt.Sprintf("Skip auto-starting gateway: %v", err))
		return
	}
	if !ready {
		logger.InfoC("gateway", fmt.Sprintf("Skip auto-starting gateway: %s", reason))
		return
	}

	pid, err := h.startGatewayLocked("starting", 0)
	if err != nil {
		logger.ErrorC("gateway", fmt.Sprintf("Failed to auto-start gateway: %v", err))
		return
	}
	logger.InfoC("gateway", fmt.Sprintf("Gateway auto-started (PID: %d)", pid))
}

// gatewayStartReady validates whether current config can start the gateway.
func (h *Handler) gatewayStartReady() (bool, string, error) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		return false, "", fmt.Errorf("failed to load config: %w", err)
	}

	modelName := strings.TrimSpace(cfg.Agents.Defaults.GetModelName())
	if modelName == "" {
		return false, "no default model configured", nil
	}

	modelCfg := lookupModelConfig(cfg, modelName)
	if modelCfg == nil {
		return false, fmt.Sprintf("default model %q is invalid", modelName), nil
	}
	if !defaultModelAllowedForModelConfig(modelCfg) {
		return false, fmt.Sprintf("default model %q is not usable for chat", modelName), nil
	}

	if !hasModelConfiguration(modelCfg) {
		return false, fmt.Sprintf("default model %q has no credentials configured", modelName), nil
	}
	if requiresRuntimeProbe(modelCfg) && !probeLocalModelAvailability(modelCfg) {
		return false, fmt.Sprintf("default model %q is not reachable", modelName), nil
	}

	return true, "", nil
}

func lookupModelConfig(cfg *config.Config, modelName string) *config.ModelConfig {
	modelCfg, err := cfg.GetModelConfig(modelName)
	if err != nil {
		return nil
	}
	return modelCfg
}

func computeConfigSignature(cfg *config.Config) string {
	if cfg == nil {
		return ""
	}
	var parts []string
	defaultModel := strings.TrimSpace(cfg.Agents.Defaults.GetModelName())
	if defaultModel != "" {
		parts = append(parts, "model:"+defaultModel)
	}
	toolSignatures := []string{}
	if cfg.Tools.ReadFile.Enabled {
		toolSignatures = append(toolSignatures, "read_file")
	}
	if cfg.Tools.WriteFile.Enabled {
		toolSignatures = append(toolSignatures, "write_file")
	}
	if cfg.Tools.ListDir.Enabled {
		toolSignatures = append(toolSignatures, "list_dir")
	}
	if cfg.Tools.EditFile.Enabled {
		toolSignatures = append(toolSignatures, "edit_file")
	}
	if cfg.Tools.AppendFile.Enabled {
		toolSignatures = append(toolSignatures, "append_file")
	}
	if cfg.Tools.Exec.Enabled {
		toolSignatures = append(toolSignatures, "exec")
	}
	if cfg.Tools.Cron.Enabled {
		toolSignatures = append(toolSignatures, "cron")
	}
	if cfg.Tools.Web.Enabled {
		toolSignatures = append(toolSignatures, "web")
		webConfig, err := json.Marshal(canonicalizeSignatureValue(reflect.ValueOf(cfg.Tools.Web)))
		if err == nil {
			parts = append(parts, "webcfg:"+string(webConfig))
		}
	}
	if cfg.Tools.WebFetch.Enabled {
		toolSignatures = append(toolSignatures, "web_fetch")
	}
	if cfg.Tools.Message.Enabled {
		toolSignatures = append(toolSignatures, "message")
	}
	if cfg.Tools.SendFile.Enabled {
		toolSignatures = append(toolSignatures, "send_file")
	}
	if cfg.Tools.FindSkills.Enabled {
		toolSignatures = append(toolSignatures, "find_skills")
	}
	if cfg.Tools.InstallSkill.Enabled {
		toolSignatures = append(toolSignatures, "install_skill")
	}
	if cfg.Tools.Spawn.Enabled {
		toolSignatures = append(toolSignatures, "spawn")
	}
	if cfg.Tools.SpawnStatus.Enabled {
		toolSignatures = append(toolSignatures, "spawn_status")
	}
	if cfg.Tools.I2C.Enabled {
		toolSignatures = append(toolSignatures, "i2c")
	}
	if cfg.Tools.SPI.Enabled {
		toolSignatures = append(toolSignatures, "spi")
	}
	if cfg.Tools.GetCurrentTime.Enabled {
		toolSignatures = append(toolSignatures, "get_current_time")
	}
	if cfg.Tools.MCP.Enabled {
		toolSignatures = append(toolSignatures, "mcp")
	}
	if cfg.Tools.MCP.Discovery.Enabled {
		toolSignatures = append(toolSignatures, "mcp_discovery")
	}
	if cfg.Tools.MCP.Discovery.UseRegex {
		toolSignatures = append(toolSignatures, "mcp_discovery_regex")
	}
	if cfg.Tools.MCP.Discovery.UseBM25 {
		toolSignatures = append(toolSignatures, "mcp_discovery_bm25")
	}
	if len(toolSignatures) > 0 {
		parts = append(parts, "tools:"+strings.Join(toolSignatures, ","))
	}
	channelSignatures := computeChannelSignatures(cfg.Channels)
	if len(channelSignatures) > 0 {
		parts = append(parts, "channels:"+strings.Join(channelSignatures, ","))
	}
	return strings.Join(parts, ";")
}

func computeChannelSignatures(channels config.ChannelsConfig) []string {
	if len(channels) == 0 {
		return nil
	}

	keys := make([]string, 0, len(channels))
	for name := range channels {
		keys = append(keys, name)
	}
	sort.Strings(keys)

	signatures := make([]string, 0, len(keys))
	for _, name := range keys {
		channel := channels[name]
		if channel == nil {
			signatures = append(signatures, name+":<nil>")
			continue
		}

		payload := struct {
			Enabled            bool                       `json:"enabled"`
			Type               string                     `json:"type"`
			AllowFrom          config.FlexibleStringSlice `json:"allow_from,omitempty"`
			ReasoningChannelID string                     `json:"reasoning_channel_id,omitempty"`
			GroupTrigger       config.GroupTriggerConfig  `json:"group_trigger,omitempty"`
			Typing             config.TypingConfig        `json:"typing,omitempty"`
			Placeholder        config.PlaceholderConfig   `json:"placeholder,omitempty"`
			Settings           json.RawMessage            `json:"settings,omitempty"`
		}{
			Enabled:            channel.Enabled,
			Type:               channel.Type,
			AllowFrom:          channel.AllowFrom,
			ReasoningChannelID: channel.ReasoningChannelID,
			GroupTrigger:       channel.GroupTrigger,
			Typing:             channel.Typing,
			Placeholder:        channel.Placeholder,
			Settings:           normalizeChannelSettings(channel),
		}

		encoded, err := json.Marshal(payload)
		if err != nil {
			signatures = append(signatures, name+":<invalid>")
			continue
		}
		signatures = append(signatures, name+":"+string(encoded))
	}

	return signatures
}

func normalizeChannelSettings(channel *config.Channel) json.RawMessage {
	if channel == nil {
		return nil
	}

	decoded, err := channel.GetDecoded()
	if err == nil && decoded != nil {
		normalized, err := json.Marshal(canonicalizeSignatureValue(reflect.ValueOf(decoded)))
		if err == nil {
			return normalized
		}
	}

	return normalizeRawJSON(channel.Settings)
}

func normalizeRawJSON(raw config.RawNode) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}

	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return bytes.TrimSpace(raw)
	}

	normalized, err := json.Marshal(value)
	if err != nil {
		return bytes.TrimSpace(raw)
	}
	return normalized
}

func canonicalizeSignatureValue(value reflect.Value) any {
	if !value.IsValid() {
		return nil
	}

	if value.CanInterface() {
		switch typed := value.Interface().(type) {
		case config.SecureString:
			return typed.String()
		case *config.SecureString:
			if typed == nil {
				return ""
			}
			return typed.String()
		case config.SecureStrings:
			return typed.Values()
		case *config.SecureStrings:
			if typed == nil {
				return nil
			}
			return typed.Values()
		}
	}

	switch value.Kind() {
	case reflect.Interface, reflect.Pointer:
		if value.IsNil() {
			return nil
		}
		return canonicalizeSignatureValue(value.Elem())
	case reflect.Struct:
		result := make(map[string]any)
		valueType := value.Type()
		for i := 0; i < value.NumField(); i++ {
			field := valueType.Field(i)
			if field.PkgPath != "" {
				continue
			}
			tag := field.Tag.Get("json")
			name := field.Name
			if tag != "" {
				if comma := strings.Index(tag, ","); comma >= 0 {
					tag = tag[:comma]
				}
				if tag == "-" {
					continue
				}
				if tag != "" {
					name = tag
				}
			}
			result[name] = canonicalizeSignatureValue(value.Field(i))
		}
		return result
	case reflect.Slice, reflect.Array:
		length := value.Len()
		result := make([]any, 0, length)
		for i := 0; i < length; i++ {
			result = append(result, canonicalizeSignatureValue(value.Index(i)))
		}
		return result
	case reflect.Map:
		if value.Type().Key().Kind() != reflect.String {
			return value.Interface()
		}
		result := make(map[string]any, value.Len())
		iter := value.MapRange()
		for iter.Next() {
			result[iter.Key().String()] = canonicalizeSignatureValue(iter.Value())
		}
		return result
	default:
		if value.CanInterface() {
			return value.Interface()
		}
		return nil
	}
}

func gatewayRestartRequiredBySignature(bootSignature, currentSignature, gatewayStatus string) bool {
	if gatewayStatus != "running" {
		return false
	}
	if bootSignature == "" || currentSignature == "" {
		return false
	}
	return bootSignature != currentSignature
}

func isCmdProcessAliveLocked(cmd *exec.Cmd) bool {
	if cmd == nil || cmd.Process == nil {
		return false
	}

	// Wait() sets ProcessState when the process exits; use it when available.
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		return false
	}

	// Windows does not support Signal(0) probing. If we still own cmd and it
	// has not reported exit, treat it as alive.
	if runtime.GOOS == "windows" {
		return true
	}

	err := cmd.Process.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}
	var errno syscall.Errno
	// EPERM means the process exists but cannot be signaled by this user.
	return errors.As(err, &errno) && errno == syscall.EPERM
}

func setGatewayRuntimeStatusLocked(status string) {
	gateway.runtimeStatus = status
	if status == "starting" || status == "restarting" {
		gateway.startupDeadline = time.Now().Add(gatewayStartupWindow)
		return
	}
	gateway.startupDeadline = time.Time{}
}

// attachToGatewayProcess attaches to an existing gateway process by PID
// and updates the gateway state accordingly.
// Assumes gateway.mu is held by the caller.
func attachToGatewayProcessLocked(pid int, cfg *config.Config) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process for PID %d: %w", pid, err)
	}

	gateway.cmd = &exec.Cmd{Process: process}
	gateway.owned = false // We didn't start this process
	setGatewayRuntimeStatusLocked("running")

	// Update bootDefaultModel and bootConfigSignature from config
	if cfg != nil {
		defaultModelName := strings.TrimSpace(cfg.Agents.Defaults.GetModelName())
		gateway.bootDefaultModel = defaultModelName
		gateway.bootConfigSignature = computeConfigSignature(cfg)
	}

	logger.InfoC("gateway", fmt.Sprintf("Attached to gateway process (PID: %d)", pid))
	return nil
}

func gatewayStatusWithoutHealthLocked() string {
	if gateway.runtimeStatus == "starting" || gateway.runtimeStatus == "restarting" {
		if gateway.startupDeadline.IsZero() || time.Now().Before(gateway.startupDeadline) {
			return gateway.runtimeStatus
		}
		return "error"
	}
	if gateway.runtimeStatus == "running" {
		// For attached processes there is no waiter goroutine; degrade stale
		// running state once the tracked process exits.
		if !isCmdProcessAliveLocked(gateway.cmd) {
			gateway.cmd = nil
			gateway.owned = false
			gateway.bootDefaultModel = ""
			gateway.bootConfigSignature = ""
			return "stopped"
		}
		return "running"
	}
	if gateway.runtimeStatus == "error" {
		return "error"
	}
	return "stopped"
}

func waitForGatewayProcessExit(cmd *exec.Cmd, timeout time.Duration) bool {
	if cmd == nil || cmd.Process == nil {
		return true
	}

	deadline := time.Now().Add(timeout)
	for {
		if !isCmdProcessAliveLocked(cmd) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(gatewayRestartPollInterval)
	}
}

// StopGateway stops the gateway process if it was started by this handler.
// This method is called during application shutdown to ensure the gateway subprocess
// is properly terminated. It only stops processes that were started by this handler,
// not processes that were attached to from existing instances.
func (h *Handler) StopGateway() {
	gateway.mu.Lock()
	defer gateway.mu.Unlock()

	// Only stop if we own the process (started it ourselves)
	if !gateway.owned || gateway.cmd == nil || gateway.cmd.Process == nil {
		return
	}

	pid, err := stopGatewayLocked()
	if err != nil {
		logger.ErrorC("gateway", fmt.Sprintf("Failed to stop gateway (PID %d): %v", pid, err))
		return
	}

	logger.InfoC("gateway", fmt.Sprintf("Gateway stopped (PID: %d)", pid))
}

// stopGatewayLocked sends a stop signal to the gateway process.
// Assumes gateway.mu is held by the caller.
// Returns the PID of the stopped process and any error encountered.
func stopGatewayLocked() (int, error) {
	if gateway.cmd == nil || gateway.cmd.Process == nil {
		return 0, nil
	}

	pid := gateway.cmd.Process.Pid
	if !gateway.owned {
		if isGateway, inspected := gatewayProcessMatcher(pid); inspected && !isGateway {
			return pid, fmt.Errorf("refuse to stop non-gateway process (PID %d)", pid)
		}
	}

	// Send SIGTERM for graceful shutdown (SIGKILL on Windows)
	var sigErr error
	if runtime.GOOS == "windows" {
		sigErr = gateway.cmd.Process.Kill()
	} else {
		sigErr = gateway.cmd.Process.Signal(syscall.SIGTERM)
	}

	if sigErr != nil {
		return pid, sigErr
	}

	logger.InfoC("gateway", fmt.Sprintf("Sent stop signal to gateway (PID: %d)", pid))
	gateway.cmd = nil
	gateway.owned = false
	gateway.bootDefaultModel = ""
	gateway.pidData = nil
	setGatewayRuntimeStatusLocked("stopped")

	return pid, nil
}

func stopGatewayProcessForRestart(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil || !isCmdProcessAliveLocked(cmd) {
		return nil
	}

	var stopErr error
	if runtime.GOOS == "windows" {
		stopErr = cmd.Process.Kill()
	} else {
		stopErr = cmd.Process.Signal(syscall.SIGTERM)
	}
	if stopErr != nil && isCmdProcessAliveLocked(cmd) {
		return fmt.Errorf("failed to stop existing gateway: %w", stopErr)
	}

	if waitForGatewayProcessExit(cmd, gatewayRestartGracePeriod) {
		return nil
	}

	if runtime.GOOS != "windows" {
		killErr := cmd.Process.Signal(syscall.SIGKILL)
		if killErr != nil && isCmdProcessAliveLocked(cmd) {
			return fmt.Errorf("failed to force-stop existing gateway: %w", killErr)
		}
		if waitForGatewayProcessExit(cmd, gatewayRestartForceKillWindow) {
			return nil
		}
	}

	return fmt.Errorf("existing gateway did not exit before restart")
}

func (h *Handler) startGatewayLocked(initialStatus string, existingPid int) (int, error) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		return 0, fmt.Errorf("failed to load config: %w", err)
	}
	defaultModelName := strings.TrimSpace(cfg.Agents.Defaults.GetModelName())

	var cmd *exec.Cmd
	var pid int

	if existingPid > 0 {
		// Attach to existing process
		pid = existingPid
		gateway.cmd = nil // Clear first to ensure clean state
		if err = attachToGatewayProcessLocked(pid, cfg); err != nil {
			logger.ErrorC("gateway", fmt.Sprintf("Failed to attach to existing gateway (PID %d): %v", pid, err))
			return 0, err
		}

		return pid, nil
	}

	// Start new process
	// Locate the picoclaw executable
	execPath := utils.FindPicoclawBinary()
	logger.InfoC("gateway", fmt.Sprintf("Starting gateway process (%s)", execPath))

	cmd = gatewayExecCommand(execPath, h.gatewayCommandArgs()...)
	applyLauncherProcAttrs(cmd)
	cmd.Env = os.Environ()
	// Forward the launcher's config path via the environment variable that
	// GetConfigPath() already reads, so the gateway sub-process uses the same
	// config file without requiring a --config flag on the gateway subcommand.
	if h.configPath != "" {
		cmd.Env = append(cmd.Env, config.EnvConfig+"="+h.configPath)
	}
	gatewayHostOverride := h.gatewayHostOverride()
	if gatewayHostOverride != "" {
		cmd.Env = append(cmd.Env, config.EnvGatewayHost+"="+gatewayHostOverride)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return 0, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return 0, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Clear old logs for this new run
	gateway.logs.Reset()

	// Ensure Pico Channel is configured before starting gateway
	changed, err := h.EnsurePicoChannel()
	if err != nil {
		logger.ErrorC("gateway", fmt.Sprintf("Warning: failed to ensure pico channel: %v", err))
		// Non-fatal: gateway can still start without pico channel
	}
	// Refresh cached pico token in case EnsurePicoChannel generated a new one.
	// Already holding gateway.mu from caller.
	if changed {
		refreshPicoTokensLocked(h.configPath)
		cfg, err = config.LoadConfig(h.configPath)
		if err != nil {
			return 0, fmt.Errorf("failed to reload config after ensuring pico channel: %w", err)
		}
		defaultModelName = strings.TrimSpace(cfg.Agents.Defaults.GetModelName())
	}

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start gateway: %w", err)
	}

	gateway.cmd = cmd
	gateway.owned = true // We started this process
	gateway.bootDefaultModel = defaultModelName
	gateway.bootConfigSignature = computeConfigSignature(cfg)
	setGatewayRuntimeStatusLocked(initialStatus)
	pid = cmd.Process.Pid
	logger.InfoC("gateway", fmt.Sprintf("Started picoclaw gateway (PID: %d) from %s", pid, execPath))

	// Capture stdout/stderr in background
	go scanPipe(stdoutPipe, gateway.logs)
	go scanPipe(stderrPipe, gateway.logs)

	// Wait for exit in background and clean up
	go func() {
		if err := cmd.Wait(); err != nil {
			logger.ErrorC("gateway", fmt.Sprintf("Gateway process exited: %v", err))
		} else {
			logger.InfoC("gateway", "Gateway process exited normally")
		}

		gateway.mu.Lock()
		if gateway.cmd == cmd {
			gateway.cmd = nil
			gateway.bootDefaultModel = ""
			gateway.bootConfigSignature = ""
			if gateway.runtimeStatus != "restarting" {
				setGatewayRuntimeStatusLocked("stopped")
			}
		}
		gateway.mu.Unlock()
	}()

	// Start a goroutine to probe pidFile and health, update runtime state once ready.
	go func() {
		healthConfirmed := false
		for i := 0; i < 30; i++ { // try for up to 15 seconds
			time.Sleep(500 * time.Millisecond)
			gateway.mu.Lock()
			stillOurs := gateway.cmd == cmd
			gateway.mu.Unlock()
			if !stillOurs {
				return
			}

			// Poll for pidFile first — once available we have port/host/token.
			if pd := ppid.ReadPidFileWithCheck(globalConfigDir()); pd != nil && pd.PID == pid {
				gateway.mu.Lock()
				if gateway.cmd == cmd {
					gateway.pidData = pd
					var picoCfg config.PicoSettings
					if bc := cfg.Channels.GetByType(config.ChannelPico); bc != nil {
						decoded, err := bc.GetDecoded()
						if err == nil && decoded != nil {
							if p, ok := decoded.(*config.PicoSettings); ok {
								picoCfg = *p
							}
						}
					}
					gateway.picoToken = picoCfg.Token.String()
					setGatewayRuntimeStatusLocked("running")
				}
				gateway.mu.Unlock()
				logger.InfoC("gateway", fmt.Sprintf("Gateway pidFile detected (PID: %d, port: %d)", pd.PID, pd.Port))
				return
			}

			// Fallback: probe health endpoint to confirm liveness.
			cfg, err := config.LoadConfig(h.configPath)
			if err != nil {
				continue
			}
			_, statusCode, err := h.getGatewayHealth(cfg, 1*time.Second)
			if err == nil && statusCode == http.StatusOK {
				gateway.mu.Lock()
				if gateway.cmd == cmd {
					setGatewayRuntimeStatusLocked("running")
				}
				gateway.mu.Unlock()
				if !healthConfirmed {
					healthConfirmed = true
					logger.InfoC("gateway", "Gateway health endpoint reachable; waiting for pid file")
				}
				continue
			}
		}
	}()

	return pid, nil
}

// handleGatewayStart starts the picoclaw gateway subprocess.
//
//	POST /api/gateway/start
func (h *Handler) handleGatewayStart(w http.ResponseWriter, r *http.Request) {
	// Check PID file first to detect an already-running gateway.
	pidData := h.sanitizeGatewayPidData(ppid.ReadPidFileWithCheck(globalConfigDir()), nil)
	if pidData != nil {
		pid := pidData.PID
		gateway.mu.Lock()
		ready, reason, err := h.gatewayStartReady()
		if err != nil {
			gateway.mu.Unlock()
			http.Error(
				w,
				fmt.Sprintf("Failed to validate gateway start conditions: %v", err),
				http.StatusInternalServerError,
			)
			return
		}
		if !ready {
			gateway.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"status":  "precondition_failed",
				"message": reason,
			})
			return
		}
		_, err = h.startGatewayLocked("starting", pid)
		if err != nil {
			gateway.mu.Unlock()
			logger.ErrorC("gateway", fmt.Sprintf("Failed to attach to running gateway (PID: %d): %v", pid, err))
			http.Error(w, fmt.Sprintf("Failed to attach to gateway: %v", err), http.StatusInternalServerError)
			return
		}
		gateway.pidData = pidData
		gateway.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"pid":    pid,
		})
		return
	}

	gateway.mu.Lock()
	defer gateway.mu.Unlock()

	if gateway.cmd != nil && gateway.cmd.Process != nil {
		gateway.cmd = nil
		setGatewayRuntimeStatusLocked("stopped")
	}

	ready, reason, err := h.gatewayStartReady()
	if err != nil {
		http.Error(
			w,
			fmt.Sprintf("Failed to validate gateway start conditions: %v", err),
			http.StatusInternalServerError,
		)
		return
	}
	if !ready {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "precondition_failed",
			"message": reason,
		})
		return
	}

	pid, err := h.startGatewayLocked("starting", 0)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to start gateway: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status": "ok",
		"pid":    pid,
	})
}

// handleGatewayStop stops the running gateway subprocess gracefully.
// Note: Unlike StopGateway (which only stops self-started processes), this API endpoint
// stops any gateway process, including attached ones. This is intentional for user control.
//
//	POST /api/gateway/stop
func (h *Handler) handleGatewayStop(w http.ResponseWriter, r *http.Request) {
	gateway.mu.Lock()
	defer gateway.mu.Unlock()

	if gateway.cmd == nil || gateway.cmd.Process == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status": "not_running",
		})
		return
	}

	pid, err := stopGatewayLocked()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to stop gateway (PID %d): %v", pid, err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status": "ok",
		"pid":    pid,
	})
}

// RestartGateway restarts the gateway process. This is a non-blocking operation
// that stops the current gateway (if running) and starts a new one.
// Returns the PID of the new gateway process or an error.
func (h *Handler) RestartGateway() (int, error) {
	ready, reason, err := h.gatewayStartReady()
	if err != nil {
		return 0, fmt.Errorf("failed to validate gateway start conditions: %w", err)
	}
	if !ready {
		return 0, &preconditionFailedError{reason: reason}
	}

	gateway.mu.Lock()
	previousCmd := gateway.cmd
	previousOwned := gateway.owned
	setGatewayRuntimeStatusLocked("restarting")
	gateway.mu.Unlock()

	if previousCmd != nil && previousCmd.Process != nil && !previousOwned {
		if isGateway, inspected := gatewayProcessMatcher(previousCmd.Process.Pid); inspected && !isGateway {
			logger.Warnf("refuse restarting non-gateway process (PID: %d)", previousCmd.Process.Pid)
			gateway.mu.Lock()
			if gateway.cmd == previousCmd {
				setGatewayRuntimeStatusLocked("running")
			}
			gateway.mu.Unlock()
			return 0, fmt.Errorf("refuse to restart non-gateway process (PID %d)", previousCmd.Process.Pid)
		}
	}

	if err = stopGatewayProcessForRestart(previousCmd); err != nil {
		gateway.mu.Lock()
		if gateway.cmd == previousCmd {
			if isCmdProcessAliveLocked(previousCmd) {
				setGatewayRuntimeStatusLocked("running")
			} else {
				gateway.cmd = nil
				gateway.bootDefaultModel = ""
				setGatewayRuntimeStatusLocked("error")
			}
		}
		gateway.mu.Unlock()
		return 0, fmt.Errorf("failed to stop gateway: %w", err)
	}

	gateway.mu.Lock()
	if gateway.cmd == previousCmd {
		gateway.cmd = nil
		gateway.bootDefaultModel = ""
	}
	pid, err := h.startGatewayLocked("restarting", 0)
	if err != nil {
		gateway.cmd = nil
		gateway.bootDefaultModel = ""
		setGatewayRuntimeStatusLocked("error")
	}
	gateway.mu.Unlock()
	if err != nil {
		return 0, fmt.Errorf("failed to start gateway: %w", err)
	}

	return pid, nil
}

// preconditionFailedError is returned when gateway restart preconditions are not met
type preconditionFailedError struct {
	reason string
}

func (e *preconditionFailedError) Error() string {
	return e.reason
}

// IsBadRequest returns true if the error should result in a 400 Bad Request status
func (e *preconditionFailedError) IsBadRequest() bool {
	return true
}

// handleGatewayRestart stops the gateway (if running) and starts a new instance.
//
//	POST /api/gateway/restart
func (h *Handler) handleGatewayRestart(w http.ResponseWriter, r *http.Request) {
	pid, err := h.RestartGateway()
	if err != nil {
		// Check if it's a precondition failed error
		var precondErr *preconditionFailedError
		if errors.As(err, &precondErr) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"status":  "precondition_failed",
				"message": precondErr.reason,
			})
			return
		}
		http.Error(w, fmt.Sprintf("Failed to restart gateway: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status": "ok",
		"pid":    pid,
	})
}

// handleGatewayClearLogs clears the in-memory gateway log buffer.
//
//	POST /api/gateway/logs/clear
func (h *Handler) handleGatewayClearLogs(w http.ResponseWriter, r *http.Request) {
	gateway.logs.Clear()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":     "cleared",
		"log_total":  0,
		"log_run_id": gateway.logs.RunID(),
	})
}

// handleGatewayStatus returns the gateway run status and health info.
//
//	GET /api/gateway/status
func (h *Handler) handleGatewayStatus(w http.ResponseWriter, r *http.Request) {
	data := h.gatewayStatusData()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) gatewayStatusData() map[string]any {
	data := map[string]any{}
	var configDefaultModel string
	cfg, cfgErr := config.LoadConfig(h.configPath)
	if cfgErr == nil && cfg != nil {
		configDefaultModel = strings.TrimSpace(cfg.Agents.Defaults.GetModelName())
		if configDefaultModel != "" {
			data["config_default_model"] = configDefaultModel
		}
	}

	// Primary detection: read PID file and check if process is alive.
	pidData := h.sanitizeGatewayPidData(ppid.ReadPidFileWithCheck(globalConfigDir()), cfg)
	if pidData != nil {
		gateway.mu.Lock()
		gateway.pidData = pidData
		if pidData.Version != "" {
			data["gateway_version"] = pidData.Version
		}
		setGatewayRuntimeStatusLocked("running")

		// Attach if we don't already track this PID.
		if gateway.cmd == nil || gateway.cmd.Process == nil || gateway.cmd.Process.Pid != pidData.PID {
			_ = attachToGatewayProcessLocked(pidData.PID, cfg)
		}

		bootDefaultModel := gateway.bootDefaultModel
		if bootDefaultModel != "" {
			data["boot_default_model"] = bootDefaultModel
		}
		data["gateway_status"] = "running"
		data["pid"] = pidData.PID
		gateway.mu.Unlock()
	} else {
		// Intentionally skip health probe here; the startup goroutine
		// (startGatewayLocked) already handles liveness detection via
		// pidFile polling and health fallback.
		gateway.mu.Lock()
		status := gatewayStatusWithoutHealthLocked()
		data["gateway_status"] = status
		// Keep last known pidData while gateway is still in a transient
		// running state; otherwise websocket proxy may lose auth token
		// during short pid-file races.
		if status == "stopped" || status == "error" {
			gateway.pidData = nil
		}
		gateway.mu.Unlock()
	}

	gatewayStatus, _ := data["gateway_status"].(string)
	currentConfigSignature := computeConfigSignature(cfg)
	gateway.mu.Lock()
	bootConfigSignature := gateway.bootConfigSignature
	gateway.mu.Unlock()
	data["gateway_restart_required"] = gatewayRestartRequiredBySignature(
		bootConfigSignature,
		currentConfigSignature,
		gatewayStatus,
	)

	ready, reason, readyErr := h.gatewayStartReady()
	if readyErr != nil {
		data["gateway_start_allowed"] = false
		data["gateway_start_reason"] = readyErr.Error()
	} else {
		data["gateway_start_allowed"] = ready
		if !ready {
			data["gateway_start_reason"] = reason
		}
	}

	return data
}

// handleGatewayLogs returns buffered gateway logs, optionally incrementally.
//
//	GET /api/gateway/logs
func (h *Handler) handleGatewayLogs(w http.ResponseWriter, r *http.Request) {
	data := gatewayLogsData(r)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// gatewayLogsData reads log_offset and log_run_id query params from the request
// and returns incremental log lines.
func gatewayLogsData(r *http.Request) map[string]any {
	data := map[string]any{}
	clientOffset := 0
	clientRunID := -1

	if v := r.URL.Query().Get("log_offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			clientOffset = n
		}
	}

	if v := r.URL.Query().Get("log_run_id"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			clientRunID = n
		}
	}

	runID := gateway.logs.RunID()

	if runID == 0 {
		data["logs"] = []string{}
		data["log_total"] = 0
		data["log_run_id"] = 0
		return data
	}

	// If runID changed, reset offset to get all logs from new run
	offset := clientOffset
	if clientRunID != runID {
		offset = 0
	}

	lines, total, runID := gateway.logs.LinesSince(offset)
	if lines == nil {
		lines = []string{}
	}

	data["logs"] = lines
	data["log_total"] = total
	data["log_run_id"] = runID
	return data
}

// scanPipe reads lines from r and appends them to buf. Returns when r reaches EOF.
func scanPipe(r io.Reader, buf *LogBuffer) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		buf.Append(scanner.Text())
	}
}
