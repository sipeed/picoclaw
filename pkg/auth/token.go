package auth

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

func LoginPasteToken(provider string, r io.Reader) (*AuthCredential, error) {
	fmt.Printf("Paste your API key or session token from %s:\n", providerDisplayName(provider))
	fmt.Print("> ")

	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("reading token: %w", err)
		}
		return nil, fmt.Errorf("no input received")
	}

	token := strings.TrimSpace(scanner.Text())
	if token == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	return &AuthCredential{
		AccessToken: token,
		Provider:    provider,
		AuthMethod:  "token",
	}, nil
}

func LoginSetupToken(r io.Reader) (*AuthCredential, error) {
	fmt.Println("Paste your setup token from `claude setup-token`:")
	fmt.Print("> ")

	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("reading token: %w", err)
		}
		return nil, fmt.Errorf("no input received")
	}

	token := strings.TrimSpace(scanner.Text())
	if token == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	if !strings.HasPrefix(token, "sk-ant-oat01-") {
		return nil, fmt.Errorf("invalid setup token: expected prefix sk-ant-oat01-")
	}

	if len(token) < 80 {
		return nil, fmt.Errorf("invalid setup token: too short (expected at least 80 characters)")
	}

	return &AuthCredential{
		AccessToken: token,
		Provider:    "anthropic",
		AuthMethod:  "oauth",
	}, nil
}

func providerDisplayName(provider string) string {
	switch provider {
	case "anthropic":
		return "console.anthropic.com"
	case "openai":
		return "platform.openai.com"
	default:
		return provider
	}
}
