package main

import "time"

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
