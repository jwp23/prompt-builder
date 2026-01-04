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
	case "copy":
		codeBlock := ExtractLastCodeBlock(lastResponse)
		if lastResponse == "" {
			return false, fmt.Errorf("No response to copy from")
		}
		if codeBlock == "" {
			return false, fmt.Errorf("No code block to copy")
		}
		if clipboardCmd == "" {
			return false, fmt.Errorf("Clipboard not available")
		}
		if err := CopyToClipboard(codeBlock, clipboardCmd); err != nil {
			return false, fmt.Errorf("Clipboard not available")
		}
		fmt.Fprintln(out, "\u2713 Copied to clipboard")
		return true, nil
	case "help":
		fmt.Fprintln(out, `Commands:
  /copy   Copy last code block to clipboard and exit
  /bye    Exit conversation
  /quit   Exit conversation
  /exit   Exit conversation
  /help   Show this help`)
		return false, nil
	default:
		return false, fmt.Errorf("Unknown command: /%s. Type /help for available commands.", cmd)
	}
}
