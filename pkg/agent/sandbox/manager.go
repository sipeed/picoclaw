package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/routing"
)

// NewFromConfig builds a sandbox instance from config and starts it before returning.
func NewFromConfig(workspace string, restrict bool, cfg *config.Config) Sandbox {
	return NewFromConfigWithAgent(workspace, restrict, cfg, routing.DefaultAgentID)
}

// NewFromConfigWithAgent builds a sandbox instance with an explicit agent ID context.
func NewFromConfigWithAgent(workspace string, restrict bool, cfg *config.Config, agentID string) Sandbox {
	mode := "all"
	scope := "agent"
	workspaceAccess := "none"
	workspaceRoot := "~/.picoclaw/sandboxes"
	image := "debian:bookworm-slim"
	containerPrefix := "picoclaw-sandbox-"
	pruneIdleHours := 24
	pruneMaxAgeDays := 7
	dockerCfg := config.AgentSandboxDockerConfig{}

	if cfg != nil {
		sb := cfg.Agents.Defaults.Sandbox
		if strings.TrimSpace(sb.Mode) != "" {
			mode = strings.TrimSpace(sb.Mode)
		}
		if strings.TrimSpace(sb.Scope) != "" {
			scope = strings.TrimSpace(sb.Scope)
		}
		if strings.TrimSpace(sb.WorkspaceAccess) != "" {
			workspaceAccess = strings.TrimSpace(sb.WorkspaceAccess)
		}
		if strings.TrimSpace(sb.WorkspaceRoot) != "" {
			workspaceRoot = strings.TrimSpace(sb.WorkspaceRoot)
		}
		if strings.TrimSpace(sb.Docker.Image) != "" {
			image = strings.TrimSpace(sb.Docker.Image)
		}
		if strings.TrimSpace(sb.Docker.ContainerPrefix) != "" {
			containerPrefix = strings.TrimSpace(sb.Docker.ContainerPrefix)
		}
		if sb.Prune.IdleHours >= 0 {
			pruneIdleHours = sb.Prune.IdleHours
		}
		if sb.Prune.MaxAgeDays >= 0 {
			pruneMaxAgeDays = sb.Prune.MaxAgeDays
		}
		dockerCfg = sb.Docker
	}

	agentID = routing.NormalizeAgentID(agentID)
	host := NewHostSandbox(workspace, restrict)
	_ = host.Start(context.Background())

	resolvedMode := normalizeSandboxMode(mode)
	if resolvedMode == "off" {
		return host
	}
	resolvedScope := normalizeSandboxScope(scope)
	normalizedAccess := normalizeWorkspaceAccess(workspaceAccess)
	workspaceRootAbs := resolveAbsPath(expandHomePath(workspaceRoot))
	agentWorkspaceAbs := resolveAbsPath(workspace)

	manager := &scopedSandboxManager{
		mode:            resolvedMode,
		scope:           resolvedScope,
		agentID:         agentID,
		host:            host,
		image:           image,
		containerPrefix: containerPrefix,
		workspaceAccess: normalizedAccess,
		workspaceRoot:   workspaceRootAbs,
		agentWorkspace:  agentWorkspaceAbs,
		pruneIdleHours:  pruneIdleHours,
		pruneMaxAgeDays: pruneMaxAgeDays,
		dockerCfg:       dockerCfg,
		scoped:          map[string]Sandbox{},
	}
	manager.fs = &managerFS{m: manager}
	if err := manager.Start(context.Background()); err != nil {
		return NewUnavailableSandbox(fmt.Errorf("container sandbox unavailable: %w", err))
	}
	return manager
}

func normalizeWorkspaceAccess(access string) string {
	v := strings.ToLower(strings.TrimSpace(access))
	switch v {
	case "ro", "rw":
		return v
	default:
		return "none"
	}
}

func normalizeSandboxMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "all", "non-main":
		return strings.ToLower(strings.TrimSpace(mode))
	default:
		return "off"
	}
}

func normalizeSandboxScope(scope string) string {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case "session", "shared":
		return strings.ToLower(strings.TrimSpace(scope))
	default:
		return "agent"
	}
}

func expandHomePath(p string) string {
	raw := strings.TrimSpace(p)
	if raw == "" {
		return raw
	}
	if raw == "~" {
		home, _ := os.UserHomeDir()
		return home
	}
	if strings.HasPrefix(raw, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, raw[2:])
	}
	return raw
}

func resolveAbsPath(p string) string {
	trimmed := strings.TrimSpace(p)
	if trimmed == "" {
		return ""
	}
	if filepath.IsAbs(trimmed) {
		return trimmed
	}
	abs, err := filepath.Abs(trimmed)
	if err != nil {
		return trimmed
	}
	return abs
}

type scopedSandboxManager struct {
	mode            string
	scope           string
	agentID         string
	host            Sandbox
	image           string
	containerPrefix string
	workspaceAccess string
	workspaceRoot   string
	agentWorkspace  string
	pruneIdleHours  int
	pruneMaxAgeDays int
	dockerCfg       config.AgentSandboxDockerConfig

	mu     sync.Mutex
	scoped map[string]Sandbox
	fs     FsBridge

	loopMu   sync.Mutex
	loopStop context.CancelFunc
	loopDone chan struct{}
}

func (m *scopedSandboxManager) Start(ctx context.Context) error {
	if m.mode == "off" {
		return nil
	}
	if _, err := m.getOrCreateSandbox(ctx, m.defaultScopeKey()); err != nil {
		return err
	}
	m.ensurePruneLoop()
	return nil
}

func (m *scopedSandboxManager) Prune(ctx context.Context) error {
	m.stopPruneLoop(ctx)

	m.mu.Lock()
	scoped := make([]Sandbox, 0, len(m.scoped))
	for _, sb := range m.scoped {
		scoped = append(scoped, sb)
	}
	m.mu.Unlock()

	var firstErr error
	for _, sb := range scoped {
		if err := sb.Prune(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (m *scopedSandboxManager) ensurePruneLoop() {
	if m.pruneIdleHours <= 0 && m.pruneMaxAgeDays <= 0 {
		return
	}
	m.loopMu.Lock()
	defer m.loopMu.Unlock()
	if m.loopStop != nil {
		return
	}

	loopCtx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	m.loopStop = cancel
	m.loopDone = done

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer func() {
			ticker.Stop()
			close(done)
		}()
		for {
			select {
			case <-loopCtx.Done():
				return
			case <-ticker.C:
				_ = m.pruneOnce(loopCtx)
			}
		}
	}()
}

func (m *scopedSandboxManager) stopPruneLoop(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	m.loopMu.Lock()
	stop := m.loopStop
	done := m.loopDone
	m.loopStop = nil
	m.loopDone = nil
	m.loopMu.Unlock()
	if stop == nil {
		return
	}
	stop()
	if done == nil {
		return
	}
	select {
	case <-done:
	case <-ctx.Done():
	}
}

func (m *scopedSandboxManager) pruneOnce(ctx context.Context) error {
	if m.pruneIdleHours <= 0 && m.pruneMaxAgeDays <= 0 {
		return nil
	}

	regPath := filepath.Join(resolvePicoClawHomeDir(), "state", "sandbox", defaultSandboxRegistryFile)
	registryMu.Lock()
	data, err := loadRegistry(regPath)
	registryMu.Unlock()
	if err != nil {
		return err
	}

	pruneCfg := ContainerSandboxConfig{
		PruneIdleHours:  m.pruneIdleHours,
		PruneMaxAgeDays: m.pruneMaxAgeDays,
	}
	now := time.Now().UnixMilli()

	m.mu.Lock()
	byContainer := make(map[string]Sandbox, len(m.scoped))
	for _, sb := range m.scoped {
		if containerSb, ok := sb.(*ContainerSandbox); ok {
			byContainer[containerSb.cfg.ContainerName] = sb
		}
	}
	m.mu.Unlock()

	var firstErr error
	for _, entry := range data.Entries {
		if !shouldPruneEntry(pruneCfg, now, entry) {
			continue
		}
		if sb, ok := byContainer[entry.ContainerName]; ok {
			if err := sb.Prune(ctx); err != nil && firstErr == nil {
				firstErr = err
			}
			continue
		}
		if err := stopAndRemoveContainerByName(ctx, entry.ContainerName); err != nil && firstErr == nil {
			firstErr = err
		}
		if err := removeRegistryEntry(regPath, entry.ContainerName); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

func (m *scopedSandboxManager) Exec(ctx context.Context, req ExecRequest) (*ExecResult, error) {
	if !m.shouldSandbox(ctx) {
		return m.host.Exec(ctx, req)
	}
	sb, err := m.getOrCreateSandbox(ctx, m.scopeKeyFromContext(ctx))
	if err != nil {
		return nil, err
	}
	return sb.Exec(ctx, req)
}

func (m *scopedSandboxManager) ExecStream(ctx context.Context, req ExecRequest, onEvent func(ExecEvent) error) (*ExecResult, error) {
	if !m.shouldSandbox(ctx) {
		return m.host.ExecStream(ctx, req, onEvent)
	}
	sb, err := m.getOrCreateSandbox(ctx, m.scopeKeyFromContext(ctx))
	if err != nil {
		return nil, err
	}
	return sb.ExecStream(ctx, req, onEvent)
}

func (m *scopedSandboxManager) Fs() FsBridge {
	return m.fs
}

func (m *scopedSandboxManager) shouldSandbox(ctx context.Context) bool {
	switch m.mode {
	case "all":
		return true
	case "non-main":
		// Phase 2 deferred: `non-main` requires stable session-key propagation
		// across all tool execution paths. For now, keep behavior disabled until
		// the execution context plumbing is finalized.
		return false
	default:
		return false
	}
}

func (m *scopedSandboxManager) mainSessionKey() string {
	return routing.BuildAgentMainSessionKey(m.agentID)
}

func (m *scopedSandboxManager) normalizeSessionKey(raw string) string {
	trimmed := strings.TrimSpace(raw)
	main := m.mainSessionKey()
	if trimmed == "" {
		return main
	}
	if strings.EqualFold(trimmed, "main") || strings.EqualFold(trimmed, main) {
		return main
	}
	if parsed := routing.ParseAgentSessionKey(trimmed); parsed != nil {
		if routing.NormalizeAgentID(parsed.AgentID) == m.agentID && strings.EqualFold(strings.TrimSpace(parsed.Rest), "main") {
			return main
		}
	}
	return trimmed
}

func (m *scopedSandboxManager) scopeKeyFromContext(ctx context.Context) string {
	sessionKey := m.normalizeSessionKey(SessionKeyFromContext(ctx))
	switch m.scope {
	case "shared":
		return "shared"
	case "session":
		return sessionKey
	default:
		if parsed := routing.ParseAgentSessionKey(sessionKey); parsed != nil {
			return "agent:" + routing.NormalizeAgentID(parsed.AgentID)
		}
		return "agent:" + m.agentID
	}
}

func (m *scopedSandboxManager) defaultScopeKey() string {
	return m.scopeKeyFromContext(WithSessionKey(context.Background(), m.mainSessionKey()))
}

func (m *scopedSandboxManager) getOrCreateSandbox(ctx context.Context, scopeKey string) (Sandbox, error) {
	m.mu.Lock()
	if sb, ok := m.scoped[scopeKey]; ok {
		m.mu.Unlock()
		return sb, nil
	}
	sb := m.buildScopedContainerSandbox(scopeKey)
	m.scoped[scopeKey] = sb
	m.mu.Unlock()

	if err := sb.Start(ctx); err != nil {
		m.mu.Lock()
		delete(m.scoped, scopeKey)
		m.mu.Unlock()
		return nil, err
	}
	return sb, nil
}

func (m *scopedSandboxManager) buildScopedContainerSandbox(scopeKey string) Sandbox {
	workspace := m.agentWorkspace
	if m.workspaceAccess == "none" || strings.TrimSpace(workspace) == "" {
		workspace = filepath.Join(m.workspaceRoot, slugScopeKey(scopeKey), "workspace")
	}
	return NewContainerSandbox(ContainerSandboxConfig{
		Image:           m.image,
		ContainerName:   strings.TrimSpace(m.containerPrefix) + slugScopeKey(scopeKey),
		ContainerPrefix: m.containerPrefix,
		Workspace:       workspace,
		AgentWorkspace:  m.agentWorkspace,
		WorkspaceAccess: m.workspaceAccess,
		WorkspaceRoot:   m.workspaceRoot,
		PruneIdleHours:  m.pruneIdleHours,
		PruneMaxAgeDays: m.pruneMaxAgeDays,
		Workdir:         m.dockerCfg.Workdir,
		ReadOnlyRoot:    m.dockerCfg.ReadOnlyRoot,
		Tmpfs:           m.dockerCfg.Tmpfs,
		Network:         m.dockerCfg.Network,
		User:            m.dockerCfg.User,
		CapDrop:         m.dockerCfg.CapDrop,
		Env:             m.dockerCfg.Env,
		SetupCommand:    m.dockerCfg.SetupCommand,
		PidsLimit:       m.dockerCfg.PidsLimit,
		Memory:          m.dockerCfg.Memory,
		MemorySwap:      m.dockerCfg.MemorySwap,
		Cpus:            m.dockerCfg.Cpus,
		Ulimits:         m.dockerCfg.Ulimits,
		SeccompProfile:  m.dockerCfg.SeccompProfile,
		ApparmorProfile: m.dockerCfg.ApparmorProfile,
		DNS:             m.dockerCfg.DNS,
		ExtraHosts:      m.dockerCfg.ExtraHosts,
		Binds:           m.dockerCfg.Binds,
	})
}

type managerFS struct {
	m *scopedSandboxManager
}

func (f *managerFS) ReadFile(ctx context.Context, path string) ([]byte, error) {
	if !f.m.shouldSandbox(ctx) {
		return f.m.host.Fs().ReadFile(ctx, path)
	}
	sb, err := f.m.getOrCreateSandbox(ctx, f.m.scopeKeyFromContext(ctx))
	if err != nil {
		return nil, err
	}
	return sb.Fs().ReadFile(ctx, path)
}

func (f *managerFS) WriteFile(ctx context.Context, path string, data []byte, mkdir bool) error {
	if !f.m.shouldSandbox(ctx) {
		return f.m.host.Fs().WriteFile(ctx, path, data, mkdir)
	}
	sb, err := f.m.getOrCreateSandbox(ctx, f.m.scopeKeyFromContext(ctx))
	if err != nil {
		return err
	}
	return sb.Fs().WriteFile(ctx, path, data, mkdir)
}

var nonAlnum = regexp.MustCompile(`[^a-z0-9._-]+`)

func slugScopeKey(scopeKey string) string {
	raw := strings.ToLower(strings.TrimSpace(scopeKey))
	if raw == "" {
		raw = "default"
	}
	safe := nonAlnum.ReplaceAllString(raw, "-")
	safe = strings.Trim(safe, "-")
	if safe == "" {
		safe = "default"
	}
	if len(safe) > 32 {
		safe = safe[:32]
	}
	sum := sha256.Sum256([]byte(raw))
	return safe + "-" + hex.EncodeToString(sum[:4])
}
