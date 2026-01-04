// commands_test.go
package main

import (
	"bytes"
	"testing"
)

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
