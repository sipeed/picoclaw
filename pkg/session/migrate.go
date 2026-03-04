package session

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// MigrateJSONSessions reads JSON session files from jsonDir and imports them

// into the given SessionStore. Successfully migrated files are renamed to

// .json.migrated so they are skipped on subsequent runs.

//

// Individual file errors are logged and skipped (the file remains for retry).

// Returns the number of sessions migrated and the first error encountered, if any.

func MigrateJSONSessions(jsonDir string, store SessionStore) (int, error) {
	entries, err := os.ReadDir(jsonDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}

		return 0, err
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

		path := filepath.Join(jsonDir, name)

		data, err := os.ReadFile(path)
		if err != nil {
			log.Printf("session migrate: read %s: %v", name, err)

			continue
		}

		var sess Session

		if err := json.Unmarshal(data, &sess); err != nil {
			log.Printf("session migrate: parse %s: %v", name, err)

			continue
		}

		if sess.Key == "" {
			log.Printf("session migrate: skip %s: empty key", name)

			continue
		}

		// Create session in store (skip if already exists)

		if existing, _ := store.Get(sess.Key); existing != nil {
			// Already migrated (perhaps from a previous partial run)

			_ = os.Rename(path, path+".migrated")

			migrated++

			continue
		}

		if err := store.Create(sess.Key, nil); err != nil {
			log.Printf("session migrate: create %s: %v", sess.Key, err)

			continue
		}

		// Import messages as a single turn

		if len(sess.Messages) > 0 {
			turn := &Turn{
				SessionKey: sess.Key,

				Kind: TurnNormal,

				Messages: sess.Messages,

				CreatedAt: sess.Created,
			}

			if err := store.Append(sess.Key, turn); err != nil {
				log.Printf("session migrate: append %s: %v", sess.Key, err)

				continue
			}
		}

		// Set summary if present

		if sess.Summary != "" {
			_ = store.SetSummary(sess.Key, sess.Summary)
		}

		// Mark as migrated

		if err := os.Rename(path, path+".migrated"); err != nil {
			log.Printf("session migrate: rename %s: %v", name, err)
		}

		migrated++
	}

	return migrated, nil
}
