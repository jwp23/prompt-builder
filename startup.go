package main

import "time"

const (
	pollInterval   = 500 * time.Millisecond
	connectTimeout = 30 * time.Second
)

func WaitForModel(client *OllamaClient, quiet bool, tty bool) error {
	if quiet || !tty {
		return nil
	}

	loaded, err := client.IsModelLoaded()
	if err != nil {
		return err
	}
	if loaded {
		return nil
	}

	// Model not loaded, poll until it is
	spinner := NewSpinnerWithTTY("Loading "+client.Model+"...", tty)
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
