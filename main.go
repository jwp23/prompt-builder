package main

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

const (
	ExitSuccess     = 0
	ExitConfigError = 1
	ExitOllamaError = 2
	ExitNoModel     = 3
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

func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "prompt-builder", "config.yaml")
}

func isTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func run(ctx context.Context, cli *CLI) error {
	_ = ctx // Context available for future cancellation support
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
	firstRequest := true
	for {
		// Wait for model on first request only
		if firstRequest {
			if err := WaitForModel(client, cli.Quiet, isTTY()); err != nil {
				return fmt.Errorf("failed to connect to Ollama: %v", err)
			}
			firstRequest = false
		}

		// Get response from LLM with streaming
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

		conv.AddAssistantMessage(response)

		// Pipe mode: output result and exit (can't continue conversation)
		if !isTTY() {
			if IsComplete(response) {
				if cli.Quiet {
					// In quiet mode, print only the extracted code block
					finalPrompt := ExtractLastCodeBlock(response)
					fmt.Println(finalPrompt)
				}
				// Non-quiet mode already streamed the response
				return nil
			}
			return fmt.Errorf("LLM requested clarification but stdin is not a TTY")
		}

		// Interactive mode: response already streamed above

		fmt.Print("> ")
		userInput, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %v", err)
		}

		userInput = strings.TrimSpace(userInput)

		if IsCommand(userInput) {
			shouldExit, err := HandleCommand(userInput, response, clipboardCmd, os.Stdout)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
			if shouldExit {
				return nil
			}
			fmt.Print("> ")
			continue
		}

		conv.AddUserMessage(userInput)
	}
}

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
