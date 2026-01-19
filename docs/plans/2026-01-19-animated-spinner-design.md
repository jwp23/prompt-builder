# Animated Loading Spinner Design

Add animated spinners for loading feedback during model startup and thinking phases.

## Requirements

- Spinner animation for "Loading \<model\>..." while waiting for model to load
- Separate spinner for "Thinking..." while waiting for first response token
- Clean transitions: each status replaces the previous on the same line
- Line clears completely when response starts streaming
- No animation in non-TTY or quiet mode

## Spinner Component

New file `spinner.go` with a reusable `Spinner` type:

```go
type Spinner struct {
    frames   []rune
    interval time.Duration
    message  string
    stopCh   chan struct{}
    doneCh   chan struct{}
}
```

**Style:** Braille dots `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`

**Speed:** ~120ms per frame

**Behavior:**
- `Start()` launches a goroutine that prints `\r<frame> <message>` cycling through frames
- `Stop()` signals stop and clears line with `\r` + spaces + `\r`
- In non-TTY mode, `Start()` prints message once without animation, `Stop()` is no-op

## Loading Model Flow

Replace `printStartupStatus()` with `waitForModel(client *OllamaClient, quiet bool, tty bool) error`:

1. If quiet or !tty: return immediately
2. Check if model loaded via `IsModelLoaded()`
   - Already loaded: skip to thinking phase
   - Error: show "Connecting..." spinner, retry with backoff
   - Not loaded: show "Loading \<model\>..." spinner
3. Poll `IsModelLoaded()` every 500ms until model appears
4. Stop spinner (clears line)
5. Return nil

**Timeout:** 30 seconds. If Ollama unreachable after timeout, exit with error message like "Could not connect to Ollama at localhost:11434".

## Thinking Phase

New method `StreamCompletionWithSpinner(prompt string, tty bool, callback func(string)) error`:

```go
func (c *OllamaClient) StreamCompletionWithSpinner(prompt string, tty bool, callback func(string)) error {
    var spinner *Spinner
    var once sync.Once

    if tty {
        spinner = NewSpinner("Thinking...")
        spinner.Start()
    }

    wrappedCallback := func(token string) {
        once.Do(func() {
            if spinner != nil {
                spinner.Stop()
            }
        })
        callback(token)
    }

    return c.streamCompletionInternal(prompt, wrappedCallback)
}
```

`sync.Once` ensures spinner stops exactly once on first token.

## Overall Flow

```
main()
├── Parse flags, load config, create client
├── Check if TTY
└── Conversation loop:
    ├── Read user input
    ├── First iteration only:
    │   └── waitForModel()  → "Loading..." or "Connecting..."
    │       └── Error → print message, exit
    ├── StreamCompletionWithSpinner()  → "Thinking..."
    └── Print response tokens
```

## Files Changed

- `spinner.go` (new) - Spinner type and animation logic
- `main.go` - Use waitForModel and StreamCompletionWithSpinner
- `ollama.go` - Add StreamCompletionWithSpinner method

## Testing

**spinner.go:**
- `Stop()` callable multiple times without panic
- `Stop()` safe if `Start()` never called
- Frame cycling logic

**waitForModel:**
- Timeout behavior (client never returns loaded)
- Immediate success (model already loaded)
- Retry logic (connection fails then succeeds)
- Quiet mode skips everything

**StreamCompletionWithSpinner:**
- Spinner stops on first token (Stop called exactly once)
- Non-TTY mode doesn't create spinner

Use existing httptest patterns from `ollama_test.go`.
