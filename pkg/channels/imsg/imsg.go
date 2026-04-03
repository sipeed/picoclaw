package imsg

import (
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
)

type IMsgChannel struct {
	*channels.BaseChannel
	config    config.IMsgConfig
	runCtx    context.Context
	runCancel context.CancelFunc
	rpcCmd    *exec.Cmd
	rpcStdin  io.WriteCloser
	wg        sync.WaitGroup
	mu        sync.Mutex
	fatalMu   sync.Mutex
	fatalLine string
	lastLine  string
	recvMode  string
	seenMu    sync.Mutex
	seenIDs   map[string]struct{}
	seenEvent map[string]time.Time
	reqSeq    atomic.Uint64
	writeMu   sync.Mutex
	pendingMu sync.Mutex
	pending   map[string]chan rpcResponse
}

func NewIMsgChannel(cfg config.IMsgConfig, messageBus *bus.MessageBus) (*IMsgChannel, error) {
	if strings.TrimSpace(cfg.IMessageCLIPath) == "" {
		cfg.IMessageCLIPath = "imsg"
	}

	base := channels.NewBaseChannel("imsg", cfg, messageBus, cfg.AllowFrom)
	return &IMsgChannel{
		BaseChannel: base,
		config:      cfg,
		seenIDs:     make(map[string]struct{}),
		seenEvent:   make(map[string]time.Time),
		pending:     make(map[string]chan rpcResponse),
	}, nil
}

type rpcResponse struct {
	Result any
	Error  *rpcError
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (c *IMsgChannel) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.IsRunning() {
		return nil
	}
	c.fatalMu.Lock()
	c.fatalLine = ""
	c.lastLine = ""
	c.fatalMu.Unlock()
	c.runCtx, c.runCancel = context.WithCancel(ctx)
	if err := c.startRPCLocked(); err != nil {
		c.runCancel()
		c.runCtx = nil
		c.runCancel = nil
		return err
	}
	c.SetRunning(true)
	logger.InfoC("imsg", "iMessage channel started")
	return nil
}

func (c *IMsgChannel) Stop(ctx context.Context) error {
	c.mu.Lock()
	if !c.IsRunning() && c.rpcCmd == nil {
		c.mu.Unlock()
		return nil
	}
	c.SetRunning(false)
	if c.runCancel != nil {
		c.runCancel()
	}
	cmd := c.rpcCmd
	stdin := c.rpcStdin
	c.rpcCmd = nil
	c.rpcStdin = nil
	c.mu.Unlock()

	if stdin != nil {
		_ = stdin.Close()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	c.wg.Wait()

	c.mu.Lock()
	c.runCtx = nil
	c.runCancel = nil
	c.mu.Unlock()

	logger.InfoC("imsg", "iMessage channel stopped")
	return nil
}

func (c *IMsgChannel) Send(ctx context.Context, msg bus.OutboundMessage) ([]string, error) {
	if !c.IsRunning() {
		return nil, channels.ErrNotRunning
	}
	paramsList := []map[string]any{
		{"chat_id": msg.ChatID, "text": msg.Content},
		{"chatId": msg.ChatID, "text": msg.Content},
		{"chat_id": msg.ChatID, "content": msg.Content},
	}
	methods := []string{"send", "message.send"}
	var lastErr error
	for _, method := range methods {
		for _, params := range paramsList {
			logger.InfoCF("imsg", "imsg rpc send request", map[string]any{
				"method": method,
				"params": params,
			})
			if _, err := c.rpcCall(ctx, method, params, 8*time.Second); err == nil {
				return nil, nil
			} else {
				lastErr = err
			}
		}
	}
	return nil, fmt.Errorf("imsg rpc send failed: %w", lastErr)
}

// SendMedia implements channels.MediaSender via imsg RPC.
func (c *IMsgChannel) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) ([]string, error) {
	if !c.IsRunning() {
		return nil, channels.ErrNotRunning
	}
	store := c.GetMediaStore()
	if store == nil {
		return nil, fmt.Errorf("no media store available: %w", channels.ErrSendFailed)
	}

	for _, part := range msg.Parts {
		localPath, err := store.Resolve(part.Ref)
		if err != nil {
			return nil, fmt.Errorf("resolve media ref %s: %w", part.Ref, err)
		}
		paramsList := []map[string]any{
			{"chat_id": msg.ChatID, "text": part.Caption, "file": localPath},
			{"chatId": msg.ChatID, "text": part.Caption, "file": localPath},
			{"chat_id": msg.ChatID, "content": part.Caption, "file": localPath},
		}
		methods := []string{"send", "message.send"}
		var lastErr error
		ok := false
		for _, method := range methods {
			for _, params := range paramsList {
				logger.InfoCF("imsg", "imsg rpc send media request", map[string]any{
					"method": method,
					"params": params,
				})
				if _, err := c.rpcCall(ctx, method, params, 12*time.Second); err == nil {
					ok = true
					break
				} else {
					lastErr = err
				}
			}
			if ok {
				break
			}
		}
		if !ok {
			return nil, fmt.Errorf("imsg rpc send media failed: %w", lastErr)
		}
	}
	return nil, nil
}

func (c *IMsgChannel) startRPCLocked() error {
	cliPath := strings.TrimSpace(c.config.IMessageCLIPath)
	if cliPath == "" {
		cliPath = "imsg"
	}

	commands := []struct {
		mode string
		args []string
	}{
		{mode: "rpc", args: []string{"rpc", "--json"}},
		{mode: "rpc", args: []string{"rpc"}},
	}
	var lastErr error
	for _, candidate := range commands {
		if err := c.startReceiveProcessLocked(cliPath, candidate.mode, candidate.args); err != nil {
			lastErr = err
			continue
		}
		if err := c.subscribeRPC(c.runCtx); err != nil {
			c.cleanupReceiveProcessLocked()
			lastErr = err
			continue
		}
		return nil
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("imsg startup failed: no receive command available")
}

func (c *IMsgChannel) cleanupReceiveProcessLocked() {
	stdin := c.rpcStdin
	cmd := c.rpcCmd
	c.rpcStdin = nil
	c.rpcCmd = nil
	if stdin != nil {
		_ = stdin.Close()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	c.wg.Wait()
}

func (c *IMsgChannel) startReceiveProcessLocked(cliPath, mode string, args []string) error {
	cmd := exec.CommandContext(c.runCtx, cliPath, args...)
	logger.InfoCF("imsg", "imsg receive command", map[string]any{
		"mode": mode,
		"cmd":  strings.Join(cmd.Args, " "),
	})
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("imsg %s stdout pipe: %w", mode, err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("imsg %s stderr pipe: %w", mode, err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("imsg %s stdin pipe: %w", mode, err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start imsg %s: %w", mode, err)
	}
	c.rpcCmd = cmd
	c.rpcStdin = stdin
	c.recvMode = mode

	exited := make(chan error, 1)
	c.wg.Add(3)
	go c.consumeRPCStdout(stdout)
	go c.consumeRPCStderr(stderr)
	go c.waitRPCExit(cmd, exited)

	// Startup probe: if RPC exits immediately, fail Start with a clear reason.
	select {
	case err := <-exited:
		c.rpcCmd = nil
		c.rpcStdin = nil
		c.wg.Wait()
		if err == nil {
			if fatal := c.getFatalLine(); fatal != "" {
				return fmt.Errorf(
					"imsg %s exited during startup (cmd=%q, fatal=%s)",
					mode,
					strings.Join(cmd.Args, " "),
					fatal,
				)
			}
			if last := c.getLastLine(); last != "" {
				return fmt.Errorf(
					"imsg %s exited during startup (cmd=%q, last output: %s)",
					mode,
					strings.Join(cmd.Args, " "),
					last,
				)
			}
			return fmt.Errorf(
				"imsg %s exited during startup (cmd=%q, no output captured)",
				mode,
				strings.Join(cmd.Args, " "),
			)
		}
		if fatal := c.getFatalLine(); fatal != "" {
			return fmt.Errorf(
				"imsg %s startup failed: %v (cmd=%q, fatal=%s)",
				mode,
				err,
				strings.Join(cmd.Args, " "),
				fatal,
			)
		}
		if last := c.getLastLine(); last != "" {
			return fmt.Errorf(
				"imsg %s startup failed: %v (cmd=%q, last output: %s)",
				mode,
				err,
				strings.Join(cmd.Args, " "),
				last,
			)
		}
		return fmt.Errorf(
			"imsg %s startup failed: %w (cmd=%q, no output captured)",
			mode,
			err,
			strings.Join(cmd.Args, " "),
		)
	case <-time.After(1200 * time.Millisecond):
	}
	logger.InfoCF("imsg", "imsg receive process started", map[string]any{
		"mode": mode,
		"cmd":  strings.Join(cmd.Args, " "),
	})
	// NOTE: History polling fallback is intentionally disabled.
	// Use pure JSON-RPC watch.subscribe for inbound messages.
	return nil
}

func (c *IMsgChannel) consumeRPCStdout(r io.Reader) {
	defer c.wg.Done()
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	for scanner.Scan() {
		c.handleRPCLine(scanner.Text())
	}
	if err := scanner.Err(); err != nil && c.IsRunning() {
		logger.WarnCF("imsg", "imsg rpc stdout scanner error", map[string]any{"error": err.Error()})
	}
}

func (c *IMsgChannel) consumeRPCStderr(r io.Reader) {
	defer c.wg.Done()
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 8*1024), 512*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		c.setLastLine(line)
		if isFatalRPCLine(line) {
			c.setFatalLine(line)
		}
		logger.DebugCF("imsg", "imsg rpc stderr", map[string]any{"line": line})
	}
}

func (c *IMsgChannel) waitRPCExit(cmd *exec.Cmd, exited chan<- error) {
	defer c.wg.Done()
	err := cmd.Wait()
	select {
	case exited <- err:
	default:
	}
	if err != nil && c.IsRunning() {
		fields := map[string]any{"error": err.Error()}
		if fatal := c.getFatalLine(); fatal != "" {
			fields["hint"] = fatal
		} else if last := c.getLastLine(); last != "" {
			fields["last_output"] = last
		}
		logger.ErrorCF("imsg", "imsg rpc exited unexpectedly", fields)
	}
}

func (c *IMsgChannel) handleRPCLine(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	c.setLastLine(line)
	var raw map[string]any
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		if isFatalRPCLine(line) {
			c.setFatalLine(line)
			logger.ErrorCF("imsg", "imsg rpc fatal output", map[string]any{"line": line})
			return
		}
		logger.DebugCF("imsg", "skip non-json rpc line", map[string]any{"line": line})
		return
	}
	// Handle JSON-RPC responses first.
	if id, ok := raw["id"]; ok && (raw["result"] != nil || raw["error"] != nil) {
		c.resolvePending(id, raw)
		return
	}

	method := strings.ToLower(anyToString(raw["method"]))
	if method != "" {
		// imsg legacy notification: method = "message"
		if strings.Contains(method, "message") || strings.Contains(method, "watch") {
			params := firstMap(raw["params"], raw["result"], raw["payload"], raw["message"], raw)
			c.handlePayload(params, line)
			return
		}
		// Non-message RPC notifications (e.g. ping/ack) should not enter agent loop.
		logger.DebugCF("imsg", "skip non-message rpc notification", map[string]any{"method": method})
		return
	}

	// Ignore JSON lines without method to avoid parsing RPC response-like payloads
	// as inbound chat messages, which can create self-reply loops.
	logger.DebugCF("imsg", "skip rpc json without method", map[string]any{"line": line})
}

func (c *IMsgChannel) handlePayload(raw map[string]any, rawLine string) {
	// Ignore messages sent by ourselves to avoid self-trigger loops.
	if deepBool(raw, "is_from_me", "from_me", "is_me", "isFromMe", "fromMe", "outgoing") {
		return
	}
	content := deepCoalesceString(raw, "content", "text", "message", "body")
	if strings.TrimSpace(content) == "" {
		return
	}

	kind := strings.ToLower(deepCoalesceString(raw, "type", "event", "method", "name"))
	if strings.Contains(kind, "outgoing") || strings.Contains(kind, "sent") {
		return
	}

	senderID := deepCoalesceString(raw, "from", "sender", "sender_id", "handle", "from_id", "participant")
	chatID := deepCoalesceString(
		raw,
		"chat_id",
		"chat",
		"conversation",
		"thread",
		"peer",
		"to",
		"chat_identifier",
		"conversation_id",
	)
	if senderID == "" && chatID == "" {
		logger.DebugCF("imsg", "imsg rpc json parsed but missing sender/chat", map[string]any{
			"kind": kind,
			"line": rawLine,
		})
		return
	}
	if senderID == "" {
		senderID = chatID
	}
	if chatID == "" {
		chatID = senderID
	}
	messageID := deepCoalesceString(raw, "id", "message_id", "guid", "rowid")
	if messageID != "" && !c.markSeen(messageID) {
		return
	}
	if messageID == "" {
		fingerprint := eventFingerprint(chatID, senderID, content, deepCoalesceString(raw, "timestamp", "date", "time"))
		if !c.markSeenEvent(fingerprint) {
			return
		}
	}

	peerKind := "direct"
	if deepBool(raw, "is_group", "group", "isGroup") {
		peerKind = "group"
	}
	peer := bus.Peer{Kind: peerKind, ID: chatID}
	sender := bus.SenderInfo{
		Platform:    "imsg",
		PlatformID:  senderID,
		CanonicalID: identity.BuildCanonicalID("imsg", senderID),
	}
	if !c.isAllowedIMsg(sender, senderID, chatID) {
		logger.DebugCF("imsg", "imsg message blocked by allow_from", map[string]any{
			"sender_id": senderID,
			"chat_id":   chatID,
		})
		c.sendNotAllowedNotice(chatID)
		return
	}
	metadata := map[string]string{"platform": "imsg"}
	if kind != "" {
		metadata["rpc_event"] = kind
	}
	metadata["recv_mode"] = c.recvMode
	if rawLine != "" {
		metadata["raw_line"] = rawLine
	}
	logger.InfoCF("imsg", "imsg inbound message received", map[string]any{
		"mode":      c.recvMode,
		"event":     kind,
		"sender_id": senderID,
		"chat_id":   chatID,
	})
	c.HandleMessage(c.runCtx, peer, messageID, senderID, chatID, content, nil, metadata, sender)
}

func (c *IMsgChannel) isAllowedIMsg(sender bus.SenderInfo, senderID, chatID string) bool {
	// 1) Canonical sender matching (preferred).
	if c.IsAllowedSender(sender) {
		return true
	}
	// 2) Legacy sender matching.
	if senderID != "" && c.IsAllowed(senderID) {
		return true
	}
	return false
}

func (c *IMsgChannel) sendNotAllowedNotice(chatID string) {
	if strings.TrimSpace(chatID) == "" || !c.IsRunning() {
		return
	}
	_, err := c.Send(c.runCtx, bus.OutboundMessage{
		Channel: "imsg",
		ChatID:  chatID,
		Content: "not allowed",
	})
	if err != nil {
		logger.DebugCF("imsg", "failed to send not-allowed notice", map[string]any{
			"chat_id": chatID,
			"error":   err.Error(),
		})
	}
}

func deepCoalesceString(v any, keys ...string) string {
	for _, k := range keys {
		if s := deepLookupString(v, k); s != "" {
			return s
		}
	}
	return ""
}

func deepLookupString(v any, key string) string {
	switch vv := v.(type) {
	case map[string]any:
		if val, ok := vv[key]; ok {
			if s := anyToString(val); s != "" {
				return s
			}
		}
		for _, child := range vv {
			if s := deepLookupString(child, key); s != "" {
				return s
			}
		}
	case []any:
		for _, child := range vv {
			if s := deepLookupString(child, key); s != "" {
				return s
			}
		}
	}
	return ""
}

func anyToString(v any) string {
	switch vv := v.(type) {
	case string:
		if strings.TrimSpace(vv) != "" {
			return vv
		}
	case json.Number:
		return vv.String()
	case float64:
		return fmt.Sprintf("%.0f", vv)
	case int, int32, int64, uint, uint32, uint64:
		return fmt.Sprintf("%v", vv)
	}
	return ""
}

func deepBool(v any, keys ...string) bool {
	for _, k := range keys {
		if b, ok := deepLookupBool(v, k); ok {
			return b
		}
	}
	return false
}

func deepLookupBool(v any, key string) (bool, bool) {
	switch vv := v.(type) {
	case map[string]any:
		if val, ok := vv[key]; ok {
			if b, okb := val.(bool); okb {
				return b, true
			}
			// Handle 0/1 style flags.
			rv := reflect.ValueOf(val)
			switch rv.Kind() {
			case reflect.Int, reflect.Int32, reflect.Int64:
				return rv.Int() != 0, true
			case reflect.Uint, reflect.Uint32, reflect.Uint64:
				return rv.Uint() != 0, true
			case reflect.Float32, reflect.Float64:
				return rv.Float() != 0, true
			}
		}
		for _, child := range vv {
			if b, ok := deepLookupBool(child, key); ok {
				return b, true
			}
		}
	case []any:
		for _, child := range vv {
			if b, ok := deepLookupBool(child, key); ok {
				return b, true
			}
		}
	}
	return false, false
}

func (c *IMsgChannel) subscribeRPC(ctx context.Context) error {
	methods := []string{"watch.subscribe", "subscribe"}
	var lastErr error
	for _, m := range methods {
		logger.InfoCF("imsg", "imsg rpc subscribe request", map[string]any{"method": m})
		if _, err := c.rpcCall(ctx, m, map[string]any{}, 6*time.Second); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	return fmt.Errorf("rpc subscribe failed: %w", lastErr)
}

func (c *IMsgChannel) rpcCall(ctx context.Context, method string, params any, timeout time.Duration) (any, error) {
	id := fmt.Sprintf("%d", c.reqSeq.Add(1))
	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	respCh := make(chan rpcResponse, 1)
	c.pendingMu.Lock()
	c.pending[id] = respCh
	c.pendingMu.Unlock()
	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
	}()

	c.writeMu.Lock()
	stdin := c.rpcStdin
	if stdin == nil {
		c.writeMu.Unlock()
		return nil, fmt.Errorf("rpc stdin unavailable")
	}
	_, wErr := io.WriteString(stdin, string(data)+"\n")
	c.writeMu.Unlock()
	if wErr != nil {
		return nil, fmt.Errorf("rpc write failed: %w", wErr)
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	select {
	case <-waitCtx.Done():
		return nil, waitCtx.Err()
	case resp := <-respCh:
		if resp.Error != nil {
			return nil, fmt.Errorf("rpc error %d: %s", resp.Error.Code, resp.Error.Message)
		}
		return resp.Result, nil
	}
}

func (c *IMsgChannel) resolvePending(id any, raw map[string]any) {
	key := anyToString(id)
	if key == "" {
		return
	}
	c.pendingMu.Lock()
	ch, ok := c.pending[key]
	c.pendingMu.Unlock()
	if !ok {
		return
	}
	resp := rpcResponse{Result: raw["result"]}
	if errObj, ok := raw["error"].(map[string]any); ok {
		resp.Error = &rpcError{
			Code:    int(toFloat(errObj["code"])),
			Message: anyToString(errObj["message"]),
		}
	}
	select {
	case ch <- resp:
	default:
	}
}

func firstMap(candidates ...any) map[string]any {
	for _, v := range candidates {
		switch vv := v.(type) {
		case map[string]any:
			if msg, ok := vv["message"].(map[string]any); ok {
				return msg
			}
			if data, ok := vv["data"].(map[string]any); ok {
				return data
			}
			if payload, ok := vv["payload"].(map[string]any); ok {
				return payload
			}
			return vv
		case []any:
			for _, item := range vv {
				if mm, ok := item.(map[string]any); ok {
					return mm
				}
			}
		}
	}
	return map[string]any{}
}

func toFloat(v any) float64 {
	switch vv := v.(type) {
	case float64:
		return vv
	case int:
		return float64(vv)
	case int64:
		return float64(vv)
	case json.Number:
		f, _ := vv.Float64()
		return f
	}
	return 0
}

func isFatalRPCLine(line string) bool {
	l := strings.ToLower(line)
	return strings.Contains(l, "permissiondenied") ||
		strings.Contains(l, "authorization denied") ||
		(strings.Contains(l, "chat.db") && strings.Contains(l, "denied"))
}

func (c *IMsgChannel) setFatalLine(line string) {
	c.fatalMu.Lock()
	defer c.fatalMu.Unlock()
	if c.fatalLine == "" {
		c.fatalLine = line
	}
}

func (c *IMsgChannel) getFatalLine() string {
	c.fatalMu.Lock()
	defer c.fatalMu.Unlock()
	return c.fatalLine
}

func (c *IMsgChannel) setLastLine(line string) {
	c.fatalMu.Lock()
	defer c.fatalMu.Unlock()
	c.lastLine = line
}

func (c *IMsgChannel) getLastLine() string {
	c.fatalMu.Lock()
	defer c.fatalMu.Unlock()
	return c.lastLine
}

func (c *IMsgChannel) markSeen(id string) bool {
	c.seenMu.Lock()
	defer c.seenMu.Unlock()
	if _, ok := c.seenIDs[id]; ok {
		return false
	}
	c.seenIDs[id] = struct{}{}
	return true
}

func (c *IMsgChannel) markSeenEvent(fp string) bool {
	c.seenMu.Lock()
	defer c.seenMu.Unlock()
	now := time.Now()
	const ttl = 2 * time.Minute
	if t, ok := c.seenEvent[fp]; ok && now.Sub(t) <= ttl {
		return false
	}
	c.seenEvent[fp] = now
	// Opportunistic cleanup to keep map bounded.
	for k, t := range c.seenEvent {
		if now.Sub(t) > ttl {
			delete(c.seenEvent, k)
		}
	}
	return true
}

func eventFingerprint(chatID, senderID, content, ts string) string {
	sum := sha1.Sum([]byte(chatID + "|" + senderID + "|" + content + "|" + ts))
	return fmt.Sprintf("%x", sum[:])
}
