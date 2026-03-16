package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/web/backend/utils"
)

// gateway holds the state for the managed gateway process.
var gateway = struct {
	mu               sync.Mutex
	cmd              *exec.Cmd
	bootDefaultModel string
	runtimeStatus    string
	startupDeadline  time.Time
	logs             *LogBuffer
	events           *EventBroadcaster
}{
	runtimeStatus: "stopped",
	logs:          NewLogBuffer(200),
	events:        NewEventBroadcaster(),
}

var (
	gatewayStartupWindow          = 15 * time.Second
	gatewayRestartGracePeriod     = 5 * time.Second
	gatewayRestartForceKillWindow = 3 * time.Second
	gatewayRestartPollInterval    = 100 * time.Millisecond
)

var gatewayHealthGet = func(url string, timeout time.Duration) (*http.Response, error) {
	client := http.Client{Timeout: timeout}
	return client.Get(url)
}

// registerGatewayRoutes binds gateway lifecycle endpoints to the ServeMux.
func (h *Handler) registerGatewayRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/gateway/status", h.handleGatewayStatus)
	mux.HandleFunc("GET /api/gateway/events", h.handleGatewayEvents)
	mux.HandleFunc("GET /api/gateway/logs", h.handleGatewayLogs)
	mux.HandleFunc("POST /api/gateway/logs/clear", h.handleGatewayClearLogs)
	mux.HandleFunc("POST /api/gateway/start", h.handleGatewayStart)
	mux.HandleFunc("POST /api/gateway/stop", h.handleGatewayStop)
	mux.HandleFunc("POST /api/gateway/restart", h.handleGatewayRestart)
}

// TryAutoStartGateway checks whether gateway start preconditions are met and
// starts it when possible. Intended to be called by the backend at startup.
func (h *Handler) TryAutoStartGateway() {
	gateway.mu.Lock()
	defer gateway.mu.Unlock()

	if isGatewayProcessAliveLocked() {
		return
	}
	if gateway.cmd != nil && gateway.cmd.Process != nil {
		gateway.cmd = nil
	}

	ready, reason, err := h.gatewayStartReady()
	if err != nil {
		log.Printf("Skip auto-starting gateway: %v", err)
		return
	}
	if !ready {
		log.Printf("Skip auto-starting gateway: %s", reason)
		return
	}

	pid, err := h.startGatewayLocked("starting")
	if err != nil {
		log.Printf("Failed to auto-start gateway: %v", err)
		return
	}
	log.Printf("Gateway auto-started (PID: %d)", pid)
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

	if !hasModelConfiguration(*modelCfg) {
		return false, fmt.Sprintf("default model %q has no credentials configured", modelName), nil
	}
	if requiresRuntimeProbe(*modelCfg) && !probeLocalModelAvailability(*modelCfg) {
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

func isGatewayProcessAliveLocked() bool {
	return isCmdProcessAliveLocked(gateway.cmd)
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

	return cmd.Process.Signal(syscall.Signal(0)) == nil
}

func setGatewayRuntimeStatusLocked(status string) {
	gateway.runtimeStatus = status
	if status == "starting" || status == "restarting" {
		gateway.startupDeadline = time.Now().Add(gatewayStartupWindow)
		return
	}
	gateway.startupDeadline = time.Time{}
}

func gatewayStatusOnHealthFailureLocked() string {
	if gateway.runtimeStatus == "starting" || gateway.runtimeStatus == "restarting" {
		if gateway.startupDeadline.IsZero() || time.Now().Before(gateway.startupDeadline) {
			return gateway.runtimeStatus
		}
		return "error"
	}
	if gateway.runtimeStatus == "running" {
		return "running"
	}
	if gateway.runtimeStatus == "error" {
		return "error"
	}
	return "error"
}

func currentGatewayStatusLocked(processAlive bool) string {
	if !processAlive {
		if gateway.runtimeStatus == "restarting" {
			if gateway.startupDeadline.IsZero() || time.Now().Before(gateway.startupDeadline) {
				return "restarting"
			}
			return "error"
		}
		if gateway.runtimeStatus == "error" {
			return "error"
		}
		return "stopped"
	}
	return gatewayStatusOnHealthFailureLocked()
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

func gatewayRestartRequired(status, bootDefaultModel, configDefaultModel string) bool {
	return status == "running" &&
		bootDefaultModel != "" &&
		configDefaultModel != "" &&
		bootDefaultModel != configDefaultModel
}

func (h *Handler) startGatewayLocked(initialStatus string) (int, error) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		return 0, fmt.Errorf("failed to load config: %w", err)
	}
	defaultModelName := strings.TrimSpace(cfg.Agents.Defaults.GetModelName())

	// Locate the picoclaw executable
	execPath := utils.FindPicoclawBinary()

	cmd := exec.Command(execPath, "gateway")
	cmd.Env = os.Environ()
	// Forward the launcher's config path via the environment variable that
	// GetConfigPath() already reads, so the gateway sub-process uses the same
	// config file without requiring a --config flag on the gateway subcommand.
	if h.configPath != "" {
		cmd.Env = append(cmd.Env, "PICOCLAW_CONFIG="+h.configPath)
	}
	if host := h.gatewayHostOverride(); host != "" {
		cmd.Env = append(cmd.Env, "PICOCLAW_GATEWAY_HOST="+host)
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
	if _, err := h.ensurePicoChannel(""); err != nil {
		log.Printf("Warning: failed to ensure pico channel: %v", err)
		// Non-fatal: gateway can still start without pico channel
	}

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start gateway: %w", err)
	}

	gateway.cmd = cmd
	gateway.bootDefaultModel = defaultModelName
	setGatewayRuntimeStatusLocked(initialStatus)
	pid := cmd.Process.Pid
	log.Printf("Started picoclaw gateway (PID: %d) from %s", pid, execPath)

	// Broadcast the launch state immediately so clients can reflect it without polling.
	gateway.events.Broadcast(GatewayEvent{
		Status:             initialStatus,
		PID:                pid,
		BootDefaultModel:   defaultModelName,
		ConfigDefaultModel: defaultModelName,
		RestartRequired:    false,
	})

	// Capture stdout/stderr in background
	go scanPipe(stdoutPipe, gateway.logs)
	go scanPipe(stderrPipe, gateway.logs)

	// Wait for exit in background and clean up
	go func() {
		if err := cmd.Wait(); err != nil {
			log.Printf("Gateway process exited: %v", err)
		} else {
			log.Printf("Gateway process exited normally")
		}

		gateway.mu.Lock()
		shouldBroadcastStopped := false
		if gateway.cmd == cmd {
			gateway.cmd = nil
			gateway.bootDefaultModel = ""
			if gateway.runtimeStatus != "restarting" {
				setGatewayRuntimeStatusLocked("stopped")
				shouldBroadcastStopped = true
			}
		}
		gateway.mu.Unlock()

		if shouldBroadcastStopped {
			gateway.events.Broadcast(GatewayEvent{
				Status:          "stopped",
				RestartRequired: false,
			})
		}
	}()

	// Start a goroutine to probe health and broadcast "running" once ready
	go func() {
		for i := 0; i < 30; i++ { // try for up to 15 seconds
			time.Sleep(500 * time.Millisecond)
			gateway.mu.Lock()
			stillOurs := gateway.cmd == cmd
			gateway.mu.Unlock()
			if !stillOurs {
				return
			}
			cfg, err := config.LoadConfig(h.configPath)
			if err != nil {
				continue
			}
			healthHost := gatewayProbeHost(h.effectiveGatewayBindHost(cfg))
			healthPort := cfg.Gateway.Port
			if healthPort == 0 {
				healthPort = 18790
			}
			healthURL := fmt.Sprintf("http://%s/health", net.JoinHostPort(healthHost, strconv.Itoa(healthPort)))
			resp, err := gatewayHealthGet(healthURL, 1*time.Second)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					gateway.mu.Lock()
					if gateway.cmd == cmd {
						setGatewayRuntimeStatusLocked("running")
					}
					gateway.mu.Unlock()
					gateway.events.Broadcast(GatewayEvent{
						Status:             "running",
						PID:                pid,
						BootDefaultModel:   defaultModelName,
						ConfigDefaultModel: defaultModelName,
						RestartRequired:    false,
					})
					return
				}
			}
		}
	}()

	return pid, nil
}

// handleGatewayStart starts the picoclaw gateway subprocess.
//
//	POST /api/gateway/start
func (h *Handler) handleGatewayStart(w http.ResponseWriter, r *http.Request) {
	gateway.mu.Lock()
	defer gateway.mu.Unlock()

	// Prevent duplicate starts
	if isGatewayProcessAliveLocked() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]any{
			"status": "already_running",
			"pid":    gateway.cmd.Process.Pid,
		})
		return
	}
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

	pid, err := h.startGatewayLocked("starting")
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

	pid := gateway.cmd.Process.Pid

	// Send SIGTERM for graceful shutdown (SIGKILL on Windows)
	var sigErr error
	if runtime.GOOS == "windows" {
		sigErr = gateway.cmd.Process.Kill()
	} else {
		sigErr = gateway.cmd.Process.Signal(syscall.SIGTERM)
	}

	if sigErr != nil {
		http.Error(w, fmt.Sprintf("Failed to stop gateway (PID %d): %v", pid, sigErr), http.StatusInternalServerError)
		return
	}

	log.Printf("Sent stop signal to gateway (PID: %d)", pid)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status": "ok",
		"pid":    pid,
	})
}

// handleGatewayRestart stops the gateway (if running) and starts a new instance.
//
//	POST /api/gateway/restart
func (h *Handler) handleGatewayRestart(w http.ResponseWriter, r *http.Request) {
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

	gateway.mu.Lock()
	previousCmd := gateway.cmd
	setGatewayRuntimeStatusLocked("restarting")
	gateway.events.Broadcast(GatewayEvent{
		Status:          "restarting",
		RestartRequired: false,
	})
	gateway.mu.Unlock()

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
		http.Error(w, fmt.Sprintf("Failed to restart gateway: %v", err), http.StatusInternalServerError)
		return
	}

	gateway.mu.Lock()
	if gateway.cmd == previousCmd {
		gateway.cmd = nil
		gateway.bootDefaultModel = ""
	}
	pid, err := h.startGatewayLocked("restarting")
	if err != nil {
		gateway.cmd = nil
		gateway.bootDefaultModel = ""
		setGatewayRuntimeStatusLocked("error")
	}
	gateway.mu.Unlock()
	if err != nil {
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
	cfg, cfgErr := config.LoadConfig(h.configPath)
	configDefaultModel := ""
	if cfgErr == nil && cfg != nil {
		configDefaultModel = strings.TrimSpace(cfg.Agents.Defaults.GetModelName())
		if configDefaultModel != "" {
			data["config_default_model"] = configDefaultModel
		}
	}

	// Check process state
	gateway.mu.Lock()
	processAlive := isGatewayProcessAliveLocked()
	bootDefaultModel := ""
	if processAlive {
		data["pid"] = gateway.cmd.Process.Pid
		if gateway.bootDefaultModel != "" {
			data["boot_default_model"] = gateway.bootDefaultModel
			bootDefaultModel = gateway.bootDefaultModel
		}
	}
	gateway.mu.Unlock()

	if !processAlive {
		gateway.mu.Lock()
		data["gateway_status"] = currentGatewayStatusLocked(false)
		gateway.mu.Unlock()
	} else {
		// Process is alive — probe its health endpoint
		host := "127.0.0.1"
		port := 18790
		if cfgErr == nil && cfg != nil {
			host = gatewayProbeHost(h.effectiveGatewayBindHost(cfg))
			if cfg.Gateway.Port != 0 {
				port = cfg.Gateway.Port
			}
		}

		url := fmt.Sprintf("http://%s/health", net.JoinHostPort(host, strconv.Itoa(port)))
		resp, err := gatewayHealthGet(url, 2*time.Second)

		if err != nil {
			gateway.mu.Lock()
			data["gateway_status"] = currentGatewayStatusLocked(true)
			gateway.mu.Unlock()
		} else {
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				gateway.mu.Lock()
				setGatewayRuntimeStatusLocked("error")
				gateway.mu.Unlock()
				data["gateway_status"] = "error"
				data["status_code"] = resp.StatusCode
			} else {
				var healthData map[string]any
				if decErr := json.NewDecoder(resp.Body).Decode(&healthData); decErr != nil {
					gateway.mu.Lock()
					setGatewayRuntimeStatusLocked("error")
					gateway.mu.Unlock()
					data["gateway_status"] = "error"
				} else {
					gateway.mu.Lock()
					setGatewayRuntimeStatusLocked("running")
					gateway.mu.Unlock()
					for k, v := range healthData {
						data[k] = v
					}
					data["gateway_status"] = "running"
				}
			}
		}
	}

	status, _ := data["gateway_status"].(string)
	data["gateway_restart_required"] = gatewayRestartRequired(
		status,
		bootDefaultModel,
		configDefaultModel,
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

// handleGatewayEvents serves an SSE stream of gateway state change events.
//
//	GET /api/gateway/events
func (h *Handler) handleGatewayEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Subscribe to gateway events
	ch := gateway.events.Subscribe()
	defer gateway.events.Unsubscribe(ch)

	// Send initial status so the client doesn't start blank
	initial := h.currentGatewayStatus()
	fmt.Fprintf(w, "data: %s\n\n", initial)
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case data, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

// currentGatewayStatus returns the current gateway status as a JSON string.
func (h *Handler) currentGatewayStatus() string {
	data := h.gatewayStatusData()
	encoded, _ := json.Marshal(data)
	return string(encoded)
}

// scanPipe reads lines from r and appends them to buf. Returns when r reaches EOF.
func scanPipe(r io.Reader, buf *LogBuffer) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		buf.Append(scanner.Text())
	}
}
