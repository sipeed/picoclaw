package chat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/logger"
)

func chatCmd(message, sessionKey, gatewayURL string, debug bool) error {
	if debug {
		logger.SetLevel(logger.DEBUG)
	}

	// Resolve gateway URL from config if not provided.
	if gatewayURL == "" {
		cfg, err := internal.LoadConfig()
		if err != nil {
			return fmt.Errorf("error loading config: %w", err)
		}
		host := cfg.Gateway.Host
		if host == "0.0.0.0" {
			host = "127.0.0.1"
		}
		gatewayURL = fmt.Sprintf("http://%s:%d", host, cfg.Gateway.Port)
	}

	// Strip trailing slash for consistency.
	gatewayURL = strings.TrimRight(gatewayURL, "/")

	// Check that the gateway is reachable.
	if err := checkGateway(gatewayURL); err != nil {
		return fmt.Errorf("gateway not available at %s: %w\n(start it with: picoclaw gateway)", gatewayURL, err)
	}

	fmt.Printf("%s Connected to gateway at %s\n", internal.Logo, gatewayURL)

	// Single-message mode.
	if message != "" {
		resp, err := sendMessage(gatewayURL, message, sessionKey)
		if err != nil {
			return err
		}
		fmt.Printf("\n%s %s\n", internal.Logo, resp)
		return nil
	}

	// Interactive REPL mode.
	fmt.Printf("%s Interactive mode (Ctrl+C or 'exit' to quit)\n\n", internal.Logo)
	interactiveMode(gatewayURL, sessionKey)
	return nil
}

// checkGateway pings the /health endpoint to confirm the gateway is up.
func checkGateway(gatewayURL string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, gatewayURL+"/health", nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// sendMessage sends a single message to the gateway /chat endpoint and returns
// the text response. The request timeout is 5 minutes to allow long-running
// tool chains.
func sendMessage(gatewayURL, message, sessionKey string) (string, error) {
	body, _ := json.Marshal(agent.ChatAPIRequest{
		Message: message,
		Session: sessionKey,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, gatewayURL+"/chat", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	var chatResp agent.ChatAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}

	if chatResp.Error != "" {
		return "", fmt.Errorf("gateway error: %s", chatResp.Error)
	}

	return chatResp.Response, nil
}

// interactiveMode starts a readline-powered REPL that sends each line to the
// gateway and prints the reply. Falls back to simple stdin reading if readline
// is unavailable.
func interactiveMode(gatewayURL, sessionKey string) {
	prompt := fmt.Sprintf("%s You: ", internal.Logo)

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          prompt,
		HistoryFile:     filepath.Join(os.TempDir(), ".picoclaw_chat_history"),
		HistoryLimit:    200,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		fmt.Printf("Error initializing readline: %v\n", err)
		fmt.Println("Falling back to simple input mode...")
		simpleInteractiveMode(gatewayURL, sessionKey)
		return
	}
	defer rl.Close()

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

		response, err := sendMessage(gatewayURL, input, sessionKey)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("\n%s %s\n\n", internal.Logo, response)
	}
}

// simpleInteractiveMode is a fallback REPL using plain bufio.
func simpleInteractiveMode(gatewayURL, sessionKey string) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("%s You: ", internal.Logo)
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

		response, err := sendMessage(gatewayURL, input, sessionKey)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("\n%s %s\n\n", internal.Logo, response)
	}
}



