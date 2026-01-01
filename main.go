package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
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
