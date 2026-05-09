package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	DefaultTimeout = 60 * time.Second
)

type PermissionPrompter interface {
	Prompt(ctx context.Context, tool, path, command string) (string, error)
}

type TerminalPrompter struct {
	in      *bufio.Reader
	out     *bufio.Writer
	timeout time.Duration
}

func NewTerminalPrompter(timeout time.Duration) *TerminalPrompter {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return &TerminalPrompter{
		in:      bufio.NewReader(os.Stdin),
		out:     bufio.NewWriter(os.Stdout),
		timeout: timeout,
	}
}

func (p *TerminalPrompter) Prompt(ctx context.Context, tool, path, command string) (string, error) {
	fmt.Fprintf(p.out, "\n[permission] Allow %s to access %s? (once/always/no): ", tool, path)
	if command != "" {
		fmt.Fprintf(p.out, "\n  Command: %s\n", command)
	}
	p.out.Flush()

	type result struct {
		value string
		err   error
	}
	ch := make(chan result, 1)

	go func() {
		line, err := p.in.ReadString('\n')
		if err != nil {
			ch <- result{"", err}
			return
		}
		choice := strings.TrimSpace(strings.ToLower(line))
		ch <- result{choice, nil}
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(p.timeout):
		return "", nil
	case r := <-ch:
		if r.err != nil {
			return "", r.err
		}
		if r.value == "once" || r.value == "always" || r.value == "no" {
			return r.value, nil
		}
		return "", nil
	}
}

type ChannelPrompter struct {
	channel string
	chatID  string
	timeout time.Duration
}

func NewChannelPrompter(channel, chatID string, timeout time.Duration) *ChannelPrompter {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return &ChannelPrompter{
		channel: channel,
		chatID:  chatID,
		timeout: timeout,
	}
}

func (p *ChannelPrompter) Prompt(ctx context.Context, tool, path, command string) (string, error) {
	content := fmt.Sprintf("[permission] Allow %s to access %s?\n", tool, path)
	if command != "" {
		content += fmt.Sprintf("Command: %s\n", command)
	}
	content += "Reply: **once** / **always** / **no**"

	_ = content
	return "", nil
}
