package main

import (
	"time"
)

const (
	pollInterval = 500 * time.Millisecond
)

func WaitForModel(client *OllamaClient, quiet bool, tty bool) error {
	if quiet || !tty {
		return nil
	}

	loaded, err := client.IsModelLoaded()
	if err == nil && loaded {
		return nil
	}

	// Model not loaded or error connecting, poll until ready
	message := "Loading " + client.Model + "..."
	if err != nil {
		message = "Connecting..."
	}

	spinner := NewSpinnerWithTTY(message, tty)
	spinner.Start()
	defer spinner.Stop()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for range ticker.C {
		loaded, err := client.IsModelLoaded()
		if err != nil {
			continue // Keep trying on errors
		}
		if loaded {
			return nil
		}
	}

	return nil
}
