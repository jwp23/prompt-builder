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
