// interfaces.go
package main

import "io"

// OllamaChatter abstracts the Ollama client for testing.
type OllamaChatter interface {
	ChatStream(messages []Message, onToken StreamCallback) (string, error)
	ChatStreamWithSpinner(messages []Message, tty bool, onToken StreamCallback) (string, error)
}

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

// Deps holds injectable dependencies for the app.
type Deps struct {
	Client    OllamaChatter
	Stdin     io.Reader
	Stdout    io.Writer
	Stderr    io.Writer
	Clipboard ClipboardWriter
	IsTTY     func() bool
}
