package imessage

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/utils"
)

// iMessageChannel implements a channel for macOS iMessage using the imsg CLI tool.
type iMessageChannel struct {
	*channels.BaseChannel
	config   config.ImessageConfig
	mu       sync.Mutex
	running  bool
	listener *Listener
	maxRowid int
}

// Listener handles listening for incoming iMessages via watch command
type Listener struct {
	ctx      context.Context
	cancel   context.CancelFunc
	cmd      *exec.Cmd
	stdout   *bufio.Reader
	lastLine string
	lastErr  error
}

func NewiMessageChannel(cfg config.ImessageConfig, b *bus.MessageBus) (*iMessageChannel, error) {
	base := channels.NewBaseChannel("imessage", cfg, b, cfg.AllowFrom)

	return &iMessageChannel{
		BaseChannel: base,
		config:      cfg,
		running:     false,
	}, nil
}

// Start begins listening for incoming iMessages and prepares to send messages
func (c *iMessageChannel) Start(ctx context.Context) error {
	//log.Printf("Starting iMessage channel...")
	//fmt.Fprintln(os.Stderr, "=== Start() called ===")

	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return fmt.Errorf("iMessage channel already running")
	}

	// Check if imsg is installed
	if err := c.checkIMsgInstalled(); err != nil {
		c.mu.Unlock()
		return fmt.Errorf("imsg not found: %w. Please install imsg: pip install imessage-reader", err)
	}

	// Validate database path
	dbPath := c.getDBPath()
	if _, err := os.Stat(dbPath); err != nil {
		c.mu.Unlock()
		return fmt.Errorf("iMessage database not found at %s: %w", dbPath, err)
	}

	// Start the listener in background
	c.listener = &Listener{}
	listenCtx, cancel := context.WithCancel(ctx)
	c.listener.ctx = listenCtx
	c.listener.cancel = cancel

	c.running = true
	c.SetRunning(true)
	c.mu.Unlock() // Unlock BEFORE starting goroutine to avoid deadlock

	//fmt.Fprintln(os.Stderr, "=== Starting goroutine ===")
	go c.listen(listenCtx)
	//fmt.Fprintln(os.Stderr, "=== Goroutine started ===")

	log.Println("iMessage channel started")

	return nil
}

// Stop stops the iMessage channel and terminates the listener
func (c *iMessageChannel) Stop(ctx context.Context) error {
	log.Println("Stopping iMessage channel...")

	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return nil
	}

	if c.listener != nil && c.listener.cancel != nil {
		c.listener.cancel()
		if c.listener.cmd != nil && c.listener.cmd.Process != nil {
			c.listener.cmd.Process.Kill()
		}
	}

	c.running = false
	c.SetRunning(false)
	log.Println("iMessage channel stopped")

	return nil
}

// Send sends an iMessage to the specified recipient
func (c *iMessageChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return fmt.Errorf("iMessage channel not running")
	}

	// Workaround: imsg CLI bug - messages starting with "---" cause "Missing value for option text" error
	// Prepend a space if message starts with "---"
	content := msg.Content
	if strings.HasPrefix(content, "---") {
		content = " " + content
	}

	// Build the imsg send command
	// Format: imsg send --to <recipient> --text "<message>"
	args := []string{
		"send",
		"--to", msg.ChatID,
		"--text", content,
	}

	cmd := exec.CommandContext(ctx, "imsg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to send iMessage: %w, output: %s", err, string(output))
	}

	//log.Printf("iMessage sent to %s: %s", msg.ChatID, utils.Truncate(msg.Content, 50))

	return nil
}

// SendMedia implements the channels.MediaSender interface.
// It sends media files (images, documents, etc.) via iMessage using the imsg CLI.
func (c *iMessageChannel) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return channels.ErrNotRunning
	}

	store := c.GetMediaStore()
	if store == nil {
		return fmt.Errorf("no media store available: %w", channels.ErrSendFailed)
	}

	for _, part := range msg.Parts {
		// Resolve the media reference to a local file path
		localPath, err := store.Resolve(part.Ref)
		if err != nil {
			log.Printf("Failed to resolve media ref %s: %v", part.Ref, err)
			continue
		}

		// Build the imsg send command with --file option
		// Format: imsg send --to <recipient> --file <path> [--text <caption>]
		args := []string{
			"send",
			"--to", msg.ChatID,
			"--file", localPath,
		}
		if part.Caption != "" {
			args = append(args, "--text", part.Caption)
		}

		cmd := exec.CommandContext(ctx, "imsg", args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("Failed to send media via iMessage: %v, output: %s", err, string(output))
			return fmt.Errorf("failed to send media: %w, output: %s", channels.ErrTemporary, string(output))
		}

		log.Printf("iMessage media sent to %s: %s", msg.ChatID, localPath)
	}

	return nil
}

// listen starts watching for incoming iMessages
func (c *iMessageChannel) listen(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in listen: %v", r)
			fmt.Fprintf(os.Stderr, "!!! PANIC in listen: %v\n", r)
		}
	}()
	
	//fmt.Fprintln(os.Stderr, "=== iMessage listener started ===")
	//fmt.Fprintln(os.Stderr, ">>> Step 1: Getting max rowid...")

	// Get the current max rowid to only watch for new messages
	c.maxRowid = c.getMaxRowid()
	//fmt.Fprintf(os.Stderr, ">>> Step 2: Got rowid %d, starting watch loop\n", c.maxRowid)

	for {
		select {
		case <-ctx.Done():
			log.Println("iMessage listener stopped by context")
			return
		default:
			// Start imsg watch command with JSON output and since-rowid to only get new messages
			args := []string{"watch", "--json"}
			if c.maxRowid > 0 {
				args = append(args, "--since-rowid", fmt.Sprintf("%d", c.maxRowid))
			}
			cmd := exec.CommandContext(ctx, "imsg", args...)
			
			// Capture stdout and stderr
			stdoutPipe, err := cmd.StdoutPipe()
			if err != nil {
				log.Printf("Error creating stdout pipe: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}

			stderrPipe, err := cmd.StderrPipe()
			if err != nil {
				log.Printf("Error creating stderr pipe: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}

			if err := cmd.Start(); err != nil {
				log.Printf("Failed to start imsg watch: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}
			log.Printf("imsg watch started successfully (pid: %d)", cmd.Process.Pid)

			c.mu.Lock()
			c.listener.cmd = cmd
			c.listener.stdout = bufio.NewReader(stdoutPipe)
			c.mu.Unlock()

			// Read errors in background
			go c.readError(stderrPipe)
			
			// Read messages
			//log.Printf("Starting to scan for messages...")
			//scanCount := 0
			for {
				//fmt.Fprintf(os.Stderr, ">>> Before Scan() call #%d\n", scanCount+1)
				hasMore := c.listener.Scan()
				//fmt.Fprintf(os.Stderr, ">>> After Scan() call #%d, hasMore=%v\n", scanCount+1, hasMore)
				//scanCount++
				
				if !hasMore {
					//log.Printf("Scan() returned false, breaking loop")
					// Check for scanner error
					if err := c.listener.ScanErr(); err != nil {
						//log.Printf("Scanner error: %v", err)
						fmt.Fprintf(os.Stderr, ">>> Scanner error: %v\n", err)
					} else {
						//log.Printf("Scanner returned false with no error (EOF)")
						fmt.Fprintln(os.Stderr, ">>> Scanner returned false with no error (EOF)")
					}
					// Check if process is still running
					if c.listener.cmd != nil && c.listener.cmd.Process != nil {
						fmt.Fprintf(os.Stderr, ">>> imsg process state: pid=%d\n", c.listener.cmd.Process.Pid)
						// Try to find the process
						checkCmd := exec.Command("ps", "-p", fmt.Sprintf("%d", c.listener.cmd.Process.Pid))
						output, _ := checkCmd.CombinedOutput()
						fmt.Fprintf(os.Stderr, ">>> ps output: %s\n", string(output))
					}
					break
				}
				
				line := c.listener.Text()
				//log.Printf("Scanned line: %s", utils.Truncate(line, 100))
				//fmt.Fprintf(os.Stderr, ">>> Scanned line: %s\n", utils.Truncate(line, 100))
				if line == "" {
					continue
				}
				c.handleMessage(line)
			}
			//log.Printf("Scanner loop ended after %d scans", scanCount)

			// Check for scanner errors
			if err := c.listener.ScanErr(); err != nil {
				log.Printf("imsg watch scanner error: %v", err)
			}

			// Wait for command to finish
			if err := cmd.Wait(); err != nil {
				//log.Printf("imsg watch exited: %v", err)
				fmt.Fprintf(os.Stderr, ">>> cmd.Wait() returned error: %v\n", err)
			} else {
				//log.Printf("imsg watch exited normally")
				fmt.Fprintln(os.Stderr, ">>> cmd.Wait() returned nil (normal exit)")
			}

			// Reconnect after a short delay
			//log.Printf("Reconnecting in 5 seconds...")
			fmt.Fprintln(os.Stderr, ">>> Reconnecting in 5 seconds...")
			select {
			case <-ctx.Done():
				//log.Printf("Context done, exiting listen loop")
				fmt.Fprintln(os.Stderr, ">>> Context done, exiting listen loop")
				return
			case <-time.After(5 * time.Second):
				//log.Printf("Reconnecting now...")
				fmt.Fprintln(os.Stderr, ">>> Reconnecting now...")
			}
		}
	}
}

// readError reads from the stderr of imsg watch
func (c *iMessageChannel) readError(stderr io.Reader) {
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		log.Printf("[imsg stderr] %s", scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Printf("[imsg stderr] scanner error: %v", err)
	}
}

// Scan returns whether the scanner has more lines to read
func (l *Listener) Scan() bool {
	if l.stdout == nil {
		return false
	}
	line, err := l.stdout.ReadString('\n')
	if err != nil {
		l.lastErr = err
		return false
	}
	l.lastLine = strings.TrimSuffix(line, "\n")
	return true
}

// Text returns the most recent line read by the scanner
func (l *Listener) Text() string {
	return l.lastLine
}

// ScanErr returns the scanner error if any
func (l *Listener) ScanErr() error {
	return l.lastErr
}

// handleMessage processes incoming iMessage lines
func (c *iMessageChannel) handleMessage(line string) {
	//log.Printf("Received iMessage line: %s", line)

	// Skip empty lines or non-JSON content (imsg may output non-JSON logs)
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	
	// Check if line starts with '{' (valid JSON object)
	if len(line) == 0 || line[0] != '{' {
		// Not a JSON object, skip it (could be a log message from imsg)
		log.Printf("[imsg] Non-JSON output: %s", utils.Truncate(line, 100))
		return
	}

	// Parse the JSON output from imsg watch
	var msg map[string]interface{}
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		log.Printf("Failed to parse iMessage JSON: %v, line: %s", err, utils.Truncate(line, 100))
		return
	}

	// Update max rowid for reconnection
	if id, ok := msg["rowid"].(float64); ok && int(id) > c.maxRowid {
		c.maxRowid = int(id)
	}

	// Skip messages from me (is_from_me)
	if isFromMe, ok := msg["is_from_me"].(bool); ok && isFromMe {
		//log.Printf("Skipping message from me: %s", utils.Truncate(fmt.Sprintf("%v", msg["text"]), 50))
		return
	}

	// Extract message fields - support both old and new field names
	senderID, _ := msg["from"].(string)
	if senderID == "" {
		if sender, ok := msg["sender"].(string); ok {
			senderID = sender
		}
	}
	if senderID == "" {
		return
	}

	// For iMessage, we need the email address for sending replies
	// imsg send --to requires an email address, not a numeric chat_id
	// So we always use senderID (email) as the ChatID for direct messages
	chatID := senderID

	content, _ := msg["text"].(string)
	if content == "" {
		return
	}

	var mediaPaths []string
	if mediaData, ok := msg["attachments"].([]interface{}); ok {
		mediaPaths = make([]string, 0, len(mediaData))
		for _, m := range mediaData {
			if path, ok := m.(string); ok {
				mediaPaths = append(mediaPaths, path)
			}
		}
	}

	metadata := make(map[string]string)
	if messageID, ok := msg["rowid"].(float64); ok {
		metadata["message_id"] = fmt.Sprintf("%.0f", messageID)
	} else if messageID, ok := msg["guid"].(string); ok {
		metadata["message_id"] = messageID
	}
	if timestamp, ok := msg["timestamp"].(string); ok {
		metadata["timestamp"] = timestamp
	} else if timestamp, ok := msg["created_at"].(string); ok {
		metadata["timestamp"] = timestamp
	}
	
	// Store the original chat_id in metadata for reference
	if chatIDNum, ok := msg["chat_id"].(float64); ok {
		metadata["original_chat_id"] = fmt.Sprintf("%.0f", chatIDNum)
	}

	// Determine peer kind
	metadata["peer_kind"] = "direct"
	metadata["peer_id"] = senderID

	//log.Printf("iMessage from %s: %s...", senderID, utils.Truncate(content, 50))

	peer := bus.Peer{Kind: "direct", ID: senderID}
	messageID := metadata["message_id"]

	c.HandleMessage(context.Background(), peer, messageID, senderID, chatID, content, mediaPaths, metadata)
}

// checkIMsgInstalled checks if the imsg CLI tool is installed
func (c *iMessageChannel) checkIMsgInstalled() error {
	_, err := exec.LookPath("imsg")
	return err
}

// getDBPath returns the database path from config, or the default path
func (c *iMessageChannel) getDBPath() string {
	if c.config.DBPath != "" {
		// Expand ~ to home directory if needed
		return expandHome(c.config.DBPath)
	}
	
	// Default path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, "Library", "Messages", "chat.db")
}

// expandHome expands ~ to the home directory
func expandHome(path string) string {
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

// getMaxRowid returns the current maximum rowid from the messages database
func (c *iMessageChannel) getMaxRowid() int {
	//fmt.Fprintln(os.Stderr, "=== getMaxRowid: starting ===")
	
	// Query the chat.db directly to get the max ROWID
	dbPath := c.getDBPath()
	if dbPath == "" {
		fmt.Fprintln(os.Stderr, "getMaxRowid: No database path configured")
		return 0
	}
	//fmt.Fprintf(os.Stderr, "getMaxRowid: dbPath=%s\n", dbPath)
	
	// Use sqlite3 to query the max ROWID
	//fmt.Fprintln(os.Stderr, "getMaxRowid: executing sqlite3 command...")
	cmd := exec.Command("sqlite3", dbPath, "SELECT MAX(ROWID) FROM message;")
	output, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "getMaxRowid: Failed to query: %v\n", err)
		return 0
	}
	//fmt.Fprintf(os.Stderr, "getMaxRowid: raw output=%s\n", string(output))

	// Parse the output
	rowidStr := strings.TrimSpace(string(output))
	rowid, err := strconv.Atoi(rowidStr)
	if err != nil {
		//fmt.Fprintf(os.Stderr, "getMaxRowid: Failed to parse: %v\n", err)
		return 0
	}

	//fmt.Fprintf(os.Stderr, "getMaxRowid: returning %d\n", rowid)
	return rowid
}

// splitLines splits a string into lines
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 {
				lines = append(lines, line)
			}
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
