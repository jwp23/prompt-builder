# Interactive Commands Design

Add slash commands to the interactive conversation loop.

## Commands

| Command | Action |
|---------|--------|
| `/copy` | Copy last code block to clipboard |
| `/bye` | Exit conversation |
| `/quit` | Exit conversation |
| `/exit` | Exit conversation |
| `/help` | List available commands |

Commands are case-insensitive. Unknown commands print an error and re-prompt.

## Behavior

### /copy

Copies the most recent code block from the LLM's last response.

```
> /copy
✓ Copied to clipboard
>
```

Error cases:
- No response yet: "No response to copy from"
- No code block in response: "No code block to copy"
- Clipboard unavailable: "Clipboard not available"

### Exit commands

All three (`/bye`, `/quit`, `/exit`) print "Goodbye" and exit with code 0.

### /help

```
> /help
Commands:
  /copy   Copy last code block to clipboard
  /bye    Exit conversation
  /quit   Exit conversation
  /exit   Exit conversation
  /help   Show this help
>
```

### Unknown commands

```
> /foo
Unknown command: /foo. Type /help for available commands.
>
```

## Implementation

### New file: commands.go

- `IsCommand(input string) bool` - Returns true if input starts with `/`
- `HandleCommand(input, lastResponse, clipboardCmd string) (shouldExit bool, err error)` - Executes command

### Changes to main.go

Insert command handling after reading user input (line 183):

```go
userInput = strings.TrimSpace(userInput)

if IsCommand(userInput) {
    shouldExit, err := HandleCommand(userInput, response, clipboardCmd)
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
    }
    if shouldExit {
        return nil
    }
    fmt.Print("> ")
    continue
}
```

### Code block extraction

Reuse `ExtractLastCodeBlock()` from detect.go.

## Tests (commands_test.go)

- `TestIsCommand` - `/copy` returns true, normal text returns false
- `TestHandleCommand` - Each command behaves correctly
- `TestHandleCommandCopy` - Copy with/without code blocks
- `TestHandleCommandUnknown` - Error for unknown commands
- `TestCommandCaseInsensitive` - `/COPY`, `/Copy` work
