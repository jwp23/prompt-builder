# prompt-builder

Go CLI tool that transforms ideas into structured prompts using local LLM.

## Testing

This project uses three levels of testing:

### Unit Tests

- Test individual functions in isolation
- Located in `*_test.go` files alongside source
- Run: `go test ./cmd/prompt-builder`

### Integration Tests

- Test `App.Run()` with mocked dependencies (OllamaClient, stdin/stdout)
- Located in `integration_test.go`
- Run with unit tests (no build tag needed)
- Use for: conversation loop, commands, error handling, pipe/TTY modes

### E2E Tests

- Test the built binary with real Ollama
- Located in `e2e_test.go` with `//go:build e2e` tag
- Run: `go test -tags=e2e ./cmd/prompt-builder`
- Requires: Ollama running locally (or set OLLAMA_HOST)
- Skip gracefully if Ollama unavailable

### When to Write Each Type

| Change | Unit | Integration | E2E |
|--------|------|-------------|-----|
| New utility function | ✓ | | |
| New interactive command | ✓ | ✓ | |
| Conversation loop change | | ✓ | |
| New CLI flag | | | ✓ |
| Exit code change | | | ✓ |
| Ollama API change | ✓ | ✓ | ✓ |
