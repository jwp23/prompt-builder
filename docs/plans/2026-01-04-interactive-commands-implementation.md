# Interactive Commands Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add slash commands (`/copy`, `/bye`, `/quit`, `/exit`, `/help`) to the interactive conversation loop.

**Architecture:** Commands are detected by leading `/` and handled before normal input processing. Command handling is isolated in `commands.go` with a simple dispatch pattern. Exit commands return `shouldExit=true`; others return `false` and print output.

**Tech Stack:** Go stdlib only (strings, fmt, errors)

**Design Doc:** `docs/plans/2026-01-04-interactive-commands-design.md`

---

## Task 1: IsCommand Function

**Files:**
- Create: `commands.go`
- Create: `commands_test.go`

**Step 1: Write the failing test for IsCommand**

Create `commands_test.go`:

```go
// commands_test.go
package main

import "testing"

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
```

**Step 2: Run test to verify it fails**

Run: `go test -run TestIsCommand -v`

Expected: FAIL - `undefined: IsCommand`

**Step 3: Write minimal implementation**

Create `commands.go`:

```go
// commands.go
package main

import "strings"

// IsCommand returns true if input starts with a slash.
func IsCommand(input string) bool {
	return strings.HasPrefix(input, "/")
}
```

**Step 4: Run test to verify it passes**

Run: `go test -run TestIsCommand -v`

Expected: PASS

**Step 5: Commit**

```bash
git add commands.go commands_test.go
git commit -m "feat: add IsCommand to detect slash commands"
```

---

## Task 2: Case-Insensitive Command Matching

**Files:**
- Modify: `commands_test.go`
- Modify: `commands.go`

**Step 1: Write the failing test for case-insensitive parsing**

Add to `commands_test.go`:

```go
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
```

**Step 2: Run test to verify it fails**

Run: `go test -run TestParseCommand -v`

Expected: FAIL - `undefined: parseCommand`

**Step 3: Write minimal implementation**

Add to `commands.go`:

```go
// parseCommand extracts the command name (lowercase, no slash) from input.
func parseCommand(input string) string {
	trimmed := strings.TrimSpace(input)
	if !strings.HasPrefix(trimmed, "/") {
		return ""
	}
	cmd := strings.TrimPrefix(trimmed, "/")
	return strings.ToLower(cmd)
}
```

**Step 4: Run test to verify it passes**

Run: `go test -run TestParseCommand -v`

Expected: PASS

**Step 5: Commit**

```bash
git add commands.go commands_test.go
git commit -m "feat: add parseCommand for case-insensitive matching"
```

---

## Task 3: Exit Commands (/bye, /quit, /exit)

**Files:**
- Modify: `commands_test.go`
- Modify: `commands.go`

**Step 1: Write the failing test for exit commands**

Add to `commands_test.go`:

```go
import (
	"bytes"
	"testing"
)

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
```

**Step 2: Run test to verify it fails**

Run: `go test -run TestHandleCommand_Exit -v`

Expected: FAIL - `undefined: HandleCommand`

**Step 3: Write minimal implementation**

Add to `commands.go`:

```go
import (
	"fmt"
	"io"
	"strings"
)

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
```

**Step 4: Run test to verify it passes**

Run: `go test -run TestHandleCommand_Exit -v`

Expected: PASS

**Step 5: Commit**

```bash
git add commands.go commands_test.go
git commit -m "feat: add exit commands /bye, /quit, /exit"
```

---

## Task 4: Unknown Command Error

**Files:**
- Modify: `commands_test.go`
- Modify: `commands.go`

**Step 1: Write the failing test for unknown commands**

Add to `commands_test.go`:

```go
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
```

**Step 2: Run test to verify it fails**

Run: `go test -run TestHandleCommand_Unknown -v`

Expected: FAIL - assertion error (currently returns no error)

**Step 3: Update implementation**

Modify the `default` case in `HandleCommand`:

```go
	default:
		return false, fmt.Errorf("Unknown command: /%s. Type /help for available commands.", cmd)
```

**Step 4: Run test to verify it passes**

Run: `go test -run TestHandleCommand_Unknown -v`

Expected: PASS

**Step 5: Commit**

```bash
git add commands.go commands_test.go
git commit -m "feat: return error for unknown commands"
```

---

## Task 5: Help Command

**Files:**
- Modify: `commands_test.go`
- Modify: `commands.go`

**Step 1: Write the failing test for /help**

Add to `commands_test.go`:

```go
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
  /copy   Copy last code block to clipboard
  /bye    Exit conversation
  /quit   Exit conversation
  /exit   Exit conversation
  /help   Show this help
`
	if out.String() != wantOutput {
		t.Errorf("HandleCommand() output = %q, want %q", out.String(), wantOutput)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -run TestHandleCommand_Help -v`

Expected: FAIL - returns unknown command error

**Step 3: Add help case to HandleCommand**

Add before the `default` case:

```go
	case "help":
		fmt.Fprintln(out, `Commands:
  /copy   Copy last code block to clipboard
  /bye    Exit conversation
  /quit   Exit conversation
  /exit   Exit conversation
  /help   Show this help`)
		return false, nil
```

**Step 4: Run test to verify it passes**

Run: `go test -run TestHandleCommand_Help -v`

Expected: PASS

**Step 5: Commit**

```bash
git add commands.go commands_test.go
git commit -m "feat: add /help command"
```

---

## Task 6: Copy Command - Success Case

**Files:**
- Modify: `commands_test.go`
- Modify: `commands.go`

**Step 1: Write the failing test for /copy success**

Add to `commands_test.go`:

```go
func TestHandleCommand_Copy_Success(t *testing.T) {
	lastResponse := "Here is your code:\n```\nfmt.Println(\"hello\")\n```\n"

	var out bytes.Buffer
	// Use "echo" as a mock clipboard command that accepts stdin
	shouldExit, err := HandleCommand("/copy", lastResponse, "cat > /dev/null", &out)

	if err != nil {
		t.Errorf("HandleCommand() error = %v", err)
	}
	if shouldExit {
		t.Error("HandleCommand() should not exit on /copy")
	}
	wantOutput := "\u2713 Copied to clipboard\n"
	if out.String() != wantOutput {
		t.Errorf("HandleCommand() output = %q, want %q", out.String(), wantOutput)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -run TestHandleCommand_Copy_Success -v`

Expected: FAIL - returns unknown command error

**Step 3: Add copy case to HandleCommand**

Add before the `help` case:

```go
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
		return false, nil
```

**Step 4: Run test to verify it passes**

Run: `go test -run TestHandleCommand_Copy_Success -v`

Expected: PASS

**Step 5: Commit**

```bash
git add commands.go commands_test.go
git commit -m "feat: add /copy command success case"
```

---

## Task 7: Copy Command - Error Cases

**Files:**
- Modify: `commands_test.go`

**Step 1: Write the failing tests for /copy error cases**

Add to `commands_test.go`:

```go
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

**Step 2: Run tests to verify they pass**

Run: `go test -run 'TestHandleCommand_Copy_No' -v`

Expected: PASS (implementation from Task 6 already handles these)

**Step 3: Commit**

```bash
git add commands_test.go
git commit -m "test: add /copy error case tests"
```

---

## Task 8: Integrate Commands into main.go

**Files:**
- Modify: `main.go:182-188`

**Step 1: Understand the integration point**

The command handling goes after reading user input (line 183) and before adding to conversation (line 188).

**Step 2: Add command handling to the conversation loop**

Replace lines 183-188 in `main.go`:

```go
		userInput, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %v", err)
		}

		conv.AddUserMessage(userInput)
```

With:

```go
		userInput, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %v", err)
		}

		userInput = strings.TrimSpace(userInput)

		if IsCommand(userInput) {
			shouldExit, err := HandleCommand(userInput, response, clipboardCmd, os.Stdout)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
			if shouldExit {
				return nil
			}
			fmt.Print("> ")
			continue
		}

		conv.AddUserMessage(userInput)
```

**Step 3: Run the build to verify it compiles**

Run: `go build`

Expected: Success

**Step 4: Run all tests**

Run: `go test -v`

Expected: All tests pass

**Step 5: Commit**

```bash
git add main.go
git commit -m "feat: integrate slash commands into conversation loop"
```

---

## Task 9: Manual Integration Test

**Step 1: Build the binary**

Run: `go build`

**Step 2: Test /help command**

Run the program and test:
```
> /help
Commands:
  /copy   Copy last code block to clipboard
  /bye    Exit conversation
  /quit   Exit conversation
  /exit   Exit conversation
  /help   Show this help
>
```

**Step 3: Test /bye command**

```
> /bye
Goodbye
(program exits)
```

**Step 4: Test unknown command**

```
> /foo
Unknown command: /foo. Type /help for available commands.
>
```

**Step 5: Test /copy (after getting a response with code block)**

```
> /copy
✓ Copied to clipboard
>
```

**Step 6: Commit any fixes if needed**

---

## Task 10: Final Cleanup and Squash (Optional)

**Step 1: Review git log**

Run: `git log --oneline -10`

**Step 2: If desired, squash commits for PR**

This is optional - discuss with Joe whether to keep granular commits or squash.

---

## Summary

| Task | Description | Files |
|------|-------------|-------|
| 1 | IsCommand function | commands.go, commands_test.go |
| 2 | Case-insensitive parsing | commands.go, commands_test.go |
| 3 | Exit commands | commands.go, commands_test.go |
| 4 | Unknown command error | commands.go, commands_test.go |
| 5 | Help command | commands.go, commands_test.go |
| 6 | Copy command success | commands.go, commands_test.go |
| 7 | Copy command errors | commands_test.go |
| 8 | main.go integration | main.go |
| 9 | Manual testing | - |
| 10 | Final cleanup | - |
