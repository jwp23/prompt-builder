# Startup Feedback Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Display status messages during startup to help users understand whether the model is loading or already processing.

**Architecture:** Add `IsModelLoaded()` method to `OllamaClient` that calls `/api/ps`. In `main.go`, check model status before `ChatStream` and print appropriate message ("Loading...", "Thinking...", or "Connecting...").

**Tech Stack:** Go, net/http, httptest for testing

---

### Task 1: Add OllamaPsResponse struct

**Files:**
- Modify: `ollama.go:25` (after OllamaStreamChunk)

**Step 1: Write the struct**

Add after `OllamaStreamChunk` (line 28):

```go
type OllamaPsModel struct {
	Name string `json:"name"`
}

type OllamaPsResponse struct {
	Models []OllamaPsModel `json:"models"`
}
```

**Step 2: Run tests to verify no breakage**

Run: `go test -v ./...`
Expected: All existing tests PASS

**Step 3: Commit**

```bash
git add ollama.go
git commit -m "feat: add OllamaPsResponse struct for /api/ps endpoint"
```

---

### Task 2: Write failing test for IsModelLoaded (model present)

**Files:**
- Modify: `ollama_test.go` (add at end)

**Step 1: Write the failing test**

```go
func TestOllamaClient_IsModelLoaded_True(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/ps" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprintln(w, `{"models":[{"name":"llama3.2"}]}`)
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "llama3.2")
	loaded, err := client.IsModelLoaded()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !loaded {
		t.Error("expected model to be loaded")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -run TestOllamaClient_IsModelLoaded_True`
Expected: FAIL with "client.IsModelLoaded undefined"

---

### Task 3: Implement IsModelLoaded method

**Files:**
- Modify: `ollama.go` (add method after ChatStream)

**Step 1: Write minimal implementation**

Add after `ChatStream` method (after line 94):

```go
func (c *OllamaClient) IsModelLoaded() (bool, error) {
	resp, err := c.client.Get(c.Host + "/api/ps")
	if err != nil {
		return false, fmt.Errorf("failed to check model status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var psResp OllamaPsResponse
	if err := json.NewDecoder(resp.Body).Decode(&psResp); err != nil {
		return false, fmt.Errorf("failed to parse response: %w", err)
	}

	for _, model := range psResp.Models {
		if model.Name == c.Model {
			return true, nil
		}
	}
	return false, nil
}
```

**Step 2: Run test to verify it passes**

Run: `go test -v -run TestOllamaClient_IsModelLoaded_True`
Expected: PASS

**Step 3: Commit**

```bash
git add ollama.go ollama_test.go
git commit -m "feat: add IsModelLoaded method to check if model is in memory"
```

---

### Task 4: Write test for IsModelLoaded (model absent)

**Files:**
- Modify: `ollama_test.go`

**Step 1: Write the test**

```go
func TestOllamaClient_IsModelLoaded_False(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"models":[{"name":"other-model"}]}`)
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "llama3.2")
	loaded, err := client.IsModelLoaded()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded {
		t.Error("expected model to not be loaded")
	}
}
```

**Step 2: Run test to verify it passes**

Run: `go test -v -run TestOllamaClient_IsModelLoaded_False`
Expected: PASS (implementation already handles this)

**Step 3: Commit**

```bash
git add ollama_test.go
git commit -m "test: add IsModelLoaded test for absent model"
```

---

### Task 5: Write test for IsModelLoaded (empty models list)

**Files:**
- Modify: `ollama_test.go`

**Step 1: Write the test**

```go
func TestOllamaClient_IsModelLoaded_EmptyList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"models":[]}`)
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "llama3.2")
	loaded, err := client.IsModelLoaded()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded {
		t.Error("expected model to not be loaded with empty list")
	}
}
```

**Step 2: Run test to verify it passes**

Run: `go test -v -run TestOllamaClient_IsModelLoaded_EmptyList`
Expected: PASS

**Step 3: Commit**

```bash
git add ollama_test.go
git commit -m "test: add IsModelLoaded test for empty models list"
```

---

### Task 6: Write test for IsModelLoaded (API error)

**Files:**
- Modify: `ollama_test.go`

**Step 1: Write the test**

```go
func TestOllamaClient_IsModelLoaded_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "llama3.2")
	_, err := client.IsModelLoaded()

	if err == nil {
		t.Error("expected error for HTTP 500")
	}
}
```

**Step 2: Run test to verify it passes**

Run: `go test -v -run TestOllamaClient_IsModelLoaded_Error`
Expected: PASS

**Step 3: Commit**

```bash
git add ollama_test.go
git commit -m "test: add IsModelLoaded test for API error"
```

---

### Task 7: Write test for IsModelLoaded (connection refused)

**Files:**
- Modify: `ollama_test.go`

**Step 1: Write the test**

```go
func TestOllamaClient_IsModelLoaded_ConnectionRefused(t *testing.T) {
	client := NewOllamaClient("http://localhost:1", "llama3.2")
	_, err := client.IsModelLoaded()

	if err == nil {
		t.Error("expected error for connection refused")
	}
}
```

**Step 2: Run test to verify it passes**

Run: `go test -v -run TestOllamaClient_IsModelLoaded_ConnectionRefused`
Expected: PASS

**Step 3: Commit**

```bash
git add ollama_test.go
git commit -m "test: add IsModelLoaded test for connection refused"
```

---

### Task 8: Add printStartupStatus function

**Files:**
- Modify: `main.go` (add function before run())

**Step 1: Write the function**

Add before the `run` function (around line 84):

```go
func printStartupStatus(client *OllamaClient, quiet bool, tty bool) {
	if quiet || !tty {
		return
	}

	loaded, err := client.IsModelLoaded()
	if err != nil {
		fmt.Println("Connecting...")
		return
	}

	if loaded {
		fmt.Println("Thinking...")
	} else {
		fmt.Printf("Loading %s...\n", client.Model)
	}
}
```

**Step 2: Run tests to verify no breakage**

Run: `go test -v ./...`
Expected: All tests PASS

**Step 3: Commit**

```bash
git add main.go
git commit -m "feat: add printStartupStatus function for user feedback"
```

---

### Task 9: Integrate printStartupStatus into run()

**Files:**
- Modify: `main.go:139` (before ChatStream call in the loop)

**Step 1: Add the call**

The status message should only print once, before the first `ChatStream` call. Add a flag and the call. Replace the conversation loop section (starting at line 137):

```go
	// Conversation loop
	reader := bufio.NewReader(os.Stdin)
	firstRequest := true
	for {
		// Show status on first request only
		if firstRequest {
			printStartupStatus(client, cli.Quiet, isTTY())
			firstRequest = false
		}

		// Get response from LLM with streaming
		response, err := client.ChatStream(conv.Messages, func(token string) error {
```

**Step 2: Run tests to verify no breakage**

Run: `go test -v ./...`
Expected: All tests PASS

**Step 3: Commit**

```bash
git add main.go
git commit -m "feat: show startup status before first LLM request"
```

---

### Task 10: Manual verification

**Step 1: Build the binary**

Run: `go build -o prompt-builder .`

**Step 2: Test with model not loaded**

First, ensure no models are loaded:
Run: `curl -s http://localhost:11434/api/ps`
Expected: `{"models":[]}`

Run: `./prompt-builder "test idea"`
Expected: Shows "Loading <model>..." before response streams

**Step 3: Test with model loaded**

After previous test, model should be loaded:
Run: `curl -s http://localhost:11434/api/ps`
Expected: Shows the model in the list

Run: `./prompt-builder "another test"`
Expected: Shows "Thinking..." before response streams

**Step 4: Test quiet mode**

Run: `./prompt-builder -q "test idea"`
Expected: No status message, just the response

**Step 5: Test pipe mode**

Run: `echo "test" | ./prompt-builder "test idea"`
Expected: No status message (not a TTY)

---

### Task 11: Final commit and verification

**Step 1: Run full test suite**

Run: `go test -v ./...`
Expected: All tests PASS

**Step 2: Verify clean git status**

Run: `git status`
Expected: Clean working tree (nothing to commit)

**Step 3: Review commits**

Run: `git log --oneline -10`
Expected: See all the commits from this implementation
