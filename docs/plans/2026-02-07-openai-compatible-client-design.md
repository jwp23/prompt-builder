# OpenAI-Compatible Client Design

## Problem

prompt-builder uses Ollama's proprietary `/api/chat` endpoint. Ollama wraps llama.cpp and adds 10-30% overhead in throughput and 2x slower model loading. Users cannot switch to faster alternatives (llama.cpp server, LM Studio, vLLM) without code changes.

## Solution

Replace the Ollama-specific client with one that speaks the OpenAI chat completions protocol (`/v1/chat/completions`). This protocol is supported by Ollama, llama.cpp, LM Studio, vLLM, and any OpenAI-compatible server. One client supports all backends.

## API Protocol

**Request** (POST `/v1/chat/completions`):
```json
{"model": "llama3.2", "messages": [{"role": "...", "content": "..."}], "stream": true}
```

**Response** — Server-Sent Events (SSE):
```
data: {"choices":[{"delta":{"content":"Hello"},"finish_reason":null}]}

data: {"choices":[{"delta":{},"finish_reason":"stop"}]}

data: [DONE]
```

Each line has a `data: ` prefix. Tokens live in `choices[0].delta.content`. The stream ends with `data: [DONE]`.

## Config Changes

`ollama_host` becomes `host`. Default stays `http://localhost:11434` so existing Ollama users need only rename the field.

Before:
```yaml
model: llama3.2
system_prompt_file: ~/.config/prompt-builder/prompt-architect.md
ollama_host: http://localhost:11434
```

After:
```yaml
model: llama3.2
system_prompt_file: ~/.config/prompt-builder/prompt-architect.md
host: http://localhost:11434
```

This is a breaking config change. Tag as v2.0.0.

## Renames

| Before | After | Reason |
|--------|-------|--------|
| `OllamaChatter` | `LLMClient` | Backend-agnostic |
| `OllamaClient` | `ChatClient` | Backend-agnostic |
| `OllamaRequest` | `ChatRequest` | Matches protocol |
| `OllamaStreamChunk` | `ChatStreamChunk` | Matches protocol |
| `ollama.go` | `llm.go` | Backend-agnostic |
| `ollama_test.go` | `llm_test.go` | Follows source file |

## Deletions

- `IsModelLoaded()` — dead code, never called from production path, and Ollama-specific (`/api/ps` has no OpenAI equivalent)
- `OllamaPsModel` and `OllamaPsResponse` — support types for `IsModelLoaded`
- Five `IsModelLoaded` tests

## What Stays the Same

- `Message` struct — `role` + `content` is identical across both protocols
- `Conversation` — manages messages, protocol-agnostic
- `Spinner` and `ChatStreamWithSpinner` — UI layer, no API coupling
- `StreamCallback` type
- `runWithDeps` and the conversation loop — uses the interface
- All integration tests — mock the interface, not the HTTP layer

## New Types

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

## Streaming Parse Logic

The SSE parsing replaces Ollama's newline-delimited JSON:

1. Read lines from response body
2. Skip empty lines (SSE uses blank lines as delimiters)
3. Strip `data: ` prefix
4. If remainder is `[DONE]`, stop
5. Unmarshal JSON into `ChatStreamChunk`
6. Extract `choices[0].delta.content` and pass to callback

## File Impact

| File | Change |
|------|--------|
| `ollama.go` → `llm.go` | Rewrite `ChatStream` for SSE, rename types, delete `IsModelLoaded` |
| `ollama_test.go` → `llm_test.go` | Update test servers to return SSE, delete `IsModelLoaded` tests |
| `config.go` | Rename `OllamaHost` to `Host` |
| `main.go` | Rename interface references, update error messages |

## Version

Breaking change (config field rename, public interface rename). Tag as **v2.0.0**.

## Testing

- Unit: `ChatStream` parses SSE correctly, handles `[DONE]`, handles empty deltas, handles errors
- Unit: Config loads `host` field with correct default
- Integration: Conversation loop works with new client through `LLMClient` interface
- E2E: Built binary connects to local server (Ollama or llama.cpp)
