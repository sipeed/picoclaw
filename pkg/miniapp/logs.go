package miniapp

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)


// apiLogsSnapshot creates a tar.gz snapshot of the current log buffer.
func (h *Handler) apiLogsSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	entries := logger.RecentLogs(logger.DEBUG, "", 300)

	snapshotDir := filepath.Join(h.workspace, "logs", "snapshots")
	if err := os.MkdirAll(snapshotDir, 0o755); err != nil {
		http.Error(w, `{"error":"cannot create snapshot dir"}`, http.StatusInternalServerError)
		return
	}

	id := time.Now().UTC().Format("20060102-150405")
	filename := fmt.Sprintf("picoclaw-logs-%s.tar.gz", id)
	snapshotPath := filepath.Join(snapshotDir, filename)

	// Create tar.gz
	f, err := os.Create(snapshotPath)
	if err != nil {
		http.Error(w, `{"error":"cannot create snapshot file"}`, http.StatusInternalServerError)
		return
	}

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	prefix := fmt.Sprintf("picoclaw-logs-%s/", id)

	// logs.json
	logsJSON, _ := json.MarshalIndent(entries, "", "  ")
	_ = tw.WriteHeader(&tar.Header{
		Name:    prefix + "logs.json",
		Size:    int64(len(logsJSON)),
		Mode:    0o644,
		ModTime: time.Now(),
	})
	_, _ = tw.Write(logsJSON)

	// metadata.json
	hostname, _ := os.Hostname()
	meta := map[string]any{
		"version":     "1",
		"hostname":    hostname,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"entry_count": len(entries),
	}
	metaJSON, _ := json.MarshalIndent(meta, "", "  ")
	_ = tw.WriteHeader(&tar.Header{
		Name:    prefix + "metadata.json",
		Size:    int64(len(metaJSON)),
		Mode:    0o644,
		ModTime: time.Now(),
	})
	_, _ = tw.Write(metaJSON)

	tw.Close()
	gw.Close()
	f.Close()

	// Cleanup old snapshots (>14 days)
	go cleanOldSnapshots(snapshotDir, 14*24*time.Hour)

	downloadURL := fmt.Sprintf("/miniapp/api/logs/snapshot/%s", id)
	writeJSON(w, map[string]string{"id": id, "download_url": downloadURL})
}

// apiLogsSnapshotDownload serves a snapshot tar.gz file.


// apiLogsSnapshotDownload serves a snapshot tar.gz file.
func (h *Handler) apiLogsSnapshotDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/miniapp/api/logs/snapshot/")
	id = filepath.Base(id) // path traversal prevention

	if id == "" || id == "." || id == ".." {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	filename := fmt.Sprintf("picoclaw-logs-%s.tar.gz", id)
	snapshotPath := filepath.Join(h.workspace, "logs", "snapshots", filename)

	if _, err := os.Stat(snapshotPath); os.IsNotExist(err) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	http.ServeFile(w, r, snapshotPath)
}

// cleanOldSnapshots removes snapshot files older than maxAge.


// cleanOldSnapshots removes snapshot files older than maxAge.
func cleanOldSnapshots(dir string, maxAge time.Duration) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-maxAge)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(dir, e.Name()))
		}
	}
}

// initDataMaxAge is the maximum age of initData before it is considered expired.

