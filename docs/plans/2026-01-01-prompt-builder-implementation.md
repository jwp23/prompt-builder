# prompt-builder Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a Go CLI that uses Ollama to transform simple ideas into structured R.G.C.O.A. prompts with auto-clipboard.

**Architecture:** Single binary CLI with modular components - config loader, Ollama HTTP client, conversation loop, output detector, and clipboard writer. Each component is independently testable.

**Tech Stack:** Go 1.21+, gopkg.in/yaml.v3, Go standard library (net/http, os, os/exec, bufio)

---

## Task 1: Project Setup

**Files:**
- Create: `go.mod`
- Create: `go.sum`
- Create: `main.go`

**Step 1: Initialize Go module**

Run:
```bash
go mod init github.com/mordant23/prompt-builder
```
Expected: `go.mod` created with module name

**Step 2: Add YAML dependency**

Run:
```bash
go get gopkg.in/yaml.v3
```
Expected: `go.sum` created, yaml.v3 added to go.mod

**Step 3: Create minimal main.go**

```go
package main

import "fmt"

func main() {
	fmt.Println("prompt-builder")
}
```

**Step 4: Verify build works**

Run:
```bash
go build -o prompt-builder .
./prompt-builder
```
Expected: Prints "prompt-builder"

**Step 5: Commit**

```bash
git add go.mod go.sum main.go
git commit -m "chore: initialize Go module with yaml dependency"
```

---

## Task 2: Config Types and Loading

**Files:**
- Create: `config.go`
- Create: `config_test.go`

**Step 1: Write failing test for config parsing**

```go
// config_test.go
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_ValidFile(t *testing.T) {
	// Create temp config file
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := `model: llama3.2
system_prompt_file: /path/to/prompt.md
ollama_host: http://localhost:11434
clipboard_cmd: wl-copy
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Model != "llama3.2" {
		t.Errorf("Model = %q, want %q", cfg.Model, "llama3.2")
	}
	if cfg.SystemPromptFile != "/path/to/prompt.md" {
		t.Errorf("SystemPromptFile = %q, want %q", cfg.SystemPromptFile, "/path/to/prompt.md")
	}
	if cfg.OllamaHost != "http://localhost:11434" {
		t.Errorf("OllamaHost = %q, want %q", cfg.OllamaHost, "http://localhost:11434")
	}
	if cfg.ClipboardCmd != "wl-copy" {
		t.Errorf("ClipboardCmd = %q, want %q", cfg.ClipboardCmd, "wl-copy")
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test -v -run TestLoadConfig_ValidFile
```
Expected: FAIL - `LoadConfig` not defined

**Step 3: Write minimal implementation**

```go
// config.go
package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Model            string `yaml:"model"`
	SystemPromptFile string `yaml:"system_prompt_file"`
	OllamaHost       string `yaml:"ollama_host"`
	ClipboardCmd     string `yaml:"clipboard_cmd"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test -v -run TestLoadConfig_ValidFile
```
Expected: PASS

**Step 5: Commit**

```bash
git add config.go config_test.go
git commit -m "feat: add config loading with YAML parsing"
```

---

## Task 3: Config Defaults and Validation

**Files:**
- Modify: `config.go`
- Modify: `config_test.go`

**Step 1: Write failing test for defaults**

Add to `config_test.go`:

```go
func TestLoadConfig_AppliesDefaults(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := `model: llama3.2
system_prompt_file: /path/to/prompt.md
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.OllamaHost != "http://localhost:11434" {
		t.Errorf("OllamaHost = %q, want default %q", cfg.OllamaHost, "http://localhost:11434")
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test -v -run TestLoadConfig_AppliesDefaults
```
Expected: FAIL - OllamaHost is empty, not default

**Step 3: Add defaults to implementation**

Update `LoadConfig` in `config.go`:

```go
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := Config{
		OllamaHost: "http://localhost:11434",
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
```

**Step 4: Run tests to verify they pass**

Run:
```bash
go test -v -run TestLoadConfig
```
Expected: All PASS

**Step 5: Commit**

```bash
git add config.go config_test.go
git commit -m "feat: add default values for config"
```

---

## Task 4: Config Path Resolution with Tilde Expansion

**Files:**
- Modify: `config.go`
- Modify: `config_test.go`

**Step 1: Write failing test for tilde expansion**

Add to `config_test.go`:

```go
func TestExpandPath_Tilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		input string
		want  string
	}{
		{"~/config.yaml", filepath.Join(home, "config.yaml")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}

	for _, tt := range tests {
		got := ExpandPath(tt.input)
		if got != tt.want {
			t.Errorf("ExpandPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test -v -run TestExpandPath_Tilde
```
Expected: FAIL - `ExpandPath` not defined

**Step 3: Write implementation**

Add to `config.go`:

```go
import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test -v -run TestExpandPath_Tilde
```
Expected: PASS

**Step 5: Commit**

```bash
git add config.go config_test.go
git commit -m "feat: add tilde expansion for config paths"
```

---

## Task 5: Detect Module - Code Block Extraction

**Files:**
- Create: `detect.go`
- Create: `detect_test.go`

**Step 1: Write failing test for code block extraction**

```go
// detect_test.go
package main

import "testing"

func TestExtractLastCodeBlock(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "single code block",
			input: `Here is your prompt:
` + "```" + `
# Role
You are an expert.
` + "```" + `
`,
			want: `# Role
You are an expert.
`,
		},
		{
			name: "multiple code blocks - returns last",
			input: `Example:
` + "```" + `
first block
` + "```" + `

Here is the final:
` + "```" + `
second block
` + "```" + `
`,
			want: `second block
`,
		},
		{
			name:  "no code block",
			input: "Just plain text",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractLastCodeBlock(tt.input)
			if got != tt.want {
				t.Errorf("ExtractLastCodeBlock() = %q, want %q", got, tt.want)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test -v -run TestExtractLastCodeBlock
```
Expected: FAIL - `ExtractLastCodeBlock` not defined

**Step 3: Write implementation**

```go
// detect.go
package main

import (
	"strings"
)

func ExtractLastCodeBlock(text string) string {
	const marker = "```"

	lastStart := strings.LastIndex(text, marker)
	if lastStart == -1 {
		return ""
	}

	// Find the opening marker for this block
	beforeLast := text[:lastStart]
	openStart := strings.LastIndex(beforeLast, marker)
	if openStart == -1 {
		return ""
	}

	// Extract content between markers
	// Skip past the opening ``` and any language identifier on that line
	contentStart := openStart + len(marker)
	if idx := strings.Index(text[contentStart:lastStart], "\n"); idx != -1 {
		contentStart += idx + 1
	}

	return text[contentStart:lastStart]
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test -v -run TestExtractLastCodeBlock
```
Expected: PASS

**Step 5: Commit**

```bash
git add detect.go detect_test.go
git commit -m "feat: add code block extraction"
```

---

## Task 6: Detect Module - Completion Detection

**Files:**
- Modify: `detect.go`
- Modify: `detect_test.go`

**Step 1: Write failing test for completion detection**

Add to `detect_test.go`:

```go
func TestIsComplete(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "code block without question - complete",
			input: "Here is your prompt:\n```\ncontent\n```\n",
			want:  true,
		},
		{
			name:  "code block with trailing question - not complete",
			input: "Here is a draft:\n```\ncontent\n```\nDoes this look right?",
			want:  false,
		},
		{
			name:  "question only - not complete",
			input: "What is your target audience?",
			want:  false,
		},
		{
			name:  "no code block no question - not complete",
			input: "Let me think about that.",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsComplete(tt.input)
			if got != tt.want {
				t.Errorf("IsComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test -v -run TestIsComplete
```
Expected: FAIL - `IsComplete` not defined

**Step 3: Write implementation**

Add to `detect.go`:

```go
func IsComplete(response string) bool {
	hasCodeBlock := strings.Contains(response, "```")
	trimmed := strings.TrimSpace(response)
	endsWithQuestion := strings.HasSuffix(trimmed, "?")
	return hasCodeBlock && !endsWithQuestion
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test -v -run TestIsComplete
```
Expected: PASS

**Step 5: Commit**

```bash
git add detect.go detect_test.go
git commit -m "feat: add conversation completion detection"
```

---

## Task 7: Clipboard Module

**Files:**
- Create: `clipboard.go`
- Create: `clipboard_test.go`

**Step 1: Write failing test for clipboard command detection**

```go
// clipboard_test.go
package main

import (
	"os/exec"
	"testing"
)

func TestDetectClipboardCmd(t *testing.T) {
	// This test verifies the detection logic
	// Actual availability depends on system
	cmd := DetectClipboardCmd("")

	// Should return something or empty string
	// Can't assert exact value as it's system-dependent
	t.Logf("Detected clipboard command: %q", cmd)

	// If a command is returned, it should be executable
	if cmd != "" {
		parts := strings.Split(cmd, " ")
		_, err := exec.LookPath(parts[0])
		if err != nil {
			t.Errorf("Detected command %q but binary not found", parts[0])
		}
	}
}

func TestDetectClipboardCmd_Override(t *testing.T) {
	cmd := DetectClipboardCmd("custom-clipboard")
	if cmd != "custom-clipboard" {
		t.Errorf("DetectClipboardCmd with override = %q, want %q", cmd, "custom-clipboard")
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test -v -run TestDetectClipboardCmd
```
Expected: FAIL - `DetectClipboardCmd` not defined

**Step 3: Write implementation**

```go
// clipboard.go
package main

import (
	"os/exec"
	"strings"
)

func DetectClipboardCmd(override string) string {
	if override != "" {
		return override
	}

	candidates := []string{
		"wl-copy",
		"xclip -selection clipboard",
		"xsel --clipboard --input",
		"pbcopy",
	}

	for _, cmd := range candidates {
		parts := strings.Split(cmd, " ")
		if _, err := exec.LookPath(parts[0]); err == nil {
			return cmd
		}
	}

	return ""
}

func CopyToClipboard(text string, cmd string) error {
	if cmd == "" {
		return nil // No clipboard available, silently skip
	}

	parts := strings.Split(cmd, " ")
	c := exec.Command(parts[0], parts[1:]...)
	c.Stdin = strings.NewReader(text)
	return c.Run()
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test -v -run TestDetectClipboardCmd
```
Expected: PASS

**Step 5: Commit**

```bash
git add clipboard.go clipboard_test.go
git commit -m "feat: add clipboard detection and copy"
```

---

## Task 8: Ollama Client - Types and Chat Request

**Files:**
- Create: `ollama.go`
- Create: `ollama_test.go`

**Step 1: Write failing test for message serialization**

```go
// ollama_test.go
package main

import (
	"encoding/json"
	"testing"
)

func TestOllamaRequest_Serialization(t *testing.T) {
	req := OllamaRequest{
		Model: "llama3.2",
		Messages: []Message{
			{Role: "system", Content: "You are helpful."},
			{Role: "user", Content: "Hello"},
		},
		Stream: false,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Verify it contains expected fields
	s := string(data)
	if !strings.Contains(s, `"model":"llama3.2"`) {
		t.Errorf("missing model field")
	}
	if !strings.Contains(s, `"role":"system"`) {
		t.Errorf("missing system role")
	}
	if !strings.Contains(s, `"stream":false`) {
		t.Errorf("missing stream field")
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test -v -run TestOllamaRequest_Serialization
```
Expected: FAIL - `OllamaRequest` not defined

**Step 3: Write implementation**

```go
// ollama.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OllamaRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type OllamaResponse struct {
	Message Message `json:"message"`
}

type OllamaClient struct {
	Host   string
	Model  string
	client *http.Client
}

func NewOllamaClient(host, model string) *OllamaClient {
	return &OllamaClient{
		Host:   host,
		Model:  model,
		client: &http.Client{},
	}
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test -v -run TestOllamaRequest_Serialization
```
Expected: PASS

**Step 5: Commit**

```bash
git add ollama.go ollama_test.go
git commit -m "feat: add Ollama client types"
```

---

## Task 9: Ollama Client - Chat Method

**Files:**
- Modify: `ollama.go`
- Modify: `ollama_test.go`

**Step 1: Write failing test for chat method**

Add to `ollama_test.go`:

```go
import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOllamaClient_Chat(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("unexpected method: %s", r.Method)
		}

		// Return mock response
		resp := OllamaResponse{
			Message: Message{Role: "assistant", Content: "Hello! How can I help?"},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "llama3.2")
	messages := []Message{
		{Role: "user", Content: "Hi"},
	}

	response, err := client.Chat(messages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response != "Hello! How can I help?" {
		t.Errorf("response = %q, want %q", response, "Hello! How can I help?")
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test -v -run TestOllamaClient_Chat
```
Expected: FAIL - `Chat` method not defined

**Step 3: Write implementation**

Add to `ollama.go`:

```go
func (c *OllamaClient) Chat(messages []Message) (string, error) {
	req := OllamaRequest{
		Model:    c.Model,
		Messages: messages,
		Stream:   false,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.client.Post(c.Host+"/api/chat", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to connect to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Ollama request failed: %s - %s", resp.Status, string(body))
	}

	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return ollamaResp.Message.Content, nil
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test -v -run TestOllamaClient_Chat
```
Expected: PASS

**Step 5: Commit**

```bash
git add ollama.go ollama_test.go
git commit -m "feat: add Ollama chat method"
```

---

## Task 10: Conversation Module

**Files:**
- Create: `conversation.go`
- Create: `conversation_test.go`

**Step 1: Write failing test for conversation state**

```go
// conversation_test.go
package main

import "testing"

func TestConversation_AddMessage(t *testing.T) {
	conv := NewConversation("You are helpful.")

	// Should start with system message
	if len(conv.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(conv.Messages))
	}
	if conv.Messages[0].Role != "system" {
		t.Errorf("first message role = %q, want %q", conv.Messages[0].Role, "system")
	}

	conv.AddUserMessage("Hello")
	if len(conv.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(conv.Messages))
	}
	if conv.Messages[1].Role != "user" {
		t.Errorf("second message role = %q, want %q", conv.Messages[1].Role, "user")
	}

	conv.AddAssistantMessage("Hi there!")
	if len(conv.Messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(conv.Messages))
	}
	if conv.Messages[2].Role != "assistant" {
		t.Errorf("third message role = %q, want %q", conv.Messages[2].Role, "assistant")
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test -v -run TestConversation_AddMessage
```
Expected: FAIL - `NewConversation` not defined

**Step 3: Write implementation**

```go
// conversation.go
package main

type Conversation struct {
	Messages []Message
}

func NewConversation(systemPrompt string) *Conversation {
	return &Conversation{
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
		},
	}
}

func (c *Conversation) AddUserMessage(content string) {
	c.Messages = append(c.Messages, Message{Role: "user", Content: content})
}

func (c *Conversation) AddAssistantMessage(content string) {
	c.Messages = append(c.Messages, Message{Role: "assistant", Content: content})
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test -v -run TestConversation_AddMessage
```
Expected: PASS

**Step 5: Commit**

```bash
git add conversation.go conversation_test.go
git commit -m "feat: add conversation state management"
```

---

## Task 11: CLI Argument Parsing

**Files:**
- Modify: `main.go`

**Step 1: Write the CLI structure**

Replace `main.go` with:

```go
package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	version = "dev"
)

type CLI struct {
	Model      string
	ConfigPath string
	NoCopy     bool
	Quiet      bool
	Idea       string
}

func parseArgs() (*CLI, error) {
	cli := &CLI{}

	flag.StringVar(&cli.Model, "model", "", "Override model from config")
	flag.StringVar(&cli.Model, "m", "", "Override model from config (shorthand)")
	flag.StringVar(&cli.ConfigPath, "config", "", "Use alternate config file")
	flag.StringVar(&cli.ConfigPath, "c", "", "Use alternate config file (shorthand)")
	flag.BoolVar(&cli.NoCopy, "no-copy", false, "Don't copy to clipboard")
	flag.BoolVar(&cli.Quiet, "quiet", false, "Suppress conversation output")
	flag.BoolVar(&cli.Quiet, "q", false, "Suppress conversation output (shorthand)")

	showVersion := flag.Bool("version", false, "Show version")
	showVersionShort := flag.Bool("v", false, "Show version (shorthand)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: prompt-builder [flags] <idea>\n\n")
		fmt.Fprintf(os.Stderr, "Transform ideas into structured prompts using R.G.C.O.A. framework.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *showVersion || *showVersionShort {
		fmt.Printf("prompt-builder %s\n", version)
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) < 1 {
		return nil, fmt.Errorf("missing required argument: <idea>")
	}
	cli.Idea = args[0]

	return cli, nil
}

func main() {
	cli, err := parseArgs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		flag.Usage()
		os.Exit(1)
	}

	fmt.Printf("Idea: %s\n", cli.Idea)
	fmt.Printf("Model: %s\n", cli.Model)
	fmt.Printf("Config: %s\n", cli.ConfigPath)
}
```

**Step 2: Verify build and basic usage**

Run:
```bash
go build -o prompt-builder . && ./prompt-builder "test idea"
```
Expected: Prints Idea, Model, Config values

Run:
```bash
./prompt-builder --help
```
Expected: Shows usage with flags

Run:
```bash
./prompt-builder -v
```
Expected: Shows version

**Step 3: Commit**

```bash
git add main.go
git commit -m "feat: add CLI argument parsing"
```

---

## Task 12: Main Integration - Config Loading

**Files:**
- Modify: `main.go`

**Step 1: Add config loading to main**

Update `main.go` - add after parseArgs:

```go
import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "prompt-builder", "config.yaml")
}

func run(cli *CLI) error {
	// Determine config path
	configPath := cli.ConfigPath
	if configPath == "" {
		configPath = defaultConfigPath()
	}
	configPath = ExpandPath(configPath)

	// Load config
	cfg, err := LoadConfig(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("config file not found: %s\n\nCreate it with:\n  mkdir -p ~/.config/prompt-builder\n  cat > ~/.config/prompt-builder/config.yaml << 'EOF'\n  model: llama3.2\n  system_prompt_file: ~/.config/prompt-builder/prompt-architect.md\n  EOF", configPath)
		}
		return fmt.Errorf("invalid config: %v", err)
	}

	// Apply CLI overrides
	if cli.Model != "" {
		cfg.Model = cli.Model
	}

	// Validate model
	if cfg.Model == "" {
		return fmt.Errorf("no model specified\n\nSet 'model' in config or use --model flag")
	}

	// Load system prompt
	promptPath := ExpandPath(cfg.SystemPromptFile)
	systemPrompt, err := os.ReadFile(promptPath)
	if err != nil {
		return fmt.Errorf("system prompt not found: %s", promptPath)
	}

	fmt.Printf("Config loaded: model=%s\n", cfg.Model)
	fmt.Printf("System prompt: %d bytes\n", len(systemPrompt))
	fmt.Printf("Idea: %s\n", cli.Idea)

	return nil
}

func main() {
	cli, err := parseArgs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		flag.Usage()
		os.Exit(1)
	}

	if err := run(cli); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

**Step 2: Verify error handling**

Run (without config):
```bash
go build -o prompt-builder . && ./prompt-builder "test"
```
Expected: Error message with config creation instructions

**Step 3: Commit**

```bash
git add main.go
git commit -m "feat: integrate config loading in main"
```

---

## Task 13: Main Integration - Conversation Loop

**Files:**
- Modify: `main.go`

**Step 1: Add conversation loop**

Update the `run` function in `main.go`:

```go
import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/term"
)

func isTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func run(cli *CLI) error {
	// ... existing config loading code ...

	// After loading system prompt, add:

	// Detect clipboard
	clipboardCmd := DetectClipboardCmd(cfg.ClipboardCmd)

	// Initialize Ollama client
	client := NewOllamaClient(cfg.OllamaHost, cfg.Model)

	// Initialize conversation
	conv := NewConversation(string(systemPrompt))

	// Prepare user's idea
	userIdea := cli.Idea
	if !isTTY() {
		// Pipe mode: ask for immediate generation
		userIdea = "Generate your best prompt without asking clarifying questions. User's idea: " + userIdea
	}
	conv.AddUserMessage(userIdea)

	// Conversation loop
	reader := bufio.NewReader(os.Stdin)
	for {
		// Get response from LLM
		response, err := client.Chat(conv.Messages)
		if err != nil {
			return fmt.Errorf("Ollama request failed: %v", err)
		}

		conv.AddAssistantMessage(response)

		// Check if conversation is complete
		if IsComplete(response) {
			// Extract final prompt
			finalPrompt := ExtractLastCodeBlock(response)

			// Print response (shows the full output including code block)
			if !cli.Quiet {
				fmt.Println(response)
			} else {
				fmt.Println(finalPrompt)
			}

			// Copy to clipboard if TTY and not disabled
			if isTTY() && !cli.NoCopy && clipboardCmd != "" {
				if err := CopyToClipboard(finalPrompt, clipboardCmd); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: clipboard copy failed: %v\n", err)
				} else {
					fmt.Fprintln(os.Stderr, "✓ Copied to clipboard")
				}
			}

			return nil
		}

		// Not complete - print response and wait for user input
		if !cli.Quiet {
			fmt.Println(response)
		}

		// In pipe mode, can't ask for input
		if !isTTY() {
			return fmt.Errorf("LLM requested clarification but stdin is not a TTY")
		}

		fmt.Print("> ")
		userInput, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %v", err)
		}

		conv.AddUserMessage(userInput)
	}
}
```

**Step 2: Add term dependency**

Run:
```bash
go get golang.org/x/term
```

**Step 3: Verify build**

Run:
```bash
go build -o prompt-builder .
```
Expected: Builds successfully

**Step 4: Commit**

```bash
git add main.go go.mod go.sum
git commit -m "feat: integrate conversation loop in main"
```

---

## Task 14: Exit Code Handling

**Files:**
- Modify: `main.go`

**Step 1: Add proper exit codes**

Update `main.go` main function:

```go
const (
	ExitSuccess      = 0
	ExitConfigError  = 1
	ExitOllamaError  = 2
	ExitNoModel      = 3
)

func main() {
	cli, err := parseArgs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		flag.Usage()
		os.Exit(ExitConfigError)
	}

	if err := run(cli); err != nil {
		errStr := err.Error()
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)

		// Determine exit code based on error type
		switch {
		case contains(errStr, "config") || contains(errStr, "system prompt"):
			os.Exit(ExitConfigError)
		case contains(errStr, "Ollama") || contains(errStr, "connect"):
			os.Exit(ExitOllamaError)
		case contains(errStr, "no model"):
			os.Exit(ExitNoModel)
		default:
			os.Exit(1)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
```

Note: Replace `contains` with `strings.Contains` - update imports:

```go
import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"
)
```

And simplify:

```go
func main() {
	cli, err := parseArgs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		flag.Usage()
		os.Exit(ExitConfigError)
	}

	if err := run(cli); err != nil {
		errStr := err.Error()
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)

		switch {
		case strings.Contains(errStr, "config") || strings.Contains(errStr, "system prompt"):
			os.Exit(ExitConfigError)
		case strings.Contains(errStr, "Ollama") || strings.Contains(errStr, "connect"):
			os.Exit(ExitOllamaError)
		case strings.Contains(errStr, "no model"):
			os.Exit(ExitNoModel)
		default:
			os.Exit(1)
		}
	}
}
```

**Step 2: Verify build**

Run:
```bash
go build -o prompt-builder .
```
Expected: Builds successfully

**Step 3: Commit**

```bash
git add main.go
git commit -m "feat: add proper exit codes"
```

---

## Task 15: Signal Handling (Ctrl+C)

**Files:**
- Modify: `main.go`

**Step 1: Add signal handling**

Add to `main.go`:

```go
import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/term"
)

func main() {
	// Set up signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
		os.Exit(130) // Standard exit code for SIGINT
	}()

	cli, err := parseArgs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		flag.Usage()
		os.Exit(ExitConfigError)
	}

	if err := run(ctx, cli); err != nil {
		// ... rest of error handling
	}
}
```

Update `run` signature to accept context (for future cancellation support):

```go
func run(ctx context.Context, cli *CLI) error {
	// ... existing code, context can be used for cancellation later
}
```

**Step 2: Verify build**

Run:
```bash
go build -o prompt-builder .
```
Expected: Builds successfully

**Step 3: Commit**

```bash
git add main.go
git commit -m "feat: add graceful signal handling"
```

---

## Task 16: Run All Tests

**Files:**
- All test files

**Step 1: Run complete test suite**

Run:
```bash
go test -v ./...
```
Expected: All tests pass

**Step 2: Run with race detector**

Run:
```bash
go test -race ./...
```
Expected: No race conditions detected

**Step 3: Commit any fixes if needed**

```bash
git add -A
git commit -m "fix: address any test issues"
```

---

## Task 17: Integration Test Setup

**Files:**
- Create: `integration_test.go`

**Step 1: Write integration test**

```go
// integration_test.go
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIntegration_ConfigLoading(t *testing.T) {
	// Create temp directory with config
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	promptPath := filepath.Join(dir, "prompt.md")

	configContent := `model: llama3.2
system_prompt_file: ` + promptPath + `
ollama_host: http://localhost:11434
`
	promptContent := `# Test System Prompt
You are a test assistant.
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(promptPath, []byte(promptContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Load config
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify config values
	if cfg.Model != "llama3.2" {
		t.Errorf("Model = %q, want %q", cfg.Model, "llama3.2")
	}

	// Load system prompt
	prompt, err := os.ReadFile(cfg.SystemPromptFile)
	if err != nil {
		t.Fatalf("failed to load system prompt: %v", err)
	}

	if len(prompt) == 0 {
		t.Error("system prompt is empty")
	}
}
```

**Step 2: Run integration test**

Run:
```bash
go test -v -run TestIntegration
```
Expected: PASS

**Step 3: Commit**

```bash
git add integration_test.go
git commit -m "test: add integration test for config loading"
```

---

## Task 18: Final Build and Verification

**Files:**
- All files

**Step 1: Clean build**

Run:
```bash
go clean
go build -o prompt-builder .
```
Expected: Clean build, binary created

**Step 2: Verify binary**

Run:
```bash
./prompt-builder --version
./prompt-builder --help
```
Expected: Version and help output displayed

**Step 3: Run all tests one more time**

Run:
```bash
go test -v ./...
```
Expected: All tests pass

**Step 4: Commit final state**

```bash
git add -A
git commit -m "chore: final build verification"
```

---

## Summary

After completing all tasks, you will have:

1. **Working CLI** with flag parsing (`--model`, `--config`, `--no-copy`, `--quiet`)
2. **Config system** loading from `~/.config/prompt-builder/config.yaml`
3. **Ollama integration** for LLM chat
4. **Conversation loop** with completion detection
5. **Clipboard integration** with auto-detection
6. **Proper error handling** with exit codes
7. **Signal handling** for graceful Ctrl+C
8. **Test coverage** for all modules

To test end-to-end, create the config files and run:
```bash
./prompt-builder "I want a custom diet plan"
```
