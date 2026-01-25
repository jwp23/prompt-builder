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
			name:  "single code block",
			input: "Here is your prompt:\n```\n# Role\nYou are an expert.\n```\n",
			want:  "# Role\nYou are an expert.\n",
		},
		{
			name:  "multiple code blocks - returns last",
			input: "Example:\n```\nfirst block\n```\n\nHere is the final:\n```\nsecond block\n```\n",
			want:  "second block\n",
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
