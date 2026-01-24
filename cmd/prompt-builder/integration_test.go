// integration_test.go
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIntegration_ConfigLoading(t *testing.T) {
	// Create temp directory with config
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	promptPath := filepath.Join(dir, "prompt.md")

	configContent := `model: llama3.2
system_prompt_file: ` + promptPath + `
ollama_host: http://localhost:11434
`
	promptContent := `# Test System Prompt
You are a test assistant.
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(promptPath, []byte(promptContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Load config
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify config values
	if cfg.Model != "llama3.2" {
		t.Errorf("Model = %q, want %q", cfg.Model, "llama3.2")
	}

	// Load system prompt
	prompt, err := os.ReadFile(cfg.SystemPromptFile)
	if err != nil {
		t.Fatalf("failed to load system prompt: %v", err)
	}

	if len(prompt) == 0 {
		t.Error("system prompt is empty")
	}
}
