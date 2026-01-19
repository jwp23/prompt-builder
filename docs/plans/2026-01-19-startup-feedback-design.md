# Startup Feedback Design

## Problem

When a user runs `./prompt-builder "idea"`, there is a pause before the chat window opens. The user receives no feedback during this time, making it unclear whether the tool is working or frozen.

The pause occurs during the first LLM request. Two factors contribute:

1. **Model loading** - Ollama loads the model into memory if not already resident
2. **Time-to-first-token** - The model processes the system prompt and user message

## Solution

Display a status message before the first LLM call. Detect whether the model is loaded and show an appropriate message:

| State | Message |
|-------|---------|
| Model not loaded | `Loading <model>...` |
| Model loaded | `Thinking...` |
| Detection failed | `Connecting...` |

The message prints on its own line. Streaming output follows below it.

## Detection

Add a method to `OllamaClient`:

```go
func (c *OllamaClient) IsModelLoaded() (bool, error)
```

This method calls `GET /api/ps` and checks whether the configured model appears in the response. Ollama's `/api/ps` endpoint returns currently loaded models:

```json
{"models":[{"name":"gpt-oss:20b",...}]}
```

Match the model name exactly against `c.Model`.

## Integration

In `main.go`, before calling `ChatStream`:

1. Call `client.IsModelLoaded()`
2. Print the appropriate message (unless `--quiet` is set or output is piped)
3. Proceed with `ChatStream` - response streams below the message

The pre-flight check is best-effort. If it fails, show "Connecting..." and let `ChatStream` proceed. If Ollama is down, `ChatStream` will return a clear error.

## Messages

"Loading <model>..." tells the user the model must load into memory first. This can take several seconds for large models.

"Thinking..." tells the user the model is ready and processing. Expect a faster response.

"Connecting..." tells the user we could not determine Ollama's state. The tool will attempt to proceed. If Ollama is unreachable, the subsequent error will explain why.

## Scope

- Interactive mode (TTY) only - pipe mode stays silent
- Respects `--quiet` flag
- No animation or progress updates in this iteration

## Testing

1. `IsModelLoaded()` returns `true` when model appears in `/api/ps` response
2. `IsModelLoaded()` returns `false` when model is absent
3. `IsModelLoaded()` returns error when API call fails
4. Correct message prints for each state (use fake HTTP server)

Follow existing patterns in `ollama_test.go` for faking the Ollama server.
