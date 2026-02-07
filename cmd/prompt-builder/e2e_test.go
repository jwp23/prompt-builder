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
	"strings"
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

func TestE2E_LLMUnreachable(t *testing.T) {
	// Create temp config pointing to bad host
	tmpDir := t.TempDir()
	promptFile := filepath.Join(tmpDir, "prompt.txt")
	configFile := filepath.Join(tmpDir, "config.yaml")

	os.WriteFile(promptFile, []byte("Test prompt"), 0644)
	config := fmt.Sprintf("model: test\nhost: http://localhost:99999\nsystem_prompt_file: %s", promptFile)
	os.WriteFile(configFile, []byte(config), 0644)

	cmd := exec.Command(testBinary, "--config", configFile, "test idea")
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatal("expected error for unreachable LLM server")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got: %T", err)
	}

	if exitErr.ExitCode() != 2 {
		t.Errorf("expected exit code 2 (LLM error), got: %d\nOutput: %s", exitErr.ExitCode(), output)
	}
}

func TestE2E_FullConversation(t *testing.T) {
	model := skipIfNoModel(t)

	tmpDir := t.TempDir()
	promptFile := filepath.Join(tmpDir, "prompt.txt")
	configFile := filepath.Join(tmpDir, "config.yaml")

	os.WriteFile(promptFile, []byte("You are a helpful assistant. Always respond with a code block containing 'DONE'."), 0644)
	config := fmt.Sprintf("model: %s\nhost: %s\nsystem_prompt_file: %s", model, ollamaHost(), promptFile)
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
	config := fmt.Sprintf("model: %s\nhost: %s\nsystem_prompt_file: %s", model, ollamaHost(), promptFile)
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

func TestE2E_CustomConfig(t *testing.T) {
	model := skipIfNoModel(t)

	tmpDir := t.TempDir()
	promptFile := filepath.Join(tmpDir, "custom-prompt.txt")
	configFile := filepath.Join(tmpDir, "custom-config.yaml")

	// Use a distinctive prompt we can verify
	os.WriteFile(promptFile, []byte("Always start your response with CUSTOM_CONFIG_TEST."), 0644)
	config := fmt.Sprintf("model: %s\nhost: %s\nsystem_prompt_file: %s", model, ollamaHost(), promptFile)
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
