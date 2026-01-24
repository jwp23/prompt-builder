# Integration and E2E Tests Design

## Overview

Add integration and E2E tests to prompt-builder for refactoring confidence and regression protection.

## Goals

1. **Refactoring confidence** - Safely restructure `main.go` and `run()` without breaking things
2. **Regression protection** - Catch bugs when dependencies change (Ollama API, clipboard tools, etc.)

## Test Organization

### File Structure

```
cmd/prompt-builder/
├── main.go             # Refactored with App struct and DI
├── e2e_test.go         # //go:build e2e
├── integration_test.go # Tests App.Run() with mocks
├── testdata/           # Fixtures
│   ├── config/
│   │   ├── valid.yaml
│   │   ├── minimal.yaml
│   │   └── custom-model.yaml
│   ├── prompts/
│   │   └── system.txt
│   └── responses/
│       ├── clarifying.txt
│       ├── complete.txt
│       └── error.txt
├── ... existing files ...
```

### Build Tags

- **Default** (`go test ./cmd/prompt-builder`) - Unit tests + integration tests. No external dependencies.
- **E2E** (`go test -tags=e2e ./cmd/prompt-builder`) - Everything including E2E. Requires Ollama.

## Refactoring for Testability

### Interfaces

```go
type OllamaClient interface {
    ChatStream(ctx context.Context, model string, msgs []Message, cb StreamCallback) (string, error)
}

type ClipboardWriter interface {
    Write(content string) error
}
```

### App Struct

```go
type App struct {
    Client    OllamaClient
    Stdin     io.Reader
    Stdout    io.Writer
    Stderr    io.Writer
    Clipboard ClipboardWriter
    IsTTY     bool
}

func (a *App) Run(ctx context.Context, cfg *Config, idea string) error
```

## Test Helpers

### Mock OllamaClient

```go
type mockOllama struct {
    responses []string
    calls     int
    err       error
}

func (m *mockOllama) ChatStream(ctx context.Context, model string, msgs []Message, cb StreamCallback) (string, error) {
    if m.err != nil {
        return "", m.err
    }
    resp := m.responses[m.calls]
    m.calls++
    for _, chunk := range strings.Split(resp, " ") {
        cb(chunk + " ")
    }
    return resp, nil
}
```

### App Builder

```go
func newTestApp(opts ...testOption) *App {
    a := &App{
        Client:    &mockOllama{},
        Stdin:     strings.NewReader(""),
        Stdout:    &bytes.Buffer{},
        Stderr:    &bytes.Buffer{},
        Clipboard: &mockClipboard{},
        IsTTY:     true,
    }
    for _, opt := range opts {
        opt(a)
    }
    return a
}

func withResponses(r ...string) testOption { ... }
func withStdin(input string) testOption { ... }
func withTTY(isTTY bool) testOption { ... }
```

## Integration Tests

Located in `integration_test.go`. Test `App.Run()` with mocked dependencies.

| Test Name | Scenario | Verification |
|-----------|----------|--------------|
| `TestRun_SingleTurnComplete` | Response contains code block | Loop exits, stdout has response |
| `TestRun_MultiTurnConversation` | First response asks question, second completes | Both responses in stdout, messages accumulate |
| `TestRun_PipeMode` | IsTTY=false | Adds "Generate without questions" prefix, exits after one response |
| `TestRun_QuietMode` | Quiet flag set | Only final code block in stdout |
| `TestRun_OllamaError` | Client returns error | Returns error, appropriate message |
| `TestRun_ContextCanceled` | Context canceled mid-stream | Exits cleanly without panic |
| `TestCommand_Copy` | User types `/copy` | Clipboard.Write called with last response |
| `TestCommand_CopyNoResponse` | `/copy` before any response | Error message, no crash |
| `TestCommand_Help` | User types `/help` | Help text in stdout |
| `TestCommand_Quit` | User types `/quit` | Loop exits cleanly |
| `TestCommand_Unknown` | User types `/foo` | Error message about unknown command |

## E2E Tests

Located in `e2e_test.go` with `//go:build e2e` tag. Test the built binary with real Ollama.

### Binary Building

```go
var testBinary string

func TestMain(m *testing.M) {
    tmp, _ := os.MkdirTemp("", "prompt-builder-test")
    testBinary = filepath.Join(tmp, "prompt-builder")

    cmd := exec.Command("go", "build", "-o", testBinary, ".")
    if err := cmd.Run(); err != nil {
        fmt.Fprintf(os.Stderr, "build failed: %v\n", err)
        os.Exit(1)
    }

    code := m.Run()
    os.RemoveAll(tmp)
    os.Exit(code)
}
```

### Ollama Availability

```go
func ollamaAvailable() bool {
    host := os.Getenv("OLLAMA_HOST")
    if host == "" {
        host = "http://localhost:11434"
    }
    resp, err := http.Get(host + "/api/tags")
    return err == nil && resp.StatusCode == 200
}

func smallestModel() string {
    // Query /api/tags, sort by size, return smallest
}
```

Tests skip (not fail) if Ollama is unavailable.

### Test Cases

| Test Name | Scenario | Verification |
|-----------|----------|--------------|
| `TestE2E_Help` | `--help` flag | Exit 0, usage text in stdout |
| `TestE2E_Version` | `--version` flag | Exit 0, version in stdout |
| `TestE2E_MissingIdea` | No argument | Exit 1, error in stderr |
| `TestE2E_OllamaUnreachable` | Bad OLLAMA_HOST | Exit 2, connection error message |
| `TestE2E_CustomConfig` | `--config` with temp file | Uses config values |
| `TestE2E_FullConversation` | Real Ollama, simple idea | Exit 0, response in stdout |
| `TestE2E_PipeMode` | Echo idea through pipe | Quiet output, exits after response |

## Implementation Order

1. Create project CLAUDE.md
2. Define interfaces (`OllamaClient`, `ClipboardWriter`)
3. Refactor `main.go` to use `App` struct with DI
4. Ensure existing tests still pass
5. Add integration tests
6. Add E2E tests with build tag

Refactoring is the critical path - integration and E2E tests depend on the `App` struct being testable.
