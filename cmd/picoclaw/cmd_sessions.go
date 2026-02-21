package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// sessionData is a minimal struct for reading session JSON files.
// We only need the fields required for display — no dependency on pkg/session.
type sessionData struct {
	Key      string          `json:"key"`
	Messages json.RawMessage `json:"messages"`
	Summary  string          `json:"summary,omitempty"`
	Created  time.Time       `json:"created"`
	Updated  time.Time       `json:"updated"`
}

// sessionMessage is a minimal struct for reading individual messages.
type sessionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func sessionsCmd() {
	args := os.Args[2:]

	if len(args) == 0 {
		sessionsHelp()
		return
	}

	subcommand := args[0]

	if subcommand == "--help" || subcommand == "-h" {
		sessionsHelp()
		return
	}

	sessionsDir := getSessionsDir()

	switch subcommand {
	case "list":
		sessionsListCmd(sessionsDir)
	case "show":
		if len(args) < 2 {
			fmt.Println("Usage: picoclaw sessions show <id>")
			os.Exit(1)
		}
		sessionsShowCmd(sessionsDir, args[1])
	case "delete":
		if len(args) < 2 {
			fmt.Println("Usage: picoclaw sessions delete <id>")
			os.Exit(1)
		}
		sessionsDeleteCmd(sessionsDir, args[1])
	case "clear":
		sessionsClearCmd(sessionsDir)
	default:
		fmt.Printf("Unknown sessions command: %s\n", subcommand)
		sessionsHelp()
		os.Exit(1)
	}
}

func sessionsHelp() {
	fmt.Println("Usage: picoclaw sessions <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  list        List all sessions")
	fmt.Println("  show <id>   Show session details")
	fmt.Println("  delete <id> Delete a session")
	fmt.Println("  clear       Delete all sessions")
}

func getSessionsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".picoclaw", "workspace", "sessions")
}

type sessionEntry struct {
	id       string
	messages int
	modTime  time.Time
	size     int64
	corrupt  bool
}

func sessionsListCmd(sessionsDir string) {
	entries, err := listSessionEntries(sessionsDir)
	if err != nil {
		fmt.Println(err)
		return
	}

	if len(entries) == 0 {
		fmt.Println("No sessions found")
		return
	}

	// Sort by modification time, most recent first
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].modTime.After(entries[j].modTime)
	})

	fmt.Println("Sessions:")
	fmt.Printf("  %-30s %8s  %s\n", "ID", "Messages", "Last Modified")
	for _, e := range entries {
		msgStr := fmt.Sprintf("%d", e.messages)
		if e.corrupt {
			msgStr = "(corrupt)"
		}
		fmt.Printf("  %-30s %8s  %s\n", e.id, msgStr, e.modTime.Format("2006-01-02 15:04"))
	}

	fmt.Printf("\n%d session(s) found\n", len(entries))
}

func sessionsShowCmd(sessionsDir, id string) {
	filePath := findSessionFile(sessionsDir, id)
	if filePath == "" {
		fmt.Printf("Session '%s' not found\n", id)
		os.Exit(1)
	}

	info, err := os.Stat(filePath)
	if err != nil {
		fmt.Printf("Error reading session: %v\n", err)
		os.Exit(1)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading session: %v\n", err)
		os.Exit(1)
	}

	var sess sessionData
	if err := json.Unmarshal(data, &sess); err != nil {
		fmt.Printf("Session: %s\n", id)
		fmt.Printf("Size: %s\n", formatSize(info.Size()))
		fmt.Printf("Last Modified: %s\n", info.ModTime().Format("2006-01-02 15:04"))
		fmt.Println("Status: corrupt (invalid JSON)")
		return
	}

	var msgs []sessionMessage
	_ = json.Unmarshal(sess.Messages, &msgs)

	fmt.Printf("Session: %s\n", sess.Key)
	fmt.Printf("Messages: %d\n", len(msgs))
	fmt.Printf("Last Modified: %s\n", info.ModTime().Format("2006-01-02 15:04"))
	fmt.Printf("Size: %s\n", formatSize(info.Size()))

	if len(msgs) > 0 {
		fmt.Println()
		start := len(msgs) - 3
		if start < 0 {
			start = 0
		}
		fmt.Println("Last messages:")
		for _, m := range msgs[start:] {
			content := strings.TrimSpace(m.Content)
			content = strings.ReplaceAll(content, "\n", " ")
			if len(content) > 80 {
				content = content[:77] + "..."
			}
			fmt.Printf("  [%s] %s\n", m.Role, content)
		}
	}
}

func sessionsDeleteCmd(sessionsDir, id string) {
	filePath := findSessionFile(sessionsDir, id)
	if filePath == "" {
		fmt.Printf("Session '%s' not found\n", id)
		os.Exit(1)
	}

	fmt.Printf("Delete session '%s'? (y/n): ", id)
	if !confirmPrompt() {
		fmt.Println("Cancelled")
		return
	}

	if err := os.Remove(filePath); err != nil {
		fmt.Printf("Error deleting session: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Deleted session %s\n", id)
}

func sessionsClearCmd(sessionsDir string) {
	entries, err := listSessionEntries(sessionsDir)
	if err != nil {
		fmt.Println(err)
		return
	}

	if len(entries) == 0 {
		fmt.Println("No sessions found")
		return
	}

	fmt.Printf("Delete all %d sessions? (y/n): ", len(entries))
	if !confirmPrompt() {
		fmt.Println("Cancelled")
		return
	}

	deleted := 0
	for _, e := range entries {
		filePath := findSessionFile(sessionsDir, e.id)
		if filePath == "" {
			continue
		}
		if err := os.Remove(filePath); err != nil {
			fmt.Printf("Error deleting session '%s': %v\n", e.id, err)
			continue
		}
		deleted++
	}

	fmt.Printf("Cleared %d session(s).\n", deleted)
}

// listSessionEntries reads the sessions directory and returns parsed entries.
func listSessionEntries(sessionsDir string) ([]sessionEntry, error) {
	if _, err := os.Stat(sessionsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("No sessions found (sessions directory does not exist)")
	}

	files, err := os.ReadDir(sessionsDir)
	if err != nil {
		return nil, fmt.Errorf("error reading sessions directory: %v", err)
	}

	var entries []sessionEntry
	for _, f := range files {
		if f.IsDir() || filepath.Ext(f.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(sessionsDir, f.Name())
		info, err := os.Stat(filePath)
		if err != nil {
			continue
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var sess sessionData
		if err := json.Unmarshal(data, &sess); err != nil {
			// Corrupt file — derive ID from filename
			id := strings.TrimSuffix(f.Name(), ".json")
			entries = append(entries, sessionEntry{
				id:      id,
				modTime: info.ModTime(),
				size:    info.Size(),
				corrupt: true,
			})
			continue
		}

		// Use the key from the JSON if present, otherwise derive from filename
		id := sess.Key
		if id == "" {
			id = strings.TrimSuffix(f.Name(), ".json")
		}

		var msgs []sessionMessage
		_ = json.Unmarshal(sess.Messages, &msgs)

		entries = append(entries, sessionEntry{
			id:       id,
			messages: len(msgs),
			modTime:  info.ModTime(),
			size:     info.Size(),
		})
	}

	return entries, nil
}

// findSessionFile locates the session file for a given ID.
// It first tries matching by the key inside the JSON, then falls back
// to matching by filename (with .json extension).
func findSessionFile(sessionsDir string, id string) string {
	files, err := os.ReadDir(sessionsDir)
	if err != nil {
		return ""
	}

	for _, f := range files {
		if f.IsDir() || filepath.Ext(f.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(sessionsDir, f.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var sess sessionData
		if err := json.Unmarshal(data, &sess); err != nil {
			// Corrupt file — match by filename
			name := strings.TrimSuffix(f.Name(), ".json")
			if name == id {
				return filePath
			}
			continue
		}

		if sess.Key == id {
			return filePath
		}
	}

	// Fallback: try direct filename match (id + .json or sanitized id + .json)
	direct := filepath.Join(sessionsDir, id+".json")
	if _, err := os.Stat(direct); err == nil {
		return direct
	}
	sanitized := filepath.Join(sessionsDir, strings.ReplaceAll(id, ":", "_")+".json")
	if _, err := os.Stat(sanitized); err == nil {
		return sanitized
	}

	return ""
}

func confirmPrompt() bool {
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

func formatSize(bytes int64) string {
	const (
		kb = 1024
		mb = kb * 1024
	)
	switch {
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
