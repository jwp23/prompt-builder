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
