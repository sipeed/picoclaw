//go:build cdp

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sipeed/picoclaw/pkg/logger"
)

const (
	cdpSendTimeout    = 30 * time.Second
	cdpConnectTimeout = 10 * time.Second
)

// CDPTarget represents a Chrome debuggable target.
type CDPTarget struct {
	Type                 string `json:"type"`
	URL                  string `json:"url"`
	Title                string `json:"title"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

// cdpMessage represents a CDP protocol message (request or response).
type cdpMessage struct {
	ID     int64            `json:"id,omitempty"`
	Method string           `json:"method,omitempty"`
	Params json.RawMessage  `json:"params,omitempty"`
	Result json.RawMessage  `json:"result,omitempty"`
	Error  *cdpErrorPayload `json:"error,omitempty"`
}

type cdpErrorPayload struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// cdpPending tracks an in-flight CDP request.
type cdpPending struct {
	ch    chan cdpResponse
	timer *time.Timer
}

type cdpResponse struct {
	Result json.RawMessage
	Err    error
}

// CDPClient is a lightweight Chrome DevTools Protocol client
// using a single WebSocket connection.
type CDPClient struct {
	conn       *websocket.Conn
	mu         sync.Mutex
	idCount    int64
	pending    map[int64]*cdpPending
	events     map[string][]cdpHandler
	eventMu    sync.RWMutex
	handlerSeq int64 // monotonic handler ID
	closed     atomic.Bool
	done       chan struct{}
}

// cdpHandler associates a unique ID with an event callback for reliable removal.
type cdpHandler struct {
	id int64
	fn func(json.RawMessage)
}

// NewCDPClient connects to a Chrome CDP endpoint.
// The endpoint can be:
//   - An HTTP URL like "http://127.0.0.1:9222" (will discover targets)
//   - A WebSocket URL like "ws://127.0.0.1:9222/devtools/page/..."
func NewCDPClient(endpoint string) (*CDPClient, error) {
	wsURL, err := resolveWSEndpoint(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve CDP endpoint: %w", err)
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: cdpConnectTimeout,
	}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to CDP at %s: %w", wsURL, err)
	}

	c := &CDPClient{
		conn:    conn,
		pending: make(map[int64]*cdpPending),
		events:  make(map[string][]cdpHandler),
		done:    make(chan struct{}),
	}

	go c.readLoop()

	return c, nil
}

// Send sends a CDP command and waits for the response.
func (c *CDPClient) Send(method string, params map[string]any) (json.RawMessage, error) {
	return c.SendWithTimeout(method, params, cdpSendTimeout)
}

// SendWithTimeout sends a CDP command with a custom timeout.
func (c *CDPClient) SendWithTimeout(method string, params map[string]any, timeout time.Duration) (json.RawMessage, error) {
	if c.closed.Load() {
		return nil, fmt.Errorf("CDP connection closed")
	}

	id := atomic.AddInt64(&c.idCount, 1)

	// Build request message
	msg := map[string]any{
		"id":     id,
		"method": method,
	}
	if params != nil {
		msg["params"] = params
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal CDP message: %w", err)
	}

	// Register pending request
	p := &cdpPending{
		ch:    make(chan cdpResponse, 1),
		timer: time.NewTimer(timeout),
	}

	c.mu.Lock()
	c.pending[id] = p
	c.mu.Unlock()

	// Clean up on return
	defer func() {
		p.timer.Stop()
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
	}()

	// Send
	c.mu.Lock()
	err = c.conn.WriteMessage(websocket.TextMessage, data)
	c.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("failed to send CDP message: %w", err)
	}

	// Wait for response or timeout
	select {
	case resp := <-p.ch:
		return resp.Result, resp.Err
	case <-p.timer.C:
		return nil, fmt.Errorf("CDP command %q timed out after %v", method, timeout)
	case <-c.done:
		return nil, fmt.Errorf("CDP connection closed while waiting for %q", method)
	}
}

// On registers an event listener for a CDP event.
// Returns a handler ID that can be passed to Off for removal.
func (c *CDPClient) On(event string, handler func(json.RawMessage)) int64 {
	c.eventMu.Lock()
	defer c.eventMu.Unlock()
	id := atomic.AddInt64(&c.handlerSeq, 1)
	c.events[event] = append(c.events[event], cdpHandler{id: id, fn: handler})
	return id
}

// Off removes an event handler by its ID.
func (c *CDPClient) Off(event string, handlerID int64) {
	c.eventMu.Lock()
	defer c.eventMu.Unlock()
	handlers := c.events[event]
	for i, h := range handlers {
		if h.id == handlerID {
			c.events[event] = append(handlers[:i], handlers[i+1:]...)
			return
		}
	}
}

// WaitForEvent blocks until the named CDP event fires or timeout expires.
func (c *CDPClient) WaitForEvent(event string, timeout time.Duration) (json.RawMessage, error) {
	ch := make(chan json.RawMessage, 1)
	var once sync.Once
	handler := func(params json.RawMessage) {
		once.Do(func() {
			ch <- params
		})
	}

	handlerID := c.On(event, handler)
	defer c.Off(event, handlerID)

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case params := <-ch:
		return params, nil
	case <-timer.C:
		return nil, fmt.Errorf("timeout waiting for CDP event %q after %v", event, timeout)
	case <-c.done:
		return nil, fmt.Errorf("CDP connection closed while waiting for event %q", event)
	}
}

// Close closes the WebSocket connection.
func (c *CDPClient) Close() error {
	if c.closed.Swap(true) {
		return nil // already closed
	}
	close(c.done)

	// Cancel all pending requests (non-blocking to avoid deadlock)
	c.mu.Lock()
	for id, p := range c.pending {
		select {
		case p.ch <- cdpResponse{Err: fmt.Errorf("CDP connection closed")}:
		default:
		}
		delete(c.pending, id)
	}
	c.mu.Unlock()

	return c.conn.Close()
}

// readLoop continuously reads WebSocket messages and dispatches them.
func (c *CDPClient) readLoop() {
	defer func() {
		// Use Swap to prevent double-close of done channel
		if !c.closed.Swap(true) {
			close(c.done)
		}
	}()

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if !c.closed.Load() {
				logger.DebugCF("tool", "CDP WebSocket read error",
					map[string]any{"error": err.Error()})
			}
			return
		}

		var msg cdpMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			logger.DebugCF("tool", "CDP failed to parse message",
				map[string]any{"error": err.Error()})
			continue
		}

		if msg.ID > 0 {
			// Response to a pending request
			c.mu.Lock()
			p, ok := c.pending[msg.ID]
			c.mu.Unlock()

			if ok {
				if msg.Error != nil {
					p.ch <- cdpResponse{
						Err: fmt.Errorf("CDP error %d: %s", msg.Error.Code, msg.Error.Message),
					}
				} else {
					p.ch <- cdpResponse{Result: msg.Result}
				}
			}
		} else if msg.Method != "" {
			// Event notification
			c.eventMu.RLock()
			handlers := make([]cdpHandler, len(c.events[msg.Method]))
			copy(handlers, c.events[msg.Method])
			c.eventMu.RUnlock()

			for _, h := range handlers {
				h.fn(msg.Params)
			}
		}
	}
}

// resolveWSEndpoint resolves a CDP endpoint to a WebSocket URL.
// If the input is already a ws:// URL, it's returned as-is.
// If it's an http:// URL, it fetches /json/version to discover the WS endpoint.
func resolveWSEndpoint(endpoint string) (string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}

	// Already a WebSocket URL
	if parsed.Scheme == "ws" || parsed.Scheme == "wss" {
		return endpoint, nil
	}

	// HTTP endpoint — discover targets
	if parsed.Scheme == "http" || parsed.Scheme == "https" {
		return discoverWSEndpoint(endpoint)
	}

	// Try as plain host:port → http
	if parsed.Scheme == "" {
		return discoverWSEndpoint("http://" + endpoint)
	}

	return "", fmt.Errorf("unsupported CDP endpoint scheme: %s", parsed.Scheme)
}

// discoverWSEndpoint fetches page targets from a Chrome HTTP endpoint.
// It prefers page-level targets over the browser-level endpoint because
// Page.enable and other page commands only work on page targets.
func discoverWSEndpoint(httpEndpoint string) (string, error) {
	client := &http.Client{Timeout: cdpConnectTimeout}

	// Try /json first to find a page target (preferred — supports Page.enable)
	resp, err := client.Get(httpEndpoint + "/json")
	if err == nil {
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err == nil {
			var targets []CDPTarget
			if err := json.Unmarshal(body, &targets); err == nil {
				// Prefer page targets
				for _, t := range targets {
					if t.Type == "page" && t.WebSocketDebuggerURL != "" {
						return t.WebSocketDebuggerURL, nil
					}
				}
				// Any target with a WS URL
				for _, t := range targets {
					if t.WebSocketDebuggerURL != "" {
						return t.WebSocketDebuggerURL, nil
					}
				}
			}
		}
	}

	// Fallback: /json/version gives browser-level WS URL
	// Note: browser-level targets don't support Page.enable, so this is
	// only useful for browser-wide operations.
	resp2, err := client.Get(httpEndpoint + "/json/version")
	if err != nil {
		return "", fmt.Errorf("failed to discover CDP targets: %w", err)
	}
	defer resp2.Body.Close()

	body2, err := io.ReadAll(resp2.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read CDP version response: %w", err)
	}

	var version struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := json.Unmarshal(body2, &version); err == nil && version.WebSocketDebuggerURL != "" {
		return version.WebSocketDebuggerURL, nil
	}

	return "", fmt.Errorf("no debuggable targets found at %s", httpEndpoint)
}

// EnablePage enables the Page domain on the CDP session.
func (c *CDPClient) EnablePage() error {
	_, err := c.Send("Page.enable", nil)
	return err
}

// EnableRuntime enables the Runtime domain on the CDP session.
func (c *CDPClient) EnableRuntime() error {
	_, err := c.Send("Runtime.enable", nil)
	return err
}

// EnableDOM enables the DOM domain on the CDP session.
func (c *CDPClient) EnableDOM() error {
	_, err := c.Send("DOM.enable", nil)
	return err
}

// Evaluate executes JavaScript in the page context and returns the result value.
func (c *CDPClient) Evaluate(ctx context.Context, expression string) (json.RawMessage, error) {
	result, err := c.Send("Runtime.evaluate", map[string]any{
		"expression":    expression,
		"returnByValue": true,
		"awaitPromise":  true,
	})
	if err != nil {
		return nil, err
	}

	// Parse the evaluation result
	var evalResult struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
		ExceptionDetails *struct {
			Exception struct {
				Description string `json:"description"`
			} `json:"exception"`
		} `json:"exceptionDetails"`
	}

	if err := json.Unmarshal(result, &evalResult); err != nil {
		return nil, fmt.Errorf("failed to parse eval result: %w", err)
	}

	if evalResult.ExceptionDetails != nil {
		return nil, fmt.Errorf("JS exception: %s", evalResult.ExceptionDetails.Exception.Description)
	}

	return evalResult.Result.Value, nil
}

// Navigate navigates the page to the given URL and waits for DOM ready.
// Uses Page.domContentEventFired (DOM parsed) instead of Page.loadEventFired
// (all resources loaded) for faster navigation — we don't need images/CSS to interact.
func (c *CDPClient) Navigate(ctx context.Context, targetURL string) error {
	// Use a channel-based one-shot listener for DOM content loaded
	domCh := make(chan struct{}, 1)
	var domOnce sync.Once
	handler := func(_ json.RawMessage) {
		domOnce.Do(func() {
			domCh <- struct{}{}
		})
	}

	handlerID := c.On("Page.domContentEventFired", handler)
	defer c.Off("Page.domContentEventFired", handlerID)

	_, err := c.Send("Page.navigate", map[string]any{
		"url": targetURL,
	})
	if err != nil {
		return fmt.Errorf("navigation failed: %w", err)
	}

	// Wait for DOM content loaded with timeout
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()

	select {
	case <-domCh:
		return nil
	case <-timer.C:
		return fmt.Errorf("page load timed out after 30s")
	case <-ctx.Done():
		return ctx.Err()
	}
}

// CaptureScreenshot takes a screenshot and returns base64-encoded PNG.
func (c *CDPClient) CaptureScreenshot(format string, quality int) (string, error) {
	params := map[string]any{
		"format": format,
	}
	if format == "jpeg" && quality > 0 {
		params["quality"] = quality
	}

	result, err := c.Send("Page.captureScreenshot", params)
	if err != nil {
		return "", err
	}

	var screenshot struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(result, &screenshot); err != nil {
		return "", fmt.Errorf("failed to parse screenshot result: %w", err)
	}

	return screenshot.Data, nil
}

// InjectScript injects JavaScript to be evaluated on every new document.
func (c *CDPClient) InjectScript(source string) error {
	_, err := c.Send("Page.addScriptToEvaluateOnNewDocument", map[string]any{
		"source": source,
	})
	return err
}

// DispatchMouseEvent sends a mouse event at the given coordinates.
func (c *CDPClient) DispatchMouseEvent(eventType string, x, y float64, button string, clickCount int) error {
	_, err := c.Send("Input.dispatchMouseEvent", map[string]any{
		"type":       eventType,
		"x":         x,
		"y":         y,
		"button":    button,
		"clickCount": clickCount,
	})
	return err
}

// InsertText inserts text at the current cursor position.
func (c *CDPClient) InsertText(text string) error {
	_, err := c.Send("Input.insertText", map[string]any{
		"text": text,
	})
	return err
}

// DispatchKeyEvent sends a keyboard event.
func (c *CDPClient) DispatchKeyEvent(eventType, key string, modifiers int) error {
	params := map[string]any{
		"type": eventType,
		"key":  key,
	}
	if modifiers > 0 {
		params["modifiers"] = modifiers
	}
	_, err := c.Send("Input.dispatchKeyEvent", params)
	return err
}
