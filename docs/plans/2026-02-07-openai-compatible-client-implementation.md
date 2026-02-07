# OpenAI-Compatible Client Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the Ollama-specific HTTP client with one that speaks the OpenAI chat completions protocol, enabling any compatible backend (llama.cpp, Ollama, LM Studio, vLLM).

**Architecture:** Rename `ollama.go` to `llm.go`. Replace the Ollama streaming format (newline-delimited JSON with `done: true`) with SSE parsing (`data: ` prefix, `[DONE]` sentinel, `choices[0].delta.content`). Rename types and interfaces to be backend-agnostic. Update config to use `host` instead of `ollama_host`. Delete dead `IsModelLoaded` code. The endpoint changes from `/api/chat` to `/v1/chat/completions`.

**Tech Stack:** Go stdlib (`net/http`, `encoding/json`, `bufio`), `gopkg.in/yaml.v3`

---

### Task 1: Rename ollama.go to llm.go and ollama_test.go to llm_test.go

This is a pure file rename with no content changes. Keeps git history clean by separating renames from edits.

**Step 1: Rename files**

```bash
cd /home/mordant23/workspace/jwp23/prompt-builder
git mv cmd/prompt-builder/ollama.go cmd/prompt-builder/llm.go
git mv cmd/prompt-builder/ollama_test.go cmd/prompt-builder/llm_test.go
```

**Step 2: Run tests to verify nothing broke**

Run: `go test ./cmd/prompt-builder`
Expected: All tests pass (rename only, no content changes)

**Step 3: Commit**

```bash
git add cmd/prompt-builder/llm.go cmd/prompt-builder/llm_test.go
git commit -m "refactor: rename ollama.go to llm.go for backend-agnostic naming"
```

---

### Task 2: Delete IsModelLoaded and its support types

Remove dead code: `IsModelLoaded()`, `OllamaPsModel`, `OllamaPsResponse`, and 5 related tests.

**Files:**
- Modify: `cmd/prompt-builder/llm.go` (delete lines 38-44, 133-155)
- Modify: `cmd/prompt-builder/llm_test.go` (delete lines 210-285)

**Step 1: Delete types from llm.go**

Delete `OllamaPsModel` and `OllamaPsResponse` structs:

```go
// DELETE these lines from llm.go:
type OllamaPsModel struct {
	Name string `json:"name"`
}

type OllamaPsResponse struct {
	Models []OllamaPsModel `json:"models"`
}
```

**Step 2: Delete IsModelLoaded method from llm.go**

Delete the entire `IsModelLoaded` method (currently lines 133-155).

**Step 3: Delete IsModelLoaded tests from llm_test.go**

Delete these 5 test functions from llm_test.go:
- `TestOllamaClient_IsModelLoaded_True`
- `TestOllamaClient_IsModelLoaded_False`
- `TestOllamaClient_IsModelLoaded_EmptyList`
- `TestOllamaClient_IsModelLoaded_Error`
- `TestOllamaClient_IsModelLoaded_ConnectionRefused`

**Step 4: Run tests**

Run: `go test ./cmd/prompt-builder`
Expected: All remaining tests pass

**Step 5: Commit**

```bash
git add cmd/prompt-builder/llm.go cmd/prompt-builder/llm_test.go
git commit -m "refactor: remove dead IsModelLoaded code and Ollama-specific types"
```

---

### Task 3: Rename interface and client types

Rename `OllamaChatter` → `LLMClient`, `OllamaClient` → `ChatClient`, `NewOllamaClient` → `NewChatClient`. Update all references.

**Files:**
- Modify: `cmd/prompt-builder/llm.go`
- Modify: `cmd/prompt-builder/llm_test.go`
- Modify: `cmd/prompt-builder/main.go`
- Modify: `cmd/prompt-builder/testhelpers_test.go`
- Modify: `cmd/prompt-builder/main_test.go`

**Step 1: Rename in llm.go**

Change the interface:
```go
// Before:
// OllamaChatter abstracts the Ollama client for testing.
type OllamaChatter interface {

// After:
// LLMClient abstracts the LLM backend for testing.
type LLMClient interface {
```

Change the struct and constructor:
```go
// Before:
type OllamaClient struct {

// After:
type ChatClient struct {
```

```go
// Before:
func NewOllamaClient(host, model string) *OllamaClient {
	return &OllamaClient{

// After:
func NewChatClient(host, model string) *ChatClient {
	return &ChatClient{
```

Update all method receivers from `OllamaClient` to `ChatClient`:
- `func (c *OllamaClient) ChatStream(` → `func (c *ChatClient) ChatStream(`
- `func (c *OllamaClient) ChatStreamWithSpinner(` → `func (c *ChatClient) ChatStreamWithSpinner(`

**Step 2: Rename in main.go**

Update `Deps` struct:
```go
// Before:
	Client       OllamaChatter

// After:
	Client       LLMClient
```

Update `run()` constructor call:
```go
// Before:
		Client:       NewOllamaClient(cfg.OllamaHost, model),

// After:
		Client:       NewChatClient(cfg.OllamaHost, model),
```

Update error message in `runWithDeps()`:
```go
// Before:
		return fmt.Errorf("Ollama request failed: %v", err)

// After:
		return fmt.Errorf("LLM request failed: %v", err)
```

Update exit code error matching in `main()`:
```go
// Before:
		case strings.Contains(errStr, "Ollama") || strings.Contains(errStr, "connect"):
			os.Exit(ExitOllamaError)

// After:
		case strings.Contains(errStr, "LLM") || strings.Contains(errStr, "connect"):
			os.Exit(ExitOllamaError)
```

Also rename the exit code constant:
```go
// Before:
	ExitOllamaError = 2

// After:
	ExitLLMError = 2
```

And update its usage:
```go
// Before:
			os.Exit(ExitOllamaError)

// After:
			os.Exit(ExitLLMError)
```

**Step 3: Rename in llm_test.go**

Update all test function names and references:
- `TestOllamaRequest_Serialization` → `TestChatRequest_Serialization`
- `TestOllamaStreamChunk_Deserialization` → `TestChatStreamChunk_Deserialization`
- `TestOllamaClient_ChatStream_HappyPath` → `TestChatClient_ChatStream_HappyPath`
- `TestOllamaClient_ChatStream_CallbackError` → `TestChatClient_ChatStream_CallbackError`
- `TestOllamaClient_ChatStream_MalformedJSON` → `TestChatClient_ChatStream_MalformedJSON`
- `TestOllamaClient_ChatStream_HTTPError` → `TestChatClient_ChatStream_HTTPError`
- `TestOllamaClient_ChatStreamWithSpinner_StopsOnFirstToken` → `TestChatClient_ChatStreamWithSpinner_StopsOnFirstToken`

Replace all `NewOllamaClient` calls with `NewChatClient` in tests.

**Step 4: Rename in main_test.go**

```go
// Before:
func TestOllamaClient_ImplementsOllamaChatter(t *testing.T) {
	var _ OllamaChatter = (*OllamaClient)(nil)
}

// After:
func TestChatClient_ImplementsLLMClient(t *testing.T) {
	var _ LLMClient = (*ChatClient)(nil)
}
```

**Step 5: Rename in testhelpers_test.go**

Update mock type comment and helper function name:
```go
// Before:
// mockOllama implements OllamaChatter for testing.
type mockOllama struct {

// After:
// mockLLM implements LLMClient for testing.
type mockLLM struct {
```

Update all `mockOllama` references to `mockLLM` throughout testhelpers_test.go.

Rename helper function:
```go
// Before:
func withOllamaError(err error) testOption {

// After:
func withLLMError(err error) testOption {
```

**Step 6: Update integration_test.go**

Update `TestRun_OllamaError` to use new names:
```go
// Before:
func TestRun_OllamaError(t *testing.T) {
	...
	deps := newTestDeps(
		withOllamaError(errors.New("connection refused")),
	...
	if !strings.Contains(err.Error(), "Ollama") {
		t.Errorf("expected Ollama error, got: %v", err)

// After:
func TestRun_LLMError(t *testing.T) {
	...
	deps := newTestDeps(
		withLLMError(errors.New("connection refused")),
	...
	if !strings.Contains(err.Error(), "LLM") {
		t.Errorf("expected LLM error, got: %v", err)
```

**Step 7: Run tests**

Run: `go test ./cmd/prompt-builder`
Expected: All tests pass

**Step 8: Commit**

```bash
git add cmd/prompt-builder/llm.go cmd/prompt-builder/llm_test.go cmd/prompt-builder/main.go cmd/prompt-builder/main_test.go cmd/prompt-builder/testhelpers_test.go cmd/prompt-builder/integration_test.go
git commit -m "refactor: rename Ollama types to backend-agnostic names"
```

---

### Task 4: Rename config field from OllamaHost to Host

**Files:**
- Modify: `cmd/prompt-builder/config.go`
- Modify: `cmd/prompt-builder/config_test.go`
- Modify: `cmd/prompt-builder/main.go`
- Modify: `cmd/prompt-builder/integration_test.go`
- Modify: `cmd/prompt-builder/e2e_test.go`

**Step 1: Write failing test for new config field name**

In `cmd/prompt-builder/config_test.go`, update `TestLoadConfig_ValidFile`:

```go
func TestLoadConfig_ValidFile(t *testing.T) {
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
```

Update `TestLoadConfig_AppliesDefaults`:

```go
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
```

**Step 2: Run tests to verify they fail**

Run: `go test -run TestLoadConfig ./cmd/prompt-builder`
Expected: FAIL — `cfg.Host undefined`

**Step 3: Update config.go**

```go
type Config struct {
	Model            string `yaml:"model"`
	SystemPromptFile string `yaml:"system_prompt_file"`
	Host             string `yaml:"host"`
	ClipboardCmd     string `yaml:"clipboard_cmd"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := Config{
		Host: "http://localhost:11434",
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
```

**Step 4: Update main.go**

```go
// Before:
		Client:       NewChatClient(cfg.OllamaHost, model),

// After:
		Client:       NewChatClient(cfg.Host, model),
```

**Step 5: Update integration_test.go config strings**

Replace all `ollama_host:` with `host:` in test config YAML strings. These appear in:
- `TestIntegration_ConfigLoading` (line 21)

**Step 6: Update e2e_test.go config strings**

Replace all `ollama_host:` with `host:` in test config YAML strings. These appear in:
- `TestE2E_OllamaUnreachable` (line 165)
- `TestE2E_FullConversation` (line 193)
- `TestE2E_PipeMode` (line 220)
- `TestE2E_CustomConfig` (line 249)

Also rename `TestE2E_OllamaUnreachable` to `TestE2E_LLMUnreachable` and update its error message check.

**Step 7: Run tests**

Run: `go test ./cmd/prompt-builder`
Expected: All tests pass

**Step 8: Commit**

```bash
git add cmd/prompt-builder/config.go cmd/prompt-builder/config_test.go cmd/prompt-builder/main.go cmd/prompt-builder/integration_test.go cmd/prompt-builder/e2e_test.go
git commit -m "refactor: rename config field ollama_host to host"
```

---

### Task 5: Replace request/response types with OpenAI format

Replace `OllamaRequest` and `OllamaStreamChunk` with `ChatRequest` and `ChatStreamChunk`.

**Files:**
- Modify: `cmd/prompt-builder/llm.go`
- Modify: `cmd/prompt-builder/llm_test.go`

**Step 1: Write failing test for new ChatRequest serialization**

Replace `TestOllamaRequest_Serialization` (already renamed to `TestChatRequest_Serialization` in Task 3) with:

```go
func TestChatRequest_Serialization(t *testing.T) {
	req := ChatRequest{
		Model: "llama3.2",
		Messages: []Message{
			{Role: "system", Content: "You are helpful."},
			{Role: "user", Content: "Hello"},
		},
		Stream: true,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	s := string(data)
	if !strings.Contains(s, `"model":"llama3.2"`) {
		t.Errorf("missing model field")
	}
	if !strings.Contains(s, `"role":"system"`) {
		t.Errorf("missing system role")
	}
	if !strings.Contains(s, `"stream":true`) {
		t.Errorf("missing stream field")
	}
}
```

**Step 2: Write failing test for ChatStreamChunk deserialization**

Replace `TestOllamaStreamChunk_Deserialization` (already renamed to `TestChatStreamChunk_Deserialization` in Task 3) with:

```go
func TestChatStreamChunk_Deserialization(t *testing.T) {
	tests := []struct {
		name         string
		json         string
		wantContent  string
		wantFinished bool
	}{
		{
			name:         "partial chunk",
			json:         `{"choices":[{"delta":{"content":"Hello"},"finish_reason":null}]}`,
			wantContent:  "Hello",
			wantFinished: false,
		},
		{
			name:         "final chunk",
			json:         `{"choices":[{"delta":{},"finish_reason":"stop"}]}`,
			wantContent:  "",
			wantFinished: true,
		},
		{
			name:         "empty delta",
			json:         `{"choices":[{"delta":{},"finish_reason":null}]}`,
			wantContent:  "",
			wantFinished: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var chunk ChatStreamChunk
			if err := json.Unmarshal([]byte(tt.json), &chunk); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			if len(chunk.Choices) == 0 {
				t.Fatal("expected at least one choice")
			}
			if chunk.Choices[0].Delta.Content != tt.wantContent {
				t.Errorf("content = %q, want %q", chunk.Choices[0].Delta.Content, tt.wantContent)
			}
			finished := chunk.Choices[0].FinishReason != nil && *chunk.Choices[0].FinishReason == "stop"
			if finished != tt.wantFinished {
				t.Errorf("finished = %v, want %v", finished, tt.wantFinished)
			}
		})
	}
}
```

**Step 3: Run tests to verify they fail**

Run: `go test -run "TestChatRequest_Serialization|TestChatStreamChunk_Deserialization" ./cmd/prompt-builder`
Expected: FAIL — `ChatRequest` and `ChatStreamChunk` undefined

**Step 4: Replace types in llm.go**

Delete:
```go
type OllamaRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type OllamaStreamChunk struct {
	Message Message `json:"message"`
	Done    bool    `json:"done"`
}
```

Add:
```go
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type ChatStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}
```

Update the `ChatStream` method to use the new types (this temporarily breaks `ChatStream` — the SSE parsing comes in Task 6):

```go
func (c *ChatClient) ChatStream(messages []Message, onToken StreamCallback) (string, error) {
	req := ChatRequest{
		Model:    c.Model,
		Messages: messages,
		Stream:   true,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.client.Post(c.Host+"/api/chat", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to connect to LLM server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("LLM request failed: %s - %s", resp.Status, string(body))
	}

	var accumulated strings.Builder
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		var chunk OllamaStreamChunk
		if err := json.Unmarshal(scanner.Bytes(), &chunk); err != nil {
			return "", fmt.Errorf("failed to parse streaming chunk: %w", err)
		}

		if err := onToken(chunk.Message.Content); err != nil {
			return "", err
		}

		accumulated.WriteString(chunk.Message.Content)

		if chunk.Done {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading stream: %w", err)
	}

	return accumulated.String(), nil
}
```

Note: This step intentionally leaves `ChatStream` using the old parsing. The type tests pass, but the streaming tests will fail until Task 6 rewrites the parsing. That is expected — we want to verify the types first.

**Step 5: Run type tests**

Run: `go test -run "TestChatRequest_Serialization|TestChatStreamChunk_Deserialization" ./cmd/prompt-builder`
Expected: PASS

**Step 6: Commit**

```bash
git add cmd/prompt-builder/llm.go cmd/prompt-builder/llm_test.go
git commit -m "feat: replace Ollama request/response types with OpenAI chat completions format"
```

---

### Task 6: Rewrite ChatStream for SSE parsing and update test server

Replace the Ollama newline-delimited JSON parsing with SSE parsing. Update the test fake server to emit SSE format. Change the endpoint from `/api/chat` to `/v1/chat/completions`.

**Files:**
- Modify: `cmd/prompt-builder/llm.go`
- Modify: `cmd/prompt-builder/llm_test.go`

**Step 1: Rewrite fakeStreamingServer in llm_test.go**

Replace the existing `fakeStreamingServer` to emit SSE format:

```go
func fakeStreamingServer(chunks []string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		for i, chunk := range chunks {
			isLast := i == len(chunks)-1
			if isLast {
				// Send the final content chunk
				fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":%q},\"finish_reason\":null}]}\n\n", chunk)
				// Send the stop chunk
				fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
				// Send the done sentinel
				fmt.Fprintf(w, "data: [DONE]\n\n")
			} else {
				fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":%q},\"finish_reason\":null}]}\n\n", chunk)
			}
			w.(http.Flusher).Flush()
		}
	}))
}
```

**Step 2: Update HTTP error test**

In `TestChatClient_ChatStream_HTTPError`, update the expected error message:

```go
	if !strings.Contains(err.Error(), "LLM request failed") {
		t.Errorf("unexpected error message: %v", err)
	}
```

**Step 3: Update malformed JSON test**

In `TestChatClient_ChatStream_MalformedJSON`, the server must emit SSE format with bad JSON:

```go
func TestChatClient_ChatStream_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, "data: not valid json")
	}))
	defer server.Close()

	client := NewChatClient(server.URL, "llama3.2")
	messages := []Message{{Role: "user", Content: "Hi"}}

	_, err := client.ChatStream(messages, func(token string) error {
		return nil
	})

	if err == nil {
		t.Error("expected error for malformed JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse streaming chunk") {
		t.Errorf("unexpected error message: %v", err)
	}
}
```

**Step 4: Run tests to verify they fail**

Run: `go test -run "TestChatClient_ChatStream" ./cmd/prompt-builder`
Expected: FAIL — `ChatStream` still uses old Ollama parsing

**Step 5: Rewrite ChatStream in llm.go**

```go
func (c *ChatClient) ChatStream(messages []Message, onToken StreamCallback) (string, error) {
	req := ChatRequest{
		Model:    c.Model,
		Messages: messages,
		Stream:   true,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.client.Post(c.Host+"/v1/chat/completions", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to connect to LLM server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("LLM request failed: %s - %s", resp.Status, string(body))
	}

	var accumulated strings.Builder
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines (SSE delimiter)
		if line == "" {
			continue
		}

		// Strip "data: " prefix
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		// Check for stream end sentinel
		if data == "[DONE]" {
			break
		}

		var chunk ChatStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return "", fmt.Errorf("failed to parse streaming chunk: %w", err)
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		content := chunk.Choices[0].Delta.Content
		if content != "" {
			if err := onToken(content); err != nil {
				return "", err
			}
			accumulated.WriteString(content)
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading stream: %w", err)
	}

	return accumulated.String(), nil
}
```

**Step 6: Run all tests**

Run: `go test ./cmd/prompt-builder`
Expected: All tests pass

**Step 7: Commit**

```bash
git add cmd/prompt-builder/llm.go cmd/prompt-builder/llm_test.go
git commit -m "feat: rewrite ChatStream for OpenAI SSE streaming protocol"
```

---

### Task 7: Update E2E tests for new config and error messages

The E2E tests use `ollama_host` in config strings and check for "Ollama" in error output. Update them for the new names.

**Files:**
- Modify: `cmd/prompt-builder/e2e_test.go`

Note: Some of these changes may already be done in Task 4. This task catches any remaining E2E references and verifies the E2E tests still work with Ollama's OpenAI-compatible endpoint.

**Step 1: Update ollamaAvailable helper**

The `ollamaAvailable` function hits `/api/tags` which is Ollama-specific. Since E2E tests still need Ollama running to test against, keep the helper but note it's Ollama-specific:

```go
// ollamaAvailable checks if Ollama is running (used for E2E test skipping).
func ollamaAvailable() bool {
	resp, err := http.Get(ollamaHost() + "/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}
```

No change needed — this is fine as-is for E2E skip logic.

**Step 2: Verify E2E_LLMUnreachable checks correct error output**

The exit code check should still work since the error string from `ChatStream` now contains "LLM" and "connect", matching the exit code logic in `main()`.

Run: `go test ./cmd/prompt-builder` (unit + integration only, no E2E tag)
Expected: All pass

**Step 3: Run E2E tests if Ollama is available**

Run: `go test -tags=e2e ./cmd/prompt-builder -v -run TestE2E`
Expected: Tests pass (or skip if Ollama unavailable). The key validation is that `TestE2E_LLMUnreachable` exits with code 2.

**Step 4: Commit (if any changes were needed)**

```bash
git add cmd/prompt-builder/e2e_test.go
git commit -m "test: update E2E tests for backend-agnostic naming"
```

---

### Task 8: Update user's config file and tag v2.0.0

**Step 1: Update the user's config file**

The user's config at `~/.config/prompt-builder/config.yaml` needs `ollama_host` renamed to `host`. Check the file and update it.

**Step 2: Update the config example in main.go error message**

In `main.go`, the missing-config error message shows example YAML. Update it:

```go
// Before:
return fmt.Errorf("config file not found: %s\n\nCreate it with:\n  mkdir -p ~/.config/prompt-builder\n  cat > ~/.config/prompt-builder/config.yaml << 'EOF'\n  model: llama3.2\n  system_prompt_file: ~/.config/prompt-builder/prompt-architect.md\n  EOF", configPath)

// After:
return fmt.Errorf("config file not found: %s\n\nCreate it with:\n  mkdir -p ~/.config/prompt-builder\n  cat > ~/.config/prompt-builder/config.yaml << 'EOF'\n  model: llama3.2\n  host: http://localhost:11434\n  system_prompt_file: ~/.config/prompt-builder/prompt-architect.md\n  EOF", configPath)
```

**Step 3: Run all tests one final time**

Run: `go test ./cmd/prompt-builder`
Expected: All pass

**Step 4: Commit and tag**

```bash
git add cmd/prompt-builder/main.go
git commit -m "feat: update config example to use host field"
git tag -a v2.0.0 -m "v2.0.0: OpenAI-compatible chat completions protocol

BREAKING: config field 'ollama_host' renamed to 'host'

- Replace Ollama-specific client with OpenAI chat completions protocol
- Support any compatible backend: llama.cpp, Ollama, LM Studio, vLLM
- Endpoint changed from /api/chat to /v1/chat/completions
- SSE streaming replaces Ollama newline-delimited JSON
- Remove dead IsModelLoaded code"
```
