// llm_test.go
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestChatRequest_Serialization(t *testing.T) {
	req := ChatRequest{
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

func TestStreamCallback_Type(t *testing.T) {
	var callback StreamCallback = func(token string) error {
		return nil
	}

	if err := callback("test"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestChatStreamChunk_Deserialization(t *testing.T) {
	tests := []struct {
		name         string
		json         string
		wantContent  string
		wantFinished bool
	}{
		{
			name:         "partial chunk",
			json:         `{"choices":[{"delta":{"content":"Hello"},"finish_reason":null}]}`,
			wantContent:  "Hello",
			wantFinished: false,
		},
		{
			name:         "final chunk",
			json:         `{"choices":[{"delta":{},"finish_reason":"stop"}]}`,
			wantContent:  "",
			wantFinished: true,
		},
		{
			name:         "empty delta",
			json:         `{"choices":[{"delta":{},"finish_reason":null}]}`,
			wantContent:  "",
			wantFinished: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var chunk ChatStreamChunk
			if err := json.Unmarshal([]byte(tt.json), &chunk); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			if len(chunk.Choices) == 0 {
				t.Fatal("expected at least one choice")
			}
			if chunk.Choices[0].Delta.Content != tt.wantContent {
				t.Errorf("content = %q, want %q", chunk.Choices[0].Delta.Content, tt.wantContent)
			}
			finished := chunk.Choices[0].FinishReason != nil && *chunk.Choices[0].FinishReason == "stop"
			if finished != tt.wantFinished {
				t.Errorf("finished = %v, want %v", finished, tt.wantFinished)
			}
		})
	}
}

func fakeStreamingServer(chunks []string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		for i, chunk := range chunks {
			isLast := i == len(chunks)-1
			if isLast {
				// Send the final content chunk
				fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":%q},\"finish_reason\":null}]}\n\n", chunk)
				// Send the stop chunk
				fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
				// Send the done sentinel
				fmt.Fprintf(w, "data: [DONE]\n\n")
			} else {
				fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":%q},\"finish_reason\":null}]}\n\n", chunk)
			}
			w.(http.Flusher).Flush()
		}
	}))
}

func TestChatClient_ChatStream_HappyPath(t *testing.T) {
	server := fakeStreamingServer([]string{"Hello", " there", "!"})
	defer server.Close()

	client := NewChatClient(server.URL, "llama3.2")
	messages := []Message{
		{Role: "user", Content: "Hi"},
	}

	var tokens []string
	response, err := client.ChatStream(messages, func(token string) error {
		tokens = append(tokens, token)
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify callback received tokens in order
	expectedTokens := []string{"Hello", " there", "!"}
	if len(tokens) != len(expectedTokens) {
		t.Errorf("got %d tokens, want %d", len(tokens), len(expectedTokens))
	}
	for i, tok := range tokens {
		if tok != expectedTokens[i] {
			t.Errorf("token[%d] = %q, want %q", i, tok, expectedTokens[i])
		}
	}

	// Verify accumulated response
	if response != "Hello there!" {
		t.Errorf("response = %q, want %q", response, "Hello there!")
	}
}

func TestChatClient_ChatStream_CallbackError(t *testing.T) {
	server := fakeStreamingServer([]string{"Hello", " there", "!"})
	defer server.Close()

	client := NewChatClient(server.URL, "llama3.2")
	messages := []Message{{Role: "user", Content: "Hi"}}

	callbackErr := fmt.Errorf("callback failed")
	callCount := 0
	_, err := client.ChatStream(messages, func(token string) error {
		callCount++
		if callCount == 2 {
			return callbackErr
		}
		return nil
	})

	if err != callbackErr {
		t.Errorf("expected callback error, got: %v", err)
	}
	if callCount != 2 {
		t.Errorf("callback called %d times, expected 2", callCount)
	}
}

func TestChatClient_ChatStream_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, "data: not valid json")
	}))
	defer server.Close()

	client := NewChatClient(server.URL, "llama3.2")
	messages := []Message{{Role: "user", Content: "Hi"}}

	_, err := client.ChatStream(messages, func(token string) error {
		return nil
	})

	if err == nil {
		t.Error("expected error for malformed JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse streaming chunk") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestChatClient_ChatStream_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "server error")
	}))
	defer server.Close()

	client := NewChatClient(server.URL, "llama3.2")
	messages := []Message{{Role: "user", Content: "Hi"}}

	_, err := client.ChatStream(messages, func(token string) error {
		return nil
	})

	if err == nil {
		t.Error("expected error for HTTP 500")
	}
	if !strings.Contains(err.Error(), "LLM request failed") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestChatClient_ChatStreamWithSpinner_StopsOnFirstToken(t *testing.T) {
	server := fakeStreamingServer([]string{"Hello", " there", "!"})
	defer server.Close()

	client := NewChatClient(server.URL, "llama3.2")
	messages := []Message{{Role: "user", Content: "Hi"}}

	var tokens []string
	response, err := client.ChatStreamWithSpinner(messages, false, func(token string) error {
		tokens = append(tokens, token)
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response != "Hello there!" {
		t.Errorf("response = %q, want %q", response, "Hello there!")
	}
	if len(tokens) != 3 {
		t.Errorf("got %d tokens, want 3", len(tokens))
	}
}

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

func TestNewSpinner(t *testing.T) {
	s := NewSpinner("Loading...")
	if s == nil {
		t.Fatal("NewSpinner returned nil")
	}
	if s.message != "Loading..." {
		t.Errorf("message = %q, want %q", s.message, "Loading...")
	}
}

func TestSpinner_StopWithoutStart(t *testing.T) {
	s := NewSpinner("Test")
	// Should not panic
	s.Stop()
}

func TestSpinner_StopMultipleTimes(t *testing.T) {
	s := NewSpinner("Test")
	// Should not panic on multiple Stop calls
	s.Stop()
	s.Stop()
	s.Stop()
}

func TestSpinner_StartStop(t *testing.T) {
	s := NewSpinner("Loading")
	s.Start()
	// Give it a moment to run
	time.Sleep(50 * time.Millisecond)
	s.Stop()
	// Should complete without hanging
}

func TestNewSpinnerWithTTY_False(t *testing.T) {
	s := NewSpinnerWithTTY("Loading", false)
	if s.tty {
		t.Error("expected tty to be false")
	}
}

func TestNewSpinnerWithTTY_True(t *testing.T) {
	s := NewSpinnerWithTTY("Loading", true)
	if !s.tty {
		t.Error("expected tty to be true")
	}
}

func TestSpinner_StartNonTTY(t *testing.T) {
	s := NewSpinnerWithTTY("Loading", false)
	s.Start() // Should be no-op, not start goroutine
	s.Stop()  // Should be safe
}
