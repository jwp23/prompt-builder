# Animated Loading Spinner Design

Add animated spinners for thinking feedback during LLM inference.

## Key Discovery

Ollama uses **lazy loading**: models don't load until the first request. Pre-checking model status with `/api/ps` doesn't work because:
- Empty model list doesn't mean the model isn't available
- Model only appears in `/api/ps` after it's loaded
- Loading happens during the first chat request, not before

**Result:** Combined loading and inference feedback into a single "Thinking..." spinner.

## Requirements

- Spinner animation for "Thinking..." while waiting for response (covers both model loading and inference)
- Clean transitions: spinner replaces on same line
- Line clears completely when response starts streaming
- No animation in non-TTY or quiet mode

## Spinner Component

File `spinner.go` with a reusable `Spinner` type:

```go
type Spinner struct {
    frames   []rune
    interval time.Duration
    message  string
    tty      bool
    stopCh   chan struct{}
    doneCh   chan struct{}
}
```

**Style:** Braille dots `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`

**Speed:** ~120ms per frame

**Behavior:**
- `Start()` launches a goroutine that prints `\r<frame> <message>` cycling through frames (only if TTY)
- `Stop()` signals stop and clears line with `\r` + spaces + `\r`
- Safe to call multiple times without panic
- In non-TTY mode, `Start()` is a no-op and `Stop()` is safe

## Thinking Phase

Method `ChatStreamWithSpinner(messages []Message, tty bool, onToken StreamCallback) (string, error)`:

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

`sync.Once` ensures spinner stops exactly once on first token.

## Overall Flow

```
main()
├── Parse flags, load config, create client
└── Conversation loop:
    ├── Read user input
    ├── ChatStreamWithSpinner()  → "Thinking..."
    │   ├── [Ollama loads model if needed]
    │   ├── [LLM generates response]
    │   └── [On first token] → spinner stops
    └── Print response tokens
```

## Files Changed

- `spinner.go` (new) - Spinner type and animation logic
- `spinner_test.go` (new) - Spinner tests
- `main.go` - Call ChatStreamWithSpinner for responses
- `ollama.go` - Add ChatStreamWithSpinner method
- Removed: `startup.go`, `startup_test.go` - WaitForModel approach not viable with lazy loading

## Testing

**spinner.go:**
- `Stop()` callable multiple times without panic
- `Stop()` safe if `Start()` never called
- Frame cycling logic
- TTY guard behavior

**ChatStreamWithSpinner:**
- Spinner stops on first token (Stop called exactly once)
- Non-TTY mode doesn't create spinner
- Returns complete response
