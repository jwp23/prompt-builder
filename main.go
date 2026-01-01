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
