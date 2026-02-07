package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
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
	ExitLLMError = 2
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

// Deps holds injectable dependencies for the app.
type Deps struct {
	Client       LLMClient
	Stdin        io.Reader
	Stdout       io.Writer
	Stderr       io.Writer
	Clipboard    ClipboardWriter
	IsTTY        func() bool
	SystemPrompt string
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

func runWithDeps(ctx context.Context, cli *CLI, deps *Deps) error {
	_ = ctx // Context available for future cancellation support

	// Initialize conversation
	conv := NewConversation(deps.SystemPrompt)

	// Prepare user's idea
	userIdea := cli.Idea
	tty := deps.IsTTY()
	if !tty {
		// Pipe mode: ask for immediate generation
		userIdea = "Generate your best prompt without asking clarifying questions. User's idea: " + userIdea
	}
	conv.AddUserMessage(userIdea)

	// Conversation loop
	reader := bufio.NewReader(deps.Stdin)
	for {
		// Get response from LLM with streaming
		response, err := deps.Client.ChatStreamWithSpinner(conv.Messages, tty && !cli.Quiet, func(token string) error {
			if !cli.Quiet {
				fmt.Fprint(deps.Stdout, token)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("LLM request failed: %v", err)
		}
		if !cli.Quiet {
			fmt.Fprintln(deps.Stdout) // newline after streaming completes
		}

		conv.AddAssistantMessage(response)

		// Pipe mode: output result and exit (can't continue conversation)
		if !tty {
			if IsComplete(response) {
				if cli.Quiet {
					// In quiet mode, print only the extracted code block
					finalPrompt := ExtractLastCodeBlock(response)
					fmt.Fprintln(deps.Stdout, finalPrompt)
				}
				// Non-quiet mode already streamed the response
				return nil
			}
			return fmt.Errorf("LLM requested clarification but stdin is not a TTY")
		}

		// Input loop: handle commands without calling LLM again
		for {
			fmt.Fprint(deps.Stdout, "> ")
			userInput, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read input: %v", err)
			}

			userInput = strings.TrimSpace(userInput)

			if IsCommand(userInput) {
				shouldExit, err := HandleCommandWithClipboard(userInput, response, deps.Clipboard, deps.Stdout)
				if err != nil {
					fmt.Fprintln(deps.Stderr, err)
				}
				if shouldExit {
					return nil
				}
				continue // Stay in input loop, don't call LLM
			}

			conv.AddUserMessage(userInput)
			break // Exit input loop, call LLM with new message
		}
	}
}

func run(ctx context.Context, cli *CLI) error {
	// Determine config path for client initialization
	configPath := cli.ConfigPath
	if configPath == "" {
		configPath = defaultConfigPath()
	}
	configPath = ExpandPath(configPath)

	cfg, err := LoadConfig(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("config file not found: %s\n\nCreate it with:\n  mkdir -p ~/.config/prompt-builder\n  cat > ~/.config/prompt-builder/config.yaml << 'EOF'\n  model: llama3.2\n  host: http://localhost:11434\n  system_prompt_file: ~/.config/prompt-builder/prompt-architect.md\n  EOF", configPath)
		}
		return fmt.Errorf("invalid config: %v", err)
	}

	// Apply CLI model override
	model := cfg.Model
	if cli.Model != "" {
		model = cli.Model
	}

	// Validate model
	if model == "" {
		return fmt.Errorf("no model specified\n\nSet 'model' in config or use --model flag")
	}

	// Load system prompt
	promptPath := ExpandPath(cfg.SystemPromptFile)
	systemPrompt, err := os.ReadFile(promptPath)
	if err != nil {
		return fmt.Errorf("system prompt not found: %s", promptPath)
	}

	// Create real dependencies
	deps := &Deps{
		Client:       NewChatClient(cfg.Host, model),
		Stdin:        os.Stdin,
		Stdout:       os.Stdout,
		Stderr:       os.Stderr,
		Clipboard:    NewClipboardWriter(DetectClipboardCmd(cfg.ClipboardCmd)),
		IsTTY:        isTTY,
		SystemPrompt: string(systemPrompt),
	}

	return runWithDeps(ctx, cli, deps)
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
		case strings.Contains(errStr, "LLM") || strings.Contains(errStr, "connect"):
			os.Exit(ExitLLMError)
		case strings.Contains(errStr, "no model"):
			os.Exit(ExitNoModel)
		default:
			os.Exit(1)
		}
	}
}
