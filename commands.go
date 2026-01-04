// commands.go
package main

import (
	"fmt"
	"io"
	"strings"
)

// IsCommand returns true if input starts with a slash.
func IsCommand(input string) bool {
	return strings.HasPrefix(input, "/")
}

// parseCommand extracts the command name (lowercase, no slash) from input.
func parseCommand(input string) string {
	trimmed := strings.TrimSpace(input)
	if !strings.HasPrefix(trimmed, "/") {
		return ""
	}
	cmd := strings.TrimPrefix(trimmed, "/")
	return strings.ToLower(cmd)
}

// HandleCommand executes a slash command.
// Returns shouldExit=true if the conversation should end.
// Writes output to the provided writer.
func HandleCommand(input, lastResponse, clipboardCmd string, out io.Writer) (shouldExit bool, err error) {
	cmd := parseCommand(input)

	switch cmd {
	case "bye", "quit", "exit":
		fmt.Fprintln(out, "Goodbye")
		return true, nil
	default:
		return false, nil
	}
}
