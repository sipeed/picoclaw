// PicoClaw - Ultra-lightweight personal AI agent
// Interactive CLI with semantic-friendly logging
//
// This file provides an enhanced interactive mode that shows real-time
// progress logs during agent processing, including:
// - Thinking process indicators
// - Tool execution status
// - API call progress
// - Final response delivery

package agent

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ergochat/readline"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// InteractiveLogger provides real-time feedback during agent processing
type InteractiveLogger struct {
	mu         sync.Mutex
	verbose    bool
	showTools  bool
	showThink  bool
	showAPI    bool
	eventSub   agent.EventSubscription
	agentLoop  *agent.AgentLoop
	lastTurnID string
}

// NewInteractiveLogger creates a new interactive logger
func NewInteractiveLogger(verbose, showTools, showThink bool) *InteractiveLogger {
	return &InteractiveLogger{
		verbose:   verbose,
		showTools: showTools,
		showThink: showThink,
		showAPI:   verbose || showThink,
	}
}

// StartEventSubscription subscribes to agent events and logs them
func (il *InteractiveLogger) StartEventSubscription(agentLoop *agent.AgentLoop) {
	il.agentLoop = agentLoop
	il.eventSub = agentLoop.SubscribeEvents(100)

	go func() {
		for evt := range il.eventSub.C {
			il.handleEvent(evt)
		}
	}()
}

// StopEventSubscription stops the event subscription
func (il *InteractiveLogger) StopEventSubscription() {
	if il.agentLoop != nil && il.eventSub.C != nil {
		il.agentLoop.UnsubscribeEvents(il.eventSub.ID)
	}
}

// handleEvent processes agent events and displays them
func (il *InteractiveLogger) handleEvent(evt agent.Event) {
	il.mu.Lock()
	defer il.mu.Unlock()

	// Track turn changes
	if evt.Meta.TurnID != il.lastTurnID {
		il.lastTurnID = evt.Meta.TurnID
	}

	switch evt.Kind {
	case agent.EventKindTurnStart:
		if il.showThink {
			payload, ok := evt.Payload.(agent.TurnStartPayload)
			if ok {
				fmt.Printf("\r\033[K%s \033[1;36m⚙️  Thinking\033[0m (channel: %s)\n", internal.Logo, payload.Channel)
			}
		}

	case agent.EventKindLLMRequest:
		if il.showAPI {
			payload, ok := evt.Payload.(agent.LLMRequestPayload)
			if ok {
				// Show basic info
				fmt.Printf("\r\033[K%s \033[1;35m📡 Calling API\033[0m: %s (messages: %d, tools: %d)\n",
					internal.Logo, payload.Model, payload.MessagesCount, payload.ToolsCount)

				// Show detailed info in verbose mode
				if il.verbose {
					if payload.UserMessagePreview != "" {
						fmt.Printf("    \033[1;36m📝 User: %s\033[0m\n",
							payload.UserMessagePreview)
					}
					if len(payload.ToolNames) > 0 {
						fmt.Printf("    \033[1;33m🔧 Tools: %s\033[0m\n",
							strings.Join(payload.ToolNames, ", "))
					}
					if payload.MaxTokens > 0 {
						fmt.Printf("    \033[1;37m⚙️  MaxTokens: %d, Temp: %.2f\033[0m\n",
							payload.MaxTokens, payload.Temperature)
					}
				}
			}
		}

	case agent.EventKindLLMResponse:
		if il.showAPI {
			payload, ok := evt.Payload.(agent.LLMResponsePayload)
			if ok {
				toolInfo := ""
				if payload.ToolCalls > 0 {
					toolInfo = fmt.Sprintf(", tool calls: %d", payload.ToolCalls)
				}
				fmt.Printf("\r\033[K%s \033[1;32m✅ API Response\033[0m (content: %d chars%s)\n",
					internal.Logo, payload.ContentLen, toolInfo)

				// Show detailed info in verbose mode
				if il.verbose {
					if payload.ContentPreview != "" {
						fmt.Printf("    \033[1;37m📄 Content: %s\033[0m\n", payload.ContentPreview)
					}
					if len(payload.ToolCallDetails) > 0 {
						for _, tc := range payload.ToolCallDetails {
							fmt.Printf("    \033[1;33m🔧 Tool: %s(%s)\033[0m\n", tc.Name, tc.Arguments)
						}
					}
					if payload.FinishReason != "" {
						fmt.Printf("    \033[1;36m🏁 Finish: %s\033[0m\n", payload.FinishReason)
					}
				}
			}
		}

	case agent.EventKindLLMRetry:
		payload, ok := evt.Payload.(agent.LLMRetryPayload)
		if ok {
			fmt.Printf("\r\033[K%s \033[1;33m🔄 Retrying API\033[0m (attempt %d/%d): %s\n",
				internal.Logo, payload.Attempt, payload.MaxRetries, payload.Reason)
		}

	case agent.EventKindToolExecStart:
		if il.showTools {
			payload, ok := evt.Payload.(agent.ToolExecStartPayload)
			if ok {
				argsStr := ""
				for k, v := range payload.Arguments {
					if argsStr != "" {
						argsStr += ", "
					}
					argsStr += fmt.Sprintf("%s=%v", k, v)
				}
				if len(argsStr) > 80 {
					argsStr = argsStr[:77] + "..."
				}
				fmt.Printf("\r\033[K%s \033[1;33m🔧 Tool Calling: %s(%s)\033[0m\n",
					internal.Logo, payload.Tool, argsStr)
			}
		}

	case agent.EventKindToolExecEnd:
		if il.showTools {
			payload, ok := evt.Payload.(agent.ToolExecEndPayload)
			if ok {
				duration := payload.Duration.Round(time.Millisecond)
				if payload.IsError {
					fmt.Printf("\r\033[K%s \033[1;31m❌ Tool Failed: %s (%v)\033[0m\n",
						internal.Logo, payload.Tool, duration)
				} else {
					fmt.Printf("\r\033[K%s \033[1;32m✅ Tool Completed: %s (%v)\033[0m\n",
						internal.Logo, payload.Tool, duration)
				}
			}
		}

	case agent.EventKindContextCompress:
		if il.verbose {
			payload, ok := evt.Payload.(agent.ContextCompressPayload)
			if ok {
				fmt.Printf("\r\033[K%s \033[1;33m🗜️  Context Compressed\033[0m: dropped %d, kept %d messages\n",
					internal.Logo, payload.DroppedMessages, payload.RemainingMessages)
			}
		}

	case agent.EventKindError:
		payload, ok := evt.Payload.(agent.ErrorPayload)
		if ok {
			fmt.Printf("\r\033[K%s \033[1;31m❌ Error [%s]: %s\033[0m\n",
				internal.Logo, payload.Stage, payload.Message)
		}

	case agent.EventKindTurnEnd:
		payload, ok := evt.Payload.(agent.TurnEndPayload)
		if ok && il.verbose {
			statusIcon := "✅"
			if payload.Status == agent.TurnEndStatusError {
				statusIcon = "❌"
			} else if payload.Status == agent.TurnEndStatusAborted {
				statusIcon = "⚠️"
			}
			fmt.Printf("\r\033[K%s \033[1;37m%s Turn Ended\033[0m: %d iterations, %v, %d chars\n",
				internal.Logo,
				statusIcon,
				payload.Iterations,
				payload.Duration.Round(time.Millisecond),
				payload.FinalContentLen)
		}
	}
}

// LogThinking logs the thinking process start
func (il *InteractiveLogger) LogThinking() {
	if !il.showThink {
		return
	}
}

// LogResponse logs the final response
func (il *InteractiveLogger) LogResponse() {
	il.mu.Lock()
	defer il.mu.Unlock()
	fmt.Printf("\r\033[K%s \033[1;34m💬 Response:\033[0m\n", internal.Logo)
}

// LogError logs an error
func (il *InteractiveLogger) LogError(err error) {
	il.mu.Lock()
	defer il.mu.Unlock()
	fmt.Printf("\r\033[K%s \033[1;31m❌ Error:\033[0m %v\n", internal.Logo, err)
}

// agentCmdWithLogging enhanced agent command with interactive logging
func agentCmdWithLogging(message, sessionKey, model string, debug, verbose, showTools, showThink bool) error {
	if sessionKey == "" {
		sessionKey = "cli:default"
	}

	cfg, err := internal.LoadConfig()
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	logger.ConfigureFromEnv()

	if debug {
		logger.SetLevel(logger.DEBUG)
		fmt.Println("🔍 Debug mode enabled")
	}

	if model != "" {
		cfg.Agents.Defaults.ModelName = model
	}

	provider, modelID, err := providers.CreateProvider(cfg)
	if err != nil {
		return fmt.Errorf("error creating provider: %w", err)
	}

	// Use the resolved model ID from provider creation
	if modelID != "" {
		cfg.Agents.Defaults.ModelName = modelID
	}

	msgBus := bus.NewMessageBus()
	defer msgBus.Close()
	agentLoop := agent.NewAgentLoop(cfg, msgBus, provider)
	defer agentLoop.Close()

	// Print agent startup info (only for interactive mode)
	startupInfo := agentLoop.GetStartupInfo()
	logger.InfoCF("agent", "Agent initialized",
		map[string]any{
			"tools_count":      startupInfo["tools"].(map[string]any)["count"],
			"skills_total":     startupInfo["skills"].(map[string]any)["total"],
			"skills_available": startupInfo["skills"].(map[string]any)["available"],
		})

	if message != "" {
		ctx := context.Background()
		response, err := agentLoop.ProcessDirect(ctx, message, sessionKey)
		if err != nil {
			return fmt.Errorf("error processing message: %w", err)
		}
		fmt.Printf("\n%s %s\n", internal.Logo, response)
		return nil
	}

	fmt.Printf("%s \033[1;32mInteractive mode\033[0m (type 'exit' to quit, Ctrl+C to interrupt)\n", internal.Logo)
	if verbose {
		fmt.Printf("%s \033[90mVerbose logging enabled\033[0m\n", internal.Logo)
	}
	if showTools {
		fmt.Printf("%s \033[90mTool execution logging enabled\033[0m\n", internal.Logo)
	}
	if showThink {
		fmt.Printf("%s \033[90mThinking process logging enabled\033[0m\n", internal.Logo)
	}
	fmt.Println()

	interactiveModeWithLogging(agentLoop, sessionKey, verbose, showTools, showThink)

	return nil
}

// interactiveModeWithLogging enhanced interactive mode with real-time logging
func interactiveModeWithLogging(agentLoop *agent.AgentLoop, sessionKey string, verbose, showTools, showThink bool) {
	prompt := fmt.Sprintf("%s \033[32mYou:\033[0m ", internal.Logo)
	il := NewInteractiveLogger(verbose, showTools, showThink)

	// Start event subscription for real-time logs
	il.StartEventSubscription(agentLoop)
	defer il.StopEventSubscription()

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          prompt,
		HistoryFile:     filepath.Join(os.TempDir(), ".picoclaw_history"),
		HistoryLimit:    100,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		fmt.Printf("Error initializing readline: %v\n", err)
		fmt.Println("Falling back to simple interactive mode...")
		simpleInteractiveModeWithLogging(agentLoop, sessionKey, verbose, showTools, showThink)
		return
	}
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt || err == io.EOF {
				fmt.Println("\n\033[1;32mGoodbye!\033[0m")
				return
			}
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}

		if input == "exit" || input == "quit" {
			fmt.Println("\n\033[1;32mGoodbye!\033[0m")
			return
		}

		ctx := context.Background()
		startTime := time.Now()

		response, err := agentLoop.ProcessDirect(ctx, input, sessionKey)
		elapsed := time.Since(startTime)

		if err != nil {
			il.LogError(err)
			continue
		}

		// Show response
		il.LogResponse()
		fmt.Printf("%s %s\n", internal.Logo, response)

		// Show timing info in verbose mode
		if verbose {
			fmt.Printf("\033[1;37m⏱️  Completed in %v\033[0m\n\n", elapsed)
		} else {
			fmt.Println()
		}
	}
}

// simpleInteractiveModeWithLogging fallback interactive mode with logging
func simpleInteractiveModeWithLogging(
	agentLoop *agent.AgentLoop,
	sessionKey string,
	verbose, showTools, showThink bool,
) {
	reader := bufio.NewReader(os.Stdin)
	il := NewInteractiveLogger(verbose, showTools, showThink)

	// Start event subscription for real-time logs
	il.StartEventSubscription(agentLoop)
	defer il.StopEventSubscription()

	for {
		fmt.Print(fmt.Sprintf("%s \033[32mYou:\033[0m ", internal.Logo))
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println("\n\033[1;32mGoodbye!\033[0m")
				return
			}
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}

		if input == "exit" || input == "quit" {
			fmt.Println("\n\033[1;32mGoodbye!\033[0m")
			return
		}

		ctx := context.Background()
		startTime := time.Now()

		response, err := agentLoop.ProcessDirect(ctx, input, sessionKey)
		elapsed := time.Since(startTime)

		if err != nil {
			il.LogError(err)
			continue
		}

		// Show response
		il.LogResponse()
		fmt.Printf("%s %s\n", internal.Logo, response)

		// Show timing info in verbose mode
		if verbose {
			fmt.Printf("\033[1;37m⏱️  Completed in %v\033[0m\n\n", elapsed)
		} else {
			fmt.Println()
		}
	}
}
