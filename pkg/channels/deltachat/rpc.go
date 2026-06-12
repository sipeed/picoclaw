package deltachat

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// rpcRequest is a single JSON-RPC 2.0 request. Delta Chat uses positional
// (array) parameters.
type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      uint64 `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params,omitempty"`
}

// rpcResponse is a single JSON-RPC 2.0 response.
type rpcResponse struct {
	ID     uint64          `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *rpcError) Error() string {
	return fmt.Sprintf("deltachat rpc error %d: %s", e.Code, e.Message)
}

// rpcClient drives a `deltachat-rpc-server` child process over stdio using
// newline-delimited JSON-RPC 2.0. The server answers requests asynchronously
// and out of order, so every call is correlated back to its caller by id.
type rpcClient struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser

	mu      sync.Mutex
	nextID  uint64
	pending map[uint64]chan rpcResponse
	closed  bool
}

// startRPC spawns the RPC server with DC_ACCOUNTS_PATH pointing at dataDir and
// begins the background read loop.
func startRPC(serverPath, dataDir string) (*rpcClient, error) {
	cmd := exec.Command(serverPath)
	cmd.Env = append(cmd.Environ(), "DC_ACCOUNTS_PATH="+dataDir)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("deltachat rpc stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("deltachat rpc stdout: %w", err)
	}
	// Let the server's logs flow to our stderr for easy diagnostics.
	cmd.Stderr = logWriter{}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start deltachat-rpc-server (%s): %w", serverPath, err)
	}

	c := &rpcClient{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		pending: make(map[uint64]chan rpcResponse),
	}
	go c.readLoop()
	return c, nil
}

// readLoop reads newline-delimited responses and dispatches them to waiters.
func (c *rpcClient) readLoop() {
	reader := bufio.NewReader(c.stdout)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			var resp rpcResponse
			if jsonErr := json.Unmarshal(line, &resp); jsonErr == nil && resp.ID != 0 {
				c.mu.Lock()
				ch := c.pending[resp.ID]
				delete(c.pending, resp.ID)
				c.mu.Unlock()
				if ch != nil {
					ch <- resp
				}
			}
		}
		if err != nil {
			c.failAll(err)
			return
		}
	}
}

// failAll wakes every pending caller with an error (used on EOF / shutdown).
func (c *rpcClient) failAll(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	for id, ch := range c.pending {
		ch <- rpcResponse{Error: &rpcError{Code: -1, Message: "rpc closed: " + err.Error()}}
		delete(c.pending, id)
	}
}

// call issues one request and blocks until the matching response arrives, the
// context is canceled, or the server goes away.
func (c *rpcClient) call(ctx context.Context, method string, params ...any) (json.RawMessage, error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, fmt.Errorf("deltachat rpc: connection closed")
	}
	c.nextID++
	id := c.nextID
	ch := make(chan rpcResponse, 1)
	c.pending[id] = ch
	c.mu.Unlock()

	req := rpcRequest{JSONRPC: "2.0", ID: id, Method: method, Params: params}
	data, err := json.Marshal(req)
	if err != nil {
		c.clearPending(id)
		return nil, err
	}
	data = append(data, '\n')

	c.mu.Lock()
	_, err = c.stdin.Write(data)
	c.mu.Unlock()
	if err != nil {
		c.clearPending(id)
		return nil, fmt.Errorf("deltachat rpc write %s: %w", method, err)
	}

	select {
	case <-ctx.Done():
		c.clearPending(id)
		return nil, ctx.Err()
	case resp := <-ch:
		if resp.Error != nil {
			return nil, resp.Error
		}
		return resp.Result, nil
	}
}

func (c *rpcClient) clearPending(id uint64) {
	c.mu.Lock()
	delete(c.pending, id)
	c.mu.Unlock()
}

// close terminates the RPC server process.
func (c *rpcClient) close() {
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()
	if c.stdin != nil {
		_ = c.stdin.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
		_ = c.cmd.Wait()
	}
}

// logWriter forwards deltachat-rpc-server stderr lines into the logger.
type logWriter struct{}

func (logWriter) Write(p []byte) (int, error) {
	logger.DebugC("deltachat", string(p))
	return len(p), nil
}
