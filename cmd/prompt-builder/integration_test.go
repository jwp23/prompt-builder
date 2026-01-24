// integration_test.go
package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
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

func TestRun_MultiTurnConversation(t *testing.T) {
	tmpDir := t.TempDir()
	promptFile := filepath.Join(tmpDir, "prompt.txt")
	configFile := filepath.Join(tmpDir, "config.yaml")

	os.WriteFile(promptFile, []byte("You are a test assistant."), 0644)
	os.WriteFile(configFile, []byte("model: test\nsystem_prompt_file: "+promptFile), 0644)

	// First response asks a question, second completes
	clarifyingResponse := "What language would you like the prompt in?"

	deps := newTestDeps(
		withResponses(clarifyingResponse),
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
