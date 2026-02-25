package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

const (
	archivePrefixPicoclaw  = "picoclaw/"
	archivePrefixWorkspace = "workspace/"
)

type backupOptions struct {
	OutputPath   string
	WithSessions bool
}

type restoreOptions struct {
	DryRun    bool
	Force     bool
	Workspace string
}

type backupEntry struct {
	SourcePath  string
	ArchivePath string
}

func backupCmd() {
	args := os.Args[2:]
	if len(args) == 0 {
		backupCreateCmd(nil)
		return
	}

	switch args[0] {
	case "create":
		backupCreateCmd(args[1:])
	case "list":
		backupListCmd(args[1:])
	case "restore":
		backupRestoreCmd(args[1:])
	case "help", "--help", "-h":
		backupHelp()
	default:
		fmt.Printf("Unknown backup command: %s\n", args[0])
		backupHelp()
	}
}

func backupHelp() {
	fmt.Println("\nBackup commands:")
	fmt.Println("  create                  Create a backup archive (default)")
	fmt.Println("  list                    Show files/directories that would be backed up")
	fmt.Println("  restore <archive>       Restore from a backup archive")
	fmt.Println()
	fmt.Println("Create options:")
	fmt.Println("  -o, --output <path>     Output tar.gz path")
	fmt.Println("  --with-sessions         Include workspace/sessions in backup")
	fmt.Println()
	fmt.Println("Restore options:")
	fmt.Println("  --dry-run               Print what would be restored without writing files")
	fmt.Println("  --force                Overwrite existing files")
	fmt.Println("  --workspace <path>      Restore workspace to this directory (default: from config)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  picoclaw backup create")
	fmt.Println("  picoclaw backup list")
	fmt.Println("  picoclaw backup create --with-sessions")
	fmt.Println("  picoclaw backup create --output ~/Desktop/picoclaw-backup.tar.gz")
	fmt.Println("  picoclaw backup restore ~/.picoclaw/backups/picoclaw-backup-20260101-120000.tar.gz")
	fmt.Println("  picoclaw backup restore backup.tar.gz --dry-run")
	fmt.Println("  picoclaw backup restore backup.tar.gz --workspace ~/my-workspace --force")
}

func backupCreateCmd(args []string) {
	opts, showHelp, err := parseBackupOptions(args)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		backupHelp()
		return
	}
	if showHelp {
		backupHelp()
		return
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error resolving home directory: %v\n", err)
		os.Exit(1)
	}

	entries := collectBackupEntries(cfg, homeDir, opts.WithSessions)
	if len(entries) == 0 {
		fmt.Println("No backup targets found. Run onboard first, then try again.")
		return
	}

	if opts.OutputPath == "" {
		opts.OutputPath = defaultBackupPath(homeDir)
	}
	opts.OutputPath = expandHomePath(opts.OutputPath, homeDir)

	if err := os.MkdirAll(filepath.Dir(opts.OutputPath), 0755); err != nil {
		fmt.Printf("Error creating backup directory: %v\n", err)
		os.Exit(1)
	}

	if err := createBackupArchive(opts.OutputPath, entries); err != nil {
		fmt.Printf("Error creating backup archive: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Backup created: %s\n", opts.OutputPath)
	fmt.Printf("  Included %d path(s)\n", len(entries))
}

func backupListCmd(args []string) {
	opts, showHelp, err := parseBackupOptions(args)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		backupHelp()
		return
	}
	if showHelp {
		backupHelp()
		return
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error resolving home directory: %v\n", err)
		os.Exit(1)
	}

	entries := collectBackupEntries(cfg, homeDir, opts.WithSessions)
	if len(entries) == 0 {
		fmt.Println("No backup targets found.")
		return
	}

	fmt.Println("\nBackup targets:")
	fmt.Println("---------------")
	for _, entry := range entries {
		fmt.Printf("  %s -> %s\n", entry.SourcePath, entry.ArchivePath)
	}
}

func backupRestoreCmd(args []string) {
	opts, archivePath, showHelp, err := parseRestoreOptions(args)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		backupHelp()
		return
	}
	if showHelp {
		backupHelp()
		return
	}
	if archivePath == "" {
		fmt.Println("Error: restore requires an archive path")
		fmt.Println("Usage: picoclaw backup restore <archive> [options]")
		os.Exit(1)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error resolving home directory: %v\n", err)
		os.Exit(1)
	}

	baseDir := filepath.Join(homeDir, ".picoclaw")
	workspaceDir := opts.Workspace
	if workspaceDir == "" {
		cfg, err := loadConfig()
		if err != nil {
			fmt.Println("Error: no config found. Either run 'picoclaw onboard' first or use --workspace <path> to specify where to restore workspace files.")
			os.Exit(1)
		}
		workspaceDir = cfg.WorkspacePath()
	} else {
		workspaceDir = expandHomePath(workspaceDir, homeDir)
	}

	if err := extractBackupArchive(archivePath, baseDir, workspaceDir, opts); err != nil {
		fmt.Printf("Error restoring backup: %v\n", err)
		os.Exit(1)
	}

	if opts.DryRun {
		fmt.Println("Dry run complete. No files were written.")
	} else {
		fmt.Println("Restore complete.")
	}
}

func parseBackupOptions(args []string) (backupOptions, bool, error) {
	opts := backupOptions{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--with-sessions":
			opts.WithSessions = true
		case "-o", "--output":
			if i+1 >= len(args) {
				return opts, false, fmt.Errorf("%s requires a value", args[i])
			}
			opts.OutputPath = args[i+1]
			i++
		case "help", "--help", "-h":
			return opts, true, nil
		default:
			return opts, false, fmt.Errorf("unknown option: %s", args[i])
		}
	}
	return opts, false, nil
}

func parseRestoreOptions(args []string) (restoreOptions, string, bool, error) {
	opts := restoreOptions{}
	var archivePath string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dry-run":
			opts.DryRun = true
		case "--force":
			opts.Force = true
		case "--workspace":
			if i+1 >= len(args) {
				return opts, "", false, fmt.Errorf("--workspace requires a value")
			}
			opts.Workspace = args[i+1]
			i++
		case "help", "--help", "-h":
			return opts, "", true, nil
		default:
			if strings.HasPrefix(args[i], "-") {
				return opts, "", false, fmt.Errorf("unknown option: %s", args[i])
			}
			if archivePath == "" {
				archivePath = args[i]
			}
		}
	}
	return opts, archivePath, false, nil
}

func defaultBackupPath(homeDir string) string {
	timestamp := time.Now().UTC().Format("20060102-150405")
	return filepath.Join(homeDir, ".picoclaw", "backups", fmt.Sprintf("picoclaw-backup-%s.tar.gz", timestamp))
}

func expandHomePath(path string, homeDir string) string {
	if path == "~" {
		return homeDir
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir, path[2:])
	}
	return path
}

func collectBackupEntries(cfg *config.Config, homeDir string, withSessions bool) []backupEntry {
	baseDir := filepath.Join(homeDir, ".picoclaw")
	workspace := cfg.WorkspacePath()

	candidates := []backupEntry{
		{
			SourcePath:  filepath.Join(baseDir, "config.json"),
			ArchivePath: filepath.ToSlash(filepath.Join("picoclaw", "config.json")),
		},
		{
			SourcePath:  filepath.Join(baseDir, "auth.json"),
			ArchivePath: filepath.ToSlash(filepath.Join("picoclaw", "auth.json")),
		},
		{
			SourcePath:  filepath.Join(workspace, "AGENTS.md"),
			ArchivePath: filepath.ToSlash(filepath.Join("workspace", "AGENTS.md")),
		},
		{
			SourcePath:  filepath.Join(workspace, "HOOKS.md"),
			ArchivePath: filepath.ToSlash(filepath.Join("workspace", "HOOKS.md")),
		},
		{
			SourcePath:  filepath.Join(workspace, "IDENTITY.md"),
			ArchivePath: filepath.ToSlash(filepath.Join("workspace", "IDENTITY.md")),
		},
		{
			SourcePath:  filepath.Join(workspace, "SOUL.md"),
			ArchivePath: filepath.ToSlash(filepath.Join("workspace", "SOUL.md")),
		},
		{
			SourcePath:  filepath.Join(workspace, "TOOLS.md"),
			ArchivePath: filepath.ToSlash(filepath.Join("workspace", "TOOLS.md")),
		},
		{
			SourcePath:  filepath.Join(workspace, "USER.md"),
			ArchivePath: filepath.ToSlash(filepath.Join("workspace", "USER.md")),
		},
		{
			SourcePath:  filepath.Join(workspace, "memory"),
			ArchivePath: filepath.ToSlash(filepath.Join("workspace", "memory")),
		},
		{
			SourcePath:  filepath.Join(workspace, "skills"),
			ArchivePath: filepath.ToSlash(filepath.Join("workspace", "skills")),
		},
		{
			SourcePath:  filepath.Join(workspace, "cron"),
			ArchivePath: filepath.ToSlash(filepath.Join("workspace", "cron")),
		},
	}

	if withSessions {
		candidates = append(candidates, backupEntry{
			SourcePath:  filepath.Join(workspace, "sessions"),
			ArchivePath: filepath.ToSlash(filepath.Join("workspace", "sessions")),
		})
	}

	existing := make([]backupEntry, 0, len(candidates))
	for _, entry := range candidates {
		if _, err := os.Stat(entry.SourcePath); err == nil {
			existing = append(existing, entry)
		}
	}
	return existing
}

func createBackupArchive(outputPath string, entries []backupEntry) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzw := gzip.NewWriter(file)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	for _, entry := range entries {
		info, err := os.Stat(entry.SourcePath)
		if err != nil {
			return err
		}
		if info.IsDir() {
			if err := addDirectoryToArchive(tw, entry.SourcePath, entry.ArchivePath); err != nil {
				return err
			}
			continue
		}
		if err := addFileToArchive(tw, entry.SourcePath, entry.ArchivePath); err != nil {
			return err
		}
	}

	return nil
}

func addDirectoryToArchive(tw *tar.Writer, sourceDir, archiveRoot string) error {
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		target := archiveRoot
		if relPath != "." {
			target = filepath.Join(archiveRoot, relPath)
		}
		target = filepath.ToSlash(target)

		if info.IsDir() {
			return addDirHeaderToArchive(tw, info, target)
		}
		if !info.Mode().IsRegular() {
			return nil
		}

		return addFileToArchive(tw, path, target)
	})
}

func addDirHeaderToArchive(tw *tar.Writer, info os.FileInfo, archivePath string) error {
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = strings.TrimSuffix(archivePath, "/") + "/"
	return tw.WriteHeader(header)
}

func addFileToArchive(tw *tar.Writer, sourcePath, archivePath string) error {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = archivePath

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	file, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(tw, file)
	return err
}

// archivePathToDest maps an archive path (e.g. picoclaw/config.json or workspace/AGENTS.md)
// to the destination path on disk. Returns empty string if the path is not under a known prefix.
func archivePathToDest(archivePath, baseDir, workspaceDir string) string {
	archivePath = filepath.ToSlash(archivePath)
	if strings.HasPrefix(archivePath, archivePrefixPicoclaw) {
		rel := strings.TrimPrefix(archivePath, archivePrefixPicoclaw)
		return filepath.Join(baseDir, filepath.FromSlash(rel))
	}
	if strings.HasPrefix(archivePath, archivePrefixWorkspace) {
		rel := strings.TrimPrefix(archivePath, archivePrefixWorkspace)
		return filepath.Join(workspaceDir, filepath.FromSlash(rel))
	}
	return ""
}

func extractBackupArchive(archivePath, baseDir, workspaceDir string, opts restoreOptions) error {
	home, _ := os.UserHomeDir()
	archivePath = expandHomePath(archivePath, home)

	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	restored := 0
	skipped := 0

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		name := filepath.ToSlash(hdr.Name)
		// Skip entries not under our known prefixes
		dest := archivePathToDest(name, baseDir, workspaceDir)
		if dest == "" {
			continue
		}

		if hdr.Typeflag == tar.TypeDir {
			if opts.DryRun {
				fmt.Printf("  [dir]  %s\n", name)
				restored++
				continue
			}
			if _, err := os.Stat(dest); err == nil && !opts.Force {
				skipped++
				continue
			}
			if err := os.MkdirAll(dest, 0755); err != nil {
				return fmt.Errorf("mkdir %s: %w", dest, err)
			}
			restored++
			continue
		}

		if hdr.Typeflag != tar.TypeReg && hdr.Typeflag != tar.TypeRegA {
			continue
		}

		if opts.DryRun {
			fmt.Printf("  [file] %s -> %s\n", name, dest)
			restored++
			continue
		}

		if _, err := os.Stat(dest); err == nil && !opts.Force {
			skipped++
			continue
		}

		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", filepath.Dir(dest), err)
		}

		mode := hdr.FileInfo().Mode()
		out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
		if err != nil {
			return fmt.Errorf("create %s: %w", dest, err)
		}
		_, err = io.Copy(out, tr)
		out.Close()
		if err != nil {
			return fmt.Errorf("write %s: %w", dest, err)
		}
		restored++
	}

	if !opts.DryRun && skipped > 0 {
		fmt.Printf("  Skipped %d existing path(s) (use --force to overwrite)\n", skipped)
	}
	fmt.Printf("  Restored %d path(s)\n", restored)
	return nil
}
