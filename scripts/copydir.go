package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: go run scripts/copydir.go <src> <dst>\n")
		os.Exit(2)
	}

	repoRoot, err := findRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "locate repo root: %v\n", err)
		os.Exit(1)
	}

	src, err := normalizePathArg(os.Args[1], repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve src path: %v\n", err)
		os.Exit(1)
	}

	dst, err := normalizePathArg(os.Args[2], repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve dst path: %v\n", err)
		os.Exit(1)
	}

	if err := ensurePathWithinRepo(repoRoot, src); err != nil {
		fmt.Fprintf(os.Stderr, "invalid src path: %v\n", err)
		os.Exit(1)
	}
	if err := ensurePathWithinRepo(repoRoot, dst); err != nil {
		fmt.Fprintf(os.Stderr, "invalid dst path: %v\n", err)
		os.Exit(1)
	}
	if samePath(repoRoot, dst) {
		fmt.Fprintln(os.Stderr, "invalid dst path: destination cannot be repo root")
		os.Exit(1)
	}

	if err := os.RemoveAll(dst); err != nil {
		fmt.Fprintf(os.Stderr, "remove %s: %v\n", dst, err)
		os.Exit(1)
	}

	if err := copyTree(src, dst); err != nil {
		fmt.Fprintf(os.Stderr, "copy %s -> %s: %v\n", src, dst, err)
		os.Exit(1)
	}
}

func findRepoRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("unable to locate copydir.go source path")
	}

	scriptDir := filepath.Dir(file)
	candidate := filepath.Clean(filepath.Join(scriptDir, ".."))
	if err := validateRepoRoot(candidate); err == nil {
		return candidate, nil
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	cur, err := filepath.Abs(wd)
	if err != nil {
		return "", err
	}

	for {
		if err := validateRepoRoot(cur); err == nil {
			return filepath.Clean(cur), nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", fmt.Errorf("could not find repository root from %s", wd)
		}
		cur = parent
	}
}

func validateRepoRoot(root string) error {
	anchors := []string{
		filepath.Join(root, "go.sum"),
		filepath.Join(root, "LICENSE"),
		filepath.Join(root, ".github"),
	}
	for _, anchor := range anchors {
		if _, err := os.Stat(anchor); err != nil {
			return fmt.Errorf("missing repo anchor %s: %w", anchor, err)
		}
	}
	return nil
}

func normalizePathArg(arg, repoRoot string) (string, error) {
	resolved := strings.ReplaceAll(arg, "${codespace}", repoRoot)
	abs, err := filepath.Abs(resolved)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

func ensurePathWithinRepo(repoRoot, path string) error {
	rel, err := filepath.Rel(repoRoot, path)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path %s is outside repository root %s", path, repoRoot)
	}
	return nil
}

func samePath(a, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b)
}

func copyTree(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("source is not a directory: %s", src)
	}

	return filepath.Walk(src, func(path string, entry os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		target := dst
		if rel != "." {
			target = filepath.Join(dst, rel)
		}

		if entry.IsDir() {
			return os.MkdirAll(target, entry.Mode())
		}

		return copyFile(path, target, entry.Mode())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Close()
}
