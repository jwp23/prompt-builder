# prompt-builder

Transform ideas into structured prompts using a local LLM.

## Quick Start

**1. Install Ollama and pull a model:**

ollama pull llama3.2
```

**2. Build:**

```bash
git clone https://github.com/mordant23/prompt-builder.git
cd prompt-builder
go build -o prompt-builder .
```

**3. Configure:**

```bash
mkdir -p ~/.config/prompt-builder

cat > ~/.config/prompt-builder/config.yaml << 'EOF'
model: llama3.2
system_prompt_file: ~/.config/prompt-builder/system-prompt.md
EOF

cat > ~/.config/prompt-builder/system-prompt.md << 'EOF'
You help create well-structured prompts. Ask clarifying questions,
then provide the final prompt in a code block.
EOF
```

**4. Run:**

```bash
./prompt-builder "a prompt for writing technical docs"
```

The tool asks clarifying questions, generates a structured prompt, and copies it to your clipboard.

## Usage

```
prompt-builder [flags] <idea>
```

| Flag | Short | Description |
|------|-------|-------------|
| `--model` | `-m` | Override model |
| `--config` | `-c` | Use alternate config file |
| `--no-copy` | | Skip clipboard copy |
| `--quiet` | `-q` | Output only the final prompt |
| `--version` | `-v` | Show version |
| `--help` | `-h` | Show help |

### Examples

```bash
# Interactive (default)
./prompt-builder "blog post about productivity"

# Different model
./prompt-builder -m mistral "sales email template"

# Save to file
./prompt-builder -q "api documentation" > prompt.md

# Pipe without clipboard
./prompt-builder "code review checklist" --no-copy > review.md
```

## Configuration

Create `~/.config/prompt-builder/config.yaml`:

```yaml
model: llama3.2
system_prompt_file: ~/.config/prompt-builder/system-prompt.md

# Optional
ollama_host: http://localhost:11434
clipboard_cmd: wl-copy
```

The tool auto-detects your clipboard command: `wl-copy` (Wayland), `xclip` (X11), or `pbcopy` (macOS).

## How It Works

1. You provide an idea
2. The LLM asks clarifying questions
3. You answer until the prompt is ready (use slash commands like `/copy` or `/help` anytime)
4. The tool extracts and copies the final prompt

The conversation ends when the response contains a code block and no trailing question.

When piped to another command, the tool generates immediately without questions.

## Interactive Commands

During a conversation, you can use these slash commands:

| Command | Action |
|---------|--------|
| `/copy` | Copy last code block to clipboard |
| `/bye` | Exit conversation |
| `/quit` | Exit conversation |
| `/exit` | Exit conversation |
| `/help` | List available commands |

Commands are case-insensitive (`/COPY`, `/Copy`, `/copy` all work).

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

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Config error |
| 2 | Ollama connection failed |
| 3 | No model specified |
| 130 | Interrupted (Ctrl+C) |

## Requirements

- Go 1.21+
- Ollama
- A clipboard tool (auto-detected)
