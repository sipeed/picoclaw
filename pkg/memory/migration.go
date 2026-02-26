package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// jsonSession mirrors pkg/session.Session for migration purposes.
type jsonSession struct {
	Key      string              `json:"key"`
	Messages []providers.Message `json:"messages"`
	Summary  string              `json:"summary,omitempty"`
	Created  time.Time           `json:"created"`
	Updated  time.Time           `json:"updated"`
}

// MigrateFromJSON reads legacy sessions/*.json files from sessionsDir,
// writes them into the Store, and renames each migrated file to
// .json.migrated as a backup. Returns the number of sessions migrated.
//
// Files that fail to parse are logged and skipped. Already-migrated
// files (.json.migrated) are ignored, making the function idempotent.
func MigrateFromJSON(
	ctx context.Context, sessionsDir string, store Store,
) (int, error) {
	entries, err := os.ReadDir(sessionsDir)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("memory: read sessions dir: %w", err)
	}

	migrated := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		// Skip already-migrated files.
		if strings.HasSuffix(name, ".migrated") {
			continue
		}

		srcPath := filepath.Join(sessionsDir, name)

		data, readErr := os.ReadFile(srcPath)
		if readErr != nil {
			log.Printf("memory: migrate: skip %s: %v", name, readErr)
			continue
		}

		var sess jsonSession
		if parseErr := json.Unmarshal(data, &sess); parseErr != nil {
			log.Printf("memory: migrate: skip %s: %v", name, parseErr)
			continue
		}

		// Use the key from the JSON content, not the filename.
		// Filenames are sanitized (":" → "_") but keys are not.
		key := sess.Key
		if key == "" {
			key = strings.TrimSuffix(name, ".json")
		}

		// Use SetHistory (atomic replace) instead of per-message
		// AddFullMessage. This makes migration idempotent: if the
		// process crashes after writing messages but before the
		// rename below, a retry replaces the partial data cleanly
		// instead of duplicating messages.
		if setErr := store.SetHistory(ctx, key, sess.Messages); setErr != nil {
			return migrated, fmt.Errorf(
				"memory: migrate %s: set history: %w",
				name, setErr,
			)
		}

		if sess.Summary != "" {
			if sumErr := store.SetSummary(ctx, key, sess.Summary); sumErr != nil {
				return migrated, fmt.Errorf(
					"memory: migrate %s: set summary: %w",
					name, sumErr,
				)
			}
		}

		// Rename to .migrated as backup (not delete).
		renameErr := os.Rename(srcPath, srcPath+".migrated")
		if renameErr != nil {
			log.Printf("memory: migrate: rename %s: %v", name, renameErr)
		}

		migrated++
	}

	return migrated, nil
}
