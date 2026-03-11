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
	mu     sync.Mutex
	cmd    *exec.Cmd
	logs   *LogBuffer
	events *EventBroadcaster
}{
	logs:   NewLogBuffer(200),
	events: NewEventBroadcaster(),
}

// registerGatewayRoutes binds gateway lifecycle endpoints to the ServeMux.
func (h *Handler) registerGatewayRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/gateway/status", h.handleGatewayStatus)
	mux.HandleFunc("GET /api/gateway/events", h.handleGatewayEvents)
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

	pid, err := h.startGatewayLocked()
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

func (h *Handler) startGatewayLocked() (int, error) {
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
	if _, err := h.ensurePicoChannel(); err != nil {
		log.Printf("Warning: failed to ensure pico channel: %v", err)
		// Non-fatal: gateway can still start without pico channel
	}

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start gateway: %w", err)
	}

	gateway.cmd = cmd
	pid := cmd.Process.Pid
	log.Printf("Started picoclaw gateway (PID: %d) from %s", pid, execPath)

	// Broadcast starting event
	gateway.events.Broadcast(GatewayEvent{Status: "starting", PID: pid})

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
		if gateway.cmd == cmd {
			gateway.cmd = nil
		}
		gateway.mu.Unlock()

		// Broadcast stopped event
		gateway.events.Broadcast(GatewayEvent{Status: "stopped"})
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
			client := http.Client{Timeout: 1 * time.Second}
			resp, err := client.Get(healthURL)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					gateway.events.Broadcast(GatewayEvent{Status: "running", PID: pid})
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

	pid, err := h.startGatewayLocked()
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
	gateway.mu.Lock()

	// Stop existing process if running
	if gateway.cmd != nil && gateway.cmd.Process != nil {
		if isCmdProcessAliveLocked(gateway.cmd) {
			// Process is alive, send SIGTERM
			if runtime.GOOS == "windows" {
				gateway.cmd.Process.Kill()
			} else {
				gateway.cmd.Process.Signal(syscall.SIGTERM)
			}

			// Wait briefly for it to exit
			gateway.mu.Unlock()
			time.Sleep(2 * time.Second)
			gateway.mu.Lock()
		}
		gateway.cmd = nil
	}

	gateway.mu.Unlock()

	// Start fresh via the existing handler
	h.handleGatewayStart(w, r)
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

// handleGatewayStatus returns the gateway run status, health info, and logs.
//
//	GET /api/gateway/status
func (h *Handler) handleGatewayStatus(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{}

	// Check process state
	gateway.mu.Lock()
	processAlive := isGatewayProcessAliveLocked()
	if processAlive {
		data["pid"] = gateway.cmd.Process.Pid
	}
	gateway.mu.Unlock()

	if !processAlive {
		data["gateway_status"] = "stopped"
	} else {
		// Process is alive — probe its health endpoint
		cfg, err := config.LoadConfig(h.configPath)
		host := "127.0.0.1"
		port := 18790
		if err == nil && cfg != nil {
			host = gatewayProbeHost(h.effectiveGatewayBindHost(cfg))
			if cfg.Gateway.Port != 0 {
				port = cfg.Gateway.Port
			}
		}

		url := fmt.Sprintf("http://%s/health", net.JoinHostPort(host, strconv.Itoa(port)))
		client := http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get(url)

		if err != nil {
			data["gateway_status"] = "starting"
		} else {
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				data["gateway_status"] = "error"
				data["status_code"] = resp.StatusCode
			} else {
				var healthData map[string]any
				if decErr := json.NewDecoder(resp.Body).Decode(&healthData); decErr != nil {
					data["gateway_status"] = "error"
				} else {
					for k, v := range healthData {
						data[k] = v
					}
					data["gateway_status"] = "running"
				}
			}
		}
	}

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

	// Append incremental log data
	appendGatewayLogs(r, data)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// appendGatewayLogs reads log_offset and log_run_id query params from the request
// and populates the response data map with incremental log lines.
func appendGatewayLogs(r *http.Request, data map[string]any) {
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
		return
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
	gateway.mu.Lock()
	defer gateway.mu.Unlock()

	data := map[string]any{
		"gateway_status": "stopped",
	}
	if isGatewayProcessAliveLocked() {
		data["gateway_status"] = "running"
		data["pid"] = gateway.cmd.Process.Pid
	}

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
