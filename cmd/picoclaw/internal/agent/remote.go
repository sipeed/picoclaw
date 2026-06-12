package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ergochat/readline"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/channels/pico"
)

const (
	remoteGeneratedSessionPrefix = "cli:"
	remoteOneShotFirstTimeout    = 30 * time.Second
	remoteOneShotIdleTimeout     = 1500 * time.Millisecond
)

var errRemoteReadlineUnavailable = errors.New("remote readline unavailable")

type remoteClient struct {
	conn *websocket.Conn
	out  *lockedWriter
	mu   sync.Mutex
}

type remoteEventResult struct {
	Displayed bool
	Err       error
}

type remoteInputResult struct {
	Line string
	Err  error
}

type lockedWriter struct {
	mu            sync.Mutex
	w             io.Writer
	refreshPrompt func()
	typingVisible bool
}

func (w *lockedWriter) Printf(format string, args ...any) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.prepareAsyncOutputLocked()
	fmt.Fprintf(w.w, format, args...)
	w.refreshPromptLocked()
}

func (w *lockedWriter) SetWriter(out io.Writer) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if out != nil {
		w.w = out
	}
}

func (w *lockedWriter) Writer() io.Writer {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.w
}

func (w *lockedWriter) SetRefreshPrompt(refresh func()) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.refreshPrompt = refresh
}

func (w *lockedWriter) StartTyping() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.typingVisible {
		return
	}
	w.clearPromptLineLocked()
	fmt.Fprintln(w.w, "[typing]")
	w.typingVisible = true
	w.refreshPromptLocked()
}

func (w *lockedWriter) StopTyping() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.typingVisible {
		w.clearTypingLineLocked()
	}
	w.refreshPromptLocked()
}

func (w *lockedWriter) prepareAsyncOutputLocked() {
	w.clearPromptLineLocked()
	if w.typingVisible {
		w.clearTypingLineLocked()
	}
}

func (w *lockedWriter) clearPromptLineLocked() {
	if w.refreshPrompt == nil {
		return
	}
	fmt.Fprint(w.w, "\r\033[2K")
}

func (w *lockedWriter) clearTypingLineLocked() {
	if !w.typingVisible {
		return
	}
	if w.refreshPrompt != nil {
		fmt.Fprint(w.w, "\r\033[2K\033[1A\r\033[2K\033[1M")
	}
	w.typingVisible = false
}

func (w *lockedWriter) refreshPromptLocked() {
	if w.refreshPrompt != nil {
		w.refreshPrompt()
	}
}

func remoteAgentCmd(
	parentCtx context.Context,
	rawURL string,
	token string,
	message string,
	sessionID string,
	in io.Reader,
	out io.Writer,
) error {
	if sessionID == "" {
		sessionID = newRemoteSessionID()
	}
	if token == "" {
		token = os.Getenv("PICO_TOKEN")
	}
	if in == nil {
		in = os.Stdin
	}
	if out == nil {
		out = os.Stdout
	}

	signalCtx, stopSignals := signal.NotifyContext(parentCtx, os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	ctx, cancel := context.WithCancel(signalCtx)
	defer cancel()

	client, err := newRemoteClient(ctx, rawURL, token, sessionID, out)
	if err != nil {
		return err
	}
	defer client.Close(websocket.CloseNormalClosure)

	go func() {
		<-ctx.Done()
		client.Close(websocket.CloseGoingAway)
	}()

	if message != "" {
		return client.RunOneShot(ctx, sessionID, message)
	}
	return client.RunInteractive(ctx, sessionID, in)
}

func newRemoteSessionID() string {
	return remoteGeneratedSessionPrefix + uuid.NewString()
}

func newRemoteClient(
	ctx context.Context,
	rawURL string,
	token string,
	sessionID string,
	out io.Writer,
) (*remoteClient, error) {
	wsURL, err := remoteURLWithSession(rawURL, sessionID)
	if err != nil {
		return nil, err
	}

	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, wsURL, remoteAuthHeader(token))
	if err != nil {
		return nil, remoteDialError(err, resp)
	}
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}

	return &remoteClient{
		conn: conn,
		out:  &lockedWriter{w: out},
	}, nil
}

func remoteDialError(err error, resp *http.Response) error {
	if resp == nil {
		return fmt.Errorf("connect remote pico websocket: %w", err)
	}
	defer resp.Body.Close()

	body := ""
	if resp.Body != nil {
		raw, readErr := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		if readErr == nil {
			body = strings.TrimSpace(string(raw))
		}
	}
	if body == "" || !remoteResponseBodyIsText(resp.Header.Get("Content-Type")) {
		return fmt.Errorf("connect remote pico websocket: %w: HTTP %s", err, resp.Status)
	}
	return fmt.Errorf("connect remote pico websocket: %w: HTTP %s: %s", err, resp.Status, body)
}

func remoteResponseBodyIsText(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = strings.TrimSpace(strings.ToLower(contentType))
	}
	return mediaType == "" ||
		strings.HasPrefix(mediaType, "text/") ||
		mediaType == "application/json" ||
		strings.HasSuffix(mediaType, "+json")
}

func (c *remoteClient) Close(code int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return
	}
	_ = c.conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(code, ""),
		time.Now().Add(time.Second),
	)
	_ = c.conn.Close()
	c.conn = nil
}

func (c *remoteClient) Send(sessionID, text string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return fmt.Errorf("remote connection is closed")
	}
	return c.conn.WriteJSON(buildRemoteMessageSend(sessionID, text))
}

func (c *remoteClient) RunOneShot(ctx context.Context, sessionID, message string) error {
	events := make(chan remoteEventResult, 8)
	go c.readLoop(ctx, events)

	if err := c.Send(sessionID, message); err != nil {
		return err
	}

	timer := time.NewTimer(remoteOneShotFirstTimeout)
	defer timer.Stop()
	seenResponse := false

	for {
		select {
		case <-ctx.Done():
			return nil
		case evt, ok := <-events:
			if !ok {
				return nil
			}
			if evt.Err != nil {
				if seenResponse || errors.Is(evt.Err, io.EOF) {
					return nil
				}
				return evt.Err
			}
			if evt.Displayed {
				seenResponse = true
				resetTimer(timer, remoteOneShotIdleTimeout)
			}
		case <-timer.C:
			if seenResponse {
				return nil
			}
			return fmt.Errorf("timed out waiting for remote response")
		}
	}
}

func (c *remoteClient) RunInteractive(ctx context.Context, sessionID string, in io.Reader) error {
	err := c.runReadlineInteractive(ctx, sessionID, in)
	if errors.Is(err, errRemoteReadlineUnavailable) {
		return c.runScannerInteractive(ctx, sessionID, in)
	}
	return err
}

func (c *remoteClient) runReadlineInteractive(ctx context.Context, sessionID string, in io.Reader) error {
	prompt := fmt.Sprintf("%s You: ", internal.Logo)
	baseOut := c.out.Writer()
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          prompt,
		HistoryFile:     filepath.Join(os.TempDir(), ".picoclaw_history"),
		HistoryLimit:    100,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
		Stdin:           in,
		Stdout:          baseOut,
		Stderr:          baseOut,
	})
	if err != nil {
		return fmt.Errorf("%w: %v", errRemoteReadlineUnavailable, err)
	}
	defer rl.Close()
	defer c.out.SetWriter(baseOut)
	defer c.out.SetRefreshPrompt(nil)
	c.out.SetWriter(baseOut)

	c.out.Printf("%s Remote mode (Ctrl+C to exit)\n", internal.Logo)
	c.out.Printf("Connected to remote Pico session %s\n\n", sessionID)
	c.out.SetRefreshPrompt(rl.Refresh)

	go func() {
		<-ctx.Done()
		_ = rl.Close()
	}()

	events := make(chan remoteEventResult, 8)
	go c.readLoop(ctx, events)

	inputs := make(chan remoteInputResult, 1)
	go readRemoteLines(rl, inputs)

	for {
		select {
		case <-ctx.Done():
			c.out.Printf("\nGoodbye!\n")
			return nil
		case evt, ok := <-events:
			if !ok {
				return nil
			}
			if evt.Err != nil {
				return evt.Err
			}
		case input, ok := <-inputs:
			if !ok {
				return nil
			}
			if input.Err != nil {
				if input.Err == readline.ErrInterrupt || errors.Is(input.Err, io.EOF) {
					c.out.Printf("\nGoodbye!\n")
					return nil
				}
				return input.Err
			}
			text := strings.TrimSpace(input.Line)
			if text == "" {
				continue
			}
			if text == "exit" || text == "quit" {
				c.out.Printf("Goodbye!\n")
				return nil
			}
			if err := c.Send(sessionID, text); err != nil {
				return err
			}
		}
	}
}

func (c *remoteClient) runScannerInteractive(ctx context.Context, sessionID string, in io.Reader) error {
	events := make(chan remoteEventResult, 8)
	go c.readLoop(ctx, events)

	lines := make(chan string)
	readErrs := make(chan error, 1)
	go scanInput(in, lines, readErrs)

	prompt := fmt.Sprintf("%s You: ", internal.Logo)
	c.out.Printf("%s Remote mode (Ctrl+C to exit)\n", internal.Logo)
	c.out.Printf("Connected to remote Pico session %s\n\n", sessionID)
	c.out.Printf("%s", prompt)
	for {
		select {
		case <-ctx.Done():
			c.out.Printf("\nGoodbye!\n")
			return nil
		case evt, ok := <-events:
			if !ok {
				return nil
			}
			if evt.Err != nil {
				return evt.Err
			}
		case err := <-readErrs:
			if err == nil {
				return nil
			}
			return err
		case line, ok := <-lines:
			if !ok {
				return nil
			}
			input := strings.TrimSpace(line)
			if input == "" {
				c.out.Printf("%s", prompt)
				continue
			}
			if input == "exit" || input == "quit" {
				c.out.Printf("Goodbye!\n")
				return nil
			}
			if err := c.Send(sessionID, input); err != nil {
				return err
			}
			c.out.Printf("%s", prompt)
		}
	}
}

func readRemoteLines(rl *readline.Instance, inputs chan<- remoteInputResult) {
	defer close(inputs)
	for {
		line, err := rl.Readline()
		inputs <- remoteInputResult{Line: line, Err: err}
		if err != nil {
			return
		}
	}
}

func (c *remoteClient) readLoop(ctx context.Context, events chan<- remoteEventResult) {
	defer close(events)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		c.mu.Lock()
		conn := c.conn
		c.mu.Unlock()
		if conn == nil {
			return
		}

		var msg pico.PicoMessage
		if err := conn.ReadJSON(&msg); err != nil {
			if ctx.Err() != nil {
				return
			}
			events <- remoteEventResult{Err: err}
			return
		}
		events <- remoteEventResult{Displayed: renderRemoteEvent(c.out, msg)}
	}
}

func scanInput(in io.Reader, lines chan<- string, errs chan<- error) {
	defer close(lines)

	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		lines <- scanner.Text()
	}
	errs <- scanner.Err()
}

func resetTimer(timer *time.Timer, duration time.Duration) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(duration)
}

func remoteURLWithSession(rawURL, sessionID string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	sessionID = strings.TrimSpace(sessionID)
	if rawURL == "" {
		return "", fmt.Errorf("remote URL is required")
	}
	if sessionID == "" {
		return "", fmt.Errorf("session ID is required")
	}
	if !strings.Contains(rawURL, "://") && !strings.HasPrefix(rawURL, "//") {
		rawURL = "ws://" + rawURL
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse remote URL: %w", err)
	}
	switch u.Scheme {
	case "":
		u.Scheme = "ws"
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	case "ws", "wss":
	default:
		return "", fmt.Errorf("unsupported remote URL scheme %q", u.Scheme)
	}
	if u.Host == "" {
		return "", fmt.Errorf("remote URL must include a host")
	}

	q := u.Query()
	q.Set("session_id", sessionID)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func remoteAuthHeader(token string) http.Header {
	header := http.Header{}
	token = strings.TrimSpace(token)
	if token != "" {
		header.Set("Authorization", "Bearer "+token)
	}
	return header
}

func buildRemoteMessageSend(sessionID, text string) pico.PicoMessage {
	return pico.PicoMessage{
		Type:      pico.TypeMessageSend,
		ID:        uuid.NewString(),
		SessionID: sessionID,
		Timestamp: time.Now().UnixMilli(),
		Payload: map[string]any{
			pico.PayloadKeyContent:    text,
			pico.PayloadKeyClientKind: pico.ClientKindRemoteCLI,
			pico.PayloadKeyClientName: "picoclaw agent --remote",
			pico.PayloadKeyTransport:  pico.TransportWebSocket,
			"sender_id":               "picoclaw-cli",
			"sender_name":             "PicoClaw CLI",
		},
	}
}

func renderRemoteEvent(out interface{ Printf(string, ...any) }, msg pico.PicoMessage) bool {
	switch msg.Type {
	case pico.TypeMessageCreate, pico.TypeMessageUpdate:
		return renderRemoteText(out, msg.Payload)
	case pico.TypeMessageDelete:
		messageID := payloadString(msg.Payload, "message_id")
		if messageID == "" {
			messageID = msg.ID
		}
		out.Printf("[message deleted: %s]\n", messageID)
		return true
	case pico.TypeTypingStart:
		startRemoteTyping(out)
		return false
	case pico.TypeTypingStop:
		stopRemoteTyping(out)
		return false
	case pico.TypeMediaCreate:
		displayed := renderRemoteText(out, msg.Payload)
		for _, media := range remoteMediaStrings(msg.Payload) {
			out.Printf("[media] %s\n", media)
			displayed = true
		}
		return displayed
	case pico.TypeError:
		code := payloadString(msg.Payload, "code")
		message := payloadString(msg.Payload, "message")
		switch {
		case code != "" && message != "":
			out.Printf("error[%s]: %s\n", code, message)
		case message != "":
			out.Printf("error: %s\n", message)
		default:
			out.Printf("error: %s\n", payloadJSON(msg.Payload))
		}
		return true
	case pico.TypePong:
		out.Printf("[pong]\n")
		return false
	default:
		out.Printf("[event %s] %s\n", msg.Type, payloadJSON(msg.Payload))
		return false
	}
}

func startRemoteTyping(out interface{ Printf(string, ...any) }) {
	if typingOut, ok := out.(interface{ StartTyping() }); ok {
		typingOut.StartTyping()
		return
	}
	out.Printf("[typing]\n")
}

func stopRemoteTyping(out interface{ Printf(string, ...any) }) {
	if typingOut, ok := out.(interface{ StopTyping() }); ok {
		typingOut.StopTyping()
	}
}

func renderRemoteText(out interface{ Printf(string, ...any) }, payload map[string]any) bool {
	if isRemoteThoughtPayload(payload) {
		return false
	}
	content := strings.TrimSpace(payloadString(payload, pico.PayloadKeyContent))
	if content == "" {
		return false
	}
	out.Printf("%s\n", content)
	return true
}

func isRemoteThoughtPayload(payload map[string]any) bool {
	kind, _ := payload[pico.PayloadKeyKind].(string)
	if strings.EqualFold(strings.TrimSpace(kind), pico.MessageKindThought) {
		return true
	}
	thought, _ := payload[pico.PayloadKeyThought].(bool)
	return thought
}

func payloadString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	switch value := payload[key].(type) {
	case string:
		return value
	case fmt.Stringer:
		return value.String()
	default:
		return ""
	}
}

func payloadJSON(payload map[string]any) string {
	if len(payload) == 0 {
		return "{}"
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func remoteMediaStrings(payload map[string]any) []string {
	var media []string
	appendString := func(value any) {
		if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
			media = append(media, s)
		}
	}

	switch values := payload["media"].(type) {
	case []any:
		for _, value := range values {
			appendString(value)
		}
	case []string:
		for _, value := range values {
			appendString(value)
		}
	case string:
		appendString(values)
	}

	if attachments, ok := payload["attachments"].([]any); ok {
		for _, raw := range attachments {
			attachment, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			appendString(attachment["url"])
		}
	}

	return media
}
