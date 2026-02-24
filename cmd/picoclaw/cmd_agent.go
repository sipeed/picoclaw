// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT

package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/chzyer/readline"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// Interactive mode identifiers
const (
	modePico   = "pico"   // Chat mode (default) - input goes to AI agent
	modeCmd    = "cmd"    // Command mode - input executed as shell commands
	modeHiPico = "hipico" // AI-assisted mode within cmd - multi-turn AI conversation
)

// cmdWorkingDir tracks the current working directory for command mode.
var cmdWorkingDir string

func init() {
	cmdWorkingDir, _ = os.Getwd()
}

func agentCmd() {
	message := ""
	sessionKey := "cli:default"
	modelOverride := ""

	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--debug", "-d":
			logger.SetLevel(logger.DEBUG)
			fmt.Println("Debug mode enabled")
		case "-m", "--message":
			if i+1 < len(args) {
				message = args[i+1]
				i++
			}
		case "-s", "--session":
			if i+1 < len(args) {
				sessionKey = args[i+1]
				i++
			}
		case "--model", "-model":
			if i+1 < len(args) {
				modelOverride = args[i+1]
				i++
			}
		}
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	if modelOverride != "" {
		cfg.Agents.Defaults.Model = modelOverride
	}

	provider, modelID, err := providers.CreateProvider(cfg)
	if err != nil {
		fmt.Printf("Error creating provider: %v\n", err)
		os.Exit(1)
	}
	// Use the resolved model ID from provider creation
	if modelID != "" {
		cfg.Agents.Defaults.Model = modelID
	}

	msgBus := bus.NewMessageBus()
	agentLoop := agent.NewAgentLoop(cfg, msgBus, provider)

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
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\n%s %s\n", logo, response)
	} else {
		fmt.Printf("%s Interactive mode (Ctrl+C to exit)\n", logo)
		fmt.Println("  /help    - show detailed help")
		fmt.Println("  /usage   - show model info and token usage")
		fmt.Println("  /cmd     - switch to command mode")
		fmt.Println("  /pico    - switch to chat mode")
		fmt.Println("  /hipico  - AI assistance in command mode")
		fmt.Println("  /byepico - end AI assistance")
		fmt.Println()
		interactiveMode(agentLoop, sessionKey)
	}
}

func interactiveMode(agentLoop *agent.AgentLoop, sessionKey string) {
	chatPrompt := fmt.Sprintf("%s You: ", logo)
	cmdPrompt := "$ "
	hipicoPrompt := fmt.Sprintf("%s> ", logo)

	mode := modePico

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          chatPrompt,
		HistoryFile:     filepath.Join(os.TempDir(), ".picoclaw_history"),
		HistoryLimit:    100,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		fmt.Printf("Error initializing readline: %v\n", err)
		fmt.Println("Falling back to simple input mode...")
		simpleInteractiveMode(agentLoop, sessionKey)
		return
	}
	defer rl.Close()

	hipicoSessionKey := "cli:hipico"

	for {
		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt || err == io.EOF {
				fmt.Println("\nGoodbye!")
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
			fmt.Println("Goodbye!")
			return
		}

		// /help and /usage work in all modes
		if input == "/help" {
			printHelp()
			continue
		}
		if input == "/usage" {
			printUsage(agentLoop)
			continue
		}

		switch mode {
		case modePico:
			if input == "/cmd" {
				mode = modeCmd
				rl.SetPrompt(cmdPrompt)
				fmt.Println("Switched to command mode. Type /pico to return to chat.")
				continue
			}

			ctx := context.Background()
			response, err := agentLoop.ProcessDirect(ctx, input, sessionKey)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}
			fmt.Printf("\n%s %s\n\n", logo, response)

		case modeCmd:
			if input == "/pico" {
				mode = modePico
				rl.SetPrompt(chatPrompt)
				fmt.Println("Switched to chat mode. Type /cmd to return to command mode.")
				continue
			}

			if strings.HasPrefix(input, "/hipico") {
				initialMsg := strings.TrimSpace(strings.TrimPrefix(input, "/hipico"))
				if initialMsg == "" {
					fmt.Println("Usage: /hipico <message>")
					fmt.Println("Example: /hipico check the log files for error messages")
					continue
				}

				mode = modeHiPico
				rl.SetPrompt(hipicoPrompt)

				contextPrefix := fmt.Sprintf("[Command mode context: working directory is %s]\n\n", cmdWorkingDir)

				fmt.Printf("\n%s AI assistance started. Type /byepico to end.\n\n", logo)

				ctx := context.Background()
				response, err := agentLoop.ProcessDirect(ctx, contextPrefix+initialMsg, hipicoSessionKey)
				if err != nil {
					fmt.Printf("Error: %v\n", err)
					mode = modeCmd
					rl.SetPrompt(cmdPrompt)
					continue
				}
				fmt.Printf("%s %s\n\n", logo, response)
				continue
			}

			executeShellCommand(input)

		case modeHiPico:
			if input == "/byepico" {
				mode = modeCmd
				rl.SetPrompt(cmdPrompt)
				fmt.Println("AI assistance ended. Back to command mode.")
				continue
			}

			if input == "/pico" {
				mode = modePico
				rl.SetPrompt(chatPrompt)
				fmt.Println("AI assistance ended. Switched to chat mode.")
				continue
			}

			ctx := context.Background()
			response, err := agentLoop.ProcessDirect(ctx, input, hipicoSessionKey)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}
			fmt.Printf("\n%s %s\n\n", logo, response)
		}
	}
}

func simpleInteractiveMode(agentLoop *agent.AgentLoop, sessionKey string) {
	reader := bufio.NewReader(os.Stdin)
	mode := modePico
	hipicoSessionKey := "cli:hipico"

	for {
		switch mode {
		case modePico:
			fmt.Printf("%s You: ", logo)
		case modeCmd:
			fmt.Print("$ ")
		case modeHiPico:
			fmt.Printf("%s> ", logo)
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println("\nGoodbye!")
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
			fmt.Println("Goodbye!")
			return
		}

		// /help and /usage work in all modes
		if input == "/help" {
			printHelp()
			continue
		}
		if input == "/usage" {
			printUsage(agentLoop)
			continue
		}

		switch mode {
		case modePico:
			if input == "/cmd" {
				mode = modeCmd
				fmt.Println("Switched to command mode. Type /pico to return to chat.")
				continue
			}

			ctx := context.Background()
			response, err := agentLoop.ProcessDirect(ctx, input, sessionKey)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}
			fmt.Printf("\n%s %s\n\n", logo, response)

		case modeCmd:
			if input == "/pico" {
				mode = modePico
				fmt.Println("Switched to chat mode. Type /cmd to return to command mode.")
				continue
			}

			if strings.HasPrefix(input, "/hipico") {
				initialMsg := strings.TrimSpace(strings.TrimPrefix(input, "/hipico"))
				if initialMsg == "" {
					fmt.Println("Usage: /hipico <message>")
					fmt.Println("Example: /hipico check the log files for error messages")
					continue
				}

				mode = modeHiPico
				contextPrefix := fmt.Sprintf("[Command mode context: working directory is %s]\n\n", cmdWorkingDir)
				fmt.Printf("\n%s AI assistance started. Type /byepico to end.\n\n", logo)

				ctx := context.Background()
				response, err := agentLoop.ProcessDirect(ctx, contextPrefix+initialMsg, hipicoSessionKey)
				if err != nil {
					fmt.Printf("Error: %v\n", err)
					mode = modeCmd
					continue
				}
				fmt.Printf("%s %s\n\n", logo, response)
				continue
			}

			executeShellCommand(input)

		case modeHiPico:
			if input == "/byepico" {
				mode = modeCmd
				fmt.Println("AI assistance ended. Back to command mode.")
				continue
			}

			if input == "/pico" {
				mode = modePico
				fmt.Println("AI assistance ended. Switched to chat mode.")
				continue
			}

			ctx := context.Background()
			response, err := agentLoop.ProcessDirect(ctx, input, hipicoSessionKey)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}
			fmt.Printf("\n%s %s\n\n", logo, response)
		}
	}
}

// printHelp outputs detailed usage information for all interactive modes.
func printHelp() {
	fmt.Printf(`%s PicoClaw Interactive Mode Help
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

PicoClaw has three interactive modes:

  1. Chat Mode (default)
     Talk to the AI agent directly. Your input is sent as a message
     and the AI responds.

  2. Command Mode
     Execute shell commands directly, like a terminal. Supports cd,
     pipes, redirects, and all standard shell features.

  3. AI-Assisted Command Mode
     A multi-turn AI conversation within command mode. The AI is aware
     of your current working directory and can help with system tasks.

Commands (available in all modes):
  /help      Show this help message
  /usage     Show model info and token usage
  exit       Exit PicoClaw
  quit       Exit PicoClaw
  Ctrl+C     Exit PicoClaw

Mode switching:
  /cmd       Switch to command mode        (from chat mode)
  /pico      Switch to chat mode           (from command / AI-assisted mode)
  /hipico <msg>  Start AI-assisted mode    (from command mode)
  /byepico   End AI assistance             (from AI-assisted mode)

Examples:
  Chat mode:
    %s You: What is the weather today?

  Command mode:
    $ ls -al /var/log
    $ cd /tmp
    $ cat error.log | grep "FATAL"

  AI-assisted mode (enter from command mode):
    $ /hipico check the log files for errors
    %s> show me more details on line 42
    %s> /byepico

`, logo, logo, logo, logo)
}

// printUsage displays current model information and accumulated token usage.
func printUsage(agentLoop *agent.AgentLoop) {
	info := agentLoop.GetUsageInfo()
	if info == nil {
		fmt.Println("No usage information available.")
		return
	}
	fmt.Printf(`%s Usage
━━━━━━━━━━━━━━━━━━━━━━
Model:              %s
Max tokens:         %d
Temperature:        %.1f

Token usage (this session):
  Prompt tokens:    %d
  Completion tokens:%d
  Total tokens:     %d
  Requests:         %d
`, logo,
		info["model"],
		info["max_tokens"],
		info["temperature"],
		info["prompt_tokens"],
		info["completion_tokens"],
		info["total_tokens"],
		info["requests"],
	)
}

// executeShellCommand runs a shell command in the current working directory
// and prints the output. It also handles the cd command to change directories.
func executeShellCommand(input string) {
	// Handle cd command specially to update working directory
	if strings.HasPrefix(input, "cd ") || input == "cd" {
		handleCd(input)
		return
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", input)
	} else {
		cmd = exec.Command("sh", "-c", input)
	}
	cmd.Dir = cmdWorkingDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	if stdout.Len() > 0 {
		fmt.Print(stdout.String())
		if !strings.HasSuffix(stdout.String(), "\n") {
			fmt.Println()
		}
	}
	if stderr.Len() > 0 {
		fmt.Print(stderr.String())
		if !strings.HasSuffix(stderr.String(), "\n") {
			fmt.Println()
		}
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			fmt.Printf("Exit code: %d\n", exitErr.ExitCode())
		} else {
			fmt.Printf("Error: %v\n", err)
		}
	}
}

// handleCd handles the cd command to change the working directory for command mode.
func handleCd(input string) {
	parts := strings.Fields(input)
	var target string

	if len(parts) < 2 {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		target = home
	} else {
		target = parts[1]
	}

	// Handle ~ expansion
	if strings.HasPrefix(target, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		if target == "~" {
			target = home
		} else if len(target) > 1 && target[1] == '/' {
			target = filepath.Join(home, target[2:])
		}
	}

	// Handle relative paths
	if !filepath.IsAbs(target) {
		target = filepath.Join(cmdWorkingDir, target)
	}

	target = filepath.Clean(target)

	info, err := os.Stat(target)
	if err != nil {
		fmt.Printf("cd: %v\n", err)
		return
	}
	if !info.IsDir() {
		fmt.Printf("cd: %s: Not a directory\n", target)
		return
	}

	cmdWorkingDir = target
}
