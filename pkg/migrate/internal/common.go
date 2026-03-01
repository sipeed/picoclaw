package internal

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func ResolveTargetHome(override string) (string, error) {
	if override != "" {
		return ExpandHome(override), nil
	}
	if envHome := os.Getenv("PICOCLAW_HOME"); envHome != "" {
		return ExpandHome(envHome), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, ".picoclaw"), nil
}

func ExpandHome(path string) string {
	if path == "" {
		return path
	}
	if path[0] == '~' {
		home, _ := os.UserHomeDir()
		if len(path) > 1 && path[1] == '/' {
			return home + path[1:]
		}
		return home
	}
	return path
}

func ResolveWorkspace(homeDir string) string {
	return filepath.Join(homeDir, "workspace")
}

func PlanWorkspaceMigration(
	srcWorkspace, dstWorkspace string,
	migrateableFiles []string,
	migrateableDirs []string,
	force bool,
) ([]Action, error) {
	var actions []Action

	for _, filename := range migrateableFiles {
		src := filepath.Join(srcWorkspace, filename)
		dst := filepath.Join(dstWorkspace, filename)
		action := planFileCopy(src, dst, force)
		if action.Type != ActionSkip || action.Description != "" {
			actions = append(actions, action)
		}
	}

	for _, dirname := range migrateableDirs {
		srcDir := filepath.Join(srcWorkspace, dirname)
		if _, err := os.Stat(srcDir); os.IsNotExist(err) {
			continue
		}
		dirActions, err := planDirCopy(srcDir, filepath.Join(dstWorkspace, dirname), force)
		if err != nil {
			return nil, err
		}
		actions = append(actions, dirActions...)
	}

	return actions, nil
}

func planFileCopy(src, dst string, force bool) Action {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return Action{
			Type:        ActionSkip,
			Source:      src,
			Target:      dst,
			Description: "source file not found",
		}
	}

	_, dstExists := os.Stat(dst)
	if dstExists == nil && !force {
		return Action{
			Type:        ActionBackup,
			Source:      src,
			Target:      dst,
			Description: "destination exists, will backup and overwrite",
		}
	}

	return Action{
		Type:        ActionCopy,
		Source:      src,
		Target:      dst,
		Description: "copy file",
	}
}

func planDirCopy(srcDir, dstDir string, force bool) ([]Action, error) {
	var actions []Action

	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		dst := filepath.Join(dstDir, relPath)

		if info.IsDir() {
			actions = append(actions, Action{
				Type:        ActionCreateDir,
				Target:      dst,
				Description: "create directory",
			})
			return nil
		}

		action := planFileCopy(path, dst, force)
		actions = append(actions, action)
		return nil
	})

	return actions, err
}

func RelPath(path, base string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return filepath.Base(path)
	}
	return rel
}

func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	info, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
