package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestParseBackupOptions(t *testing.T) {
	opts, showHelp, err := parseBackupOptions([]string{"--with-sessions", "-o", "~/x.tar.gz"})
	if err != nil {
		t.Fatalf("parseBackupOptions returned error: %v", err)
	}
	if showHelp {
		t.Fatalf("expected showHelp=false")
	}
	if !opts.WithSessions {
		t.Fatalf("expected WithSessions=true")
	}
	if opts.OutputPath != "~/x.tar.gz" {
		t.Fatalf("unexpected OutputPath: %q", opts.OutputPath)
	}
}

func TestParseBackupOptionsHelp(t *testing.T) {
	_, showHelp, err := parseBackupOptions([]string{"--help"})
	if err != nil {
		t.Fatalf("parseBackupOptions returned error: %v", err)
	}
	if !showHelp {
		t.Fatalf("expected showHelp=true for --help")
	}
}

func TestParseRestoreOptions(t *testing.T) {
	opts, archive, showHelp, err := parseRestoreOptions([]string{"backup.tar.gz", "--dry-run", "--force"})
	if err != nil {
		t.Fatalf("parseRestoreOptions returned error: %v", err)
	}
	if showHelp {
		t.Fatalf("expected showHelp=false")
	}
	if archive != "backup.tar.gz" {
		t.Fatalf("unexpected archive path: %q", archive)
	}
	if !opts.DryRun || !opts.Force {
		t.Fatalf("expected DryRun and Force true, got DryRun=%v Force=%v", opts.DryRun, opts.Force)
	}
}

func TestParseRestoreOptionsWorkspace(t *testing.T) {
	opts, archive, _, err := parseRestoreOptions([]string{"x.tar.gz", "--workspace", "~/my-ws"})
	if err != nil {
		t.Fatalf("parseRestoreOptions: %v", err)
	}
	if archive != "x.tar.gz" || opts.Workspace != "~/my-ws" {
		t.Fatalf("archive=%q workspace=%q", archive, opts.Workspace)
	}
}

func TestCollectBackupEntries(t *testing.T) {
	homeDir := t.TempDir()
	workspace := filepath.Join(homeDir, "workspace")
	if err := os.MkdirAll(workspace, 0755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}

	mustWriteFile(t, filepath.Join(homeDir, ".picoclaw", "config.json"), "{}")
	mustWriteFile(t, filepath.Join(homeDir, ".picoclaw", "auth.json"), "{}")
	mustWriteFile(t, filepath.Join(workspace, "AGENTS.md"), "# AGENTS")
	mustWriteFile(t, filepath.Join(workspace, "HOOKS.md"), "# HOOKS")
	mustWriteFile(t, filepath.Join(workspace, "IDENTITY.md"), "# IDENTITY")
	mustWriteFile(t, filepath.Join(workspace, "SOUL.md"), "# SOUL")
	mustWriteFile(t, filepath.Join(workspace, "TOOLS.md"), "# TOOLS")
	mustWriteFile(t, filepath.Join(workspace, "USER.md"), "# USER")
	if err := os.MkdirAll(filepath.Join(workspace, "memory"), 0755); err != nil {
		t.Fatalf("mkdir memory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(workspace, "skills"), 0755); err != nil {
		t.Fatalf("mkdir skills: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(workspace, "cron"), 0755); err != nil {
		t.Fatalf("mkdir cron: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(workspace, "sessions"), 0755); err != nil {
		t.Fatalf("mkdir sessions: %v", err)
	}

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: workspace,
			},
		},
	}

	entriesNoSessions := collectBackupEntries(cfg, homeDir, false)
	if !hasArchivePath(entriesNoSessions, "workspace/AGENTS.md") {
		t.Fatalf("expected workspace/AGENTS.md in backup entries")
	}
	if hasArchivePath(entriesNoSessions, "workspace/sessions") {
		t.Fatalf("did not expect workspace/sessions without --with-sessions")
	}

	entriesWithSessions := collectBackupEntries(cfg, homeDir, true)
	if !hasArchivePath(entriesWithSessions, "workspace/sessions") {
		t.Fatalf("expected workspace/sessions with --with-sessions")
	}
}

func TestBackupRestoreRoundtrip(t *testing.T) {
	homeDir := t.TempDir()
	workspace := filepath.Join(homeDir, "workspace")
	if err := os.MkdirAll(workspace, 0755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}

	mustWriteFile(t, filepath.Join(homeDir, ".picoclaw", "config.json"), `{"agents":{"defaults":{"workspace":"`+workspace+`"}}}`)
	mustWriteFile(t, filepath.Join(homeDir, ".picoclaw", "auth.json"), `{}`)
	mustWriteFile(t, filepath.Join(workspace, "AGENTS.md"), "# AGENTS content")
	mustWriteFile(t, filepath.Join(workspace, "USER.md"), "# USER content")

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: workspace,
			},
		},
	}

	entries := collectBackupEntries(cfg, homeDir, false)
	if len(entries) < 4 {
		t.Fatalf("expected at least 4 entries, got %d", len(entries))
	}

	archivePath := filepath.Join(t.TempDir(), "backup.tar.gz")
	if err := createBackupArchive(archivePath, entries); err != nil {
		t.Fatalf("createBackupArchive: %v", err)
	}

	restoreBase := t.TempDir()
	restoreWorkspace := filepath.Join(restoreBase, "restored-ws")
	if err := os.MkdirAll(restoreWorkspace, 0755); err != nil {
		t.Fatalf("mkdir restore workspace: %v", err)
	}

	opts := restoreOptions{Force: true}
	if err := extractBackupArchive(archivePath, filepath.Join(restoreBase, ".picoclaw"), restoreWorkspace, opts); err != nil {
		t.Fatalf("extractBackupArchive: %v", err)
	}

	// Verify restored files
	configPath := filepath.Join(restoreBase, ".picoclaw", "config.json")
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("restored config.json missing: %v", err)
	}
	agentsPath := filepath.Join(restoreWorkspace, "AGENTS.md")
	data, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("reading restored AGENTS.md: %v", err)
	}
	if string(data) != "# AGENTS content" {
		t.Errorf("AGENTS.md content = %q, want %q", string(data), "# AGENTS content")
	}
	userPath := filepath.Join(restoreWorkspace, "USER.md")
	data, err = os.ReadFile(userPath)
	if err != nil {
		t.Fatalf("reading restored USER.md: %v", err)
	}
	if string(data) != "# USER content" {
		t.Errorf("USER.md content = %q, want %q", string(data), "# USER content")
	}
}

func TestArchivePathToDest(t *testing.T) {
	base := "/home/.picoclaw"
	ws := "/home/workspace"
	if got := archivePathToDest("picoclaw/config.json", base, ws); got != filepath.Join(base, "config.json") {
		t.Errorf("picoclaw/config.json -> %q", got)
	}
	if got := archivePathToDest("workspace/AGENTS.md", base, ws); got != filepath.Join(ws, "AGENTS.md") {
		t.Errorf("workspace/AGENTS.md -> %q", got)
	}
	if got := archivePathToDest("unknown/foo", base, ws); got != "" {
		t.Errorf("unknown prefix should return empty, got %q", got)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir parent for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func hasArchivePath(entries []backupEntry, archivePath string) bool {
	for _, entry := range entries {
		if entry.ArchivePath == archivePath {
			return true
		}
	}
	return false
}
