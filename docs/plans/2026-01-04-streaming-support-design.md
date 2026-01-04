# Streaming Support Design

## Problem

The interactive chat feels slow because the current implementation uses non-streaming mode (`Stream: false`). Users see nothing while the GPU works, then the entire response appears at once. This creates poor UX even when token generation is fast.

## Solution

Implement streaming mode in the Ollama client so tokens display as they're generated.

## API Changes

Add a streaming method to `OllamaClient`:

```go
type StreamCallback func(token string) error

func (c *OllamaClient) ChatStream(messages []Message, onToken StreamCallback) (string, error)
```

**Behavior:**
- Sets `Stream: true` in the request
- Reads newline-delimited JSON chunks from the response body
- Calls `onToken(token)` for each chunk's content
- Accumulates and returns the full response when done (needed for conversation history)
- If `onToken` returns an error, stops streaming and returns that error

## Ollama Streaming Format

```json
{"message":{"role":"assistant","content":"Hello"},"done":false}
{"message":{"role":"assistant","content":" there"},"done":false}
{"message":{"role":"assistant","content":"!"},"done":true}
```

Each chunk has partial content. The `done: true` chunk signals completion.

## Implementation

**New struct for streaming responses:**

```go
type OllamaStreamChunk struct {
    Message Message `json:"message"`
    Done    bool    `json:"done"`
}
```

**Implementation flow:**
1. POST to `/api/chat` with `Stream: true`
2. Create a `bufio.Scanner` on `resp.Body`
3. For each line:
   - Unmarshal into `OllamaStreamChunk`
   - Call `onToken(chunk.Message.Content)`
   - Append content to accumulator
   - If `chunk.Done`, break
4. Return accumulated full response

**Error handling:**
- HTTP errors: return immediately
- JSON parse errors on a chunk: return error with context
- Callback errors: stop streaming, return the error
- Connection drops mid-stream: scanner returns error, propagate it

## Integration in main.go

Replace the current `Chat()` call:

```go
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

**Key points:**
- Callback prints each token with `fmt.Print` (tokens include their own whitespace)
- After streaming, print newline so `> ` prompt starts on fresh line
- In quiet mode, skip printing (still accumulate for extraction)
- Pipe mode keeps streaming for incremental downstream processing

## Testing Strategy

**Unit tests for `ChatStream`:**
- Fake HTTP server returning chunked responses
- Verify callback receives tokens in order
- Verify accumulated response matches full content
- Error cases: malformed JSON, connection drop, callback error

**Test helper:**

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

## Files to Modify

- `ollama.go` - Add `OllamaStreamChunk` struct and `ChatStream` method
- `main.go` - Replace `Chat()` call with `ChatStream()` and callback
- `ollama_test.go` - Add streaming tests with fake server

## Out of Scope

- Issue 2 (model answering its own prompt) - separate system prompt change
- Progress spinner for initial connection
- Cancellation support via context
