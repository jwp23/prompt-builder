// testhelpers_test.go
package main

import (
	"bytes"
	"errors"
	"strings"
)

// mockOllama implements OllamaChatter for testing.
type mockOllama struct {
	responses []string
	calls     int
	err       error
}

func (m *mockOllama) ChatStream(messages []Message, onToken StreamCallback) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if m.calls >= len(m.responses) {
		return "", errors.New("no more mock responses")
	}
	resp := m.responses[m.calls]
	m.calls++

	// Simulate streaming by calling callback with chunks
	for _, chunk := range strings.Split(resp, " ") {
		if err := onToken(chunk + " "); err != nil {
			return "", err
		}
	}
	return resp, nil
}

func (m *mockOllama) ChatStreamWithSpinner(messages []Message, tty bool, onToken StreamCallback) (string, error) {
	return m.ChatStream(messages, onToken)
}

// mockClipboard implements ClipboardWriter for testing.
type mockClipboard struct {
	written string
	err     error
}

func (m *mockClipboard) Write(text string) error {
	if m.err != nil {
		return m.err
	}
	m.written = text
	return nil
}

// testOption configures a test Deps.
type testOption func(*Deps)

// newTestDeps creates Deps with mocks for testing.
func newTestDeps(opts ...testOption) *Deps {
	d := &Deps{
		Client:       &mockOllama{},
		Stdin:        strings.NewReader(""),
		Stdout:       &bytes.Buffer{},
		Stderr:       &bytes.Buffer{},
		Clipboard:    &mockClipboard{},
		IsTTY:        func() bool { return true },
		SystemPrompt: "You are a test assistant.",
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

func withResponses(responses ...string) testOption {
	return func(d *Deps) {
		d.Client = &mockOllama{responses: responses}
	}
}

func withOllamaError(err error) testOption {
	return func(d *Deps) {
		d.Client = &mockOllama{err: err}
	}
}

func withStdin(input string) testOption {
	return func(d *Deps) {
		d.Stdin = strings.NewReader(input)
	}
}

func withTTY(tty bool) testOption {
	return func(d *Deps) {
		d.IsTTY = func() bool { return tty }
	}
}

// stdout returns the captured stdout as string.
func stdout(d *Deps) string {
	return d.Stdout.(*bytes.Buffer).String()
}

// stderr returns the captured stderr as string.
func stderr(d *Deps) string {
	return d.Stderr.(*bytes.Buffer).String()
}

// clipboardWritten returns what was written to clipboard.
func clipboardWritten(d *Deps) string {
	if m, ok := d.Clipboard.(*mockClipboard); ok {
		return m.written
	}
	return ""
}
