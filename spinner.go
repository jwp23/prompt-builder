package main

import (
	"fmt"
	"strings"
	"time"
)

var spinnerFrames = []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}

type Spinner struct {
	frames   []rune
	interval time.Duration
	message  string
	stopCh   chan struct{}
	doneCh   chan struct{}
}

func NewSpinner(message string) *Spinner {
	return &Spinner{
		frames:   spinnerFrames,
		interval: 120 * time.Millisecond,
		message:  message,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

func (s *Spinner) Stop() {
	select {
	case <-s.stopCh:
		// Already stopped
		return
	default:
		close(s.stopCh)
	}
}

func (s *Spinner) Start() {
	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		defer close(s.doneCh)

		frame := 0
		for {
			select {
			case <-s.stopCh:
				s.clearLine()
				return
			case <-ticker.C:
				fmt.Printf("\r%c %s", s.frames[frame], s.message)
				frame = (frame + 1) % len(s.frames)
			}
		}
	}()
}

func (s *Spinner) clearLine() {
	// Clear the line: carriage return, spaces, carriage return
	clearLen := len(s.message) + 3 // frame + space + message
	fmt.Printf("\r%s\r", strings.Repeat(" ", clearLen))
}
