// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT

package main

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// spinner displays a rotating progress indicator on stderr while the agent
// processes a request. Output goes to stderr so that piped stdout is not
// affected.
type spinner struct {
	frames  []rune
	text    string
	stop    chan struct{}
	stopped chan struct{}
	once    sync.Once
}

func newSpinner(text string) *spinner {
	return &spinner{
		frames:  []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'},
		text:    text,
		stop:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
}

// Start launches the spinner animation in a background goroutine.
func (s *spinner) Start() {
	go func() {
		defer close(s.stopped)
		i := 0
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-s.stop:
				// Clear the spinner line
				fmt.Fprintf(os.Stderr, "\r\033[K")
				return
			case <-ticker.C:
				frame := s.frames[i%len(s.frames)]
				fmt.Fprintf(os.Stderr, "\r%c %s", frame, s.text)
				i++
			}
		}
	}()
}

// Stop signals the spinner to halt and waits for it to finish.
func (s *spinner) Stop() {
	s.once.Do(func() {
		close(s.stop)
		<-s.stopped
	})
}
