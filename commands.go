// commands.go
package main

import "strings"

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
