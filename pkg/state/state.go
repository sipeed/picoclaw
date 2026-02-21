package state

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// State represents the persistent state for a workspace.
// It includes information about the last active channel/chat.
type State struct {
	// LastChannel is the last channel used for communication
	LastChannel string `json:"last_channel,omitempty"`

	// LastChatID is the last chat ID used for communication
	LastChatID string `json:"last_chat_id,omitempty"`

	// LastMainChannel is the last channel used by a "main" client type.
	// Used by heartbeat to always target the main (Android app) session.
	LastMainChannel string `json:"last_main_channel,omitempty"`

	// ChannelChatIDs maps each channel name to the last known chatID.
	// Used for cross-channel messaging (e.g. WS user sending to Discord).
	ChannelChatIDs map[string]string `json:"channel_chat_ids,omitempty"`

	// Timestamp is the last time this state was updated
	Timestamp time.Time `json:"timestamp"`
}

// Manager manages persistent state with atomic saves.
type Manager struct {
	workspace string
	state     *State
	mu        sync.RWMutex
	stateFile string
}

// NewManager creates a new state manager for the given workspace.
func NewManager(workspace string) *Manager {
	stateDir := filepath.Join(workspace, "state")
	stateFile := filepath.Join(stateDir, "state.json")
	oldStateFile := filepath.Join(workspace, "state.json")

	// Create state directory if it doesn't exist
	os.MkdirAll(stateDir, 0755)

	sm := &Manager{
		workspace: workspace,
		stateFile: stateFile,
		state:     &State{},
	}

	// Try to load from new location first
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		// New file doesn't exist, try migrating from old location
		if data, err := os.ReadFile(oldStateFile); err == nil {
			if err := json.Unmarshal(data, sm.state); err == nil {
				// Migrate to new location
				sm.saveAtomic()
				log.Printf("[INFO] state: migrated state from %s to %s", oldStateFile, stateFile)
			}
		}
	} else {
		// Load from new location
		sm.load()
	}

	return sm
}

// SetLastChannel atomically updates the last channel and saves the state.
// This method uses a temp file + rename pattern for atomic writes,
// ensuring that the state file is never corrupted even if the process crashes.
func (sm *Manager) SetLastChannel(channel string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Update state
	sm.state.LastChannel = channel
	sm.state.Timestamp = time.Now()

	// Also update per-channel chatID mapping (channel format: "name:chatID")
	sm.updateChannelChatID(channel)

	// Atomic save using temp file + rename
	if err := sm.saveAtomic(); err != nil {
		return fmt.Errorf("failed to save state atomically: %w", err)
	}

	return nil
}

// SetLastChatID atomically updates the last chat ID and saves the state.
func (sm *Manager) SetLastChatID(chatID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Update state
	sm.state.LastChatID = chatID
	sm.state.Timestamp = time.Now()

	// Atomic save using temp file + rename
	if err := sm.saveAtomic(); err != nil {
		return fmt.Errorf("failed to save state atomically: %w", err)
	}

	return nil
}

// GetLastChannel returns the last channel from the state.
func (sm *Manager) GetLastChannel() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state.LastChannel
}

// GetLastChatID returns the last chat ID from the state.
func (sm *Manager) GetLastChatID() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state.LastChatID
}

// SetLastChannelWithType atomically updates the last channel and, if the
// client type is "main", also updates LastMainChannel. Both fields are
// persisted in a single atomic write.
func (sm *Manager) SetLastChannelWithType(channel, clientType string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.state.LastChannel = channel
	sm.state.Timestamp = time.Now()

	if clientType == "main" {
		sm.state.LastMainChannel = channel
	}

	// Also update per-channel chatID mapping (channel format: "name:chatID")
	sm.updateChannelChatID(channel)

	if err := sm.saveAtomic(); err != nil {
		return fmt.Errorf("failed to save state atomically: %w", err)
	}

	return nil
}

// GetLastMainChannel returns the last channel used by a "main" client type.
func (sm *Manager) GetLastMainChannel() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state.LastMainChannel
}

// updateChannelChatID parses "name:chatID" and updates ChannelChatIDs.
// Must be called with the lock held.
func (sm *Manager) updateChannelChatID(channelKey string) {
	if parts := strings.SplitN(channelKey, ":", 2); len(parts) == 2 {
		if sm.state.ChannelChatIDs == nil {
			sm.state.ChannelChatIDs = make(map[string]string)
		}
		sm.state.ChannelChatIDs[parts[0]] = parts[1]
	}
}

// SetChannelChatID records the last known chatID for a given channel name.
// This enables cross-channel messaging by resolving the target chatID
// when the AI specifies a channel but no chatID.
func (sm *Manager) SetChannelChatID(channel, chatID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.state.ChannelChatIDs == nil {
		sm.state.ChannelChatIDs = make(map[string]string)
	}
	sm.state.ChannelChatIDs[channel] = chatID
	sm.state.Timestamp = time.Now()

	if err := sm.saveAtomic(); err != nil {
		return fmt.Errorf("failed to save state atomically: %w", err)
	}
	return nil
}

// GetChannelChatID returns the last known chatID for the given channel name.
func (sm *Manager) GetChannelChatID(channel string) string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if sm.state.ChannelChatIDs == nil {
		return ""
	}
	return sm.state.ChannelChatIDs[channel]
}

// GetTimestamp returns the timestamp of the last state update.
func (sm *Manager) GetTimestamp() time.Time {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state.Timestamp
}

// saveAtomic performs an atomic save using temp file + rename.
// This ensures that the state file is never corrupted:
// 1. Write to a temp file
// 2. Rename temp file to target (atomic on POSIX systems)
// 3. If rename fails, cleanup the temp file
//
// Must be called with the lock held.
func (sm *Manager) saveAtomic() error {
	// Create temp file in the same directory as the target
	tempFile := sm.stateFile + ".tmp"

	// Marshal state to JSON
	data, err := json.MarshalIndent(sm.state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to temp file
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Atomic rename from temp to target
	if err := os.Rename(tempFile, sm.stateFile); err != nil {
		// Cleanup temp file if rename fails
		os.Remove(tempFile)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// load loads the state from disk.
func (sm *Manager) load() error {
	data, err := os.ReadFile(sm.stateFile)
	if err != nil {
		// File doesn't exist yet, that's OK
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read state file: %w", err)
	}

	if err := json.Unmarshal(data, sm.state); err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return nil
}
