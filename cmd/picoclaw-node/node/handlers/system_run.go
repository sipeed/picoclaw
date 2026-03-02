package handlers

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/openclaw/go_node/config"
	"github.com/openclaw/go_node/gateway"
)

const outputCap = 200_000 // max combined stdout+stderr bytes

// SystemRunParams matches openclaw node-host invoke-types SystemRunParams
type SystemRunParams struct {
	Command    []string          `json:"command"`
	RawCommand *string           `json:"rawCommand,omitempty"`
	CWD        *string           `json:"cwd,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
	TimeoutMs  *int64            `json:"timeoutMs,omitempty"`
}

// RunResult matches openclaw invoke-types RunResult
type RunResult struct {
	ExitCode  *int   `json:"exitCode,omitempty"`
	TimedOut  bool   `json:"timedOut"`
	Success   bool   `json:"success"`
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr"`
	Error     string `json:"error,omitempty"`
	Truncated bool   `json:"truncated"`
}

// SystemRunHandler handles system.run command
type SystemRunHandler struct {
	cfg config.ExecConfig
}

// NewSystemRunHandler creates a handler with exec security config
func NewSystemRunHandler(cfg config.ExecConfig) *SystemRunHandler {
	return &SystemRunHandler{cfg: cfg}
}

// Handle executes shell command and returns result
func (h *SystemRunHandler) Handle(req gateway.InvokeRequest) gateway.InvokeResult {
	var params SystemRunParams
	paramsJSON := strings.TrimSpace(req.ParamsJSON)
	if paramsJSON == "" {
		return gateway.InvokeResult{
			OK: false,
			Error: &gateway.ErrorShape{
				Code:    "INVALID_REQUEST",
				Message: "paramsJSON required",
			},
		}
	}
	if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
		return gateway.InvokeResult{
			OK: false,
			Error: &gateway.ErrorShape{
				Code:    "INVALID_REQUEST",
				Message: "invalid paramsJSON: " + err.Error(),
			},
		}
	}

	argv := params.Command
	if len(argv) == 0 && params.RawCommand != nil {
		if !h.cfg.AllowAllCommands && len(h.cfg.AllowedCommands) > 0 {
			return gateway.InvokeResult{
				OK: false,
				Error: &gateway.ErrorShape{
					Code:    "RAW_COMMAND_NOT_ALLOWED",
					Message: "RAW_COMMAND_NOT_ALLOWED: rawCommand disabled when allowedCommands is set (unless allowAllCommands=true)",
				},
			}
		}
		raw := strings.TrimSpace(*params.RawCommand)
		if raw == "" {
			return gateway.InvokeResult{
				OK: false,
				Error: &gateway.ErrorShape{
					Code:    "INVALID_REQUEST",
					Message: "command required",
				},
			}
		}
		shell, args := shellExec()
		argv = append([]string{shell}, append(args, raw)...)
	}
	if len(argv) == 0 {
		return gateway.InvokeResult{
			OK: false,
			Error: &gateway.ErrorShape{
				Code:    "INVALID_REQUEST",
				Message: "command required",
			},
		}
	}

	if errRes := h.checkAllowed(argv[0]); errRes != nil {
		return *errRes
	}

	cwd, errRes := h.resolveCWD(params.CWD)
	if errRes != nil {
		return *errRes
	}

	log.Printf("go_node: system.run argv=%v cwd=%s", argv, cwd)

	timeoutMs := int64(60_000)
	if params.TimeoutMs != nil && *params.TimeoutMs > 0 {
		timeoutMs = *params.TimeoutMs
	}

	result := runCommand(argv, &cwd, params.Env, timeoutMs)
	payload, _ := json.Marshal(result)
	return gateway.InvokeResult{
		OK:          result.Success,
		PayloadJSON: string(payload),
	}
}

func (h *SystemRunHandler) checkAllowed(cmd string) *gateway.InvokeResult {
	if h.cfg.AllowAllCommands || len(h.cfg.AllowedCommands) == 0 {
		return nil // ok, allow all
	}
	name := filepath.Base(strings.TrimSpace(cmd))
	for _, allowed := range h.cfg.AllowedCommands {
		if name == strings.TrimSpace(allowed) {
			return nil // ok
		}
	}
	return &gateway.InvokeResult{
		OK: false,
		Error: &gateway.ErrorShape{
			Code:    "COMMAND_NOT_ALLOWED",
			Message: "COMMAND_NOT_ALLOWED: " + name + " is not in allowedCommands",
		},
	}
}

func (h *SystemRunHandler) resolveCWD(paramCWD *string) (string, *gateway.InvokeResult) {
	workDir := strings.TrimSpace(h.cfg.WorkDir)
	if workDir == "" {
		if paramCWD != nil && strings.TrimSpace(*paramCWD) != "" {
			abs, err := filepath.Abs(*paramCWD)
			if err != nil {
				return "", &gateway.InvokeResult{
					OK: false,
					Error: &gateway.ErrorShape{
						Code:    "INVALID_REQUEST",
						Message: "invalid cwd: " + err.Error(),
					},
				}
			}
			return abs, nil
		}
		wd, _ := os.Getwd()
		return wd, nil
	}
	if paramCWD == nil || strings.TrimSpace(*paramCWD) == "" {
		if fi, err := os.Stat(workDir); err != nil || !fi.IsDir() {
			return "", &gateway.InvokeResult{
				OK: false,
				Error: &gateway.ErrorShape{
					Code:    "WORKDIR_INVALID",
					Message: "WORKDIR_INVALID: exec.workDir must be an existing directory: " + workDir,
				},
			}
		}
		return workDir, nil
	}
	if fi, err := os.Stat(workDir); err != nil || !fi.IsDir() {
		return "", &gateway.InvokeResult{
			OK: false,
			Error: &gateway.ErrorShape{
				Code:    "WORKDIR_INVALID",
				Message: "WORKDIR_INVALID: exec.workDir must be an existing directory: " + workDir,
			},
		}
	}
	req := strings.TrimSpace(*paramCWD)
	var abs string
	if filepath.IsAbs(req) {
		abs = req
	} else {
		abs = filepath.Join(workDir, req)
	}
	abs, err := filepath.Abs(abs)
	if err != nil {
		return "", &gateway.InvokeResult{
			OK: false,
			Error: &gateway.ErrorShape{
				Code:    "INVALID_REQUEST",
				Message: "invalid cwd: " + err.Error(),
			},
		}
	}
	rel, err := filepath.Rel(workDir, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", &gateway.InvokeResult{
			OK: false,
			Error: &gateway.ErrorShape{
				Code:    "CWD_OUTSIDE_WORKDIR",
				Message: "CWD_OUTSIDE_WORKDIR: cwd must be under workDir " + workDir,
			},
		}
	}
	return abs, nil
}

func shellExec() (shell string, args []string) {
	if runtime.GOOS == "windows" {
		return "cmd.exe", []string{"/c"}
	}
	return "/bin/sh", []string{"-c"}
}

func runCommand(argv []string, cwd *string, env map[string]string, timeoutMs int64) RunResult {
	cmd := exec.Command(argv[0], argv[1:]...)
	if cwd != nil && *cwd != "" {
		cmd.Dir = *cwd
	}
	if len(env) > 0 {
		cmd.Env = envToSlice(env)
	}

	var stdout, stderr bytes.Buffer
	stdoutCap := &capWriter{w: &stdout, max: outputCap}
	stderrCap := &capWriter{w: &stderr, max: outputCap}
	cmd.Stdout = stdoutCap
	cmd.Stderr = stderrCap

	err := cmd.Start()
	if err != nil {
		return RunResult{
			Success: false,
			Stdout:  stdout.String(),
			Stderr:  stderr.String(),
			Error:   err.Error(),
		}
	}

	timeout := time.Duration(timeoutMs) * time.Millisecond
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case waitErr := <-done:
		if waitErr != nil {
			if exitErr, ok := waitErr.(*exec.ExitError); ok {
				code := exitErr.ExitCode()
				return RunResult{
					ExitCode:  &code,
					TimedOut:  false,
					Success:   false,
					Stdout:    truncateOutput(stdout.String(), outputCap),
					Stderr:    truncateOutput(stderr.String(), outputCap),
					Truncated: stdoutCap.truncated || stderrCap.truncated,
				}
			}
			return RunResult{
				Success:   false,
				Stdout:    truncateOutput(stdout.String(), outputCap),
				Stderr:    truncateOutput(stderr.String(), outputCap),
				Error:     waitErr.Error(),
				Truncated: stdoutCap.truncated || stderrCap.truncated,
			}
		}
		ec := 0
		return RunResult{
			ExitCode:  &ec,
			TimedOut:  false,
			Success:   true,
			Stdout:    truncateOutput(stdout.String(), outputCap),
			Stderr:    truncateOutput(stderr.String(), outputCap),
			Truncated: stdoutCap.truncated || stderrCap.truncated,
		}
	case <-time.After(timeout):
		cmd.Process.Kill()
		<-done
		return RunResult{
			TimedOut:  true,
			Success:   false,
			Stdout:    truncateOutput(stdout.String(), outputCap),
			Stderr:    truncateOutput(stderr.String(), outputCap),
			Error:     "command timeout",
			Truncated: stdoutCap.truncated || stderrCap.truncated,
		}
	}
}

func envToSlice(m map[string]string) []string {
	base := os.Environ()
	overrides := make(map[string]string)
	for k, v := range m {
		overrides[k] = v
	}
	if len(overrides) == 0 {
		return base
	}
	seen := make(map[string]bool)
	var out []string
	for _, s := range base {
		if idx := strings.IndexByte(s, '='); idx > 0 {
			key := s[:idx]
			if v, ok := overrides[key]; ok {
				out = append(out, key+"="+v)
				seen[key] = true
				continue
			}
		}
		out = append(out, s)
	}
	for k, v := range overrides {
		if !seen[k] {
			out = append(out, k+"="+v)
		}
	}
	return out
}

func truncateOutput(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return "... (truncated) " + s[len(s)-max:]
}

type capWriter struct {
	w         *bytes.Buffer
	max       int
	written   int
	truncated bool
}

func (c *capWriter) Write(p []byte) (n int, err error) {
	if c.written >= c.max {
		c.truncated = true
		return len(p), nil
	}
	rem := c.max - c.written
	if len(p) > rem {
		c.w.Write(p[:rem])
		c.written += rem
		c.truncated = true
		return len(p), nil
	}
	c.w.Write(p)
	c.written += len(p)
	return len(p), nil
}
