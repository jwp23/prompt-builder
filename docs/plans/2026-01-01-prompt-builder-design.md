# prompt-builder Design

A CLI tool that transforms simple ideas into structured prompts using the R.G.C.O.A. framework via a local LLM.

## Overview & Architecture

**prompt-builder** is a Go CLI that uses a local LLM (via Ollama) to transform simple ideas into structured prompts using the R.G.C.O.A. framework.

**Core flow:**
```
┌─────────────────┐     ┌─────────────┐     ┌─────────────┐
│  User runs CLI  │────▶│   Ollama    │────▶│ Final prompt│
│  with idea      │◀────│   (LLM)     │     │ to clipboard│
└─────────────────┘     └─────────────┘     └─────────────┘
        │                      │
        │    conversation      │
        └──────────────────────┘
```

**Components:**
- **CLI binary** (`prompt-builder`) - handles args, config, I/O
- **Ollama client** - HTTP calls to `localhost:11434`
- **Conversation loop** - manages back-and-forth with LLM
- **Output detector** - identifies final prompt (code block + no trailing question)
- **Clipboard writer** - auto-copies via `wl-copy` (with fallback detection)

**Dependencies:**
- Go standard library (HTTP, JSON, I/O)
- `gopkg.in/yaml.v3` for config parsing
- No heavy frameworks - keep it minimal

**File structure:**
```
prompt-builder/
├── main.go           # Entry point, arg parsing
├── config.go         # Config loading
├── ollama.go         # Ollama API client
├── conversation.go   # Conversation loop logic
├── clipboard.go      # Clipboard integration
└── detect.go         # Final prompt detection
```

## CLI Interface

**Basic usage:**
```bash
prompt-builder "I want a custom diet plan"
```

**Flags:**
```
prompt-builder [flags] <idea>

Flags:
  --model, -m <name>    Override model from config (e.g., -m mistral)
  --config, -c <path>   Use alternate config file
  --no-copy             Print final prompt but don't copy to clipboard
  --quiet, -q           Suppress conversation, only output final prompt
  --help, -h            Show help
  --version, -v         Show version
```

**Behavior modes:**

| Scenario | Conversation | Final Prompt | Clipboard |
|----------|--------------|--------------|-----------|
| TTY (normal) | visible | visible | auto-copy |
| TTY + `--no-copy` | visible | visible | skip |
| TTY + `--quiet` | hidden | visible | auto-copy |
| Piped (`| cat`) | hidden | stdout only | skip |

**Examples:**
```bash
# Normal interactive use
prompt-builder "blog post about productivity"

# Override model
prompt-builder -m mistral "sales email"

# Scripting - just get the prompt
prompt-builder -q "api documentation" > prompt.md

# Pipe to file without clipboard
prompt-builder "code review checklist" --no-copy > review.md
```

**Exit codes:**
- `0` - success
- `1` - config error (missing file, invalid YAML)
- `2` - Ollama connection failed
- `3` - no model configured/specified

## Configuration

**Location:** `~/.config/prompt-builder/config.yaml`

**Config schema:**
```yaml
# Required
model: llama3.2
system_prompt_file: ~/.config/prompt-builder/prompt-architect.md

# Optional
ollama_host: http://localhost:11434  # default
clipboard_cmd: wl-copy               # auto-detected if omitted
```

**System prompt file:** `~/.config/prompt-builder/prompt-architect.md`
- Plain markdown file containing the R.G.C.O.A. system prompt
- Easy to edit with any text editor
- Can swap different system prompts by changing the path

**Config resolution order:**
1. CLI flags (highest priority)
2. Config file
3. Defaults (lowest priority)

**Defaults:**
- `ollama_host`: `http://localhost:11434`
- `clipboard_cmd`: auto-detect (`wl-copy` → `xclip` → `pbcopy`)

**Missing config behavior:**
```
$ prompt-builder "idea"
Error: config file not found: ~/.config/prompt-builder/config.yaml

Create it with:
  mkdir -p ~/.config/prompt-builder
  cat > ~/.config/prompt-builder/config.yaml << 'EOF'
  model: llama3.2
  system_prompt_file: ~/.config/prompt-builder/prompt-architect.md
  EOF
```

**Missing model behavior:**
```
$ prompt-builder "idea"
Error: no model specified

Set 'model' in config or use --model flag
```

## Conversation Flow

**Initialization:**
1. Parse CLI args
2. Load config file
3. Read system prompt from file
4. Connect to Ollama, verify model exists

**Conversation loop:**
```
┌─────────────────────────────────────────────────┐
│ Send: system prompt + user's idea               │
└──────────────────────┬──────────────────────────┘
                       ▼
┌─────────────────────────────────────────────────┐
│ Receive LLM response                            │
├─────────────────────────────────────────────────┤
│ Has code block + no trailing question?          │
│   YES → Extract prompt, done                    │
│   NO  → Print response, wait for user input     │
└──────────────────────┬──────────────────────────┘
                       ▼
┌─────────────────────────────────────────────────┐
│ User types response (stdin)                     │
│ Send to LLM, loop back                          │
└─────────────────────────────────────────────────┘
```

**"Done" detection logic:**
```go
func isComplete(response string) bool {
    hasCodeBlock := strings.Contains(response, "```")
    endsWithQuestion := strings.HasSuffix(strings.TrimSpace(response), "?")
    return hasCodeBlock && !endsWithQuestion
}
```

**Pipe mode (non-TTY):**
- Prepend to user's idea: "Generate your best prompt without asking clarifying questions."
- Single request/response, no loop
- Extract code block, output to stdout

**Ctrl+C handling:**
- Graceful shutdown
- No clipboard write on interrupt

## Output Handling

**Code block extraction:**
```go
func extractLastCodeBlock(response string) string {
    // Find all ``` fenced blocks
    // Return contents of the last one (without the ``` markers)
    // Preserve inner formatting (markdown, etc.)
}
```

**Example extraction:**
```
LLM output:
  Here is your final prompt:
  ```
  # Role
  You are a nutrition expert...

  # Goal
  Create a personalized diet plan...
  ```

Extracted:
  # Role
  You are a nutrition expert...

  # Goal
  Create a personalized diet plan...
```

**Clipboard auto-detection:**
```go
func detectClipboardCmd() string {
    // Check in order:
    // 1. Config override (clipboard_cmd)
    // 2. wl-copy (Wayland)
    // 3. xclip -selection clipboard (X11)
    // 4. xsel --clipboard (X11 fallback)
    // 5. pbcopy (macOS)
    // Return empty string if none found
}
```

**Clipboard failure handling:**
- If clipboard command not found: warn but still print prompt
- If clipboard command fails: warn but still print prompt
- User always sees the prompt regardless of clipboard status

**TTY output format:**
```
$ prompt-builder "diet plan"
What dietary restrictions do you have?
> vegetarian, no nuts

Here is your final prompt:
[prompt displayed]

✓ Copied to clipboard
```

## Error Handling

**Startup errors:**

| Error | Message | Exit Code |
|-------|---------|-----------|
| Config not found | `Error: config not found: <path>` + creation example | 1 |
| Invalid YAML | `Error: invalid config: <parse error>` | 1 |
| System prompt file not found | `Error: system prompt not found: <path>` | 1 |
| Model not specified | `Error: no model specified` + usage hint | 3 |

**Runtime errors:**

| Error | Message | Exit Code |
|-------|---------|-----------|
| Ollama not running | `Error: cannot connect to Ollama at <host>` | 2 |
| Model not installed | `Error: model '<name>' not found in Ollama` + `ollama pull <name>` hint | 2 |
| Ollama request failed | `Error: Ollama request failed: <details>` | 2 |

**Graceful degradation:**
- Clipboard unavailable → warn, continue: `Warning: clipboard not available, prompt printed above`
- Clipboard command fails → warn, continue: `Warning: clipboard copy failed: <error>`

**Interrupt handling (Ctrl+C):**
- Clean exit, no error message
- No partial clipboard write
- Exit code 130 (standard for SIGINT)

**Timeout:**
- No hard timeout on LLM responses (could be slow on large models)
- User can Ctrl+C if stuck
