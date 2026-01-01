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
ollama_host: http://localhost:11434
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
	if cfg.OllamaHost != "http://localhost:11434" {
		t.Errorf("OllamaHost = %q, want %q", cfg.OllamaHost, "http://localhost:11434")
	}
	if cfg.ClipboardCmd != "wl-copy" {
		t.Errorf("ClipboardCmd = %q, want %q", cfg.ClipboardCmd, "wl-copy")
	}
}
