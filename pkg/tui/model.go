// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT

package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/sipeed/picoclaw/pkg/agent"
)

const (
	tickInterval       = 200 * time.Millisecond
	thinkingFrameCount = 4
	headerHeight       = 1
	statusBarHeight    = 1
	textareaHeight     = 3
	// chromeHeight accounts for header, status bar, textarea, and separators
	chromeHeight     = headerHeight + statusBarHeight + textareaHeight + 2
	maxSessionKeyLen = 15
)

// thinkingFrames are the animation frames for the thinking indicator
var thinkingFrames = [thinkingFrameCount]string{"⠋", "⠙", "⠹", "⠸"}

// tickMsg drives the thinking animation and event polling
type tickMsg time.Time

// chatMessage represents a single message in the chat history
type chatMessage struct {
	role     string // "user", "assistant", "tool", "system"
	content  string
	toolName string
	toolID   string
	toolDone bool
	toolErr  bool
}

// Model is the main bubbletea model for the PicoClaw TUI
type Model struct {
	viewport    viewport.Model
	textarea    textarea.Model
	messages    []chatMessage
	agentLoop   *agent.AgentLoop
	sessionKey  string
	modelName   string
	thinking    bool
	thinkFrame  int
	width       int
	height      int
	ready       bool
	eventBridge *eventBridge
	renderer    *glamour.TermRenderer
	quitting    bool
}

// NewModel creates a new TUI model wired to the given agent loop
func NewModel(agentLoop *agent.AgentLoop, sessionKey, modelName string) Model {
	ta := textarea.New()
	ta.Placeholder = "Type a message... (Enter to send, Alt+Enter for newline)"
	ta.Prompt = "│ "
	ta.CharLimit = 0 // No limit
	ta.SetHeight(textareaHeight)
	ta.ShowLineNumbers = false

	// Enter sends, Alt+Enter inserts newline
	ta.KeyMap.InsertNewline = key.NewBinding(key.WithKeys("alt+enter"))

	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()

	eb := newEventBridge()
	agentLoop.SetEventListener(eb)

	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(0), // will be updated on resize
	)

	return Model{
		textarea:    ta,
		agentLoop:   agentLoop,
		sessionKey:  sessionKey,
		modelName:   modelName,
		eventBridge: eb,
		renderer:    renderer,
	}
}

// Init returns the initial command for the TUI
func (m Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, tickCmd())
}

// Update handles incoming messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)

	case tickMsg:
		return m.handleTick()

	case ThinkingStartedMsg:
		m.thinking = true
		m.updateViewport()
		return m, nil

	case ToolCallStartedMsg:
		m.messages = append(m.messages, chatMessage{
			role:     "tool",
			content:  fmt.Sprintf("Running %s...", msg.Name),
			toolName: msg.Name,
			toolID:   msg.ID,
		})
		m.updateViewport()
		return m, nil

	case ToolCallCompletedMsg:
		for i := len(m.messages) - 1; i >= 0; i-- {
			if m.messages[i].role == "tool" && m.messages[i].toolID == msg.ID {
				m.messages[i].toolDone = true
				m.messages[i].toolErr = msg.IsError
				if msg.IsError {
					m.messages[i].content = fmt.Sprintf("%s failed", msg.Name)
				} else {
					m.messages[i].content = fmt.Sprintf("%s done", msg.Name)
				}
				break
			}
		}
		m.updateViewport()
		return m, nil

	case ResponseMsg:
		m.messages = append(m.messages, chatMessage{
			role:    "assistant",
			content: msg.Content,
		})
		m.thinking = false
		m.updateViewport()
		return m, nil

	case ErrorMsg:
		m.messages = append(m.messages, chatMessage{
			role:    "system",
			content: fmt.Sprintf("Error: %v", msg.Err),
		})
		m.thinking = false
		m.updateViewport()
		return m, nil

	case SlashCommandResultMsg:
		m.messages = append(m.messages, chatMessage{
			role:    "system",
			content: msg.Result,
		})
		m.thinking = false
		m.updateViewport()
		return m, nil
	}

	// Pass remaining messages to textarea
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the TUI
func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}
	if !m.ready {
		return "Initializing...\n"
	}

	header := m.renderHeader()
	sep := separatorStyle.Render(strings.Repeat("─", m.width))

	return fmt.Sprintf(
		"%s\n%s\n%s\n%s\n%s",
		header,
		m.viewport.View(),
		sep,
		m.textarea.View(),
		m.renderStatusBar(),
	)
}

// handleKeyMsg processes keyboard input
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		m.quitting = true
		return m, tea.Quit

	case tea.KeyPgUp:
		m.viewport.PageUp()
		return m, nil

	case tea.KeyPgDown:
		m.viewport.PageDown()
		return m, nil

	case tea.KeyEnter:
		input := strings.TrimSpace(m.textarea.Value())
		if input == "" {
			return m, nil
		}
		m.textarea.Reset()

		// Add user message to chat
		m.messages = append(m.messages, chatMessage{
			role:    "user",
			content: input,
		})
		m.thinking = true
		m.updateViewport()

		return m, m.processMessage(input)
	}

	// Pass to textarea for regular typing
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

// handleWindowSize initializes or resizes the viewport and textarea
func (m Model) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	vpHeight := msg.Height - chromeHeight
	if vpHeight < 1 {
		vpHeight = 1
	}

	if !m.ready {
		m.viewport = viewport.New(msg.Width, vpHeight)
		m.ready = true
	} else {
		m.viewport.Width = msg.Width
		m.viewport.Height = vpHeight
	}

	m.textarea.SetWidth(msg.Width)

	// Recreate renderer with updated word wrap
	if r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(msg.Width-4),
	); err == nil {
		m.renderer = r
	}

	m.updateViewport()

	return m, nil
}

// handleTick advances the thinking animation and polls agent events
func (m Model) handleTick() (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{tickCmd()}

	if m.thinking {
		m.thinkFrame = (m.thinkFrame + 1) % thinkingFrameCount
		m.updateViewport()
	}

	// Poll events from the agent bridge
	eventCmds := m.pollEvents()
	cmds = append(cmds, eventCmds...)

	return m, tea.Batch(cmds...)
}

// processMessage sends user input to the agent loop in a goroutine
func (m Model) processMessage(input string) tea.Cmd {
	agentLoop := m.agentLoop
	sessionKey := m.sessionKey

	return func() tea.Msg {
		ctx := context.Background()
		response, err := agentLoop.ProcessDirect(ctx, input, sessionKey)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		if strings.HasPrefix(input, "/") {
			return SlashCommandResultMsg{Result: response}
		}
		return ResponseMsg{Content: response}
	}
}

// pollEvents drains the event bridge channel, returning tea commands
func (m Model) pollEvents() []tea.Cmd {
	var cmds []tea.Cmd

	for {
		select {
		case event := <-m.eventBridge.events:
			cmd := m.convertEvent(event)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		default:
			return cmds
		}
	}
}

// convertEvent turns an agent event into a tea.Cmd that returns the matching message
func (m Model) convertEvent(event agent.AgentEvent) tea.Cmd {
	switch event.Type {
	case agent.EventThinkingStarted:
		return func() tea.Msg { return ThinkingStartedMsg{} }

	case agent.EventToolCallStarted:
		if data, ok := event.Data.(agent.ToolCallStartedData); ok {
			return func() tea.Msg {
				return ToolCallStartedMsg{
					ID:   data.ID,
					Name: data.Name,
					Args: data.Args,
				}
			}
		}

	case agent.EventToolCallCompleted:
		if data, ok := event.Data.(agent.ToolCallCompletedData); ok {
			return func() tea.Msg {
				return ToolCallCompletedMsg{
					ID:      data.ID,
					Name:    data.Name,
					Result:  data.Result,
					IsError: data.IsError,
				}
			}
		}

	case agent.EventResponseComplete:
		// The response is also returned by ProcessDirect, so we don't
		// duplicate it here. The tick-based polling of EventResponseComplete
		// is only used for streaming scenarios in the future.
		return nil

	case agent.EventError:
		if data, ok := event.Data.(agent.ErrorData); ok {
			return func() tea.Msg { return ErrorMsg{Err: data.Err} }
		}
	}
	return nil
}

// updateViewport renders all messages and sets the viewport content
func (m *Model) updateViewport() {
	var sb strings.Builder

	for _, msg := range m.messages {
		switch msg.role {
		case "user":
			label := userLabelStyle.Render("You:")
			sb.WriteString(label + " " + msg.content + "\n\n")

		case "assistant":
			label := assistantLabelStyle.Render("Assistant:")
			rendered := msg.content
			if m.renderer != nil {
				if r, err := m.renderer.Render(msg.content); err == nil {
					rendered = strings.TrimSpace(r)
				}
			}
			sb.WriteString(label + "\n" + rendered + "\n\n")

		case "tool":
			var styled string
			switch {
			case msg.toolErr:
				styled = toolErrorStyle.Render("✗ " + msg.content)
			case msg.toolDone:
				styled = toolDoneStyle.Render("✓ " + msg.content)
			default:
				styled = toolRunningStyle.Render("⟳ " + msg.content)
			}
			sb.WriteString("  " + styled + "\n")

		case "system":
			sb.WriteString(msg.content + "\n\n")
		}
	}

	// Append thinking indicator if active
	if m.thinking {
		frame := thinkingFrames[m.thinkFrame]
		sb.WriteString(thinkingStyle.Render(frame+" Thinking...") + "\n")
	}

	m.viewport.SetContent(sb.String())
	m.viewport.GotoBottom()
}

// renderHeader returns the header line
func (m Model) renderHeader() string {
	title := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true).
		Render("PicoClaw")

	session := truncateSessionKey(m.sessionKey)

	right := lipgloss.NewStyle().
		Foreground(secondaryColor).
		Render(fmt.Sprintf("[%s] %s", session, m.modelName))

	gap := m.width - lipgloss.Width(title) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}

	return title + strings.Repeat(" ", gap) + right
}

// renderStatusBar returns the status bar at the bottom
func (m Model) renderStatusBar() string {
	left := m.modelName

	// Count user and assistant messages
	msgCount := 0
	for _, msg := range m.messages {
		if msg.role == "user" || msg.role == "assistant" {
			msgCount++
		}
	}
	right := fmt.Sprintf("messages: %d | session: %s", msgCount, truncateSessionKey(m.sessionKey))

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 2 // padding
	if gap < 1 {
		gap = 1
	}

	bar := left + strings.Repeat(" ", gap) + right
	return statusBarStyle.Render(bar)
}

// truncateSessionKey shortens the session key for display
func truncateSessionKey(key string) string {
	if len(key) <= maxSessionKeyLen {
		return key
	}
	return key[:maxSessionKeyLen] + "…"
}

// tickCmd returns a command that sends a tickMsg after the tick interval
func tickCmd() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
