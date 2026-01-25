// slash.go
package main

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// ClipboardWriter abstracts clipboard operations for testing.
type ClipboardWriter interface {
	Write(text string) error
}

// clipboardFunc adapts a function to ClipboardWriter.
type clipboardFunc struct {
	cmd string
}

func (c *clipboardFunc) Write(text string) error {
	return CopyToClipboard(text, c.cmd)
}

// NewClipboardWriter creates a ClipboardWriter from a command string.
func NewClipboardWriter(cmd string) ClipboardWriter {
	return &clipboardFunc{cmd: cmd}
}

// DetectClipboardCmd returns the clipboard command to use.
func DetectClipboardCmd(override string) string {
	if override != "" {
		return override
	}

	candidates := []string{
		"wl-copy",
		"xclip -selection clipboard",
		"xsel --clipboard --input",
		"pbcopy",
	}

	for _, cmd := range candidates {
		parts := strings.Split(cmd, " ")
		if _, err := exec.LookPath(parts[0]); err == nil {
			return cmd
		}
	}

	return ""
}

// CopyToClipboard copies text to the clipboard using the given command.
func CopyToClipboard(text string, cmd string) error {
	if cmd == "" {
		return nil // No clipboard available, silently skip
	}

	parts := strings.Split(cmd, " ")
	c := exec.Command(parts[0], parts[1:]...)
	c.Stdin = strings.NewReader(text)
	return c.Run()
}

// ExtractLastCodeBlock extracts the content of the last code block from text.
func ExtractLastCodeBlock(text string) string {
	const marker = "```"

	lastStart := strings.LastIndex(text, marker)
	if lastStart == -1 {
		return ""
	}

	// Find the opening marker for this block
	beforeLast := text[:lastStart]
	openStart := strings.LastIndex(beforeLast, marker)
	if openStart == -1 {
		return ""
	}

	// Extract content between markers
	// Skip past the opening ``` and any language identifier on that line
	contentStart := openStart + len(marker)
	if idx := strings.Index(text[contentStart:lastStart], "\n"); idx != -1 {
		contentStart += idx + 1
	}

	return text[contentStart:lastStart]
}

// IsComplete returns true if the response contains a code block and doesn't end with a question.
func IsComplete(response string) bool {
	hasCodeBlock := strings.Contains(response, "```")
	trimmed := strings.TrimSpace(response)
	endsWithQuestion := strings.HasSuffix(trimmed, "?")
	return hasCodeBlock && !endsWithQuestion
}

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

// HandleCommandWithClipboard executes a slash command.
func HandleCommandWithClipboard(input, lastResponse string, clipboard ClipboardWriter, out io.Writer) (shouldExit bool, err error) {
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
		if clipboard == nil {
			return false, fmt.Errorf("Clipboard not available")
		}
		if err := clipboard.Write(codeBlock); err != nil {
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
