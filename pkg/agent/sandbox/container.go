package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-units"
	"github.com/sipeed/picoclaw/pkg/config"
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
	cfg      ContainerSandboxConfig
	cli      *client.Client
	startErr error
	fs       FsBridge
	hash     string
}

const defaultSandboxRegistryFile = "containers.json"

// NewContainerSandbox creates a container sandbox with normalized defaults and precomputed config hash.
func NewContainerSandbox(cfg ContainerSandboxConfig) *ContainerSandbox {
	if strings.TrimSpace(cfg.Image) == "" {
		cfg.Image = "debian:bookworm-slim"
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
	if len(cfg.CapDrop) == 0 {
		cfg.CapDrop = []string{"ALL"}
	}
	if cfg.Env == nil {
		cfg.Env = map[string]string{"LANG": "C.UTF-8"}
	}
	cfg.WorkspaceAccess = normalizeWorkspaceAccess(cfg.WorkspaceAccess)
	cfg.WorkspaceRoot = strings.TrimSpace(cfg.WorkspaceRoot)
	sb := &ContainerSandbox{cfg: cfg}
	sb.hash = computeContainerConfigHash(cfg)
	sb.fs = &containerFS{sb: sb}
	return sb
}

// Start initializes docker connectivity and validates sandbox runtime requirements.
func (c *ContainerSandbox) Start(ctx context.Context) error {
	if err := validateSandboxSecurity(c.cfg); err != nil {
		c.startErr = err
		return err
	}
	c.cfg.Env = sanitizeEnvVars(c.cfg.Env)
	if strings.TrimSpace(c.cfg.Workspace) != "" && c.cfg.WorkspaceAccess == "none" {
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
		rc, pullErr := c.cli.ImagePull(ctx, c.cfg.Image, image.PullOptions{})
		if pullErr != nil {
			c.startErr = fmt.Errorf("docker image unavailable (%s): %w", c.cfg.Image, pullErr)
			return c.startErr
		}
		defer rc.Close()
		_, _ = io.Copy(io.Discard, rc)
	}

	c.startErr = nil
	return nil
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

func (c *ContainerSandbox) ExecStream(ctx context.Context, req ExecRequest, onEvent func(ExecEvent) error) (*ExecResult, error) {
	if c.startErr != nil {
		return nil, c.startErr
	}
	execCtx := ctx
	cancel := func() {}
	if req.TimeoutMs > 0 {
		execCtx, cancel = context.WithTimeout(ctx, durationMs(req.TimeoutMs))
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
	if _, err := stdcopy.StdCopy(stdoutWriter, stderrWriter, attach.Reader); err != nil && err != io.EOF {
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

	hashMismatch := existing != nil && existing.ConfigHash != "" && existing.ConfigHash != c.hash
	if hashMismatch {
		hot := inspect.State.Running && (now-existing.LastUsedAtMs) < int64((5*time.Minute)/time.Millisecond)
		if !hot {
			_ = c.cli.ContainerRemove(ctx, c.cfg.ContainerName, container.RemoveOptions{Force: true})
			_ = removeRegistryEntry(regPath, c.cfg.ContainerName)
			return c.createAndStart(ctx)
		}
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
	if workspace != "" {
		abs, err := filepath.Abs(workspace)
		if err == nil {
			switch c.cfg.WorkspaceAccess {
			case "ro":
				binds = append(binds, fmt.Sprintf("%s:%s:ro", abs, c.cfg.Workdir))
			case "rw":
				binds = append(binds, fmt.Sprintf("%s:%s:rw", abs, c.cfg.Workdir))
			default:
				binds = append(binds, fmt.Sprintf("%s:%s", abs, c.cfg.Workdir))
			}
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
	return filepath.Join(resolvePicoClawHomeDir(), "state", "sandbox")
}

func resolvePicoClawHomeDir() string {
	if envHome := strings.TrimSpace(os.Getenv("PICOCLAW_HOME")); envHome != "" {
		if abs := resolveAbsPath(expandHomePath(envHome)); strings.TrimSpace(abs) != "" {
			return abs
		}
	}
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		return filepath.Join(home, ".picoclaw")
	}
	return filepath.Join(osTempDir(), ".picoclaw")
}

func (c *ContainerSandbox) stopAndRemoveContainer(ctx context.Context, containerName string) error {
	timeout := 10
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

	timeout := 10
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
	return nil, fmt.Errorf("file not found in container: %s", containerPath)
}

func (f *containerFS) WriteFile(ctx context.Context, p string, data []byte, mkdir bool) error {
	if err := f.sb.ensureContainer(ctx); err != nil {
		return err
	}
	containerPath, err := resolveContainerPathWithRoot(f.sb.cfg.Workdir, p)
	if err != nil {
		return err
	}
	dir := path.Dir(containerPath)
	base := path.Base(containerPath)

	if mkdir {
		hostDir, ok := f.sb.hostDirForContainerPath(dir)
		if ok {
			if err := os.MkdirAll(hostDir, 0o755); err != nil {
				return fmt.Errorf("host mkdir failed: %w", err)
			}
		} else {
			_, err := f.sb.Exec(ctx, ExecRequest{
				Command: "mkdir -p " + shellEscape(dir),
			})
			if err != nil {
				return err
			}
		}
	}

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	if err := tw.WriteHeader(&tar.Header{
		Name: base,
		Mode: 0644,
		Size: int64(len(data)),
	}); err != nil {
		_ = tw.Close()
		return fmt.Errorf("tar header write failed: %w", err)
	}
	if _, err := tw.Write(data); err != nil {
		_ = tw.Close()
		return fmt.Errorf("tar content write failed: %w", err)
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("tar close failed: %w", err)
	}

	if err := f.sb.cli.CopyToContainer(ctx, f.sb.cfg.ContainerName, dir, &buf, container.CopyToContainerOptions{
		AllowOverwriteDirWithFile: true,
	}); err != nil {
		return fmt.Errorf("docker copy to container failed: %w", err)
	}
	return nil
}

func (c *ContainerSandbox) hostDirForContainerPath(containerDir string) (string, bool) {
	if c.cfg.WorkspaceAccess == "ro" {
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
	}
	raw, _ := json.Marshal(payload)
	return computeConfigHash(string(raw))
}
