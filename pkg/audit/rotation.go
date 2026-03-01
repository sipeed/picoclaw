package audit

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

// rotatingWriter handles file I/O with rotation based on size and age.
//
// Rotation strategy:
//   - Daily rotation: New file created at midnight
//   - Size-based: File rotated when exceeding MaxSizeMB
//   - Cleanup: Old files deleted after MaxAgeDays
//   - Backup limit: Only MaxBackups files retained
//   - Compression: Old files optionally gzip compressed
//
// The filename pattern is: audit-DDMMYYYY.log[.N][.gz]
type rotatingWriter struct {
	baseDir    string
	baseName   string
	extension  string
	rotation   config.RotationConfig
	format     string

	// Current file state
	currentFile *os.File
	currentSize int64
	currentDate string

	// Synchronization
	mu sync.Mutex
}

// newRotatingWriter creates a new rotating writer.
func newRotatingWriter(baseDir string, rotation config.RotationConfig, format string) (*rotatingWriter, error) {
	rw := &rotatingWriter{
		baseDir:   baseDir,
		baseName:  "audit-",
		extension: ".log",
		rotation:  rotation,
		format:    format,
	}

	// Open initial file
	if err := rw.rotate(); err != nil {
		return nil, err
	}

	return rw, nil
}

// Write writes data to the current log file, rotating if necessary.
func (rw *rotatingWriter) Write(data []byte) error {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	// Check if rotation needed
	today := time.Now().Format("02012006")
	needsRotation := false

	if today != rw.currentDate {
		// Day changed
		needsRotation = true
	} else if rw.rotation.MaxSizeMB > 0 {
		// Check size limit
		maxSize := int64(rw.rotation.MaxSizeMB) * 1024 * 1024
		if rw.currentSize+int64(len(data)) > maxSize {
			needsRotation = true
		}
	}

	if needsRotation {
		if err := rw.rotate(); err != nil {
			return err
		}
	}

	// Write data
	n, err := rw.currentFile.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to log file: %w", err)
	}

	rw.currentSize += int64(n)
	return nil
}

// Close closes the current log file.
func (rw *rotatingWriter) Close() error {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	if rw.currentFile != nil {
		return rw.currentFile.Close()
	}
	return nil
}

// rotate performs log rotation:
// 1. Close current file
// 2. Compress old file if enabled
// 3. Open new file for today
// 4. Clean up old files
func (rw *rotatingWriter) rotate() error {
	// Close current file if open
	if rw.currentFile != nil {
		if err := rw.currentFile.Close(); err != nil {
			return fmt.Errorf("failed to close current log file: %w", err)
		}

		// Compress the old file if compression is enabled
		if rw.rotation.Compress {
			oldPath := rw.currentFile.Name()
			go rw.compressFile(oldPath) // Async compression
		}
	}

	// Update current date
	rw.currentDate = time.Now().Format("02012006")

	// Generate new filename
	newPath := filepath.Join(rw.baseDir, rw.baseName+rw.currentDate+rw.extension)

	// Check if file already exists (from previous run)
	// If so, find next available number
	if _, err := os.Stat(newPath); err == nil {
		newPath = rw.findNextAvailableName()
	}

	// Create new file with secure permissions
	file, err := os.OpenFile(newPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("failed to create new log file: %w", err)
	}

	rw.currentFile = file
	rw.currentSize = 0

	// Clean up old files asynchronously
	go rw.cleanup()

	return nil
}

// findNextAvailableName finds the next available numbered filename.
func (rw *rotatingWriter) findNextAvailableName() string {
	base := filepath.Join(rw.baseDir, rw.baseName+rw.currentDate)
	
	for i := 1; i < 1000; i++ {
		candidate := fmt.Sprintf("%s.%d%s", base, i, rw.extension)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
	
	// Fallback with timestamp
	return fmt.Sprintf("%s.%d%s", base, time.Now().Unix(), rw.extension)
}

// compressFile compresses a log file with gzip.
func (rw *rotatingWriter) compressFile(path string) {
	src, err := os.Open(path)
	if err != nil {
		return
	}
	defer src.Close()

	dstPath := path + ".gz"
	dst, err := os.Create(dstPath)
	if err != nil {
		return
	}
	defer dst.Close()

	gz := gzip.NewWriter(dst)
	defer gz.Close()

	if _, err := io.Copy(gz, src); err != nil {
		// Clean up partial file
		os.Remove(dstPath)
		return
	}

	// Remove original file after successful compression
	src.Close()
	os.Remove(path)
}

// cleanup removes old log files based on rotation settings.
func (rw *rotatingWriter) cleanup() {
	files, err := rw.listLogFiles()
	if err != nil {
		return
	}

	// Sort by modification time (oldest first)
	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime.Before(files[j].ModTime)
	})

	now := time.Now()

	for _, file := range files {
		shouldDelete := false

		// Check age limit
		if rw.rotation.MaxAgeDays > 0 {
			maxAge := time.Duration(rw.rotation.MaxAgeDays) * 24 * time.Hour
			if now.Sub(file.ModTime) > maxAge {
				shouldDelete = true
			}
		}

		// Check backup limit (keep most recent MaxBackups)
		if rw.rotation.MaxBackups > 0 && len(files) > rw.rotation.MaxBackups {
			shouldDelete = true
		}

		if shouldDelete {
			os.Remove(file.Path)
			// Remove from list so count is accurate
			files = files[1:]
		}
	}
}

// logFileInfo holds information about a log file for cleanup.
type logFileInfo struct {
	Path    string
	ModTime time.Time
}

// listLogFiles returns all audit log files in the base directory.
func (rw *rotatingWriter) listLogFiles() ([]logFileInfo, error) {
	entries, err := os.ReadDir(rw.baseDir)
	if err != nil {
		return nil, err
	}

	var files []logFileInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Match audit-*.log or audit-*.log.gz
		if !strings.HasPrefix(name, rw.baseName) {
			continue
		}
		if !strings.HasSuffix(name, rw.extension) && !strings.HasSuffix(name, rw.extension+".gz") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		files = append(files, logFileInfo{
			Path:    filepath.Join(rw.baseDir, name),
			ModTime: info.ModTime(),
		})
	}

	return files, nil
}
