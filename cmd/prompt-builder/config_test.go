// config_test.go
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_ValidFile(t *testing.T) {
	// Create temp config file
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := `model: llama3.2
system_prompt_file: /path/to/prompt.md
host: http://localhost:11434
clipboard_cmd: wl-copy
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Model != "llama3.2" {
		t.Errorf("Model = %q, want %q", cfg.Model, "llama3.2")
	}
	if cfg.SystemPromptFile != "/path/to/prompt.md" {
		t.Errorf("SystemPromptFile = %q, want %q", cfg.SystemPromptFile, "/path/to/prompt.md")
	}
	if cfg.Host != "http://localhost:11434" {
		t.Errorf("Host = %q, want %q", cfg.Host, "http://localhost:11434")
	}
	if cfg.ClipboardCmd != "wl-copy" {
		t.Errorf("ClipboardCmd = %q, want %q", cfg.ClipboardCmd, "wl-copy")
	}
}

func TestLoadConfig_AppliesDefaults(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := `model: llama3.2
system_prompt_file: /path/to/prompt.md
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Host != "http://localhost:11434" {
		t.Errorf("Host = %q, want default %q", cfg.Host, "http://localhost:11434")
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestExpandPath_Tilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		input string
		want  string
	}{
		{"~/config.yaml", filepath.Join(home, "config.yaml")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}

	for _, tt := range tests {
		got := ExpandPath(tt.input)
		if got != tt.want {
			t.Errorf("ExpandPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
