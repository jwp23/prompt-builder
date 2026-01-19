package main

import (
	"fmt"
	"time"
)

const (
	pollInterval   = 500 * time.Millisecond
	connectTimeout = 30 * time.Second
)

func WaitForModel(client *OllamaClient, quiet bool, tty bool) error {
	return WaitForModelWithTimeout(client, quiet, tty, connectTimeout)
}

func WaitForModelWithTimeout(client *OllamaClient, quiet bool, tty bool, timeout time.Duration) error {
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

	deadline := time.Now().Add(timeout)

	for range ticker.C {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for model to load")
		}

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
