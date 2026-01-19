package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWaitForModel_AlreadyLoaded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"models":[{"name":"llama3.2"}]}`)
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "llama3.2")
	err := WaitForModel(client, false, true)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
