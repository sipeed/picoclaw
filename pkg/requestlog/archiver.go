package requestlog

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

type Archiver struct {
	config   Config
	logDir   string
	stopChan chan struct{}
}

func NewArchiver(cfg Config, logDir string) *Archiver {
	return &Archiver{
		config:   cfg,
		logDir:   logDir,
		stopChan: make(chan struct{}),
	}
}

func (a *Archiver) Start() error {
	if !a.config.Enabled {
		return nil
	}

	interval, err := parseInterval(a.config.ArchiveInterval)
	if err != nil {
		logger.ErrorCF("requestlog.archiver", "Invalid archive interval", map[string]any{
			"interval": a.config.ArchiveInterval,
			"error":    err.Error(),
		})
		return err
	}

	go a.runScheduler(interval)
	return nil
}

func (a *Archiver) Stop() {
	close(a.stopChan)
}

func (a *Archiver) runScheduler(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-a.stopChan:
			return
		case <-ticker.C:
			if err := a.Archive(); err != nil {
				logger.ErrorCF("requestlog.archiver", "Archive failed", map[string]any{
					"error": err.Error(),
				})
			}
		}
	}
}

func (a *Archiver) Archive() error {
	files, err := os.ReadDir(a.logDir)
	if err != nil {
		return err
	}

	now := time.Now()
	cutoff := now.AddDate(0, 0, -a.config.RetentionDays)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		if !strings.HasSuffix(name, ".jsonl") {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		if info.ModTime().After(cutoff) {
			continue
		}

		if a.config.CompressArchive {
			if err := a.compressFile(filepath.Join(a.logDir, name)); err != nil {
				logger.ErrorCF("requestlog.archiver", "Failed to compress file", map[string]any{
					"file":  name,
					"error": err.Error(),
				})
			}
		}
	}

	return a.cleanupOldFiles()
}

func (a *Archiver) compressFile(srcPath string) error {
	dstPath := srcPath + ".tar.gz"

	if _, err := os.Stat(dstPath); err == nil {
		return nil
	}

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	gzWriter := gzip.NewWriter(dstFile)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	info, err := srcFile.Stat()
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}

	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}

	if _, err := io.Copy(tarWriter, srcFile); err != nil {
		return err
	}

	if err := os.Remove(srcPath); err != nil {
		logger.WarnCF("requestlog.archiver", "Failed to remove original file after compression", map[string]any{
			"file":  srcPath,
			"error": err.Error(),
		})
	}

	logger.InfoCF("requestlog.archiver", "File compressed", map[string]any{
		"source":     srcPath,
		"compressed": dstPath,
	})
	return nil
}

func (a *Archiver) cleanupOldFiles() error {
	files, err := os.ReadDir(a.logDir)
	if err != nil {
		return err
	}

	type fileInfo struct {
		name    string
		modTime time.Time
		size    int64
	}

	var fileInfos []fileInfo
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			continue
		}
		fileInfos = append(fileInfos, fileInfo{
			name:    file.Name(),
			modTime: info.ModTime(),
			size:    info.Size(),
		})
	}

	sort.Slice(fileInfos, func(i, j int) bool {
		return fileInfos[i].modTime.Before(fileInfos[j].modTime)
	})

	totalSize := int64(0)
	for _, fi := range fileInfos {
		totalSize += fi.size
	}

	maxSize := int64(a.config.MaxFiles) * int64(a.config.MaxFileSizeMB) * 1024 * 1024

	for totalSize > maxSize && len(fileInfos) > 0 {
		oldest := fileInfos[0]
		fileInfos = fileInfos[1:]

		path := filepath.Join(a.logDir, oldest.name)
		if err := os.Remove(path); err != nil {
			logger.WarnCF("requestlog.archiver", "Failed to remove old file", map[string]any{
				"file":  oldest.name,
				"error": err.Error(),
			})
			continue
		}

		totalSize -= oldest.size
		logger.InfoCF("requestlog.archiver", "Old file removed", map[string]any{
			"file": oldest.name,
		})
	}

	return nil
}

func parseInterval(intervalStr string) (time.Duration, error) {
	intervalStr = strings.TrimSpace(intervalStr)
	if intervalStr == "" {
		return 24 * time.Hour, nil
	}

	d, err := time.ParseDuration(intervalStr)
	if err == nil {
		return d, nil
	}

	var num int
	var unit string
	_, err = fmt.Sscanf(intervalStr, "%d%s", &num, &unit)
	if err != nil {
		return 0, err
	}

	switch unit {
	case "h":
		return time.Duration(num) * time.Hour, nil
	case "d", "day", "days":
		return time.Duration(num) * 24 * time.Hour, nil
	case "m", "min", "mins":
		return time.Duration(num) * time.Minute, nil
	default:
		return 0, fmt.Errorf("unknown time unit: %s", unit)
	}
}
