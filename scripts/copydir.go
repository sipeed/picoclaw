package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: go run scripts/copydir.go <src> <dst>\n")
		os.Exit(2)
	}

	src := os.Args[1]
	dst := os.Args[2]

	if err := os.RemoveAll(dst); err != nil {
		fmt.Fprintf(os.Stderr, "remove %s: %v\n", dst, err)
		os.Exit(1)
	}

	if err := copyTree(src, dst); err != nil {
		fmt.Fprintf(os.Stderr, "copy %s -> %s: %v\n", src, dst, err)
		os.Exit(1)
	}
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
