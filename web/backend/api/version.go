package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/web/backend/utils"
)

type systemVersionResponse struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit,omitempty"`
	BuildTime string `json:"build_time,omitempty"`
	GoVersion string `json:"go_version"`
}

type cachedSystemVersion struct {
	value      systemVersionResponse
	gatewayPID int
}

type systemVersionCache struct {
	mu         sync.Mutex
	current    cachedSystemVersion
	hasCurrent bool
	inflightCh chan struct{}
}

func newSystemVersionCache() *systemVersionCache {
	return &systemVersionCache{}
}

var (
	// 15 seconds matches the gateway startup window used elsewhere in launcher flow,
	// giving slow/embedded hosts enough time for first command invocation while
	// staying independent from cross-file init ordering.
	versionCmdTimeout           = 15 * time.Second
	maxVersionResolveAttempts   = 3
	findPicoclawBinaryForInfo   = resolveGatewayBinaryForVersionInfo
	runPicoclawVersionOutput    = executePicoclawVersion
	currentGatewayVersionState  = gatewayVersionState
	launcherBuildInfoForVersion = fallbackSystemVersionInfoFromConfig
	versionInfoCache            = newSystemVersionCache()
	ansiEscapePattern           = regexp.MustCompile(`\x1b\[[0-9;]*m`)
	versionLinePattern          = regexp.MustCompile(
		`^(?:[^A-Za-z0-9]*\s*)?picoclaw(?:\.exe)?\s+([^\s(]+)` +
			`(?:\s+\(git:\s*([^)]+)\))?\s*$`,
	)
)

func (h *Handler) registerVersionRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/system/version", h.handleGetVersion)
}

// handleGetVersion returns runtime version information for web clients.
func (h *Handler) handleGetVersion(w http.ResponseWriter, r *http.Request) {
	versionInfo := h.resolveSystemVersionInfo(r.Context())

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(versionInfo); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// resolveSystemVersionInfo prefers the actual picoclaw binary version output,
// and falls back to launcher build metadata when command execution fails.
func (h *Handler) resolveSystemVersionInfo(ctx context.Context) systemVersionResponse {
	for range maxVersionResolveAttempts {
		gatewayPID, gatewayAlive := currentGatewayVersionState()
		if cached, ok := versionInfoCache.get(gatewayPID, gatewayAlive); ok {
			return cached
		}

		leader, ok := versionInfoCache.waitOrStart(ctx)
		if !ok {
			return fallbackSystemVersionInfo()
		}
		if !leader {
			continue
		}

		resolved := h.resolveSystemVersionInfoUncached(ctx)
		gatewayPID, gatewayAlive = currentGatewayVersionState()
		versionInfoCache.finishResolve(resolved, gatewayPID, gatewayAlive)
		return resolved
	}

	return fallbackSystemVersionInfo()
}

func (h *Handler) resolveSystemVersionInfoUncached(ctx context.Context) systemVersionResponse {
	if ctx == nil {
		ctx = context.Background()
	}

	fallback := fallbackSystemVersionInfo()

	execPath := strings.TrimSpace(findPicoclawBinaryForInfo())
	if execPath == "" {
		return fallback
	}

	cmdCtx, cancel := context.WithTimeout(ctx, versionCmdTimeout)
	defer cancel()

	output, err := runPicoclawVersionOutput(cmdCtx, execPath)
	if err != nil {
		return fallback
	}

	parsed, ok := parsePicoclawVersionOutput(output)
	if !ok {
		return fallback
	}

	if parsed.GoVersion == "" {
		parsed.GoVersion = fallback.GoVersion
		if parsed.GoVersion == "" {
			parsed.GoVersion = runtime.Version()
		}
	}

	return parsed
}

func fallbackSystemVersionInfo() systemVersionResponse {
	return launcherBuildInfoForVersion()
}

func fallbackSystemVersionInfoFromConfig() systemVersionResponse {
	buildTime, goVer := config.FormatBuildInfo()
	return systemVersionResponse{
		Version:   config.GetVersion(),
		GitCommit: config.GitCommit,
		BuildTime: buildTime,
		GoVersion: goVer,
	}
}

// resolveGatewayBinaryForVersionInfo uses the same executable as the launcher
// gateway start path when available, then falls back to launcher binary lookup.
// This keeps version probing aligned with the actual gateway startup behavior,
// so web and gateway do not drift onto different binaries.
func resolveGatewayBinaryForVersionInfo() string {
	gateway.mu.Lock()
	cmd := gateway.cmd
	gateway.mu.Unlock()

	if cmd != nil {
		if execPath := strings.TrimSpace(cmd.Path); execPath != "" {
			return execPath
		}
	}

	return utils.FindPicoclawBinary()
}

func gatewayVersionState() (int, bool) {
	gateway.mu.Lock()
	defer gateway.mu.Unlock()

	if gateway.cmd == nil || gateway.cmd.Process == nil {
		return 0, false
	}
	pid := gateway.cmd.Process.Pid
	if pid <= 0 {
		return 0, false
	}

	return pid, isCmdProcessAliveLocked(gateway.cmd)
}

func (c *systemVersionCache) get(gatewayPID int, gatewayAlive bool) (systemVersionResponse, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.hasCurrent && (!gatewayAlive || gatewayPID <= 0 || gatewayPID != c.current.gatewayPID) {
		c.clearCurrentLocked()
	}

	if c.hasCurrent {
		return c.current.value, true
	}

	return systemVersionResponse{}, false
}

func (c *systemVersionCache) waitOrStart(ctx context.Context) (bool, bool) {
	if ctx == nil {
		ctx = context.Background()
	}
	if ctx.Err() != nil {
		return false, false
	}

	c.mu.Lock()
	if c.inflightCh == nil {
		c.inflightCh = make(chan struct{})
		c.mu.Unlock()
		return true, true
	}
	waitCh := c.inflightCh
	c.mu.Unlock()

	select {
	case <-waitCh:
		return false, true
	case <-ctx.Done():
		return false, false
	}
}

func (c *systemVersionCache) finishResolve(value systemVersionResponse, gatewayPID int, gatewayAlive bool) {
	c.mu.Lock()
	if gatewayAlive && gatewayPID > 0 {
		c.current = cachedSystemVersion{value: value, gatewayPID: gatewayPID}
		c.hasCurrent = true
	} else {
		c.clearCurrentLocked()
	}

	inflightCh := c.inflightCh
	c.inflightCh = nil
	c.mu.Unlock()

	if inflightCh != nil {
		close(inflightCh)
	}
}

func (c *systemVersionCache) clearCurrentLocked() {
	c.hasCurrent = false
	c.current = cachedSystemVersion{}
}

func (c *systemVersionCache) resetForTest() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.current = cachedSystemVersion{}
	c.hasCurrent = false
	if c.inflightCh != nil {
		close(c.inflightCh)
		c.inflightCh = nil
	}
}

// executePicoclawVersion runs the version subcommand against the
// discovered picoclaw executable.
func executePicoclawVersion(ctx context.Context, execPath string) (string, error) {
	out, err := exec.CommandContext(ctx, execPath, "version").CombinedOutput()
	if err == nil {
		return string(out), nil
	}

	return string(out), fmt.Errorf("failed to execute version command: %w", err)
}

// parsePicoclawVersionOutput extracts version/build/go fields from CLI output.
// It accepts banner/ANSI-decorated output and only requires the version line.
func parsePicoclawVersionOutput(raw string) (systemVersionResponse, bool) {
	var result systemVersionResponse

	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := strings.TrimSpace(ansiEscapePattern.ReplaceAllString(scanner.Text(), ""))
		if line == "" {
			continue
		}

		if match := versionLinePattern.FindStringSubmatch(line); len(match) > 0 {
			candidateVersion := strings.TrimSpace(match[1])
			if !isLikelyVersionValue(candidateVersion) {
				continue
			}
			result.Version = candidateVersion
			if len(match) > 2 {
				result.GitCommit = strings.TrimSpace(match[2])
			}
			continue
		}

		if buildValue, ok := strings.CutPrefix(line, "Build:"); ok {
			result.BuildTime = strings.TrimSpace(buildValue)
			continue
		}

		if goValue, ok := strings.CutPrefix(line, "Go:"); ok {
			result.GoVersion = strings.TrimSpace(goValue)
		}
	}

	if err := scanner.Err(); err != nil {
		return systemVersionResponse{}, false
	}

	if result.Version == "" {
		return systemVersionResponse{}, false
	}

	return result, true
}

func isLikelyVersionValue(value string) bool {
	v := strings.TrimSpace(strings.ToLower(value))
	if v == "" {
		return false
	}
	if v == "dev" {
		return true
	}

	// Accept git-like short/long hashes even when they contain only letters (a-f).
	if len(v) >= 7 && len(v) <= 40 {
		allHex := true
		for _, ch := range v {
			if (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') {
				continue
			}
			allHex = false
			break
		}
		if allHex {
			return true
		}
	}

	for _, ch := range v {
		if ch >= '0' && ch <= '9' {
			return true
		}
	}
	return false
}
