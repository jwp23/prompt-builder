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
