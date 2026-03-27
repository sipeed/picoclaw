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

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/web/backend/utils"
)

type systemVersionResponse struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit,omitempty"`
	BuildTime string `json:"build_time,omitempty"`
	GoVersion string `json:"go_version"`
}

var (
	// Reuse the launcher gateway startup window so embedded/slow devices
	// have enough time for first-run command initialization.
	versionCmdTimeout         = gatewayStartupWindow
	findPicoclawBinaryForInfo = utils.FindPicoclawBinary
	runPicoclawVersionOutput  = executePicoclawVersion
	versionLinePattern        = regexp.MustCompile(`\bpicoclaw\s+([^\s(]+)(?:\s+\(git:\s*([^)]+)\))?`)
	ansiEscapePattern         = regexp.MustCompile(`\x1b\[[0-9;]*m`)
)

func (h *Handler) registerVersionRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/system/version", h.handleGetVersion)
}

// handleGetVersion returns runtime version information for web clients.
func (h *Handler) handleGetVersion(w http.ResponseWriter, _ *http.Request) {
	versionInfo := h.resolveSystemVersionInfo()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(versionInfo)
}

// resolveSystemVersionInfo prefers the actual picoclaw binary version output,
// and falls back to launcher build metadata when command execution fails.
func (h *Handler) resolveSystemVersionInfo() systemVersionResponse {
	buildTime, goVer := config.FormatBuildInfo()
	fallback := systemVersionResponse{
		Version:   config.GetVersion(),
		GitCommit: config.GitCommit,
		BuildTime: buildTime,
		GoVersion: goVer,
	}

	execPath := strings.TrimSpace(findPicoclawBinaryForInfo())
	if execPath == "" {
		return fallback
	}

	ctx, cancel := context.WithTimeout(context.Background(), versionCmdTimeout)
	defer cancel()

	output, err := runPicoclawVersionOutput(ctx, execPath)
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
			result.Version = strings.TrimSpace(match[1])
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

	if result.Version == "" {
		return systemVersionResponse{}, false
	}

	return result, true
}
