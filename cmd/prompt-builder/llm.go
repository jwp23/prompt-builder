// llm.go
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// LLMClient abstracts the LLM backend for testing.
type LLMClient interface {
	ChatStream(messages []Message, onToken StreamCallback) (string, error)
	ChatStreamWithSpinner(messages []Message, tty bool, onToken StreamCallback) (string, error)
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type ChatStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

type StreamCallback func(token string) error

type ChatClient struct {
	Host   string
	Model  string
	client *http.Client
}

func NewChatClient(host, model string) *ChatClient {
	return &ChatClient{
		Host:   host,
		Model:  model,
		client: &http.Client{},
	}
}

func (c *ChatClient) ChatStream(messages []Message, onToken StreamCallback) (string, error) {
	req := ChatRequest{
		Model:    c.Model,
		Messages: messages,
		Stream:   true,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.client.Post(c.Host+"/v1/chat/completions", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to connect to LLM server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("LLM request failed: %s - %s", resp.Status, string(body))
	}

	var accumulated strings.Builder
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines (SSE delimiter)
		if line == "" {
			continue
		}

		// Strip "data: " prefix
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		// Check for stream end sentinel
		if data == "[DONE]" {
			break
		}

		var chunk ChatStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return "", fmt.Errorf("failed to parse streaming chunk: %w", err)
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		content := chunk.Choices[0].Delta.Content
		if content != "" {
			if err := onToken(content); err != nil {
				return "", err
			}
			accumulated.WriteString(content)
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading stream: %w", err)
	}

	return accumulated.String(), nil
}

func (c *ChatClient) ChatStreamWithSpinner(messages []Message, tty bool, onToken StreamCallback) (string, error) {
	var spinner *Spinner
	var once sync.Once

	if tty {
		spinner = NewSpinnerWithTTY("Thinking...", tty)
		spinner.Start()
	}

	wrappedCallback := func(token string) error {
		once.Do(func() {
			if spinner != nil {
				spinner.Stop()
			}
		})
		return onToken(token)
	}

	return c.ChatStream(messages, wrappedCallback)
}

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

var spinnerFrames = []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}

type Spinner struct {
	frames   []rune
	interval time.Duration
	message  string
	tty      bool
	stopCh   chan struct{}
	doneCh   chan struct{}
}

func NewSpinner(message string) *Spinner {
	return NewSpinnerWithTTY(message, true)
}

func NewSpinnerWithTTY(message string, tty bool) *Spinner {
	return &Spinner{
		frames:   spinnerFrames,
		interval: 120 * time.Millisecond,
		message:  message,
		tty:      tty,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

func (s *Spinner) Stop() {
	select {
	case <-s.stopCh:
		// Already stopped
		return
	default:
		close(s.stopCh)
	}
}

func (s *Spinner) Start() {
	if !s.tty {
		return
	}
	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		defer close(s.doneCh)

		frame := 0
		for {
			select {
			case <-s.stopCh:
				s.clearLine()
				return
			case <-ticker.C:
				fmt.Printf("\r%c %s", s.frames[frame], s.message)
				frame = (frame + 1) % len(s.frames)
			}
		}
	}()
}

func (s *Spinner) clearLine() {
	// Clear the line: carriage return, spaces, carriage return
	clearLen := len(s.message) + 3 // frame + space + message
	fmt.Printf("\r%s\r", strings.Repeat(" ", clearLen))
}
