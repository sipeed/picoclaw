package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// Client is the transport-agnostic MCP client contract.
type Client interface {
	Start(ctx context.Context) error
	ListTools(ctx context.Context) ([]RemoteTool, error)
	CallTool(ctx context.Context, toolName string, arguments map[string]any) (CallResult, error)
	Close() error
}

// StdioClient speaks MCP over stdio (JSON-RPC framed with Content-Length headers).
type StdioClient struct {
	config ServerConfig
	mode   string

	mu      sync.Mutex
	writeMu sync.Mutex

	started bool
	closed  bool

	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	stderr  io.ReadCloser
	waitCh  chan struct{}
	pending map[string]chan rpcResponse

	nextID uint64
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      string `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type rpcResponseEnvelope struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
	Method  string          `json:"method,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	result json.RawMessage
	rpcErr *rpcError
	err    error
}

type initializeParams struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]any         `json:"capabilities"`
	ClientInfo      map[string]interface{} `json:"clientInfo"`
}

func NewStdioClient(config ServerConfig) *StdioClient {
	return &StdioClient{
		config: config,
		mode:   normalizeProtocol(config.Protocol),
	}
}

func (c *StdioClient) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		return nil
	}
	if strings.TrimSpace(c.config.Command) == "" {
		c.mu.Unlock()
		return fmt.Errorf("mcp server %q command is empty", c.config.Name)
	}

	cmd := exec.Command(c.config.Command, c.config.Args...)
	if c.config.WorkingDir != "" {
		cmd.Dir = c.config.WorkingDir
	}
	cmd.Env = buildProcessEnv(c.config.Env)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		c.mu.Unlock()
		return fmt.Errorf("create stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		c.mu.Unlock()
		return fmt.Errorf("create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		c.mu.Unlock()
		return fmt.Errorf("create stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		c.mu.Unlock()
		return fmt.Errorf("start process: %w", err)
	}

	c.started = true
	c.closed = false
	c.cmd = cmd
	c.stdin = stdin
	c.stdout = stdout
	c.stderr = stderr
	c.waitCh = make(chan struct{})
	c.pending = make(map[string]chan rpcResponse)
	c.mu.Unlock()

	go c.readLoop()
	go c.waitLoop()
	go c.drainStderr()

	initCtx, cancel := withTimeoutIfMissing(ctx, c.config.InitTimeout())
	defer cancel()

	_, err = c.request(initCtx, "initialize", initializeParams{
		ProtocolVersion: "2024-11-05",
		Capabilities: map[string]any{
			"tools": map[string]any{},
		},
		ClientInfo: map[string]any{
			"name":    "picoclaw",
			"version": "dev",
		},
	})
	if err != nil {
		_ = c.Close()
		return fmt.Errorf("initialize failed: %w", err)
	}

	if err := c.notify("notifications/initialized", map[string]any{}); err != nil {
		_ = c.Close()
		return fmt.Errorf("initialized notification failed: %w", err)
	}

	return nil
}

func (c *StdioClient) ListTools(ctx context.Context) ([]RemoteTool, error) {
	if err := c.Start(ctx); err != nil {
		return nil, err
	}

	type listToolsResponse struct {
		Tools []struct {
			Name        string         `json:"name"`
			Description string         `json:"description,omitempty"`
			InputSchema map[string]any `json:"inputSchema"`
		} `json:"tools"`
		NextCursor string `json:"nextCursor,omitempty"`
	}

	allTools := make([]RemoteTool, 0, 8)
	cursor := ""

	for page := 0; page < maxToolListPages; page++ {
		params := map[string]any{}
		if cursor != "" {
			params["cursor"] = cursor
		}

		callCtx, cancel := withTimeoutIfMissing(ctx, c.config.CallTimeout())
		raw, err := c.request(callCtx, "tools/list", params)
		cancel()
		if err != nil {
			return nil, err
		}

		var response listToolsResponse
		if err := json.Unmarshal(raw, &response); err != nil {
			return nil, fmt.Errorf("decode tools/list response: %w", err)
		}

		for _, tool := range response.Tools {
			allTools = append(allTools, RemoteTool{
				Name:        tool.Name,
				Description: tool.Description,
				InputSchema: tool.InputSchema,
			})
		}

		if response.NextCursor == "" {
			return allTools, nil
		}
		cursor = response.NextCursor
	}

	return nil, fmt.Errorf("tools/list exceeded %d pages", maxToolListPages)
}

func (c *StdioClient) CallTool(ctx context.Context, toolName string, arguments map[string]any) (CallResult, error) {
	if err := c.Start(ctx); err != nil {
		return CallResult{}, err
	}

	callCtx, cancel := withTimeoutIfMissing(ctx, c.config.CallTimeout())
	defer cancel()

	raw, err := c.request(callCtx, "tools/call", map[string]any{
		"name":      toolName,
		"arguments": arguments,
	})
	if err != nil {
		return CallResult{}, err
	}

	return formatCallPayload(raw, c.config.ResponseLimit())
}

func (c *StdioClient) Close() error {
	c.mu.Lock()
	if !c.started || c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	cmd := c.cmd
	stdin := c.stdin
	waitCh := c.waitCh
	c.mu.Unlock()

	c.failPending(errors.New("mcp client closed"))

	if stdin != nil {
		_ = stdin.Close()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}

	if waitCh != nil {
		select {
		case <-waitCh:
		case <-time.After(2 * time.Second):
		}
	}
	return nil
}

func (c *StdioClient) request(ctx context.Context, method string, params any) (json.RawMessage, error) {
	id := strconv.FormatUint(atomic.AddUint64(&c.nextID, 1), 10)
	responseCh := make(chan rpcResponse, 1)

	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, fmt.Errorf("mcp server %q is closed", c.config.Name)
	}
	c.pending[id] = responseCh
	c.mu.Unlock()

	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}
	if err := c.writeMessage(req); err != nil {
		c.removePending(id)
		return nil, err
	}

	select {
	case <-ctx.Done():
		c.removePending(id)
		return nil, ctx.Err()
	case response := <-responseCh:
		if response.err != nil {
			return nil, response.err
		}
		if response.rpcErr != nil {
			return nil, fmt.Errorf("mcp error %d: %s", response.rpcErr.Code, response.rpcErr.Message)
		}
		return response.result, nil
	}
}

func (c *StdioClient) notify(method string, params any) error {
	req := rpcRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	return c.writeMessage(req)
}

func (c *StdioClient) writeMessage(payload any) error {
	c.mu.Lock()
	if c.closed || c.stdin == nil {
		c.mu.Unlock()
		return fmt.Errorf("mcp server %q is not writable", c.config.Name)
	}
	stdin := c.stdin
	c.mu.Unlock()

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal json-rpc payload: %w", err)
	}

	if c.mode == ProtocolJSONLines {
		c.writeMu.Lock()
		defer c.writeMu.Unlock()

		if _, err := stdin.Write(append(data, '\n')); err != nil {
			return fmt.Errorf("write jsonl body: %w", err)
		}
		return nil
	}

	frameHeader := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if _, err := io.WriteString(stdin, frameHeader); err != nil {
		return fmt.Errorf("write frame header: %w", err)
	}
	if _, err := stdin.Write(data); err != nil {
		return fmt.Errorf("write frame body: %w", err)
	}
	return nil
}

func (c *StdioClient) readLoop() {
	if c.mode == ProtocolJSONLines {
		c.readJSONLLoop()
		return
	}

	c.readMCPFrameLoop()
}

func (c *StdioClient) readMCPFrameLoop() {
	reader := bufio.NewReader(c.stdout)

	for {
		payload, err := readFramePayload(reader)
		if err != nil {
			c.failPending(err)
			return
		}

		var envelope rpcResponseEnvelope
		if err := json.Unmarshal(payload, &envelope); err != nil {
			continue
		}
		c.dispatchResponse(envelope)
	}
}

func (c *StdioClient) readJSONLLoop() {
	scanner := bufio.NewScanner(c.stdout)
	scanner.Buffer(make([]byte, 0, defaultScannerBufferBytes), maxFrameBytes)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var envelope rpcResponseEnvelope
		if err := json.Unmarshal([]byte(line), &envelope); err != nil {
			continue
		}
		c.dispatchResponse(envelope)
	}

	if err := scanner.Err(); err != nil {
		c.failPending(err)
		return
	}
	c.failPending(io.EOF)
}

func (c *StdioClient) dispatchResponse(envelope rpcResponseEnvelope) {
	if len(envelope.ID) == 0 {
		return
	}

	id, ok := parseRPCID(envelope.ID)
	if !ok {
		return
	}

	c.mu.Lock()
	responseCh := c.pending[id]
	if responseCh != nil {
		delete(c.pending, id)
	}
	c.mu.Unlock()

	if responseCh == nil {
		return
	}

	response := rpcResponse{
		result: envelope.Result,
		rpcErr: envelope.Error,
	}
	select {
	case responseCh <- response:
	default:
	}
}

func (c *StdioClient) waitLoop() {
	c.mu.Lock()
	cmd := c.cmd
	waitCh := c.waitCh
	serverName := c.config.Name
	c.mu.Unlock()

	if cmd == nil {
		if waitCh != nil {
			close(waitCh)
		}
		return
	}

	err := cmd.Wait()
	if waitCh != nil {
		close(waitCh)
	}
	if err != nil {
		logger.WarnCF("mcp", "MCP process exited with error",
			map[string]any{
				"server": serverName,
				"error":  err.Error(),
			})
	}
}

func (c *StdioClient) drainStderr() {
	c.mu.Lock()
	stderr := c.stderr
	serverName := c.config.Name
	c.mu.Unlock()

	if stderr == nil {
		return
	}

	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		logger.DebugCF("mcp", "MCP server stderr",
			map[string]any{
				"server": serverName,
				"line":   line,
			})
	}
}

func (c *StdioClient) failPending(err error) {
	c.mu.Lock()
	pending := c.pending
	c.pending = make(map[string]chan rpcResponse)
	c.mu.Unlock()

	if len(pending) == 0 {
		return
	}

	for _, ch := range pending {
		select {
		case ch <- rpcResponse{err: err}:
		default:
		}
	}
}

func (c *StdioClient) removePending(id string) {
	c.mu.Lock()
	delete(c.pending, id)
	c.mu.Unlock()
}

func readFramePayload(reader *bufio.Reader) ([]byte, error) {
	contentLength := -1

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		trimmed := strings.TrimRight(line, "\r\n")
		if trimmed == "" {
			break
		}

		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			continue
		}
		headerName := strings.TrimSpace(strings.ToLower(parts[0]))
		if headerName != "content-length" {
			continue
		}
		value := strings.TrimSpace(parts[1])
		length, err := strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("invalid content-length %q: %w", value, err)
		}
		contentLength = length
	}

	if contentLength <= 0 {
		return nil, fmt.Errorf("missing content-length")
	}
	if contentLength > maxFrameBytes {
		return nil, fmt.Errorf("frame too large (%d bytes)", contentLength)
	}

	payload := make([]byte, contentLength)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func parseRPCID(raw json.RawMessage) (string, bool) {
	var stringID string
	if err := json.Unmarshal(raw, &stringID); err == nil {
		return stringID, true
	}

	var numberID float64
	if err := json.Unmarshal(raw, &numberID); err == nil {
		return strconv.FormatInt(int64(numberID), 10), true
	}

	return "", false
}

func withTimeoutIfMissing(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if _, hasDeadline := parent.Deadline(); hasDeadline {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, timeout)
}

func buildProcessEnv(custom map[string]string) []string {
	base := os.Environ()
	if len(custom) == 0 {
		return base
	}

	keys := make([]string, 0, len(custom))
	for key := range custom {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	env := make([]string, 0, len(base)+len(keys))
	env = append(env, base...)
	for _, key := range keys {
		env = append(env, key+"="+custom[key])
	}
	return env
}

func normalizeProtocol(protocol string) string {
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case "", ProtocolMCPFrames:
		return ProtocolMCPFrames
	case ProtocolJSONLines:
		return ProtocolJSONLines
	default:
		return ProtocolMCPFrames
	}
}
