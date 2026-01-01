// ollama_test.go
package main

import (
	"encoding/json"
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
