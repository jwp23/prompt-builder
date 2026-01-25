# File Consolidation Design

## Problem

The `cmd/prompt-builder/` directory has 10 source files for ~900 lines of code. Many files are small (22-40 lines) and the fragmentation makes it hard to understand where to look for specific functionality.

## Goal

Reduce cognitive load by consolidating related files, organized by domain. A future reader should understand where to go without reading the entire codebase.

## Final Structure

```
cmd/prompt-builder/
├── main.go              # Entry point, CLI, Deps, app orchestration
├── main_test.go
├── ollama.go            # OllamaChatter, OllamaClient, Message, Conversation, Spinner
├── ollama_test.go
├── slash.go             # ClipboardWriter, slash commands, clipboard, code block extraction
├── slash_test.go
├── config.go            # Config loading, path expansion (unchanged)
├── config_test.go       # (unchanged)
├── integration_test.go  # Full app flow tests (unchanged)
├── e2e_test.go          # Binary tests (unchanged)
├── testhelpers_test.go  # Shared test utilities (unchanged)
└── testdata/            # Test fixtures (unchanged)
```

## File Consolidation Map

### main.go

Absorbs: `interfaces.go` (partial), `conversation.go` (no - goes to ollama.go)

**Keeps:**
- `CLI` struct, `parseArgs()`
- `defaultConfigPath()`, `isTTY()`
- `run()`, `runWithDeps()`
- `main()`

**Moves in from interfaces.go:**
- `Deps` struct

### ollama.go

Absorbs: `spinner.go`, `conversation.go`, partial `interfaces.go`

**Keeps:**
- `Message`, `OllamaRequest`, `OllamaStreamChunk`, `OllamaPsResponse`
- `StreamCallback`
- `OllamaClient` and all methods

**Moves in from interfaces.go:**
- `OllamaChatter` interface

**Moves in from conversation.go:**
- `Conversation` struct and methods

**Moves in from spinner.go:**
- `Spinner` struct and methods

### slash.go (renamed from commands.go)

Absorbs: `clipboard.go`, `detect.go`, partial `interfaces.go`

**Keeps:**
- `IsCommand()`, `parseCommand()`
- `HandleCommand()`, `HandleCommandWithClipboard()`

**Moves in from interfaces.go:**
- `ClipboardWriter` interface
- `clipboardFunc` struct
- `NewClipboardWriter()`

**Moves in from clipboard.go:**
- `CopyToClipboard()`
- `DetectClipboardCmd()`

**Moves in from detect.go:**
- `ExtractLastCodeBlock()`
- `IsComplete()`

### config.go

Unchanged.

## Test File Consolidation

Tests follow their source files:

| Deleted Test File | Tests Move To |
|-------------------|---------------|
| `interfaces_test.go` | Split: Deps tests → `main_test.go`, ClipboardWriter tests → `slash_test.go` |
| `conversation_test.go` | `ollama_test.go` |
| `spinner_test.go` | `ollama_test.go` |
| `clipboard_test.go` | `slash_test.go` |
| `detect_test.go` | `slash_test.go` |
| `commands_test.go` | Rename to `slash_test.go` |

## Files Deleted

Source files:
- `interfaces.go`
- `conversation.go`
- `spinner.go`
- `clipboard.go`
- `detect.go`
- `commands.go` (content moves to `slash.go`)

Test files:
- `interfaces_test.go`
- `conversation_test.go`
- `spinner_test.go`
- `clipboard_test.go`
- `detect_test.go`
- `commands_test.go` (content moves to `slash_test.go`)

## Design Decisions

1. **Domain grouping over technical grouping** - Files organized by what they do for the user, not by technical concern.

2. **Interfaces live with implementations** - `OllamaChatter` in `ollama.go`, `ClipboardWriter` in `slash.go`. Easier to find and maintain.

3. **Spinner stays in ollama.go** - It's only used during LLM response waiting. Keeping it with the code that uses it is pragmatic.

4. **No separate tests folder** - Go convention is tests next to code. Build tags handle e2e separation.

5. **slash.go naming** - Distinguishes interactive `/commands` from CLI arguments in `main.go`.
