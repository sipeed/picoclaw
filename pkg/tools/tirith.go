package tools

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// TirithConfig holds Tirith security scanner settings (tools-internal).
// Mapped from config.TirithConfig at ExecTool construction time.
type TirithConfig struct {
	Enabled  bool
	BinPath  string
	Timeout  int
	FailOpen bool
}

const (
	tirithRepo          = "sheeki03/tirith"
	tirithMarkerTTL     = 24 * time.Hour
	tirithInstallTimeout = 30 * time.Second
)

var (
	tirithResolvedPath string
	tirithInstallFailed bool
	tirithMu           sync.Mutex
)

// tirithGuard scans a command with tirith for content-level threats.
// Call AFTER guardCommand() — cheap regex guards reject known-bad first.
// Returns empty string if allowed, error message if blocked.
func tirithGuard(command string, cfg TirithConfig) string {
	if !cfg.Enabled {
		return ""
	}

	binPath := resolveTirithPath(cfg.BinPath)

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 5
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	shellFlag := "posix"
	if runtime.GOOS == "windows" {
		shellFlag = "powershell"
	}

	cmd := exec.CommandContext(ctx, binPath, "check", "--json", "--non-interactive",
		"--shell", shellFlag, "--", command)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			switch exitErr.ExitCode() {
			case 1: // block
				return fmt.Sprintf("Command blocked by Tirith security scan: %s", tirithSummarize(stdout.Bytes()))
			case 2: // warn — log and allow
				log.Printf("[tirith] warning: %s", tirithSummarize(stdout.Bytes()))
				return ""
			}
		}
		// Spawn failure / timeout
		if cfg.FailOpen {
			log.Printf("[tirith] unavailable (fail-open): %v", err)
			return ""
		}
		return fmt.Sprintf("Tirith security scan failed (fail-closed): %v", err)
	}
	// Exit code 0 — allow
	return ""
}

func tirithSummarize(jsonOutput []byte) string {
	if len(jsonOutput) == 0 {
		return "security issue detected"
	}
	var data struct {
		Findings []struct {
			Severity string `json:"severity"`
			Title    string `json:"title"`
		} `json:"findings"`
	}
	if err := json.Unmarshal(jsonOutput, &data); err != nil {
		return "security issue detected (details unavailable)"
	}
	parts := make([]string, 0, len(data.Findings))
	for _, f := range data.Findings {
		if f.Title != "" {
			parts = append(parts, fmt.Sprintf("[%s] %s", f.Severity, f.Title))
		}
	}
	if len(parts) == 0 {
		return "security issue detected"
	}
	return strings.Join(parts, "; ")
}

// resolveTirithPath resolves the tirith binary, auto-installing if needed.
func resolveTirithPath(configured string) string {
	tirithMu.Lock()
	defer tirithMu.Unlock()

	if tirithResolvedPath != "" {
		return tirithResolvedPath
	}
	if tirithInstallFailed {
		return configured
	}

	explicit := configured != "tirith" && configured != ""

	if explicit {
		if isExecutable(configured) {
			tirithResolvedPath = configured
			return configured
		}
		if p, err := exec.LookPath(configured); err == nil {
			tirithResolvedPath = p
			return p
		}
		tirithInstallFailed = true
		return configured
	}

	// Default: PATH → ~/.picoclaw/bin → auto-install
	if p, err := exec.LookPath("tirith"); err == nil {
		tirithResolvedPath = p
		return p
	}

	binName := "tirith"
	if runtime.GOOS == "windows" {
		binName = "tirith.exe"
	}
	localBin := filepath.Join(tirithBinDir(), binName)
	if isExecutable(localBin) {
		tirithResolvedPath = localBin
		return localBin
	}

	// Check failure marker
	if tirithIsFailedOnDisk() {
		tirithInstallFailed = true
		return configured
	}

	installed := tirithInstall()
	if installed != "" {
		tirithResolvedPath = installed
		return installed
	}

	tirithInstallFailed = true
	tirithMarkFailed()
	return configured
}

func tirithBinDir() string {
	home, _ := os.UserHomeDir()
	d := filepath.Join(home, ".picoclaw", "bin")
	_ = os.MkdirAll(d, 0o755)
	return d
}

func tirithMarkerPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".picoclaw", ".tirith-install-failed")
}

func tirithIsFailedOnDisk() bool {
	info, err := os.Stat(tirithMarkerPath())
	if err != nil {
		return false
	}
	return time.Since(info.ModTime()) < tirithMarkerTTL
}

func tirithMarkFailed() {
	p := tirithMarkerPath()
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	_ = os.WriteFile(p, []byte("failed"), 0o644)
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if runtime.GOOS == "windows" {
		return !info.IsDir()
	}
	return !info.IsDir() && info.Mode()&0o111 != 0
}

func tirithInstall() string {
	target, ext := tirithDetectTarget()
	if target == "" {
		log.Printf("[tirith] auto-install: unsupported platform %s/%s", runtime.GOOS, runtime.GOARCH)
		return ""
	}

	archiveName := fmt.Sprintf("tirith-%s%s", target, ext)
	baseURL := fmt.Sprintf("https://github.com/%s/releases/latest/download", tirithRepo)

	tmpDir, err := os.MkdirTemp("", "tirith-install-")
	if err != nil {
		return ""
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, archiveName)
	checksumsPath := filepath.Join(tmpDir, "checksums.txt")

	log.Printf("[tirith] downloading latest release for %s...", target)

	if err := tirithDownload(baseURL+"/"+archiveName, archivePath); err != nil {
		log.Printf("[tirith] download failed: %v", err)
		return ""
	}
	if err := tirithDownload(baseURL+"/checksums.txt", checksumsPath); err != nil {
		log.Printf("[tirith] checksums download failed: %v", err)
		return ""
	}

	if !tirithVerifyChecksum(archivePath, checksumsPath, archiveName) {
		log.Printf("[tirith] checksum verification failed")
		return ""
	}

	binName := "tirith"
	if runtime.GOOS == "windows" {
		binName = "tirith.exe"
	}

	extractedPath := tirithExtract(tmpDir, archivePath, ext, binName)
	if extractedPath == "" {
		return ""
	}

	dest := filepath.Join(tirithBinDir(), binName)
	if err := os.Rename(extractedPath, dest); err != nil {
		// Cross-device rename fallback
		src, err2 := os.ReadFile(extractedPath)
		if err2 != nil {
			return ""
		}
		if err2 := os.WriteFile(dest, src, 0o755); err2 != nil {
			return ""
		}
	}
	_ = os.Chmod(dest, 0o755)

	log.Printf("[tirith] installed to %s (SHA-256 verified)", dest)
	return dest
}

func tirithDetectTarget() (string, string) {
	var plat string
	switch runtime.GOOS {
	case "darwin":
		plat = "apple-darwin"
	case "linux":
		plat = "unknown-linux-gnu"
	case "windows":
		plat = "pc-windows-msvc"
	default:
		return "", ""
	}

	var arch string
	switch runtime.GOARCH {
	case "amd64":
		arch = "x86_64"
	case "arm64":
		arch = "aarch64"
	default:
		return "", ""
	}

	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	return arch + "-" + plat, ext
}

func tirithDownload(url, dest string) error {
	client := &http.Client{Timeout: tirithInstallTimeout}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "token "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func tirithVerifyChecksum(archivePath, checksumsPath, archiveName string) bool {
	data, err := os.ReadFile(checksumsPath)
	if err != nil {
		return false
	}
	var expected string
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), "  ", 2)
		if len(parts) == 2 && parts[1] == archiveName {
			expected = parts[0]
			break
		}
	}
	if expected == "" {
		return false
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return false
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false
	}
	return hex.EncodeToString(h.Sum(nil)) == expected
}

func tirithExtract(tmpDir, archivePath, ext, binName string) string {
	if ext == ".zip" {
		return tirithExtractZip(tmpDir, archivePath, binName)
	}
	return tirithExtractTarGz(tmpDir, archivePath, binName)
}

func tirithExtractTarGz(tmpDir, archivePath, binName string) string {
	cmd := exec.Command("tar", "xzf", archivePath, "-C", tmpDir)
	if err := cmd.Run(); err != nil {
		return ""
	}
	// Find the binary
	candidates := []string{
		filepath.Join(tmpDir, binName),
	}
	for _, c := range candidates {
		if isExecutable(c) {
			return c
		}
	}
	// Walk for it
	var found string
	_ = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && info.Name() == binName && !info.IsDir() {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

func tirithExtractZip(tmpDir, archivePath, binName string) string {
	cmd := exec.Command("unzip", "-o", archivePath, "-d", tmpDir)
	if err := cmd.Run(); err != nil {
		return ""
	}
	var found string
	_ = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && info.Name() == binName && !info.IsDir() {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	return found
}
