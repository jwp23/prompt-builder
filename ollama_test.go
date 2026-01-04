// ollama_test.go
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestStreamCallback_Type(t *testing.T) {
	var callback StreamCallback = func(token string) error {
		return nil
	}

	if err := callback("test"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOllamaStreamChunk_Deserialization(t *testing.T) {
	tests := []struct {
		name        string
		json        string
		wantRole    string
		wantContent string
		wantDone    bool
	}{
		{
			name:        "partial chunk",
			json:        `{"message":{"role":"assistant","content":"Hello"},"done":false}`,
			wantRole:    "assistant",
			wantContent: "Hello",
			wantDone:    false,
		},
		{
			name:        "final chunk",
			json:        `{"message":{"role":"assistant","content":"!"},"done":true}`,
			wantRole:    "assistant",
			wantContent: "!",
			wantDone:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var chunk OllamaStreamChunk
			if err := json.Unmarshal([]byte(tt.json), &chunk); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			if chunk.Message.Role != tt.wantRole {
				t.Errorf("role = %q, want %q", chunk.Message.Role, tt.wantRole)
			}
			if chunk.Message.Content != tt.wantContent {
				t.Errorf("content = %q, want %q", chunk.Message.Content, tt.wantContent)
			}
			if chunk.Done != tt.wantDone {
				t.Errorf("done = %v, want %v", chunk.Done, tt.wantDone)
			}
		})
	}
}

func fakeStreamingServer(chunks []string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for i, chunk := range chunks {
			done := i == len(chunks)-1
			fmt.Fprintf(w, `{"message":{"role":"assistant","content":%q},"done":%v}`+"\n",
				chunk, done)
			w.(http.Flusher).Flush()
		}
	}))
}

func TestOllamaClient_ChatStream_HappyPath(t *testing.T) {
	server := fakeStreamingServer([]string{"Hello", " there", "!"})
	defer server.Close()

	client := NewOllamaClient(server.URL, "llama3.2")
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

func TestOllamaClient_ChatStream_CallbackError(t *testing.T) {
	server := fakeStreamingServer([]string{"Hello", " there", "!"})
	defer server.Close()

	client := NewOllamaClient(server.URL, "llama3.2")
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

func TestOllamaClient_ChatStream_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "not valid json")
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "llama3.2")
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

func TestOllamaClient_ChatStream_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "server error")
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "llama3.2")
	messages := []Message{{Role: "user", Content: "Hi"}}

	_, err := client.ChatStream(messages, func(token string) error {
		return nil
	})

	if err == nil {
		t.Error("expected error for HTTP 500")
	}
	if !strings.Contains(err.Error(), "Ollama request failed") {
		t.Errorf("unexpected error message: %v", err)
	}
}

