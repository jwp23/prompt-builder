# Streaming Support Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement streaming mode in the Ollama client so tokens display as they're generated, improving UX.

**Architecture:** Add a `ChatStream` method to `OllamaClient` that reads newline-delimited JSON chunks from Ollama's streaming API. Each chunk's content is passed to a callback for immediate display, while accumulating the full response for conversation history. Replace the `Chat()` call in `main.go` with `ChatStream()`.

**Tech Stack:** Go, `bufio.Scanner` for line-by-line reading, `net/http/httptest` for mock servers in tests.

---

## Task 1: Add OllamaStreamChunk Struct

**Files:**
- Modify: `ollama.go:23-25` (after `OllamaResponse`)

**Step 1: Write the failing test**

Add to `ollama_test.go`:

```go
func TestOllamaStreamChunk_Deserialization(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		wantRole string
		wantContent string
		wantDone bool
	}{
		{
			name:        "partial chunk",
			json:        `{"message":{"role":"assistant","content":"Hello"},"done":false}`,
			wantRole:    "assistant",
			wantContent: "Hello",
			wantDone:    false,
		},
		{
			name:        "final chunk",
			json:        `{"message":{"role":"assistant","content":"!"},"done":true}`,
			wantRole:    "assistant",
			wantContent: "!",
			wantDone:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var chunk OllamaStreamChunk
			if err := json.Unmarshal([]byte(tt.json), &chunk); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			if chunk.Message.Role != tt.wantRole {
				t.Errorf("role = %q, want %q", chunk.Message.Role, tt.wantRole)
			}
			if chunk.Message.Content != tt.wantContent {
				t.Errorf("content = %q, want %q", chunk.Message.Content, tt.wantContent)
			}
			if chunk.Done != tt.wantDone {
				t.Errorf("done = %v, want %v", chunk.Done, tt.wantDone)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -run TestOllamaStreamChunk_Deserialization -v`
Expected: FAIL with "undefined: OllamaStreamChunk"

**Step 3: Write minimal implementation**

Add to `ollama.go` after `OllamaResponse`:

```go
type OllamaStreamChunk struct {
	Message Message `json:"message"`
	Done    bool    `json:"done"`
}
```

**Step 4: Run test to verify it passes**

Run: `go test -run TestOllamaStreamChunk_Deserialization -v`
Expected: PASS

**Step 5: Commit**

```bash
git add ollama.go ollama_test.go
git commit -m "feat: add OllamaStreamChunk struct for streaming responses"
```

---

## Task 2: Add StreamCallback Type

**Files:**
- Modify: `ollama.go:27` (after `OllamaStreamChunk`)

**Step 1: Write the failing test**

Add to `ollama_test.go`:

```go
func TestStreamCallback_Type(t *testing.T) {
	var callback StreamCallback = func(token string) error {
		return nil
	}

	if err := callback("test"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -run TestStreamCallback_Type -v`
Expected: FAIL with "undefined: StreamCallback"

**Step 3: Write minimal implementation**

Add to `ollama.go` after `OllamaStreamChunk`:

```go
type StreamCallback func(token string) error
```

**Step 4: Run test to verify it passes**

Run: `go test -run TestStreamCallback_Type -v`
Expected: PASS

**Step 5: Commit**

```bash
git add ollama.go ollama_test.go
git commit -m "feat: add StreamCallback type for streaming token handler"
```

---

## Task 3: Create Streaming Test Helper

**Files:**
- Modify: `ollama_test.go`

**Step 1: Add the test helper function**

Add to `ollama_test.go`:

```go
func fakeStreamingServer(chunks []string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for i, chunk := range chunks {
			done := i == len(chunks)-1
			fmt.Fprintf(w, `{"message":{"role":"assistant","content":%q},"done":%v}`+"\n",
				chunk, done)
			w.(http.Flusher).Flush()
		}
	}))
}
```

Note: You'll need to add `"fmt"` to the imports if not already present.

**Step 2: Verify the helper compiles**

Run: `go build ./...`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add ollama_test.go
git commit -m "test: add fakeStreamingServer helper for streaming tests"
```

---

## Task 4: Implement ChatStream Method - Happy Path

**Files:**
- Modify: `ollama.go` (add `ChatStream` method)
- Modify: `ollama_test.go` (add test)

**Step 1: Write the failing test**

Add to `ollama_test.go`:

```go
func TestOllamaClient_ChatStream_HappyPath(t *testing.T) {
	server := fakeStreamingServer([]string{"Hello", " there", "!"})
	defer server.Close()

	client := NewOllamaClient(server.URL, "llama3.2")
	messages := []Message{
		{Role: "user", Content: "Hi"},
	}

	var tokens []string
	response, err := client.ChatStream(messages, func(token string) error {
		tokens = append(tokens, token)
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify callback received tokens in order
	expectedTokens := []string{"Hello", " there", "!"}
	if len(tokens) != len(expectedTokens) {
		t.Errorf("got %d tokens, want %d", len(tokens), len(expectedTokens))
	}
	for i, tok := range tokens {
		if tok != expectedTokens[i] {
			t.Errorf("token[%d] = %q, want %q", i, tok, expectedTokens[i])
		}
	}

	// Verify accumulated response
	if response != "Hello there!" {
		t.Errorf("response = %q, want %q", response, "Hello there!")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -run TestOllamaClient_ChatStream_HappyPath -v`
Expected: FAIL with "client.ChatStream undefined"

**Step 3: Write minimal implementation**

Add to `ollama.go` after the `Chat` method:

```go
func (c *OllamaClient) ChatStream(messages []Message, onToken StreamCallback) (string, error) {
	req := OllamaRequest{
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
		return "", fmt.Errorf("failed to connect to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Ollama request failed: %s - %s", resp.Status, string(body))
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

Note: Add `"bufio"` and `"strings"` to imports if not present.

**Step 4: Run test to verify it passes**

Run: `go test -run TestOllamaClient_ChatStream_HappyPath -v`
Expected: PASS

**Step 5: Run all tests to check for regressions**

Run: `go test -v`
Expected: All tests PASS

**Step 6: Commit**

```bash
git add ollama.go ollama_test.go
git commit -m "feat: implement ChatStream method for streaming responses"
```

---

## Task 5: Add ChatStream Error Handling Tests

**Files:**
- Modify: `ollama_test.go`

**Step 1: Write test for callback error**

Add to `ollama_test.go`:

```go
func TestOllamaClient_ChatStream_CallbackError(t *testing.T) {
	server := fakeStreamingServer([]string{"Hello", " there", "!"})
	defer server.Close()

	client := NewOllamaClient(server.URL, "llama3.2")
	messages := []Message{{Role: "user", Content: "Hi"}}

	callbackErr := fmt.Errorf("callback failed")
	callCount := 0
	_, err := client.ChatStream(messages, func(token string) error {
		callCount++
		if callCount == 2 {
			return callbackErr
		}
		return nil
	})

	if err != callbackErr {
		t.Errorf("expected callback error, got: %v", err)
	}
	if callCount != 2 {
		t.Errorf("callback called %d times, expected 2", callCount)
	}
}
```

**Step 2: Run test to verify it passes**

Run: `go test -run TestOllamaClient_ChatStream_CallbackError -v`
Expected: PASS (implementation already handles this)

**Step 3: Write test for malformed JSON**

Add to `ollama_test.go`:

```go
func TestOllamaClient_ChatStream_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "not valid json")
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "llama3.2")
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

**Step 4: Run test to verify it passes**

Run: `go test -run TestOllamaClient_ChatStream_MalformedJSON -v`
Expected: PASS

**Step 5: Write test for HTTP error**

Add to `ollama_test.go`:

```go
func TestOllamaClient_ChatStream_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "server error")
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "llama3.2")
	messages := []Message{{Role: "user", Content: "Hi"}}

	_, err := client.ChatStream(messages, func(token string) error {
		return nil
	})

	if err == nil {
		t.Error("expected error for HTTP 500")
	}
	if !strings.Contains(err.Error(), "Ollama request failed") {
		t.Errorf("unexpected error message: %v", err)
	}
}
```

**Step 6: Run all streaming tests**

Run: `go test -run "ChatStream" -v`
Expected: All PASS

**Step 7: Run full test suite**

Run: `go test -v`
Expected: All PASS

**Step 8: Commit**

```bash
git add ollama_test.go
git commit -m "test: add error handling tests for ChatStream"
```

---

## Task 6: Integrate ChatStream in main.go

**Files:**
- Modify: `main.go:139-144` (replace `Chat()` call with `ChatStream()`)

**Step 1: Update the run function**

Replace the block at line 139-144:

```go
// Get response from LLM
response, err := client.Chat(conv.Messages)
if err != nil {
	return fmt.Errorf("Ollama request failed: %v", err)
}
```

With:

```go
// Get response from LLM with streaming
response, err := client.ChatStream(conv.Messages, func(token string) error {
	if !cli.Quiet {
		fmt.Print(token)
	}
	return nil
})
if err != nil {
	return fmt.Errorf("Ollama request failed: %v", err)
}
if !cli.Quiet {
	fmt.Println() // newline after streaming completes
}
```

**Step 2: Build to verify no compile errors**

Run: `go build -o prompt-builder .`
Expected: Build succeeds

**Step 3: Run all tests**

Run: `go test -v`
Expected: All PASS

**Step 4: Commit**

```bash
git add main.go
git commit -m "feat: integrate streaming in conversation loop"
```

---

## Task 7: Manual Integration Test

**Step 1: Test interactive mode with streaming**

Run: `./prompt-builder "a simple test prompt"`
Expected: Tokens appear incrementally as they're generated

**Step 2: Test quiet mode**

Run: `./prompt-builder -q "a simple test prompt"`
Expected: No output during generation, only final result

**Step 3: Test pipe mode**

Run: `echo "test" | ./prompt-builder "a simple test prompt" --no-copy`
Expected: Streaming output (incremental tokens visible)

**Step 4: Document results**

Note any issues found for follow-up. If all works, proceed to final commit.

---

## Task 8: Final Cleanup and Commit

**Step 1: Run full test suite**

Run: `go test -v`
Expected: All PASS

**Step 2: Check for unused imports**

Run: `go build ./...`
Expected: No warnings

**Step 3: Verify the original Chat method is still available**

The non-streaming `Chat()` method should remain for backward compatibility or future use cases where streaming isn't wanted. No changes needed if it wasn't removed.

**Step 4: Final commit if any cleanup was needed**

```bash
git add -A
git commit -m "chore: streaming support complete"
```

---

## Summary

| Task | Description |
|------|-------------|
| 1 | Add `OllamaStreamChunk` struct |
| 2 | Add `StreamCallback` type |
| 3 | Create test helper `fakeStreamingServer` |
| 4 | Implement `ChatStream` method |
| 5 | Add error handling tests |
| 6 | Integrate in `main.go` |
| 7 | Manual integration test |
| 8 | Final cleanup |

**Total commits:** 7-8 incremental commits following TDD.
