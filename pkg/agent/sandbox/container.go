package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-units"

	"github.com/sipeed/picoclaw/internal/infra"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// ContainerSandboxConfig defines runtime and docker settings for container sandbox execution.
type ContainerSandboxConfig struct {
	Image           string
	ContainerName   string
	ContainerPrefix string
	Workspace       string
	AgentWorkspace  string
	WorkspaceAccess string
	WorkspaceRoot   string
	PruneIdleHours  int
	PruneMaxAgeDays int
	Workdir         string
	ReadOnlyRoot    bool
	Tmpfs           []string
	Network         string
	User            string
	CapDrop         []string
	Env             map[string]string
	SetupCommand    string
	PidsLimit       int64
	Memory          string
	MemorySwap      string
	Cpus            float64
	Ulimits         map[string]config.AgentSandboxDockerUlimitValue
	SeccompProfile  string
	ApparmorProfile string
	DNS             []string
	ExtraHosts      []string
	Binds           []string
}

// ContainerSandbox executes commands and filesystem operations inside a managed docker container.
type ContainerSandbox struct {
	mu       sync.Mutex
	cfg      ContainerSandboxConfig
	cli      *client.Client
	startErr error
	fs       FsBridge
	hash     string
}

const (
	defaultSandboxRegistryFile = "containers.json"
	DefaultSandboxImage        = "picoclaw-sandbox:bookworm-slim"
	FallbackSandboxImage       = "debian:bookworm-slim"
)

// NewContainerSandbox creates a container sandbox with normalized defaults and precomputed config hash.
func NewContainerSandbox(cfg ContainerSandboxConfig) *ContainerSandbox {
	if strings.TrimSpace(cfg.Image) == "" {
		cfg.Image = DefaultSandboxImage
	}
	if strings.TrimSpace(cfg.ContainerPrefix) == "" {
		cfg.ContainerPrefix = "picoclaw-sandbox-"
	}
	if strings.TrimSpace(cfg.ContainerName) == "" {
		cfg.ContainerName = cfg.ContainerPrefix + "default"
	}
	if strings.TrimSpace(cfg.Workdir) == "" {
		cfg.Workdir = "/workspace"
	}
	if len(cfg.Tmpfs) == 0 {
		cfg.Tmpfs = []string{"/tmp", "/var/tmp", "/run"}
	}
	if strings.TrimSpace(cfg.Network) == "" {
		cfg.Network = "none"
	}
	if cfg.CapDrop == nil {
		cfg.CapDrop = []string{"ALL"}
	}
	if cfg.Env == nil {
		cfg.Env = map[string]string{"LANG": "C.UTF-8"}
	}
	cfg.Env = sanitizeEnvVars(cfg.Env)
	if len(cfg.Env) == 0 {
		cfg.Env = map[string]string{"LANG": "C.UTF-8"}
	}
	cfg.WorkspaceAccess = string(normalizeWorkspaceAccess(config.WorkspaceAccess(cfg.WorkspaceAccess)))
	cfg.WorkspaceRoot = strings.TrimSpace(cfg.WorkspaceRoot)
	sb := &ContainerSandbox{cfg: cfg}
	sb.hash = computeContainerConfigHash(cfg)
	sb.fs = &containerFS{sb: sb}
	return sb
}

// Start initializes docker connectivity and validates sandbox runtime requirements.
func (c *ContainerSandbox) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.startErr != nil {
		return c.startErr
	}
	if c.cli != nil {
		return nil
	}

	if err := validateSandboxSecurity(c.cfg); err != nil {
		c.startErr = err
		return err
	}
	if strings.TrimSpace(c.cfg.Workspace) != "" && c.cfg.WorkspaceAccess == string(config.WorkspaceAccessNone) {
		if err := os.MkdirAll(c.cfg.Workspace, 0o755); err != nil {
			c.startErr = fmt.Errorf("sandbox workspace init failed: %w", err)
			return c.startErr
		}
	}
	if strings.TrimSpace(c.cfg.WorkspaceRoot) != "" {
		if err := os.MkdirAll(c.cfg.WorkspaceRoot, 0o755); err != nil {
			c.startErr = fmt.Errorf("sandbox workspace_root init failed: %w", err)
			return c.startErr
		}
	}

	if c.cli == nil {
		cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			c.startErr = fmt.Errorf("docker client init failed: %w", err)
			return c.startErr
		}
		c.cli = cli
	}

	if _, err := c.cli.Ping(ctx); err != nil {
		c.startErr = fmt.Errorf("docker daemon unavailable: %w", err)
		return c.startErr
	}

	if _, err := c.cli.ImageInspect(ctx, c.cfg.Image); err != nil {
		if c.cfg.Image == DefaultSandboxImage {
			// If default image is missing, try to pull fallback and tag it
			rc, pullErr := c.cli.ImagePull(ctx, FallbackSandboxImage, image.PullOptions{})
			if pullErr != nil {
				c.startErr = fmt.Errorf("docker fallback image unavailable (%s): %w", FallbackSandboxImage, pullErr)
				return c.startErr
			}
			defer rc.Close()
			if _, err := io.Copy(io.Discard, rc); err != nil {
				c.startErr = fmt.Errorf("failed to pull fallback image: %w", err)
				return c.startErr
			}

			// Tag debian:bookworm-slim as picoclaw-sandbox:bookworm-slim
			if err := c.cli.ImageTag(ctx, FallbackSandboxImage, DefaultSandboxImage); err != nil {
				c.startErr = fmt.Errorf("failed to tag fallback image: %w", err)
				return c.startErr
			}
		} else {
			// For non-default images, just try to pull directly
			rc, pullErr := c.cli.ImagePull(ctx, c.cfg.Image, image.PullOptions{})
			if pullErr != nil {
				c.startErr = fmt.Errorf("docker image unavailable (%s): %w", c.cfg.Image, pullErr)
				return c.startErr
			}
			defer rc.Close()
			if _, err := io.Copy(io.Discard, rc); err != nil {
				c.startErr = fmt.Errorf("failed to pull image: %w", err)
				return c.startErr
			}
		}
	}

	return nil
}

// Resolve returns the container sandbox itself.
func (c *ContainerSandbox) Resolve(ctx context.Context) (Sandbox, error) {
	return c, nil
}

// Prune reclaims container sandbox resources.
// This is the container-specific cleanup boundary where implementations should
// stop and remove this sandbox container.
func (c *ContainerSandbox) Prune(ctx context.Context) error {
	containerName := strings.TrimSpace(c.cfg.ContainerName)
	if containerName == "" {
		return nil
	}

	var firstErr error
	if c.cli != nil {
		if err := c.stopAndRemoveContainer(ctx, containerName); err != nil {
			firstErr = err
		}
	}
	if err := removeRegistryEntry(c.registryPath(), containerName); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

// Exec ensures the container is ready and runs the requested command inside the sandbox.
func (c *ContainerSandbox) Exec(ctx context.Context, req ExecRequest) (*ExecResult, error) {
	return aggregateExecStream(func(onEvent func(ExecEvent) error) (*ExecResult, error) {
		return c.ExecStream(ctx, req, onEvent)
	})
}

func (c *ContainerSandbox) ExecStream(
	ctx context.Context,
	req ExecRequest,
	onEvent func(ExecEvent) error,
) (*ExecResult, error) {
	if c.startErr != nil {
		return nil, c.startErr
	}
	execCtx := ctx
	cancel := func() {}
	if req.TimeoutMs > 0 {
		execCtx, cancel = context.WithTimeout(ctx, time.Duration(req.TimeoutMs)*time.Millisecond)
	} else if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		execCtx, cancel = context.WithTimeout(ctx, 30*time.Second)
	}
	defer cancel()
	if err := c.ensureContainer(execCtx); err != nil {
		return nil, err
	}

	cmd, wd, err := c.buildExecCommand(req)
	if err != nil {
		return nil, err
	}

	execResp, err := c.cli.ContainerExecCreate(execCtx, c.cfg.ContainerName, container.ExecOptions{
		Cmd:          cmd,
		WorkingDir:   wd,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return nil, fmt.Errorf("docker exec create failed: %w", err)
	}

	attach, err := c.cli.ContainerExecAttach(execCtx, execResp.ID, container.ExecStartOptions{})
	if err != nil {
		return nil, fmt.Errorf("docker exec attach failed: %w", err)
	}

	// Prevent stdcopy.StdCopy from blocking indefinitely if the container hangs.
	// We force close the hijacked connection when the context times out.
	go func() {
		<-execCtx.Done()
		attach.Close()
	}()
	defer attach.Close()

	var stdout, stderr bytes.Buffer
	stdoutWriter := &execStreamWriter{
		eventType: ExecEventStdout,
		onEvent:   onEvent,
		buffer:    &stdout,
	}
	stderrWriter := &execStreamWriter{
		eventType: ExecEventStderr,
		onEvent:   onEvent,
		buffer:    &stderr,
	}
	_, err = stdcopy.StdCopy(stdoutWriter, stderrWriter, attach.Reader)
	if err != nil && err != io.EOF {
		if execCtx.Err() != nil {
			return nil, execCtx.Err()
		}
		return nil, fmt.Errorf("docker exec output read failed: %w", err)
	}

	exitCode, err := c.waitExecDone(execCtx, execResp.ID)
	if err != nil {
		return nil, err
	}
	if onEvent != nil {
		if err := onEvent(ExecEvent{Type: ExecEventExit, ExitCode: exitCode}); err != nil {
			return nil, err
		}
	}

	return &ExecResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}, nil
}

// Fs returns the filesystem bridge bound to the sandbox container.
func (c *ContainerSandbox) Fs() FsBridge {
	if c.startErr != nil {
		return &errorFS{err: c.startErr}
	}
	return c.fs
}

func (c *ContainerSandbox) GetWorkspace(ctx context.Context) string {
	return c.cfg.Workdir
}

func (c *ContainerSandbox) ensureContainer(ctx context.Context) error {
	inspect, err := c.cli.ContainerInspect(ctx, c.cfg.ContainerName)
	if err != nil {
		return c.createAndStart(ctx)
	}

	now := time.Now().UnixMilli()
	regPath := c.registryPath()
	registryMu.Lock()
	data, regErr := loadRegistry(regPath)
	registryMu.Unlock()
	if regErr != nil {
		return fmt.Errorf("sandbox registry load failed: %w", regErr)
	}

	var existing *registryEntry
	for i := range data.Entries {
		if data.Entries[i].ContainerName == c.cfg.ContainerName {
			existing = &data.Entries[i]
			break
		}
	}

	if c.cfg.WorkspaceAccess == string(config.WorkspaceAccessNone) && strings.TrimSpace(c.cfg.Workspace) != "" &&
		strings.TrimSpace(c.cfg.AgentWorkspace) != "" {
		if err := syncAgentWorkspace(c.cfg.AgentWorkspace, c.cfg.Workspace); err != nil {
			logger.WarnCF("sandbox", "failed to sync agent workspace", map[string]any{"error": err})
		}
	}

	hashMismatch := existing != nil && existing.ConfigHash != "" && existing.ConfigHash != c.hash
	if hashMismatch {
		hot := inspect.State.Running && (now-existing.LastUsedAtMs) < int64((5*time.Minute)/time.Millisecond)
		if !hot {
			_ = c.cli.ContainerRemove(ctx, c.cfg.ContainerName, container.RemoveOptions{Force: true})
			_ = removeRegistryEntry(regPath, c.cfg.ContainerName)
			return c.createAndStart(ctx)
		}
		// Container is actively running; recreating it now would disrupt
		// in-flight work. Log a warning so operators can detect configuration drift.
		// The container will be recreated on the next cold start or prune cycle.
		logger.WarnCF(
			"sandbox",
			"container config hash mismatch but container is hot; skipping recreate",
			map[string]any{
				"container": c.cfg.ContainerName,
				"want":      c.hash,
				"got":       existing.ConfigHash,
			},
		)
	}

	if !inspect.State.Running {
		if err := c.cli.ContainerStart(ctx, c.cfg.ContainerName, container.StartOptions{}); err != nil {
			return fmt.Errorf("docker container start failed: %w", err)
		}
	}

	createdAt := now
	if existing != nil && existing.CreatedAtMs > 0 {
		createdAt = existing.CreatedAtMs
	}
	return upsertRegistryEntry(regPath, registryEntry{
		ContainerName: c.cfg.ContainerName,
		Image:         c.cfg.Image,
		ConfigHash:    c.hash,
		CreatedAtMs:   createdAt,
		LastUsedAtMs:  now,
	})
}

func (c *ContainerSandbox) createAndStart(ctx context.Context) error {
	cfg := &container.Config{
		Image:      c.cfg.Image,
		Cmd:        []string{"sleep", "infinity"},
		Entrypoint: []string{},
		WorkingDir: c.cfg.Workdir,
		User:       strings.TrimSpace(c.cfg.User),
		Env:        c.containerEnv(),
	}
	hostCfg, err := c.hostConfig()
	if err != nil {
		return err
	}
	resp, createErr := c.cli.ContainerCreate(ctx, cfg, hostCfg, &network.NetworkingConfig{}, nil, c.cfg.ContainerName)
	if createErr != nil {
		return fmt.Errorf("docker container create failed: %w", createErr)
	}
	if resp.ID == "" {
		return fmt.Errorf("docker container create returned empty id")
	}
	if err := c.cli.ContainerStart(ctx, c.cfg.ContainerName, container.StartOptions{}); err != nil {
		return fmt.Errorf("docker container start failed: %w", err)
	}
	if err := c.runSetupCommand(ctx); err != nil {
		_ = c.cli.ContainerRemove(ctx, c.cfg.ContainerName, container.RemoveOptions{Force: true})
		return err
	}

	now := time.Now().UnixMilli()
	return upsertRegistryEntry(c.registryPath(), registryEntry{
		ContainerName: c.cfg.ContainerName,
		Image:         c.cfg.Image,
		ConfigHash:    c.hash,
		CreatedAtMs:   now,
		LastUsedAtMs:  now,
	})
}

func (c *ContainerSandbox) binds() []string {
	binds := make([]string, 0, 1+len(c.cfg.Binds))
	workspace := strings.TrimSpace(c.cfg.Workspace)

	// Determine the effective host directory to mount
	var hostDir string
	if workspace != "" {
		if abs, err := filepath.Abs(workspace); err == nil {
			hostDir = abs
		}
	}

	if hostDir != "" {
		if c.cfg.WorkspaceAccess == string(config.WorkspaceAccessNone) {
			// Ensure the isolated directory exists on the host so Docker doesn't create it as root
			_ = os.MkdirAll(hostDir, 0o755)
		}
		// Add :Z flag for SELinux (Podman) to label the content with a private unshared label.
		// This fixes errors like: "crun: getcwd: Operation not permitted: OCI permission denied"
		switch config.WorkspaceAccess(c.cfg.WorkspaceAccess) {
		case config.WorkspaceAccessRO:
			binds = append(binds, fmt.Sprintf("%s:%s:ro,Z", hostDir, c.cfg.Workdir))
		case config.WorkspaceAccessRW, config.WorkspaceAccessNone:
			binds = append(binds, fmt.Sprintf("%s:%s:rw,Z", hostDir, c.cfg.Workdir))
		default:
			// Default to no mount for unknown access types
		}
	}
	for _, bind := range c.cfg.Binds {
		if strings.TrimSpace(bind) != "" {
			binds = append(binds, strings.TrimSpace(bind))
		}
	}
	return binds
}

func (c *ContainerSandbox) registryPath() string {
	return filepath.Join(c.sandboxStateDir(), defaultSandboxRegistryFile)
}

func (c *ContainerSandbox) sandboxStateDir() string {
	return filepath.Join(infra.ResolveHomeDir(), "sandbox")
}

func (c *ContainerSandbox) stopAndRemoveContainer(ctx context.Context, containerName string) error {
	timeout := 5
	_ = c.cli.ContainerStop(ctx, containerName, container.StopOptions{Timeout: &timeout})
	if err := c.cli.ContainerRemove(ctx, containerName, container.RemoveOptions{Force: true}); err != nil {
		return err
	}
	return nil
}

func stopAndRemoveContainerByName(ctx context.Context, containerName string) error {
	name := strings.TrimSpace(containerName)
	if name == "" {
		return nil
	}
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()

	timeout := 5
	_ = cli.ContainerStop(ctx, name, container.StopOptions{Timeout: &timeout})
	if err := cli.ContainerRemove(ctx, name, container.RemoveOptions{Force: true}); err != nil {
		return err
	}
	return nil
}

func shouldPruneEntry(cfg ContainerSandboxConfig, nowMs int64, e registryEntry) bool {
	idleMs := nowMs - e.LastUsedAtMs
	ageMs := nowMs - e.CreatedAtMs
	return (cfg.PruneIdleHours > 0 && idleMs > int64(cfg.PruneIdleHours)*int64(time.Hour/time.Millisecond)) ||
		(cfg.PruneMaxAgeDays > 0 && ageMs > int64(cfg.PruneMaxAgeDays)*24*int64(time.Hour/time.Millisecond))
}

func osTempDir() string {
	if d := strings.TrimSpace(os.TempDir()); d != "" {
		return d
	}
	return "."
}

func (c *ContainerSandbox) buildExecCommand(req ExecRequest) ([]string, string, error) {
	workingDir := c.cfg.Workdir
	if strings.TrimSpace(req.WorkingDir) != "" {
		resolved, err := resolveContainerPathWithRoot(c.cfg.Workdir, req.WorkingDir)
		if err != nil {
			return nil, "", err
		}
		workingDir = resolved
	}

	if len(req.Args) > 0 {
		return append([]string{req.Command}, req.Args...), workingDir, nil
	}
	if strings.TrimSpace(req.Command) == "" {
		return nil, "", fmt.Errorf("empty command")
	}
	return []string{"sh", "-lc", req.Command}, workingDir, nil
}

func (c *ContainerSandbox) waitExecDone(ctx context.Context, execID string) (int, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return 1, ctx.Err()
		case <-ticker.C:
			ins, err := c.cli.ContainerExecInspect(ctx, execID)
			if err != nil {
				return 1, fmt.Errorf("docker exec inspect failed: %w", err)
			}
			if !ins.Running {
				return ins.ExitCode, nil
			}
		}
	}
}

type execStreamWriter struct {
	eventType ExecEventType
	onEvent   func(ExecEvent) error
	buffer    *bytes.Buffer
}

func (w *execStreamWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	chunk := append([]byte(nil), p...)
	if w.buffer != nil {
		_, _ = w.buffer.Write(chunk)
	}
	if w.onEvent != nil {
		if err := w.onEvent(ExecEvent{Type: w.eventType, Chunk: chunk}); err != nil {
			return 0, err
		}
	}
	return len(p), nil
}

type containerFS struct {
	sb *ContainerSandbox
}

func (f *containerFS) ReadFile(ctx context.Context, p string) ([]byte, error) {
	if err := f.sb.ensureContainer(ctx); err != nil {
		return nil, err
	}
	containerPath, err := resolveContainerPathWithRoot(f.sb.cfg.Workdir, p)
	if err != nil {
		return nil, err
	}

	rc, _, err := f.sb.cli.CopyFromContainer(ctx, f.sb.cfg.ContainerName, containerPath)
	if err != nil {
		return nil, fmt.Errorf("docker copy from container failed: %w", err)
	}
	defer rc.Close()

	tr := tar.NewReader(rc)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tar read failed: %w", err)
		}
		if hdr.Typeflag == tar.TypeReg || hdr.Typeflag == tar.TypeRegA {
			content, err := io.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("tar file read failed: %w", err)
			}
			return content, nil
		}
	}
	return nil, fmt.Errorf("file not found in container %s: %w", containerPath, fs.ErrNotExist)
}

func (f *containerFS) WriteFile(ctx context.Context, p string, data []byte, mkdir bool) error {
	if err := f.sb.ensureContainer(ctx); err != nil {
		return err
	}
	containerPath, err := resolveContainerPathWithRoot(f.sb.cfg.Workdir, p)
	if err != nil {
		return err
	}

	// Build a script that optionally creates the parent directory and then writes the file via cat.
	// This ensures proper ownership and permissions as the configured container user.
	script := `set -eu; cat >"$1"`
	if mkdir {
		script = `set -eu; dir=$(dirname -- "$1"); if [ "$dir" != "." ]; then mkdir -p -- "$dir"; fi; cat >"$1"`
	}

	execResp, err := f.sb.cli.ContainerExecCreate(ctx, f.sb.cfg.ContainerName, container.ExecOptions{
		Cmd:          []string{"sh", "-c", script, "picoclaw-fs-write", containerPath},
		User:         f.sb.cfg.User, // Use configured user to preserve ownership
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return fmt.Errorf("docker exec create failed: %w", err)
	}

	attach, err := f.sb.cli.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{})
	if err != nil {
		return fmt.Errorf("docker exec attach failed: %w", err)
	}
	defer attach.Close()

	// Write data to the hijacked connection's stdin
	if _, err = attach.Conn.Write(data); err != nil {
		return fmt.Errorf("failed to write data to container: %w", err)
	}

	// Close the write side of the connection so cat receives EOF and terminates
	if conn, ok := attach.Conn.(interface{ CloseWrite() error }); ok {
		_ = conn.CloseWrite()
	}

	// Wait for the exec process to complete and check the exit code
	var stdout, stderr bytes.Buffer
	_, _ = stdcopy.StdCopy(&stdout, &stderr, attach.Reader)

	inspect, err := f.sb.cli.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return fmt.Errorf("failed to inspect write process: %w", err)
	}
	if inspect.ExitCode != 0 {
		return fmt.Errorf("file write failed with code %d: %s", inspect.ExitCode, stderr.String())
	}

	return nil
}

func (f *containerFS) ReadDir(ctx context.Context, p string) ([]os.DirEntry, error) {
	if err := f.sb.ensureContainer(ctx); err != nil {
		return nil, err
	}
	containerPath, err := resolveContainerPathWithRoot(f.sb.cfg.Workdir, p)
	if err != nil {
		return nil, err
	}

	rc, _, err := f.sb.cli.CopyFromContainer(ctx, f.sb.cfg.ContainerName, containerPath)
	if err != nil {
		return nil, fmt.Errorf("docker copy from container failed: %w", err)
	}
	defer rc.Close()

	var entries []os.DirEntry
	tr := tar.NewReader(rc)
	entries, err = parseTopLevelDirEntriesFromTar(tr)
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func parseTopLevelDirEntriesFromTar(tr *tar.Reader) ([]os.DirEntry, error) {
	var entries []os.DirEntry
	seen := make(map[string]struct{})
	rootName := ""
	first := true

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tar read failed: %w", err)
		}

		cleanName := path.Clean(hdr.Name)
		if first {
			first = false
			rootName = cleanName
			continue
		}

		relName := cleanName
		if rootName != "" && rootName != "." {
			if relName == rootName {
				continue
			}
			prefix := rootName + "/"
			relName = strings.TrimPrefix(relName, prefix)
		}
		relName = strings.TrimPrefix(relName, "./")
		if relName == "" || relName == "." {
			continue
		}
		if strings.Contains(relName, "/") {
			// CopyFromContainer is recursive; keep only immediate children.
			continue
		}
		if _, ok := seen[relName]; ok {
			continue
		}
		seen[relName] = struct{}{}

		entries = append(entries, &containerDirEntry{
			name: relName,
			info: hdr.FileInfo(),
		})
	}

	return entries, nil
}

type containerDirEntry struct {
	name string
	info os.FileInfo
}

func (d *containerDirEntry) Name() string               { return d.name }
func (d *containerDirEntry) IsDir() bool                { return d.info.IsDir() }
func (d *containerDirEntry) Type() os.FileMode          { return d.info.Mode().Type() }
func (d *containerDirEntry) Info() (os.FileInfo, error) { return d.info, nil }

func (c *ContainerSandbox) hostDirForContainerPath(containerDir string) (string, bool) {
	if c.cfg.WorkspaceAccess == string(config.WorkspaceAccessRO) {
		return "", false
	}
	workspace := strings.TrimSpace(c.cfg.Workspace)
	if workspace == "" {
		return "", false
	}
	workdir := path.Clean(c.cfg.Workdir)
	if workdir == "" || workdir == "." || workdir == "/" {
		workdir = "/workspace"
	}
	clean := path.Clean(containerDir)
	if clean == workdir {
		abs, err := filepath.Abs(workspace)
		if err != nil {
			return "", false
		}
		return abs, true
	}
	if !strings.HasPrefix(clean, workdir+"/") {
		return "", false
	}
	rel := strings.TrimPrefix(clean, workdir+"/")
	abs, err := filepath.Abs(workspace)
	if err != nil {
		return "", false
	}
	return filepath.Join(abs, filepath.FromSlash(rel)), true
}

func resolveContainerPath(p string) (string, error) {
	return resolveContainerPathWithRoot("/workspace", p)
}

func resolveContainerPathWithRoot(root, p string) (string, error) {
	raw := strings.TrimSpace(p)
	if raw == "" {
		return "", fmt.Errorf("path is required")
	}
	base := path.Clean(strings.TrimSpace(root))
	if base == "" || base == "." || base == "/" {
		base = "/workspace"
	}
	var candidate string
	if strings.HasPrefix(raw, "/") {
		candidate = path.Clean(raw)
	} else {
		candidate = path.Clean(path.Join(base, raw))
	}
	if candidate != base && !strings.HasPrefix(candidate, base+"/") {
		return "", fmt.Errorf("access denied: path is outside container workspace")
	}
	return candidate, nil
}

func shellEscape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

func (c *ContainerSandbox) containerEnv() []string {
	if len(c.cfg.Env) == 0 {
		return nil
	}
	keys := make([]string, 0, len(c.cfg.Env))
	for k := range c.cfg.Env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k+"="+c.cfg.Env[k])
	}
	return out
}

func (c *ContainerSandbox) hostConfig() (*container.HostConfig, error) {
	hostCfg := &container.HostConfig{
		Binds:          c.binds(),
		ReadonlyRootfs: c.cfg.ReadOnlyRoot,
		NetworkMode:    container.NetworkMode(strings.TrimSpace(c.cfg.Network)),
		CapDrop:        c.cfg.CapDrop,
		DNS:            c.cfg.DNS,
		ExtraHosts:     c.cfg.ExtraHosts,
		SecurityOpt:    []string{"no-new-privileges"},
	}
	if c.cfg.PidsLimit > 0 {
		p := c.cfg.PidsLimit
		hostCfg.Resources.PidsLimit = &p
	}

	tmpfs := map[string]string{}
	for _, entry := range c.cfg.Tmpfs {
		e := strings.TrimSpace(entry)
		if e == "" {
			continue
		}
		parts := strings.SplitN(e, ":", 2)
		mountPoint := strings.TrimSpace(parts[0])
		if mountPoint == "" {
			continue
		}
		opts := ""
		if len(parts) > 1 {
			opts = strings.TrimSpace(parts[1])
		}
		tmpfs[mountPoint] = opts
	}
	if len(tmpfs) > 0 {
		hostCfg.Tmpfs = tmpfs
	}

	if sec := strings.TrimSpace(c.cfg.SeccompProfile); sec != "" {
		hostCfg.SecurityOpt = append(hostCfg.SecurityOpt, "seccomp="+sec)
	}
	if app := strings.TrimSpace(c.cfg.ApparmorProfile); app != "" {
		hostCfg.SecurityOpt = append(hostCfg.SecurityOpt, "apparmor="+app)
	}
	if mem := strings.TrimSpace(c.cfg.Memory); mem != "" {
		v, err := parseByteLimit(mem)
		if err != nil {
			return nil, fmt.Errorf("invalid docker.memory: %w", err)
		}
		hostCfg.Memory = v
	}
	if swap := strings.TrimSpace(c.cfg.MemorySwap); swap != "" {
		v, err := parseByteLimit(swap)
		if err != nil {
			return nil, fmt.Errorf("invalid docker.memory_swap: %w", err)
		}
		hostCfg.MemorySwap = v
	}
	if c.cfg.Cpus > 0 {
		hostCfg.NanoCPUs = int64(math.Round(c.cfg.Cpus * 1_000_000_000))
	}
	if len(c.cfg.Ulimits) > 0 {
		keys := make([]string, 0, len(c.cfg.Ulimits))
		for k := range c.cfg.Ulimits {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		hostCfg.Resources.Ulimits = make([]*container.Ulimit, 0, len(keys))
		for _, name := range keys {
			ul := c.cfg.Ulimits[name]
			value, ok := buildDockerUlimit(name, ul)
			if ok {
				hostCfg.Resources.Ulimits = append(hostCfg.Resources.Ulimits, value)
			}
		}
	}
	return hostCfg, nil
}

func parseByteLimit(raw string) (int64, error) {
	if n, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64); err == nil {
		return n, nil
	}
	return units.RAMInBytes(strings.TrimSpace(raw))
}

func buildDockerUlimit(name string, in config.AgentSandboxDockerUlimitValue) (*container.Ulimit, bool) {
	n := strings.TrimSpace(name)
	if n == "" {
		return nil, false
	}
	if in.Value != nil {
		v := *in.Value
		return &container.Ulimit{Name: n, Soft: v, Hard: v}, true
	}
	if in.Soft == nil && in.Hard == nil {
		return nil, false
	}
	soft := int64(0)
	hard := int64(0)
	if in.Soft != nil {
		soft = *in.Soft
	}
	if in.Hard != nil {
		hard = *in.Hard
	}
	if in.Soft == nil {
		soft = hard
	}
	if in.Hard == nil {
		hard = soft
	}
	return &container.Ulimit{Name: n, Soft: soft, Hard: hard}, true
}

func (c *ContainerSandbox) runSetupCommand(ctx context.Context) error {
	cmd := strings.TrimSpace(c.cfg.SetupCommand)
	if cmd == "" {
		return nil
	}
	execResp, err := c.cli.ContainerExecCreate(ctx, c.cfg.ContainerName, container.ExecOptions{
		Cmd:          []string{"sh", "-lc", cmd},
		AttachStdout: true,
		AttachStderr: true,
		WorkingDir:   c.cfg.Workdir,
	})
	if err != nil {
		return fmt.Errorf("docker setup_command create failed: %w", err)
	}
	attach, err := c.cli.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{})
	if err != nil {
		return fmt.Errorf("docker setup_command attach failed: %w", err)
	}
	defer attach.Close()
	var stdout, stderr bytes.Buffer
	_, _ = stdcopy.StdCopy(&stdout, &stderr, attach.Reader)
	exitCode, err := c.waitExecDone(ctx, execResp.ID)
	if err != nil {
		return fmt.Errorf("docker setup_command wait failed: %w", err)
	}
	if exitCode != 0 {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if msg == "" {
			msg = "unknown error"
		}
		return fmt.Errorf("docker setup_command failed (exit=%d): %s", exitCode, msg)
	}
	return nil
}

func computeContainerConfigHash(cfg ContainerSandboxConfig) string {
	type hashUlimit struct {
		Name  string `json:"name"`
		Value *int64 `json:"value,omitempty"`
		Soft  *int64 `json:"soft,omitempty"`
		Hard  *int64 `json:"hard,omitempty"`
	}
	ulimits := make([]hashUlimit, 0, len(cfg.Ulimits))
	for k, v := range cfg.Ulimits {
		name := strings.TrimSpace(k)
		if name == "" {
			continue
		}
		ulimits = append(ulimits, hashUlimit{
			Name:  name,
			Value: v.Value,
			Soft:  v.Soft,
			Hard:  v.Hard,
		})
	}
	sort.Slice(ulimits, func(i, j int) bool { return ulimits[i].Name < ulimits[j].Name })

	envKeys := make([]string, 0, len(cfg.Env))
	for k := range cfg.Env {
		envKeys = append(envKeys, k)
	}
	sort.Strings(envKeys)
	envPairs := make([][2]string, 0, len(envKeys))
	for _, k := range envKeys {
		envPairs = append(envPairs, [2]string{k, cfg.Env[k]})
	}

	payload := struct {
		Image           string       `json:"image"`
		ContainerPrefix string       `json:"container_prefix"`
		Workspace       string       `json:"workspace"`
		AgentWorkspace  string       `json:"agent_workspace"`
		WorkspaceAccess string       `json:"workspace_access"`
		WorkspaceRoot   string       `json:"workspace_root"`
		Workdir         string       `json:"workdir"`
		ReadOnlyRoot    bool         `json:"read_only_root"`
		Tmpfs           []string     `json:"tmpfs"`
		Network         string       `json:"network"`
		User            string       `json:"user"`
		CapDrop         []string     `json:"cap_drop"`
		Env             [][2]string  `json:"env"`
		SetupCommand    string       `json:"setup_command"`
		PidsLimit       int64        `json:"pids_limit"`
		Memory          string       `json:"memory"`
		MemorySwap      string       `json:"memory_swap"`
		Cpus            float64      `json:"cpus"`
		Ulimits         []hashUlimit `json:"ulimits"`
		SeccompProfile  string       `json:"seccomp_profile"`
		ApparmorProfile string       `json:"apparmor_profile"`
		DNS             []string     `json:"dns"`
		ExtraHosts      []string     `json:"extra_hosts"`
		Binds           []string     `json:"binds"`
		Cmd             []string     `json:"cmd"`
		Entrypoint      []string     `json:"entrypoint"`
	}{
		Image:           strings.TrimSpace(cfg.Image),
		ContainerPrefix: strings.TrimSpace(cfg.ContainerPrefix),
		Workspace:       strings.TrimSpace(cfg.Workspace),
		AgentWorkspace:  strings.TrimSpace(cfg.AgentWorkspace),
		WorkspaceAccess: strings.TrimSpace(cfg.WorkspaceAccess),
		WorkspaceRoot:   strings.TrimSpace(cfg.WorkspaceRoot),
		Workdir:         strings.TrimSpace(cfg.Workdir),
		ReadOnlyRoot:    cfg.ReadOnlyRoot,
		Tmpfs:           cfg.Tmpfs,
		Network:         strings.TrimSpace(cfg.Network),
		User:            strings.TrimSpace(cfg.User),
		CapDrop:         cfg.CapDrop,
		Env:             envPairs,
		SetupCommand:    strings.TrimSpace(cfg.SetupCommand),
		PidsLimit:       cfg.PidsLimit,
		Memory:          strings.TrimSpace(cfg.Memory),
		MemorySwap:      strings.TrimSpace(cfg.MemorySwap),
		Cpus:            cfg.Cpus,
		Ulimits:         ulimits,
		SeccompProfile:  strings.TrimSpace(cfg.SeccompProfile),
		ApparmorProfile: strings.TrimSpace(cfg.ApparmorProfile),
		DNS:             cfg.DNS,
		ExtraHosts:      cfg.ExtraHosts,
		Binds:           cfg.Binds,
		Cmd:             []string{"sleep", "infinity"},
		Entrypoint:      []string{},
	}
	raw, _ := json.Marshal(payload)
	return computeConfigHash(string(raw))
}
