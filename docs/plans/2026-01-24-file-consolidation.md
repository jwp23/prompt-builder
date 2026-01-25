# File Consolidation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Reduce `cmd/prompt-builder/` from 10 source files to 4 by consolidating related code by domain.

**Architecture:** Pure refactoring—no logic changes. Move code between files, update imports, delete empty files. Tests must pass after each task.

**Tech Stack:** Go 1.21+

---

## Task 1: Consolidate ollama.go (absorb spinner.go, conversation.go, OllamaChatter interface)

**Files:**
- Modify: `cmd/prompt-builder/ollama.go`
- Modify: `cmd/prompt-builder/interfaces.go` (remove OllamaChatter)
- Delete: `cmd/prompt-builder/spinner.go`
- Delete: `cmd/prompt-builder/conversation.go`

**Step 1: Add OllamaChatter interface to ollama.go**

Add at the top of ollama.go, after the imports:

```go
// OllamaChatter abstracts the Ollama client for testing.
type OllamaChatter interface {
	ChatStream(messages []Message, onToken StreamCallback) (string, error)
	ChatStreamWithSpinner(messages []Message, tty bool, onToken StreamCallback) (string, error)
}
```

**Step 2: Add Conversation struct and methods to ollama.go**

Add at the end of ollama.go:

```go
type Conversation struct {
	Messages []Message
}

func NewConversation(systemPrompt string) *Conversation {
	return &Conversation{
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
		},
	}
}

func (c *Conversation) AddUserMessage(content string) {
	c.Messages = append(c.Messages, Message{Role: "user", Content: content})
}

func (c *Conversation) AddAssistantMessage(content string) {
	c.Messages = append(c.Messages, Message{Role: "assistant", Content: content})
}
```

**Step 3: Add Spinner code to ollama.go**

Add after the Conversation code:

```go
var spinnerFrames = []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}

type Spinner struct {
	frames   []rune
	interval time.Duration
	message  string
	tty      bool
	stopCh   chan struct{}
	doneCh   chan struct{}
}

func NewSpinner(message string) *Spinner {
	return NewSpinnerWithTTY(message, true)
}

func NewSpinnerWithTTY(message string, tty bool) *Spinner {
	return &Spinner{
		frames:   spinnerFrames,
		interval: 120 * time.Millisecond,
		message:  message,
		tty:      tty,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

func (s *Spinner) Stop() {
	select {
	case <-s.stopCh:
		// Already stopped
		return
	default:
		close(s.stopCh)
	}
}

func (s *Spinner) Start() {
	if !s.tty {
		return
	}
	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		defer close(s.doneCh)

		frame := 0
		for {
			select {
			case <-s.stopCh:
				s.clearLine()
				return
			case <-ticker.C:
				fmt.Printf("\r%c %s", s.frames[frame], s.message)
				frame = (frame + 1) % len(s.frames)
			}
		}
	}()
}

func (s *Spinner) clearLine() {
	// Clear the line: carriage return, spaces, carriage return
	clearLen := len(s.message) + 3 // frame + space + message
	fmt.Printf("\r%s\r", strings.Repeat(" ", clearLen))
}
```

**Step 4: Update ollama.go imports**

Ensure imports include `fmt`, `strings`, and `time` (for Spinner).

**Step 5: Remove OllamaChatter from interfaces.go**

Edit `interfaces.go` to remove the OllamaChatter interface (lines 7-10).

**Step 6: Delete spinner.go and conversation.go**

```bash
rm cmd/prompt-builder/spinner.go
rm cmd/prompt-builder/conversation.go
```

**Step 7: Run tests**

```bash
go test ./cmd/prompt-builder/...
```

Expected: All tests pass.

**Step 8: Commit**

```bash
git add cmd/prompt-builder/ollama.go cmd/prompt-builder/interfaces.go
git rm cmd/prompt-builder/spinner.go cmd/prompt-builder/conversation.go
git commit -m "refactor: consolidate spinner and conversation into ollama.go"
```

---

## Task 2: Consolidate ollama_test.go (absorb spinner_test.go, conversation_test.go)

**Files:**
- Modify: `cmd/prompt-builder/ollama_test.go`
- Delete: `cmd/prompt-builder/spinner_test.go`
- Delete: `cmd/prompt-builder/conversation_test.go`

**Step 1: Add conversation tests to ollama_test.go**

Add at the end of ollama_test.go:

```go
func TestConversation_AddMessage(t *testing.T) {
	conv := NewConversation("You are helpful.")

	// Should start with system message
	if len(conv.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(conv.Messages))
	}
	if conv.Messages[0].Role != "system" {
		t.Errorf("first message role = %q, want %q", conv.Messages[0].Role, "system")
	}

	conv.AddUserMessage("Hello")
	if len(conv.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(conv.Messages))
	}
	if conv.Messages[1].Role != "user" {
		t.Errorf("second message role = %q, want %q", conv.Messages[1].Role, "user")
	}

	conv.AddAssistantMessage("Hi there!")
	if len(conv.Messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(conv.Messages))
	}
	if conv.Messages[2].Role != "assistant" {
		t.Errorf("third message role = %q, want %q", conv.Messages[2].Role, "assistant")
	}
}
```

**Step 2: Add spinner tests to ollama_test.go**

Add after conversation tests:

```go
func TestNewSpinner(t *testing.T) {
	s := NewSpinner("Loading...")
	if s == nil {
		t.Fatal("NewSpinner returned nil")
	}
	if s.message != "Loading..." {
		t.Errorf("message = %q, want %q", s.message, "Loading...")
	}
}

func TestSpinner_StopWithoutStart(t *testing.T) {
	s := NewSpinner("Test")
	// Should not panic
	s.Stop()
}

func TestSpinner_StopMultipleTimes(t *testing.T) {
	s := NewSpinner("Test")
	// Should not panic on multiple Stop calls
	s.Stop()
	s.Stop()
	s.Stop()
}

func TestSpinner_StartStop(t *testing.T) {
	s := NewSpinner("Loading")
	s.Start()
	// Give it a moment to run
	time.Sleep(50 * time.Millisecond)
	s.Stop()
	// Should complete without hanging
}

func TestNewSpinnerWithTTY_False(t *testing.T) {
	s := NewSpinnerWithTTY("Loading", false)
	if s.tty {
		t.Error("expected tty to be false")
	}
}

func TestNewSpinnerWithTTY_True(t *testing.T) {
	s := NewSpinnerWithTTY("Loading", true)
	if !s.tty {
		t.Error("expected tty to be true")
	}
}

func TestSpinner_StartNonTTY(t *testing.T) {
	s := NewSpinnerWithTTY("Loading", false)
	s.Start() // Should be no-op, not start goroutine
	s.Stop()  // Should be safe
}
```

**Step 3: Update ollama_test.go imports**

Ensure imports include `time`.

**Step 4: Delete spinner_test.go and conversation_test.go**

```bash
rm cmd/prompt-builder/spinner_test.go
rm cmd/prompt-builder/conversation_test.go
```

**Step 5: Run tests**

```bash
go test ./cmd/prompt-builder/...
```

Expected: All tests pass.

**Step 6: Commit**

```bash
git add cmd/prompt-builder/ollama_test.go
git rm cmd/prompt-builder/spinner_test.go cmd/prompt-builder/conversation_test.go
git commit -m "refactor: consolidate spinner and conversation tests into ollama_test.go"
```

---

## Task 3: Create slash.go (rename commands.go, absorb clipboard.go, detect.go, ClipboardWriter)

**Files:**
- Create: `cmd/prompt-builder/slash.go` (rename from commands.go)
- Modify: `cmd/prompt-builder/interfaces.go` (remove ClipboardWriter)
- Delete: `cmd/prompt-builder/commands.go`
- Delete: `cmd/prompt-builder/clipboard.go`
- Delete: `cmd/prompt-builder/detect.go`

**Step 1: Create slash.go with all consolidated code**

Create `cmd/prompt-builder/slash.go` with this content:

```go
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

// HandleCommandWithClipboard is like HandleCommand but uses ClipboardWriter interface.
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
```

**Step 2: Remove ClipboardWriter from interfaces.go**

Edit `interfaces.go` to remove ClipboardWriter interface, clipboardFunc struct, and NewClipboardWriter function (the entire file should now only contain Deps).

**Step 3: Delete old files**

```bash
rm cmd/prompt-builder/commands.go
rm cmd/prompt-builder/clipboard.go
rm cmd/prompt-builder/detect.go
```

**Step 4: Run tests**

```bash
go test ./cmd/prompt-builder/...
```

Expected: All tests pass.

**Step 5: Commit**

```bash
git add cmd/prompt-builder/slash.go cmd/prompt-builder/interfaces.go
git rm cmd/prompt-builder/commands.go cmd/prompt-builder/clipboard.go cmd/prompt-builder/detect.go
git commit -m "refactor: create slash.go consolidating commands, clipboard, and detect"
```

---

## Task 4: Create slash_test.go (rename commands_test.go, absorb clipboard_test.go, detect_test.go)

**Files:**
- Create: `cmd/prompt-builder/slash_test.go`
- Delete: `cmd/prompt-builder/commands_test.go`
- Delete: `cmd/prompt-builder/clipboard_test.go`
- Delete: `cmd/prompt-builder/detect_test.go`

**Step 1: Create slash_test.go with all consolidated tests**

Create `cmd/prompt-builder/slash_test.go` with this content:

```go
// slash_test.go
package main

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

func TestDetectClipboardCmd(t *testing.T) {
	// This test verifies the detection logic
	// Actual availability depends on system
	cmd := DetectClipboardCmd("")

	// Should return something or empty string
	// Can't assert exact value as it's system-dependent
	t.Logf("Detected clipboard command: %q", cmd)

	// If a command is returned, it should be executable
	if cmd != "" {
		parts := strings.Split(cmd, " ")
		_, err := exec.LookPath(parts[0])
		if err != nil {
			t.Errorf("Detected command %q but binary not found", parts[0])
		}
	}
}

func TestDetectClipboardCmd_Override(t *testing.T) {
	cmd := DetectClipboardCmd("custom-clipboard")
	if cmd != "custom-clipboard" {
		t.Errorf("DetectClipboardCmd with override = %q, want %q", cmd, "custom-clipboard")
	}
}

func TestExtractLastCodeBlock(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "single code block",
			input: "Here is your prompt:\n```\n# Role\nYou are an expert.\n```\n",
			want: "# Role\nYou are an expert.\n",
		},
		{
			name: "multiple code blocks - returns last",
			input: "Example:\n```\nfirst block\n```\n\nHere is the final:\n```\nsecond block\n```\n",
			want: "second block\n",
		},
		{
			name:  "no code block",
			input: "Just plain text",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractLastCodeBlock(tt.input)
			if got != tt.want {
				t.Errorf("ExtractLastCodeBlock() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsComplete(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "code block without question - complete",
			input: "Here is your prompt:\n```\ncontent\n```\n",
			want:  true,
		},
		{
			name:  "code block with trailing question - not complete",
			input: "Here is a draft:\n```\ncontent\n```\nDoes this look right?",
			want:  false,
		},
		{
			name:  "question only - not complete",
			input: "What is your target audience?",
			want:  false,
		},
		{
			name:  "no code block no question - not complete",
			input: "Let me think about that.",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsComplete(tt.input)
			if got != tt.want {
				t.Errorf("IsComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsCommand(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"slash command", "/copy", true},
		{"slash with text", "/help", true},
		{"normal text", "hello", false},
		{"empty string", "", false},
		{"just slash", "/", true},
		{"slash in middle", "foo /bar", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsCommand(tt.input)
			if got != tt.want {
				t.Errorf("IsCommand(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantCmd string
	}{
		{"lowercase", "/copy", "copy"},
		{"uppercase", "/COPY", "copy"},
		{"mixed case", "/Copy", "copy"},
		{"with whitespace", "  /HELP  ", "help"},
		{"exit", "/exit", "exit"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseCommand(tt.input)
			if got != tt.wantCmd {
				t.Errorf("parseCommand(%q) = %q, want %q", tt.input, got, tt.wantCmd)
			}
		})
	}
}

func TestHandleCommand_Exit(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantExit   bool
		wantOutput string
	}{
		{"bye", "/bye", true, "Goodbye\n"},
		{"quit", "/quit", true, "Goodbye\n"},
		{"exit", "/exit", true, "Goodbye\n"},
		{"BYE uppercase", "/BYE", true, "Goodbye\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			shouldExit, err := HandleCommand(tt.input, "", "", &out)
			if err != nil {
				t.Errorf("HandleCommand() error = %v", err)
			}
			if shouldExit != tt.wantExit {
				t.Errorf("HandleCommand() shouldExit = %v, want %v", shouldExit, tt.wantExit)
			}
			if out.String() != tt.wantOutput {
				t.Errorf("HandleCommand() output = %q, want %q", out.String(), tt.wantOutput)
			}
		})
	}
}

func TestHandleCommand_Unknown(t *testing.T) {
	var out bytes.Buffer
	shouldExit, err := HandleCommand("/foo", "", "", &out)

	if err == nil {
		t.Error("HandleCommand() expected error for unknown command")
	}
	if shouldExit {
		t.Error("HandleCommand() should not exit on unknown command")
	}
	wantErr := "Unknown command: /foo. Type /help for available commands."
	if err.Error() != wantErr {
		t.Errorf("HandleCommand() error = %q, want %q", err.Error(), wantErr)
	}
}

func TestHandleCommand_Help(t *testing.T) {
	var out bytes.Buffer
	shouldExit, err := HandleCommand("/help", "", "", &out)

	if err != nil {
		t.Errorf("HandleCommand() error = %v", err)
	}
	if shouldExit {
		t.Error("HandleCommand() should not exit on /help")
	}

	wantOutput := `Commands:
  /copy   Copy last code block to clipboard and exit
  /bye    Exit conversation
  /quit   Exit conversation
  /exit   Exit conversation
  /help   Show this help
`
	if out.String() != wantOutput {
		t.Errorf("HandleCommand() output = %q, want %q", out.String(), wantOutput)
	}
}

func TestHandleCommand_Copy_Success(t *testing.T) {
	lastResponse := "Here is your code:\n```\nfmt.Println(\"hello\")\n```\n"

	var out bytes.Buffer
	// Use "cat" as a mock clipboard command that accepts stdin
	shouldExit, err := HandleCommand("/copy", lastResponse, "cat", &out)

	if err != nil {
		t.Errorf("HandleCommand() error = %v", err)
	}
	if !shouldExit {
		t.Error("HandleCommand() should exit on /copy")
	}
	wantOutput := "\u2713 Copied to clipboard\n"
	if out.String() != wantOutput {
		t.Errorf("HandleCommand() output = %q, want %q", out.String(), wantOutput)
	}
}

func TestHandleCommand_Copy_NoResponse(t *testing.T) {
	var out bytes.Buffer
	_, err := HandleCommand("/copy", "", "cat", &out)

	if err == nil {
		t.Error("HandleCommand() expected error when no response")
	}
	wantErr := "No response to copy from"
	if err.Error() != wantErr {
		t.Errorf("HandleCommand() error = %q, want %q", err.Error(), wantErr)
	}
}

func TestHandleCommand_Copy_NoCodeBlock(t *testing.T) {
	var out bytes.Buffer
	_, err := HandleCommand("/copy", "Just plain text", "cat", &out)

	if err == nil {
		t.Error("HandleCommand() expected error when no code block")
	}
	wantErr := "No code block to copy"
	if err.Error() != wantErr {
		t.Errorf("HandleCommand() error = %q, want %q", err.Error(), wantErr)
	}
}

func TestHandleCommand_Copy_NoClipboard(t *testing.T) {
	lastResponse := "```\ncode\n```"
	var out bytes.Buffer
	_, err := HandleCommand("/copy", lastResponse, "", &out)

	if err == nil {
		t.Error("HandleCommand() expected error when clipboard unavailable")
	}
	wantErr := "Clipboard not available"
	if err.Error() != wantErr {
		t.Errorf("HandleCommand() error = %q, want %q", err.Error(), wantErr)
	}
}
```

**Step 2: Delete old test files**

```bash
rm cmd/prompt-builder/commands_test.go
rm cmd/prompt-builder/clipboard_test.go
rm cmd/prompt-builder/detect_test.go
```

**Step 3: Run tests**

```bash
go test ./cmd/prompt-builder/...
```

Expected: All tests pass.

**Step 4: Commit**

```bash
git add cmd/prompt-builder/slash_test.go
git rm cmd/prompt-builder/commands_test.go cmd/prompt-builder/clipboard_test.go cmd/prompt-builder/detect_test.go
git commit -m "refactor: create slash_test.go consolidating command, clipboard, and detect tests"
```

---

## Task 5: Consolidate main.go (absorb Deps from interfaces.go)

**Files:**
- Modify: `cmd/prompt-builder/main.go`
- Delete: `cmd/prompt-builder/interfaces.go`

**Step 1: Add Deps struct to main.go**

Add after the CLI struct in main.go (around line 34):

```go
// Deps holds injectable dependencies for the app.
type Deps struct {
	Client    OllamaChatter
	Stdin     io.Reader
	Stdout    io.Writer
	Stderr    io.Writer
	Clipboard ClipboardWriter
	IsTTY     func() bool
}
```

**Step 2: Update main.go imports**

Ensure `io` is in the imports (it should already be there via other usage, but verify).

**Step 3: Delete interfaces.go**

```bash
rm cmd/prompt-builder/interfaces.go
```

**Step 4: Run tests**

```bash
go test ./cmd/prompt-builder/...
```

Expected: All tests pass.

**Step 5: Commit**

```bash
git add cmd/prompt-builder/main.go
git rm cmd/prompt-builder/interfaces.go
git commit -m "refactor: move Deps struct to main.go, delete interfaces.go"
```

---

## Task 6: Consolidate main_test.go (absorb interfaces_test.go)

**Files:**
- Modify: `cmd/prompt-builder/main_test.go` (create if doesn't exist)
- Delete: `cmd/prompt-builder/interfaces_test.go`

**Step 1: Check if main_test.go exists**

```bash
ls cmd/prompt-builder/main_test.go
```

If it doesn't exist, create it. If it does, append to it.

**Step 2: Add interface compliance tests to main_test.go**

Create or append to `cmd/prompt-builder/main_test.go`:

```go
package main

import (
	"context"
	"testing"
)

func TestOllamaClient_ImplementsOllamaChatter(t *testing.T) {
	var _ OllamaChatter = (*OllamaClient)(nil)
}

func TestRunWithDeps_Exists(t *testing.T) {
	// Just verify the function signature exists
	var _ func(context.Context, *CLI, *Deps) error = runWithDeps
}
```

**Step 3: Delete interfaces_test.go**

```bash
rm cmd/prompt-builder/interfaces_test.go
```

**Step 4: Run tests**

```bash
go test ./cmd/prompt-builder/...
```

Expected: All tests pass.

**Step 5: Commit**

```bash
git add cmd/prompt-builder/main_test.go
git rm cmd/prompt-builder/interfaces_test.go
git commit -m "refactor: move interface tests to main_test.go, delete interfaces_test.go"
```

---

## Task 7: Final verification and cleanup

**Step 1: Verify file structure**

```bash
ls cmd/prompt-builder/*.go | grep -v _test
```

Expected output (4 files):
```
cmd/prompt-builder/config.go
cmd/prompt-builder/main.go
cmd/prompt-builder/ollama.go
cmd/prompt-builder/slash.go
```

**Step 2: Verify test file structure**

```bash
ls cmd/prompt-builder/*_test.go
```

Expected output:
```
cmd/prompt-builder/config_test.go
cmd/prompt-builder/e2e_test.go
cmd/prompt-builder/integration_test.go
cmd/prompt-builder/main_test.go
cmd/prompt-builder/ollama_test.go
cmd/prompt-builder/slash_test.go
cmd/prompt-builder/testhelpers_test.go
```

**Step 3: Run all tests**

```bash
go test ./cmd/prompt-builder/...
```

Expected: All tests pass.

**Step 4: Run go fmt**

```bash
go fmt ./cmd/prompt-builder/...
```

**Step 5: Final commit if any formatting changes**

```bash
git status
# If changes:
git add -A
git commit -m "style: go fmt"
```

**Step 6: Verify line counts**

```bash
wc -l cmd/prompt-builder/*.go | grep -v _test | sort -n
```

Should show 4 source files with reasonable line counts.
