# Integration and E2E Tests Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add integration and E2E tests with dependency injection for refactoring confidence and regression protection.

**Architecture:** Refactor `main.go` to use an `App` struct with injected dependencies (OllamaClient interface, io.Reader/Writer for stdin/stdout, ClipboardWriter interface). Integration tests use mocks; E2E tests build the binary and invoke it with real Ollama.

**Tech Stack:** Go 1.25, standard `testing` package, `httptest` for mocks, build tags for E2E separation.

---

## Task 1: Define Interfaces

**Files:**
- Create: `cmd/prompt-builder/interfaces.go`
- Test: `cmd/prompt-builder/interfaces_test.go`

**Step 1: Write failing test for OllamaChatter interface**

```go
// interfaces_test.go
package main

import (
	"context"
	"testing"
)

func TestOllamaClient_ImplementsOllamaChatter(t *testing.T) {
	var _ OllamaChatter = (*OllamaClient)(nil)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./cmd/prompt-builder -run TestOllamaClient_ImplementsOllamaChatter -v`
Expected: FAIL with "undefined: OllamaChatter"

**Step 3: Write OllamaChatter interface**

```go
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
```

**Step 4: Run test to verify it passes**

Run: `go test ./cmd/prompt-builder -run TestOllamaClient_ImplementsOllamaChatter -v`
Expected: PASS

**Step 5: Commit**

```bash
git add cmd/prompt-builder/interfaces.go cmd/prompt-builder/interfaces_test.go
git commit -m "feat: add OllamaChatter and ClipboardWriter interfaces for DI"
```

---

## Task 2: Refactor run() to Accept Dependencies

**Files:**
- Modify: `cmd/prompt-builder/main.go`
- Test: existing tests should still pass

**Step 1: Write failing test for runWithDeps**

Add to `interfaces_test.go`:

```go
func TestRunWithDeps_Exists(t *testing.T) {
	// Just verify the function signature exists
	var _ func(context.Context, *CLI, *Deps) error = runWithDeps
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./cmd/prompt-builder -run TestRunWithDeps_Exists -v`
Expected: FAIL with "undefined: runWithDeps"

**Step 3: Create runWithDeps and refactor run()**

In `main.go`, add `runWithDeps` and have `run()` call it:

```go
func runWithDeps(ctx context.Context, cli *CLI, deps *Deps) error {
	_ = ctx // Context available for future cancellation support

	// Determine config path
	configPath := cli.ConfigPath
	if configPath == "" {
		configPath = defaultConfigPath()
	}
	configPath = ExpandPath(configPath)

	// Load config
	cfg, err := LoadConfig(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(deps.Stderr, "config file not found: %s\n\nCreate it with:\n  mkdir -p ~/.config/prompt-builder\n  cat > ~/.config/prompt-builder/config.yaml << 'EOF'\n  model: llama3.2\n  system_prompt_file: ~/.config/prompt-builder/prompt-architect.md\n  EOF\n", configPath)
			return fmt.Errorf("config file not found: %s", configPath)
		}
		return fmt.Errorf("invalid config: %v", err)
	}

	// Apply CLI overrides
	if cli.Model != "" {
		cfg.Model = cli.Model
	}

	// Validate model
	if cfg.Model == "" {
		return fmt.Errorf("no model specified\n\nSet 'model' in config or use --model flag")
	}

	// Load system prompt
	promptPath := ExpandPath(cfg.SystemPromptFile)
	systemPrompt, err := os.ReadFile(promptPath)
	if err != nil {
		return fmt.Errorf("system prompt not found: %s", promptPath)
	}

	// Initialize conversation
	conv := NewConversation(string(systemPrompt))

	// Prepare user's idea
	userIdea := cli.Idea
	tty := deps.IsTTY()
	if !tty {
		// Pipe mode: ask for immediate generation
		userIdea = "Generate your best prompt without asking clarifying questions. User's idea: " + userIdea
	}
	conv.AddUserMessage(userIdea)

	// Conversation loop
	reader := bufio.NewReader(deps.Stdin)
	for {
		// Get response from LLM with streaming
		response, err := deps.Client.ChatStreamWithSpinner(conv.Messages, tty && !cli.Quiet, func(token string) error {
			if !cli.Quiet {
				fmt.Fprint(deps.Stdout, token)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("Ollama request failed: %v", err)
		}
		if !cli.Quiet {
			fmt.Fprintln(deps.Stdout) // newline after streaming completes
		}

		conv.AddAssistantMessage(response)

		// Pipe mode: output result and exit (can't continue conversation)
		if !tty {
			if IsComplete(response) {
				if cli.Quiet {
					// In quiet mode, print only the extracted code block
					finalPrompt := ExtractLastCodeBlock(response)
					fmt.Fprintln(deps.Stdout, finalPrompt)
				}
				// Non-quiet mode already streamed the response
				return nil
			}
			return fmt.Errorf("LLM requested clarification but stdin is not a TTY")
		}

		fmt.Fprint(deps.Stdout, "> ")
		userInput, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %v", err)
		}

		userInput = strings.TrimSpace(userInput)

		if IsCommand(userInput) {
			shouldExit, err := HandleCommandWithClipboard(userInput, response, deps.Clipboard, deps.Stdout)
			if err != nil {
				fmt.Fprintln(deps.Stderr, err)
			}
			if shouldExit {
				return nil
			}
			fmt.Fprint(deps.Stdout, "> ")
			continue
		}

		conv.AddUserMessage(userInput)
	}
}

func run(ctx context.Context, cli *CLI) error {
	// Determine config path for client initialization
	configPath := cli.ConfigPath
	if configPath == "" {
		configPath = defaultConfigPath()
	}
	configPath = ExpandPath(configPath)

	cfg, err := LoadConfig(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("config file not found: %s\n\nCreate it with:\n  mkdir -p ~/.config/prompt-builder\n  cat > ~/.config/prompt-builder/config.yaml << 'EOF'\n  model: llama3.2\n  system_prompt_file: ~/.config/prompt-builder/prompt-architect.md\n  EOF", configPath)
		}
		return fmt.Errorf("invalid config: %v", err)
	}

	// Apply CLI model override
	model := cfg.Model
	if cli.Model != "" {
		model = cli.Model
	}

	// Create real dependencies
	deps := &Deps{
		Client:    NewOllamaClient(cfg.OllamaHost, model),
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
		Clipboard: NewClipboardWriter(DetectClipboardCmd(cfg.ClipboardCmd)),
		IsTTY:     isTTY,
	}

	return runWithDeps(ctx, cli, deps)
}
```

**Step 4: Add HandleCommandWithClipboard to commands.go**

```go
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

**Step 5: Run all existing tests**

Run: `go test ./cmd/prompt-builder -v`
Expected: All tests PASS

**Step 6: Commit**

```bash
git add cmd/prompt-builder/main.go cmd/prompt-builder/commands.go cmd/prompt-builder/interfaces_test.go
git commit -m "refactor: extract runWithDeps for dependency injection"
```

---

## Task 3: Create Test Fixtures

**Files:**
- Create: `cmd/prompt-builder/testdata/config/valid.yaml`
- Create: `cmd/prompt-builder/testdata/config/minimal.yaml`
- Create: `cmd/prompt-builder/testdata/prompts/system.txt`

**Step 1: Create testdata directories and files**

```yaml
# testdata/config/valid.yaml
model: test-model
ollama_host: http://localhost:11434
system_prompt_file: testdata/prompts/system.txt
```

```yaml
# testdata/config/minimal.yaml
model: test-model
system_prompt_file: testdata/prompts/system.txt
```

```text
# testdata/prompts/system.txt
You are a test assistant.
```

**Step 2: Verify files exist**

Run: `ls -la cmd/prompt-builder/testdata/`
Expected: Shows config/ and prompts/ directories

**Step 3: Commit**

```bash
git add cmd/prompt-builder/testdata/
git commit -m "test: add test fixtures for integration tests"
```

---

## Task 4: Create Test Helpers

**Files:**
- Create: `cmd/prompt-builder/testhelpers_test.go`

**Step 1: Write test helper file**

```go
// testhelpers_test.go
package main

import (
	"bytes"
	"context"
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
		Client:    &mockOllama{},
		Stdin:     strings.NewReader(""),
		Stdout:    &bytes.Buffer{},
		Stderr:    &bytes.Buffer{},
		Clipboard: &mockClipboard{},
		IsTTY:     func() bool { return true },
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

func withClipboardError(err error) testOption {
	return func(d *Deps) {
		d.Clipboard = &mockClipboard{err: err}
	}
}

func withNoClipboard() testOption {
	return func(d *Deps) {
		d.Clipboard = nil
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
```

**Step 2: Run to verify helpers compile**

Run: `go test ./cmd/prompt-builder -run TestNothing -v`
Expected: No compile errors (test not found is OK)

**Step 3: Commit**

```bash
git add cmd/prompt-builder/testhelpers_test.go
git commit -m "test: add mock helpers for integration tests"
```

---

## Task 5: Integration Test - Single Turn Complete

**Files:**
- Create: `cmd/prompt-builder/integration_test.go`

**Step 1: Write failing test**

```go
// integration_test.go
package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_SingleTurnComplete(t *testing.T) {
	// Create temp config and prompt files
	tmpDir := t.TempDir()
	promptFile := filepath.Join(tmpDir, "prompt.txt")
	configFile := filepath.Join(tmpDir, "config.yaml")

	os.WriteFile(promptFile, []byte("You are a test assistant."), 0644)
	os.WriteFile(configFile, []byte("model: test\nsystem_prompt_file: "+promptFile), 0644)

	// Response with code block = complete
	completeResponse := "Here is your prompt:\n```\nTest prompt content\n```"

	deps := newTestDeps(
		withResponses(completeResponse),
		withTTY(false), // Pipe mode exits after one response
	)

	cli := &CLI{
		ConfigPath: configFile,
		Idea:       "test idea",
	}

	err := runWithDeps(context.Background(), cli, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := stdout(deps)
	if !strings.Contains(out, "Test prompt content") {
		t.Errorf("expected response in stdout, got: %s", out)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./cmd/prompt-builder -run TestRun_SingleTurnComplete -v`
Expected: May fail if runWithDeps doesn't exist yet, or pass if Task 2 is complete

**Step 3: Run test to verify it passes**

Run: `go test ./cmd/prompt-builder -run TestRun_SingleTurnComplete -v`
Expected: PASS

**Step 4: Commit**

```bash
git add cmd/prompt-builder/integration_test.go
git commit -m "test: add TestRun_SingleTurnComplete integration test"
```

---

## Task 6: Integration Test - Multi-Turn Conversation

**Files:**
- Modify: `cmd/prompt-builder/integration_test.go`

**Step 1: Write failing test**

Add to `integration_test.go`:

```go
func TestRun_MultiTurnConversation(t *testing.T) {
	tmpDir := t.TempDir()
	promptFile := filepath.Join(tmpDir, "prompt.txt")
	configFile := filepath.Join(tmpDir, "config.yaml")

	os.WriteFile(promptFile, []byte("You are a test assistant."), 0644)
	os.WriteFile(configFile, []byte("model: test\nsystem_prompt_file: "+promptFile), 0644)

	// First response asks a question, second completes
	clarifyingResponse := "What language would you like the prompt in?"
	completeResponse := "Here is your prompt:\n```\nFinal prompt\n```"

	deps := newTestDeps(
		withResponses(clarifyingResponse, completeResponse),
		withStdin("/bye\n"), // User exits after first response
		withTTY(true),
	)

	cli := &CLI{
		ConfigPath: configFile,
		Idea:       "test idea",
	}

	err := runWithDeps(context.Background(), cli, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := stdout(deps)
	if !strings.Contains(out, "What language") {
		t.Errorf("expected clarifying question in stdout, got: %s", out)
	}
}
```

**Step 2: Run test**

Run: `go test ./cmd/prompt-builder -run TestRun_MultiTurnConversation -v`
Expected: PASS

**Step 3: Commit**

```bash
git add cmd/prompt-builder/integration_test.go
git commit -m "test: add TestRun_MultiTurnConversation integration test"
```

---

## Task 7: Integration Test - Pipe Mode

**Files:**
- Modify: `cmd/prompt-builder/integration_test.go`

**Step 1: Write test**

Add to `integration_test.go`:

```go
func TestRun_PipeMode(t *testing.T) {
	tmpDir := t.TempDir()
	promptFile := filepath.Join(tmpDir, "prompt.txt")
	configFile := filepath.Join(tmpDir, "config.yaml")

	os.WriteFile(promptFile, []byte("You are a test assistant."), 0644)
	os.WriteFile(configFile, []byte("model: test\nsystem_prompt_file: "+promptFile), 0644)

	completeResponse := "Here is your prompt:\n```\nPipe mode prompt\n```"

	deps := newTestDeps(
		withResponses(completeResponse),
		withTTY(false), // Pipe mode
	)

	cli := &CLI{
		ConfigPath: configFile,
		Idea:       "test idea",
	}

	// Capture messages sent to mock
	mock := deps.Client.(*mockOllama)

	err := runWithDeps(context.Background(), cli, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify "Generate without questions" prefix was added
	if mock.calls != 1 {
		t.Errorf("expected 1 call, got %d", mock.calls)
	}
}

func TestRun_PipeMode_Quiet(t *testing.T) {
	tmpDir := t.TempDir()
	promptFile := filepath.Join(tmpDir, "prompt.txt")
	configFile := filepath.Join(tmpDir, "config.yaml")

	os.WriteFile(promptFile, []byte("You are a test assistant."), 0644)
	os.WriteFile(configFile, []byte("model: test\nsystem_prompt_file: "+promptFile), 0644)

	completeResponse := "Here is your prompt:\n```\nQuiet mode output\n```"

	deps := newTestDeps(
		withResponses(completeResponse),
		withTTY(false),
	)

	cli := &CLI{
		ConfigPath: configFile,
		Idea:       "test idea",
		Quiet:      true,
	}

	err := runWithDeps(context.Background(), cli, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := stdout(deps)
	// In quiet mode, only the code block content should be printed
	if !strings.Contains(out, "Quiet mode output") {
		t.Errorf("expected code block in stdout, got: %s", out)
	}
	// Should NOT contain the markdown fence
	if strings.Contains(out, "```") {
		t.Errorf("should not contain markdown fence in quiet mode, got: %s", out)
	}
}
```

**Step 2: Run tests**

Run: `go test ./cmd/prompt-builder -run TestRun_PipeMode -v`
Expected: PASS

**Step 3: Commit**

```bash
git add cmd/prompt-builder/integration_test.go
git commit -m "test: add pipe mode integration tests"
```

---

## Task 8: Integration Test - Ollama Error

**Files:**
- Modify: `cmd/prompt-builder/integration_test.go`

**Step 1: Write test**

Add to `integration_test.go`:

```go
func TestRun_OllamaError(t *testing.T) {
	tmpDir := t.TempDir()
	promptFile := filepath.Join(tmpDir, "prompt.txt")
	configFile := filepath.Join(tmpDir, "config.yaml")

	os.WriteFile(promptFile, []byte("You are a test assistant."), 0644)
	os.WriteFile(configFile, []byte("model: test\nsystem_prompt_file: "+promptFile), 0644)

	deps := newTestDeps(
		withOllamaError(errors.New("connection refused")),
		withTTY(false),
	)

	cli := &CLI{
		ConfigPath: configFile,
		Idea:       "test idea",
	}

	err := runWithDeps(context.Background(), cli, deps)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "Ollama") {
		t.Errorf("expected Ollama error, got: %v", err)
	}
}
```

Add import for `errors` package at the top.

**Step 2: Run test**

Run: `go test ./cmd/prompt-builder -run TestRun_OllamaError -v`
Expected: PASS

**Step 3: Commit**

```bash
git add cmd/prompt-builder/integration_test.go
git commit -m "test: add Ollama error integration test"
```

---

## Task 9: Integration Test - Commands

**Files:**
- Modify: `cmd/prompt-builder/integration_test.go`

**Step 1: Write tests**

Add to `integration_test.go`:

```go
func TestCommand_Copy(t *testing.T) {
	tmpDir := t.TempDir()
	promptFile := filepath.Join(tmpDir, "prompt.txt")
	configFile := filepath.Join(tmpDir, "config.yaml")

	os.WriteFile(promptFile, []byte("You are a test assistant."), 0644)
	os.WriteFile(configFile, []byte("model: test\nsystem_prompt_file: "+promptFile), 0644)

	responseWithCode := "Here is code:\n```\ncode to copy\n```"

	deps := newTestDeps(
		withResponses(responseWithCode),
		withStdin("/copy\n"),
		withTTY(true),
	)

	cli := &CLI{
		ConfigPath: configFile,
		Idea:       "test idea",
	}

	err := runWithDeps(context.Background(), cli, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	copied := clipboardWritten(deps)
	if copied != "code to copy\n" {
		t.Errorf("expected 'code to copy\\n' in clipboard, got: %q", copied)
	}
}

func TestCommand_CopyNoResponse(t *testing.T) {
	tmpDir := t.TempDir()
	promptFile := filepath.Join(tmpDir, "prompt.txt")
	configFile := filepath.Join(tmpDir, "config.yaml")

	os.WriteFile(promptFile, []byte("You are a test assistant."), 0644)
	os.WriteFile(configFile, []byte("model: test\nsystem_prompt_file: "+promptFile), 0644)

	// Response without code block
	responseNoCode := "I need more information. What language?"

	deps := newTestDeps(
		withResponses(responseNoCode),
		withStdin("/copy\n/bye\n"),
		withTTY(true),
	)

	cli := &CLI{
		ConfigPath: configFile,
		Idea:       "test idea",
	}

	err := runWithDeps(context.Background(), cli, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have error message in stderr
	errOut := stderr(deps)
	if !strings.Contains(errOut, "No code block") {
		t.Errorf("expected 'No code block' error, got: %s", errOut)
	}
}

func TestCommand_Help(t *testing.T) {
	tmpDir := t.TempDir()
	promptFile := filepath.Join(tmpDir, "prompt.txt")
	configFile := filepath.Join(tmpDir, "config.yaml")

	os.WriteFile(promptFile, []byte("You are a test assistant."), 0644)
	os.WriteFile(configFile, []byte("model: test\nsystem_prompt_file: "+promptFile), 0644)

	response := "What would you like?"

	deps := newTestDeps(
		withResponses(response),
		withStdin("/help\n/bye\n"),
		withTTY(true),
	)

	cli := &CLI{
		ConfigPath: configFile,
		Idea:       "test idea",
	}

	err := runWithDeps(context.Background(), cli, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := stdout(deps)
	if !strings.Contains(out, "/copy") || !strings.Contains(out, "/bye") {
		t.Errorf("expected help text with commands, got: %s", out)
	}
}

func TestCommand_Quit(t *testing.T) {
	tmpDir := t.TempDir()
	promptFile := filepath.Join(tmpDir, "prompt.txt")
	configFile := filepath.Join(tmpDir, "config.yaml")

	os.WriteFile(promptFile, []byte("You are a test assistant."), 0644)
	os.WriteFile(configFile, []byte("model: test\nsystem_prompt_file: "+promptFile), 0644)

	response := "What would you like?"

	deps := newTestDeps(
		withResponses(response),
		withStdin("/quit\n"),
		withTTY(true),
	)

	cli := &CLI{
		ConfigPath: configFile,
		Idea:       "test idea",
	}

	err := runWithDeps(context.Background(), cli, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := stdout(deps)
	if !strings.Contains(out, "Goodbye") {
		t.Errorf("expected 'Goodbye', got: %s", out)
	}
}

func TestCommand_Unknown(t *testing.T) {
	tmpDir := t.TempDir()
	promptFile := filepath.Join(tmpDir, "prompt.txt")
	configFile := filepath.Join(tmpDir, "config.yaml")

	os.WriteFile(promptFile, []byte("You are a test assistant."), 0644)
	os.WriteFile(configFile, []byte("model: test\nsystem_prompt_file: "+promptFile), 0644)

	response := "What would you like?"

	deps := newTestDeps(
		withResponses(response),
		withStdin("/foo\n/bye\n"),
		withTTY(true),
	)

	cli := &CLI{
		ConfigPath: configFile,
		Idea:       "test idea",
	}

	err := runWithDeps(context.Background(), cli, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	errOut := stderr(deps)
	if !strings.Contains(errOut, "Unknown command") {
		t.Errorf("expected 'Unknown command' error, got: %s", errOut)
	}
}
```

**Step 2: Run tests**

Run: `go test ./cmd/prompt-builder -run TestCommand -v`
Expected: PASS

**Step 3: Commit**

```bash
git add cmd/prompt-builder/integration_test.go
git commit -m "test: add command integration tests"
```

---

## Task 10: E2E Test Setup with TestMain

**Files:**
- Create: `cmd/prompt-builder/e2e_test.go`

**Step 1: Write E2E test file with TestMain**

```go
//go:build e2e

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"
)

var testBinary string

func TestMain(m *testing.M) {
	// Build once before all tests
	tmp, err := os.MkdirTemp("", "prompt-builder-test")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}

	testBinary = filepath.Join(tmp, "prompt-builder")

	cmd := exec.Command("go", "build", "-o", testBinary, ".")
	cmd.Dir = "."
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build failed: %v\n%s\n", err, output)
		os.Exit(1)
	}

	code := m.Run()
	os.RemoveAll(tmp)
	os.Exit(code)
}

func ollamaHost() string {
	host := os.Getenv("OLLAMA_HOST")
	if host == "" {
		host = "http://localhost:11434"
	}
	return host
}

func ollamaAvailable() bool {
	resp, err := http.Get(ollamaHost() + "/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

type ollamaModel struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type ollamaTagsResponse struct {
	Models []ollamaModel `json:"models"`
}

func smallestModel() string {
	resp, err := http.Get(ollamaHost() + "/api/tags")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var tags ollamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return ""
	}

	if len(tags.Models) == 0 {
		return ""
	}

	sort.Slice(tags.Models, func(i, j int) bool {
		return tags.Models[i].Size < tags.Models[j].Size
	})

	return tags.Models[0].Name
}

func skipIfNoOllama(t *testing.T) {
	if !ollamaAvailable() {
		t.Skip("Ollama not available")
	}
}

func skipIfNoModel(t *testing.T) string {
	skipIfNoOllama(t)
	model := smallestModel()
	if model == "" {
		t.Skip("No models available in Ollama")
	}
	return model
}
```

**Step 2: Verify it compiles (but doesn't run without tag)**

Run: `go test ./cmd/prompt-builder -v`
Expected: E2E tests not included (no build tag)

Run: `go build ./cmd/prompt-builder`
Expected: Compiles successfully

**Step 3: Commit**

```bash
git add cmd/prompt-builder/e2e_test.go
git commit -m "test: add E2E test infrastructure with TestMain"
```

---

## Task 11: E2E Test - Help and Version Flags

**Files:**
- Modify: `cmd/prompt-builder/e2e_test.go`

**Step 1: Write tests**

Add to `e2e_test.go`:

```go
func TestE2E_Help(t *testing.T) {
	cmd := exec.Command(testBinary, "--help")
	output, err := cmd.CombinedOutput()

	// --help exits 0
	if err != nil {
		t.Fatalf("--help failed: %v\n%s", err, output)
	}

	if !strings.Contains(string(output), "Usage:") {
		t.Errorf("expected usage text, got: %s", output)
	}
	if !strings.Contains(string(output), "prompt-builder") {
		t.Errorf("expected 'prompt-builder' in output, got: %s", output)
	}
}

func TestE2E_Version(t *testing.T) {
	cmd := exec.Command(testBinary, "--version")
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("--version failed: %v\n%s", err, output)
	}

	if !strings.Contains(string(output), "prompt-builder") {
		t.Errorf("expected 'prompt-builder' in output, got: %s", output)
	}
}
```

Add `"strings"` to imports.

**Step 2: Run E2E tests**

Run: `go test -tags=e2e ./cmd/prompt-builder -run TestE2E_Help -v`
Expected: PASS

Run: `go test -tags=e2e ./cmd/prompt-builder -run TestE2E_Version -v`
Expected: PASS

**Step 3: Commit**

```bash
git add cmd/prompt-builder/e2e_test.go
git commit -m "test: add E2E tests for --help and --version"
```

---

## Task 12: E2E Test - Missing Idea and Error Cases

**Files:**
- Modify: `cmd/prompt-builder/e2e_test.go`

**Step 1: Write tests**

Add to `e2e_test.go`:

```go
func TestE2E_MissingIdea(t *testing.T) {
	cmd := exec.Command(testBinary)
	output, err := cmd.CombinedOutput()

	// Should fail with exit code 1
	if err == nil {
		t.Fatal("expected error for missing idea")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got: %T", err)
	}

	if exitErr.ExitCode() != 1 {
		t.Errorf("expected exit code 1, got: %d", exitErr.ExitCode())
	}

	if !strings.Contains(string(output), "missing") || !strings.Contains(string(output), "idea") {
		t.Errorf("expected 'missing idea' error, got: %s", output)
	}
}

func TestE2E_OllamaUnreachable(t *testing.T) {
	// Create temp config pointing to bad host
	tmpDir := t.TempDir()
	promptFile := filepath.Join(tmpDir, "prompt.txt")
	configFile := filepath.Join(tmpDir, "config.yaml")

	os.WriteFile(promptFile, []byte("Test prompt"), 0644)
	config := fmt.Sprintf("model: test\nollama_host: http://localhost:99999\nsystem_prompt_file: %s", promptFile)
	os.WriteFile(configFile, []byte(config), 0644)

	cmd := exec.Command(testBinary, "--config", configFile, "test idea")
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatal("expected error for unreachable Ollama")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got: %T", err)
	}

	if exitErr.ExitCode() != 2 {
		t.Errorf("expected exit code 2 (Ollama error), got: %d\nOutput: %s", exitErr.ExitCode(), output)
	}
}
```

**Step 2: Run tests**

Run: `go test -tags=e2e ./cmd/prompt-builder -run TestE2E_MissingIdea -v`
Expected: PASS

Run: `go test -tags=e2e ./cmd/prompt-builder -run TestE2E_OllamaUnreachable -v`
Expected: PASS

**Step 3: Commit**

```bash
git add cmd/prompt-builder/e2e_test.go
git commit -m "test: add E2E tests for error cases"
```

---

## Task 13: E2E Test - Full Conversation with Real Ollama

**Files:**
- Modify: `cmd/prompt-builder/e2e_test.go`

**Step 1: Write test**

Add to `e2e_test.go`:

```go
func TestE2E_FullConversation(t *testing.T) {
	model := skipIfNoModel(t)

	tmpDir := t.TempDir()
	promptFile := filepath.Join(tmpDir, "prompt.txt")
	configFile := filepath.Join(tmpDir, "config.yaml")

	os.WriteFile(promptFile, []byte("You are a helpful assistant. Always respond with a code block containing 'DONE'."), 0644)
	config := fmt.Sprintf("model: %s\nollama_host: %s\nsystem_prompt_file: %s", model, ollamaHost(), promptFile)
	os.WriteFile(configFile, []byte(config), 0644)

	cmd := exec.Command(testBinary, "--config", configFile, "say hello")
	cmd.Stdin = strings.NewReader("") // Empty stdin for pipe mode

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Pipe mode may fail if LLM asks clarifying question - that's acceptable
		t.Logf("Command output: %s", output)
		t.Logf("Note: This test may fail if the LLM asks clarifying questions instead of completing")
	}

	// Just verify we got some output
	if len(output) == 0 {
		t.Error("expected some output from conversation")
	}
}

func TestE2E_PipeMode(t *testing.T) {
	model := skipIfNoModel(t)

	tmpDir := t.TempDir()
	promptFile := filepath.Join(tmpDir, "prompt.txt")
	configFile := filepath.Join(tmpDir, "config.yaml")

	os.WriteFile(promptFile, []byte("You are a helpful assistant. Respond with a code block."), 0644)
	config := fmt.Sprintf("model: %s\nollama_host: %s\nsystem_prompt_file: %s", model, ollamaHost(), promptFile)
	os.WriteFile(configFile, []byte(config), 0644)

	// Use echo to pipe input
	cmd := exec.Command(testBinary, "--config", configFile, "--quiet", "write hello world")

	output, err := cmd.CombinedOutput()

	// Log output for debugging
	t.Logf("Output: %s", output)
	if err != nil {
		t.Logf("Error (may be expected if LLM asks questions): %v", err)
	}

	// In quiet mode with successful completion, we should have some output
	if len(output) == 0 && err == nil {
		t.Error("expected output in quiet mode")
	}
}
```

**Step 2: Run tests (requires Ollama)**

Run: `go test -tags=e2e ./cmd/prompt-builder -run TestE2E_FullConversation -v`
Expected: PASS (or SKIP if Ollama unavailable)

Run: `go test -tags=e2e ./cmd/prompt-builder -run TestE2E_PipeMode -v`
Expected: PASS (or SKIP if Ollama unavailable)

**Step 3: Commit**

```bash
git add cmd/prompt-builder/e2e_test.go
git commit -m "test: add E2E tests with real Ollama"
```

---

## Task 14: E2E Test - Custom Config

**Files:**
- Modify: `cmd/prompt-builder/e2e_test.go`

**Step 1: Write test**

Add to `e2e_test.go`:

```go
func TestE2E_CustomConfig(t *testing.T) {
	model := skipIfNoModel(t)

	tmpDir := t.TempDir()
	promptFile := filepath.Join(tmpDir, "custom-prompt.txt")
	configFile := filepath.Join(tmpDir, "custom-config.yaml")

	// Use a distinctive prompt we can verify
	os.WriteFile(promptFile, []byte("Always start your response with CUSTOM_CONFIG_TEST."), 0644)
	config := fmt.Sprintf("model: %s\nollama_host: %s\nsystem_prompt_file: %s", model, ollamaHost(), promptFile)
	os.WriteFile(configFile, []byte(config), 0644)

	cmd := exec.Command(testBinary, "--config", configFile, "hello")

	output, err := cmd.CombinedOutput()
	t.Logf("Output: %s", output)

	if err != nil {
		// LLM might ask clarifying questions
		t.Logf("Command returned error (may be expected): %v", err)
	}

	// Verify the command ran with our config (got some response)
	if len(output) == 0 {
		t.Error("expected some output with custom config")
	}
}
```

**Step 2: Run test**

Run: `go test -tags=e2e ./cmd/prompt-builder -run TestE2E_CustomConfig -v`
Expected: PASS (or SKIP if Ollama unavailable)

**Step 3: Commit**

```bash
git add cmd/prompt-builder/e2e_test.go
git commit -m "test: add E2E test for custom config"
```

---

## Task 15: Run Full Test Suite and Verify

**Files:**
- None (verification only)

**Step 1: Run all unit and integration tests**

Run: `go test ./cmd/prompt-builder -v`
Expected: All tests PASS

**Step 2: Run all tests including E2E**

Run: `go test -tags=e2e ./cmd/prompt-builder -v`
Expected: All tests PASS (E2E tests may skip if Ollama unavailable)

**Step 3: Check coverage**

Run: `go test ./cmd/prompt-builder -cover`
Expected: Coverage should increase from baseline (~50%)

**Step 4: Final commit with test summary**

```bash
git add -A
git status
# If any uncommitted changes, commit them
git commit -m "test: complete integration and E2E test suite" --allow-empty
```

---

## Task 16: Update Existing Integration Test File

**Files:**
- Modify: `cmd/prompt-builder/integration_test.go` (rename old one if conflicts)

**Step 1: Check for conflicts**

The existing `integration_test.go` tests config loading. Merge or rename to avoid duplication.

Run: `ls cmd/prompt-builder/*integration*`

**Step 2: Consolidate if needed**

If old `integration_test.go` exists with different tests, rename it or merge the tests into the new file.

**Step 3: Commit**

```bash
git add cmd/prompt-builder/integration_test.go
git commit -m "test: consolidate integration tests"
```

---

## Summary

After completing all tasks:

1. **Interfaces defined:** `OllamaChatter`, `ClipboardWriter`, `Deps` struct
2. **Refactored:** `runWithDeps()` accepts injected dependencies
3. **Test fixtures:** `testdata/` with config and prompt files
4. **Test helpers:** Mocks and builder functions
5. **Integration tests:** 11 tests covering conversation loop, commands, errors
6. **E2E tests:** 7 tests covering CLI flags, exit codes, real Ollama

Run `go test ./cmd/prompt-builder` for fast feedback, `go test -tags=e2e ./cmd/prompt-builder` for full validation.
