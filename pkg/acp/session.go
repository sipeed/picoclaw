package acp

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

// Session represents a running ACP (Agent Client Protocol) harness session.
type Session struct {
	Key     string // e.g. agent:<agentId>:acp:<uuid>
	AgentID string
	Mode    string // "run" or "session"
	Command string
	Args    []string
	Cwd     string
	Label   string

	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	stderr    io.ReadCloser
	cancel    context.CancelFunc
	ctx       context.Context

	mu        sync.RWMutex
	outputBuf []string
	isActive  bool
	err       error
}

func NewSession(key, agentID, mode, command, cwd, label string, args []string) *Session {
	return &Session{
		Key:       key,
		AgentID:   agentID,
		Mode:      mode,
		Command:   command,
		Args:      args,
		Cwd:       cwd,
		Label:     label,
		outputBuf: make([]string, 0),
	}
}

func (s *Session) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.cmd = exec.CommandContext(s.ctx, s.Command, s.Args...)
	if s.Cwd != "" {
		s.cmd.Dir = s.Cwd
	}

	var err error
	s.stdin, err = s.cmd.StdinPipe()
	if err != nil {
		return err
	}
	s.stdout, err = s.cmd.StdoutPipe()
	if err != nil {
		return err
	}
	s.stderr, err = s.cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := s.cmd.Start(); err != nil {
		return err
	}

	s.isActive = true

	// Read stdout
	go s.readStream(s.stdout, "OUT")
	// Read stderr
	go s.readStream(s.stderr, "ERR")

	go func() {
		err := s.cmd.Wait()
		s.mu.Lock()
		defer s.mu.Unlock()
		s.isActive = false
		s.err = err
	}()

	return nil
}

func (s *Session) readStream(r io.Reader, prefix string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		s.mu.Lock()
		s.outputBuf = append(s.outputBuf, fmt.Sprintf("[%s] %s", prefix, line))
		// Keep last 1000 lines
		if len(s.outputBuf) > 1000 {
			s.outputBuf = s.outputBuf[1:]
		}
		s.mu.Unlock()
	}
}

func (s *Session) Write(input string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.isActive {
		return fmt.Errorf("session is not active")
	}
	_, err := fmt.Fprintln(s.stdin, input)
	return err
}

func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
	if s.stdin != nil {
		s.stdin.Close()
	}
	return nil
}

func (s *Session) GetOutput() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, len(s.outputBuf))
	copy(out, s.outputBuf)
	return out
}

func (s *Session) Status() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.isActive {
		return "running"
	}
	if s.err != nil {
		return fmt.Sprintf("exited with error: %v", s.err)
	}
	return "finished"
}
