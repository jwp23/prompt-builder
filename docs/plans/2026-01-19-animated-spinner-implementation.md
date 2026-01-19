# Animated Spinner Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add braille dot spinner animations for "Loading <model>..." and "Thinking..." states.

**Architecture:** Spinner is a standalone component in `spinner.go`. Loading flow polls `/api/ps` with animated feedback. Thinking spinner wraps the streaming callback and stops on first token.

**Tech Stack:** Go standard library only (time, sync, fmt)

---

## Task 1: Spinner Type and Constructor

**Files:**
- Create: `spinner.go`
- Create: `spinner_test.go`

**Step 1: Write the failing test for NewSpinner**

```go
// spinner_test.go
package main

import "testing"

func TestNewSpinner(t *testing.T) {
	s := NewSpinner("Loading...")
	if s == nil {
		t.Fatal("NewSpinner returned nil")
	}
	if s.message != "Loading..." {
		t.Errorf("message = %q, want %q", s.message, "Loading...")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -run TestNewSpinner -v`
Expected: FAIL - undefined: NewSpinner

**Step 3: Write minimal implementation**

```go
// spinner.go
package main

import "time"

var spinnerFrames = []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}

type Spinner struct {
	frames   []rune
	interval time.Duration
	message  string
	stopCh   chan struct{}
	doneCh   chan struct{}
}

func NewSpinner(message string) *Spinner {
	return &Spinner{
		frames:   spinnerFrames,
		interval: 120 * time.Millisecond,
		message:  message,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test -run TestNewSpinner -v`
Expected: PASS

**Step 5: Commit**

```bash
git add spinner.go spinner_test.go
git commit -m "feat(spinner): add Spinner type and constructor"
```

---

## Task 2: Spinner Stop Safety

**Files:**
- Modify: `spinner.go`
- Modify: `spinner_test.go`

**Step 1: Write failing test for Stop being safe without Start**

```go
func TestSpinner_StopWithoutStart(t *testing.T) {
	s := NewSpinner("Test")
	// Should not panic
	s.Stop()
}
```

**Step 2: Run test to verify it fails**

Run: `go test -run TestSpinner_StopWithoutStart -v`
Expected: FAIL - undefined: s.Stop

**Step 3: Write minimal Stop implementation**

Add to `spinner.go`:

```go
func (s *Spinner) Stop() {
	select {
	case <-s.stopCh:
		// Already stopped
		return
	default:
		close(s.stopCh)
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test -run TestSpinner_StopWithoutStart -v`
Expected: PASS

**Step 5: Write test for multiple Stop calls**

```go
func TestSpinner_StopMultipleTimes(t *testing.T) {
	s := NewSpinner("Test")
	// Should not panic on multiple Stop calls
	s.Stop()
	s.Stop()
	s.Stop()
}
```

**Step 6: Run test to verify it passes**

Run: `go test -run TestSpinner_StopMultipleTimes -v`
Expected: PASS (already handles this case)

**Step 7: Commit**

```bash
git add spinner.go spinner_test.go
git commit -m "feat(spinner): add safe Stop method"
```

---

## Task 3: Spinner Start and Animation Loop

**Files:**
- Modify: `spinner.go`
- Modify: `spinner_test.go`

**Step 1: Write failing test for Start**

```go
func TestSpinner_StartStop(t *testing.T) {
	s := NewSpinner("Loading")
	s.Start()
	// Give it a moment to run
	time.Sleep(50 * time.Millisecond)
	s.Stop()
	// Should complete without hanging
}
```

Add import: `"time"`

**Step 2: Run test to verify it fails**

Run: `go test -run TestSpinner_StartStop -v -timeout 5s`
Expected: FAIL - undefined: s.Start

**Step 3: Write Start implementation**

Add imports and implementation to `spinner.go`:

```go
import (
	"fmt"
	"strings"
	"time"
)

func (s *Spinner) Start() {
	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		defer close(s.doneCh)

		frame := 0
		for {
			select {
			case <-s.stopCh:
				s.clearLine()
				return
			case <-ticker.C:
				fmt.Printf("\r%c %s", s.frames[frame], s.message)
				frame = (frame + 1) % len(s.frames)
			}
		}
	}()
}

func (s *Spinner) clearLine() {
	// Clear the line: carriage return, spaces, carriage return
	clearLen := len(s.message) + 3 // frame + space + message
	fmt.Printf("\r%s\r", strings.Repeat(" ", clearLen))
}
```

**Step 4: Run test to verify it passes**

Run: `go test -run TestSpinner_StartStop -v -timeout 5s`
Expected: PASS

**Step 5: Commit**

```bash
git add spinner.go spinner_test.go
git commit -m "feat(spinner): add Start method with animation loop"
```

---

## Task 4: Spinner TTY Guard

**Files:**
- Modify: `spinner.go`
- Modify: `spinner_test.go`

**Step 1: Write failing test for TTY behavior**

```go
func TestNewSpinnerWithTTY_False(t *testing.T) {
	s := NewSpinnerWithTTY("Loading", false)
	if s.tty {
		t.Error("expected tty to be false")
	}
}

func TestNewSpinnerWithTTY_True(t *testing.T) {
	s := NewSpinnerWithTTY("Loading", true)
	if !s.tty {
		t.Error("expected tty to be true")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test -run TestNewSpinnerWithTTY -v`
Expected: FAIL - undefined: NewSpinnerWithTTY

**Step 3: Add tty field and NewSpinnerWithTTY**

Update `spinner.go`:

```go
type Spinner struct {
	frames   []rune
	interval time.Duration
	message  string
	tty      bool
	stopCh   chan struct{}
	doneCh   chan struct{}
}

func NewSpinner(message string) *Spinner {
	return NewSpinnerWithTTY(message, true)
}

func NewSpinnerWithTTY(message string, tty bool) *Spinner {
	return &Spinner{
		frames:   spinnerFrames,
		interval: 120 * time.Millisecond,
		message:  message,
		tty:      tty,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}
```

**Step 4: Run tests to verify they pass**

Run: `go test -run TestNewSpinnerWithTTY -v`
Expected: PASS

**Step 5: Write test for non-TTY Start behavior**

```go
func TestSpinner_StartNonTTY(t *testing.T) {
	s := NewSpinnerWithTTY("Loading", false)
	s.Start() // Should be no-op, not start goroutine
	s.Stop()  // Should be safe
}
```

**Step 6: Update Start to check TTY**

```go
func (s *Spinner) Start() {
	if !s.tty {
		return
	}
	go func() {
		// ... rest unchanged
	}()
}
```

**Step 7: Run all spinner tests**

Run: `go test -run TestSpinner -v`
Expected: PASS

**Step 8: Commit**

```bash
git add spinner.go spinner_test.go
git commit -m "feat(spinner): add TTY guard for non-interactive mode"
```

---

## Task 5: WaitForModel - Basic Success Path

**Files:**
- Create: `startup.go`
- Create: `startup_test.go`

**Step 1: Write failing test for model already loaded**

```go
// startup_test.go
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWaitForModel_AlreadyLoaded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"models":[{"name":"llama3.2"}]}`)
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "llama3.2")
	err := WaitForModel(client, false, true)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -run TestWaitForModel_AlreadyLoaded -v`
Expected: FAIL - undefined: WaitForModel

**Step 3: Write minimal WaitForModel**

```go
// startup.go
package main

func WaitForModel(client *OllamaClient, quiet bool, tty bool) error {
	if quiet || !tty {
		return nil
	}

	loaded, err := client.IsModelLoaded()
	if err != nil {
		return err
	}
	if loaded {
		return nil
	}

	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test -run TestWaitForModel_AlreadyLoaded -v`
Expected: PASS

**Step 5: Commit**

```bash
git add startup.go startup_test.go
git commit -m "feat(startup): add WaitForModel basic structure"
```

---

## Task 6: WaitForModel - Quiet Mode Skip

**Files:**
- Modify: `startup_test.go`

**Step 1: Write test for quiet mode**

```go
func TestWaitForModel_QuietMode(t *testing.T) {
	// Server that would fail if called
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be called in quiet mode")
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "llama3.2")
	err := WaitForModel(client, true, true) // quiet=true

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWaitForModel_NonTTY(t *testing.T) {
	// Server that would fail if called
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be called in non-TTY mode")
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "llama3.2")
	err := WaitForModel(client, false, false) // tty=false

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

**Step 2: Run tests to verify they pass**

Run: `go test -run "TestWaitForModel_QuietMode|TestWaitForModel_NonTTY" -v`
Expected: PASS (already implemented)

**Step 3: Commit**

```bash
git add startup_test.go
git commit -m "test(startup): add quiet mode and non-TTY tests"
```

---

## Task 7: WaitForModel - Polling Until Loaded

**Files:**
- Modify: `startup.go`
- Modify: `startup_test.go`

**Step 1: Write test for polling behavior**

```go
func TestWaitForModel_PollsUntilLoaded(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 3 {
			fmt.Fprintln(w, `{"models":[]}`) // Not loaded yet
		} else {
			fmt.Fprintln(w, `{"models":[{"name":"llama3.2"}]}`) // Now loaded
		}
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "llama3.2")
	err := WaitForModel(client, false, true)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount < 3 {
		t.Errorf("expected at least 3 calls, got %d", callCount)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -run TestWaitForModel_PollsUntilLoaded -v -timeout 5s`
Expected: FAIL (returns immediately without polling)

**Step 3: Add polling loop to WaitForModel**

Update `startup.go`:

```go
package main

import "time"

const (
	pollInterval   = 500 * time.Millisecond
	connectTimeout = 30 * time.Second
)

func WaitForModel(client *OllamaClient, quiet bool, tty bool) error {
	if quiet || !tty {
		return nil
	}

	loaded, err := client.IsModelLoaded()
	if err != nil {
		return err
	}
	if loaded {
		return nil
	}

	// Model not loaded, poll until it is
	spinner := NewSpinnerWithTTY("Loading "+client.Model+"...", tty)
	spinner.Start()
	defer spinner.Stop()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for range ticker.C {
		loaded, err := client.IsModelLoaded()
		if err != nil {
			continue // Keep trying on errors
		}
		if loaded {
			return nil
		}
	}

	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test -run TestWaitForModel_PollsUntilLoaded -v -timeout 10s`
Expected: PASS

**Step 5: Commit**

```bash
git add startup.go startup_test.go
git commit -m "feat(startup): add polling loop with spinner"
```

---

## Task 8: WaitForModel - Connection Timeout

**Files:**
- Modify: `startup.go`
- Modify: `startup_test.go`

**Step 1: Write test for timeout**

```go
func TestWaitForModel_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"models":[]}`) // Never becomes loaded
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "llama3.2")
	err := WaitForModelWithTimeout(client, false, true, 100*time.Millisecond)

	if err == nil {
		t.Fatal("expected timeout error")
	}
	if err.Error() != "timeout waiting for model to load" {
		t.Errorf("unexpected error: %v", err)
	}
}
```

Add import: `"time"`

**Step 2: Run test to verify it fails**

Run: `go test -run TestWaitForModel_Timeout -v`
Expected: FAIL - undefined: WaitForModelWithTimeout

**Step 3: Add timeout support**

Update `startup.go`:

```go
package main

import (
	"fmt"
	"time"
)

const (
	pollInterval   = 500 * time.Millisecond
	connectTimeout = 30 * time.Second
)

func WaitForModel(client *OllamaClient, quiet bool, tty bool) error {
	return WaitForModelWithTimeout(client, quiet, tty, connectTimeout)
}

func WaitForModelWithTimeout(client *OllamaClient, quiet bool, tty bool, timeout time.Duration) error {
	if quiet || !tty {
		return nil
	}

	loaded, err := client.IsModelLoaded()
	if err == nil && loaded {
		return nil
	}

	// Model not loaded or error connecting, poll until ready
	message := "Loading " + client.Model + "..."
	if err != nil {
		message = "Connecting..."
	}

	spinner := NewSpinnerWithTTY(message, tty)
	spinner.Start()
	defer spinner.Stop()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	deadline := time.Now().Add(timeout)

	for range ticker.C {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for model to load")
		}

		loaded, err := client.IsModelLoaded()
		if err != nil {
			continue // Keep trying on errors
		}
		if loaded {
			return nil
		}
	}

	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test -run TestWaitForModel_Timeout -v`
Expected: PASS

**Step 5: Commit**

```bash
git add startup.go startup_test.go
git commit -m "feat(startup): add connection timeout"
```

---

## Task 9: WaitForModel - Connection Error with Retry

**Files:**
- Modify: `startup_test.go`

**Step 1: Write test for connection error then success**

```go
func TestWaitForModel_ConnectionErrorThenSuccess(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 3 {
			w.WriteHeader(http.StatusInternalServerError) // Simulate error
		} else {
			fmt.Fprintln(w, `{"models":[{"name":"llama3.2"}]}`)
		}
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "llama3.2")
	err := WaitForModelWithTimeout(client, false, true, 5*time.Second)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount < 3 {
		t.Errorf("expected at least 3 calls, got %d", callCount)
	}
}
```

**Step 2: Run test to verify it passes**

Run: `go test -run TestWaitForModel_ConnectionErrorThenSuccess -v -timeout 10s`
Expected: PASS (already handles this)

**Step 3: Commit**

```bash
git add startup_test.go
git commit -m "test(startup): add connection retry test"
```

---

## Task 10: Integrate WaitForModel in main.go

**Files:**
- Modify: `main.go`

**Step 1: Replace printStartupStatus with WaitForModel**

In `main.go`, find lines 159-163:

```go
		// Show status on first request only
		if firstRequest {
			printStartupStatus(client, cli.Quiet, isTTY())
			firstRequest = false
		}
```

Replace with:

```go
		// Wait for model on first request only
		if firstRequest {
			if err := WaitForModel(client, cli.Quiet, isTTY()); err != nil {
				return fmt.Errorf("failed to connect to Ollama: %v", err)
			}
			firstRequest = false
		}
```

**Step 2: Remove printStartupStatus function**

Delete lines 85-101 (the `printStartupStatus` function).

**Step 3: Run all tests**

Run: `go test ./... -v`
Expected: PASS

**Step 4: Manual test**

Run: `go build && ./prompt-builder "test idea"`
Expected: See spinner animation while loading model

**Step 5: Commit**

```bash
git add main.go
git commit -m "feat: integrate WaitForModel spinner in main"
```

---

## Task 11: Thinking Spinner - Wrapper Callback

**Files:**
- Modify: `ollama.go`
- Modify: `ollama_test.go`

**Step 1: Write failing test for ChatStreamWithSpinner**

```go
func TestOllamaClient_ChatStreamWithSpinner_StopsOnFirstToken(t *testing.T) {
	server := fakeStreamingServer([]string{"Hello", " there", "!"})
	defer server.Close()

	client := NewOllamaClient(server.URL, "llama3.2")
	messages := []Message{{Role: "user", Content: "Hi"}}

	var tokens []string
	response, err := client.ChatStreamWithSpinner(messages, false, func(token string) error {
		tokens = append(tokens, token)
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response != "Hello there!" {
		t.Errorf("response = %q, want %q", response, "Hello there!")
	}
	if len(tokens) != 3 {
		t.Errorf("got %d tokens, want 3", len(tokens))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -run TestOllamaClient_ChatStreamWithSpinner -v`
Expected: FAIL - undefined: ChatStreamWithSpinner

**Step 3: Add ChatStreamWithSpinner method**

Add to `ollama.go`:

```go
import "sync"
```

Then add the method:

```go
func (c *OllamaClient) ChatStreamWithSpinner(messages []Message, tty bool, onToken StreamCallback) (string, error) {
	var spinner *Spinner
	var once sync.Once

	if tty {
		spinner = NewSpinnerWithTTY("Thinking...", tty)
		spinner.Start()
	}

	wrappedCallback := func(token string) error {
		once.Do(func() {
			if spinner != nil {
				spinner.Stop()
			}
		})
		return onToken(token)
	}

	return c.ChatStream(messages, wrappedCallback)
}
```

**Step 4: Run test to verify it passes**

Run: `go test -run TestOllamaClient_ChatStreamWithSpinner -v`
Expected: PASS

**Step 5: Commit**

```bash
git add ollama.go ollama_test.go
git commit -m "feat(ollama): add ChatStreamWithSpinner method"
```

---

## Task 12: Integrate Thinking Spinner in main.go

**Files:**
- Modify: `main.go`

**Step 1: Replace ChatStream with ChatStreamWithSpinner**

In `main.go`, find lines 165-171:

```go
		// Get response from LLM with streaming
		response, err := client.ChatStream(conv.Messages, func(token string) error {
			if !cli.Quiet {
				fmt.Print(token)
			}
			return nil
		})
```

Replace with:

```go
		// Get response from LLM with streaming
		response, err := client.ChatStreamWithSpinner(conv.Messages, isTTY() && !cli.Quiet, func(token string) error {
			if !cli.Quiet {
				fmt.Print(token)
			}
			return nil
		})
```

**Step 2: Run all tests**

Run: `go test ./... -v`
Expected: PASS

**Step 3: Manual test**

Run: `go build && ./prompt-builder "test idea"`
Expected: See "Thinking..." spinner that stops when response starts

**Step 4: Commit**

```bash
git add main.go
git commit -m "feat: integrate Thinking spinner in main"
```

---

## Task 13: Final Cleanup and Full Test

**Step 1: Run all tests**

Run: `go test ./... -v`
Expected: All PASS

**Step 2: Run linter if available**

Run: `go vet ./...`
Expected: No issues

**Step 3: Manual end-to-end test**

Test scenarios:
1. Model not loaded: `go build && ./prompt-builder "test"` → See "Loading..." spinner
2. Model loaded: Run again → See "Thinking..." spinner
3. Quiet mode: `./prompt-builder -q "test"` → No spinners visible
4. Pipe mode: `echo "test" | ./prompt-builder "test"` → No spinners visible

**Step 4: Commit**

```bash
git add -A
git commit -m "feat: complete animated spinner implementation"
```
